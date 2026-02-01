package worker

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/petabytecl/gaz/backoff"
)

// Default backoff configuration for supervisor.
const (
	defaultMaxInterval         = 5 * time.Minute
	defaultMultiplier          = 2.0
	defaultRandomizationFactor = 0.5
)

// supervisor wraps a single worker with panic recovery, restart logic,
// and circuit breaker protection. It is created by the Manager for each
// registered worker (or pool instance).
type supervisor struct {
	worker  Worker
	opts    *WorkerOptions
	backoff *backoff.ExponentialBackOff
	logger  *slog.Logger

	// Circuit breaker state
	failures    int
	windowStart time.Time

	// Last error for dead letter reporting
	lastError error

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	wg     sync.WaitGroup

	// Callback for critical worker failure
	onCriticalFail func()
}

// newSupervisor creates a new supervisor for the given worker.
func newSupervisor(w Worker, opts *WorkerOptions, logger *slog.Logger, onCriticalFail func()) *supervisor {
	return &supervisor{
		worker: w,
		opts:   opts,
		backoff: backoff.NewExponentialBackOff(
			backoff.WithInitialInterval(1*time.Second),
			backoff.WithMaxInterval(defaultMaxInterval),
			backoff.WithMultiplier(defaultMultiplier),
			backoff.WithRandomizationFactor(defaultRandomizationFactor),
		),
		logger:         logger.With(slog.String("worker", w.Name())),
		done:           make(chan struct{}),
		onCriticalFail: onCriticalFail,
	}
}

// start begins supervising the worker. It returns immediately.
// The supervision runs until the context is cancelled or the circuit breaker trips.
func (s *supervisor) start(ctx context.Context) {
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.windowStart = time.Now()

	s.wg.Add(1)
	go s.supervise()
}

// stop signals the supervisor to stop and waits for completion.
func (s *supervisor) stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// wait returns a channel that closes when the supervisor has stopped.
func (s *supervisor) wait() <-chan struct{} {
	return s.done
}

// supervise is the main supervision loop. It runs the worker with panic recovery,
// restarts on panic with exponential backoff, and trips the circuit breaker
// after too many failures.
func (s *supervisor) supervise() {
	defer s.wg.Done()
	defer close(s.done)

	for {
		// Check if context is cancelled before starting
		select {
		case <-s.ctx.Done():
			s.logger.Info("supervisor stopping", slog.String("reason", "context cancelled"))
			return
		default:
		}

		// Run worker with panic recovery
		startTime := time.Now()
		panicked := s.runWithRecovery()

		if !panicked {
			// Worker exited cleanly (Stop was called or it finished)
			s.logger.Info("worker stopped normally")
			return
		}

		// Worker panicked - check circuit breaker
		s.failures++

		// Reset circuit breaker window if it has expired
		if time.Since(s.windowStart) > s.opts.CircuitWindow {
			s.failures = 1
			s.windowStart = time.Now()
		}

		// Check if circuit breaker should trip
		if s.failures >= s.opts.MaxRestarts {
			s.logger.Error("circuit breaker tripped",
				slog.Int("failures", s.failures),
				slog.Duration("window", s.opts.CircuitWindow),
			)

			// Invoke dead letter handler if configured
			s.invokeDeadLetterHandler()

			if s.opts.Critical && s.onCriticalFail != nil {
				s.logger.Error("critical worker failed, triggering shutdown")
				s.onCriticalFail()
			}
			return
		}

		// Check if worker ran long enough to reset backoff (stable run)
		runDuration := time.Since(startTime)
		if runDuration >= s.opts.StableRunPeriod {
			s.logger.Info("worker ran stable period, resetting backoff",
				slog.Duration("ran", runDuration),
				slog.Duration("stable_period", s.opts.StableRunPeriod),
			)
			s.backoff.Reset()
		}

		// Calculate restart delay
		delay := s.backoff.NextBackOff()
		s.logger.Warn("worker will restart",
			slog.Int("failures", s.failures),
			slog.Int("max_restarts", s.opts.MaxRestarts),
			slog.Duration("delay", delay),
		)

		// Wait for delay or context cancellation
		select {
		case <-time.After(delay):
			// Continue to restart
		case <-s.ctx.Done():
			s.logger.Info("supervisor stopping during restart delay")
			return
		}
	}
}

// runWithRecovery runs the worker and recovers from any panic.
// Returns true if the worker panicked or failed to start, false if it exited normally.
func (s *supervisor) runWithRecovery() (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			s.logger.Error("worker panicked",
				slog.Any("panic", r),
				slog.String("stack", string(stack)),
			)
			s.lastError = fmt.Errorf("panic: %v", r)
			panicked = true
		}
	}()

	s.logger.Info("worker OnStart")
	if err := s.worker.OnStart(s.ctx); err != nil {
		s.logger.Error("worker failed to start", slog.Any("error", err))
		s.lastError = err
		// Treat start failure as a panic-equivalent (triggers restart logic)
		panicked = true
		return panicked
	}

	// Wait for context cancellation (shutdown signal)
	<-s.ctx.Done()

	s.logger.Info("worker OnStop")
	if err := s.worker.OnStop(s.ctx); err != nil {
		s.logger.Warn("worker stop error", slog.Any("error", err))
		// Continue with shutdown even on error (stop errors are non-fatal)
	}

	return false
}

// invokeDeadLetterHandler calls the dead letter handler with panic recovery.
// This ensures a buggy handler doesn't crash the supervisor.
func (s *supervisor) invokeDeadLetterHandler() {
	if s.opts.OnDeadLetter == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("dead letter handler panicked",
				slog.Any("panic", r),
			)
		}
	}()

	info := DeadLetterInfo{
		WorkerName:    s.worker.Name(),
		FinalError:    s.lastError,
		PanicCount:    s.failures,
		CircuitWindow: s.opts.CircuitWindow,
		Timestamp:     time.Now(),
	}
	s.opts.OnDeadLetter(info)
}
