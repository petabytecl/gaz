package cron

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
)

// Resolver defines the interface for resolving job instances from a container.
// This abstraction allows the cron package to resolve fresh job instances
// without directly depending on the di package.
type Resolver interface {
	// ResolveByName resolves a service by its registered type name.
	// The opts parameter is reserved for future use (pass nil).
	// Returns the resolved instance and any error encountered.
	ResolveByName(name string, opts []string) (any, error)
}

// diJobWrapper wraps a CronJob type to implement cronx's Job interface.
// It resolves a fresh job instance from the container for each execution,
// providing transient lifecycle semantics as specified in CONTEXT.md.
//
// The wrapper handles:
//   - Fresh instance resolution per execution (transient per run)
//   - Panic recovery with stack trace logging
//   - Context with optional timeout
//   - Structured logging of job execution
//   - Thread-safe status tracking for health checks
type diJobWrapper struct {
	resolver    Resolver
	serviceName string        // Type name for container resolution
	jobName     string        // Human-readable name for logging
	schedule    string        // Schedule expression (for reference)
	timeout     time.Duration // Job timeout duration
	appCtx      context.Context
	logger      *slog.Logger

	mu      sync.Mutex
	running bool
	lastRun time.Time
	lastErr error
}

// NewJobWrapper creates a new DI-aware job wrapper.
//
// Parameters:
//   - resolver: Container interface for resolving job instances
//   - serviceName: Type name for container resolution (e.g., "*MyJob")
//   - jobName: Human-readable name for logging
//   - schedule: Cron schedule expression (for reference/logging)
//   - timeout: Job execution timeout (0 for no timeout)
//   - appCtx: Parent context (cancelled on shutdown)
//   - logger: Logger for structured logging
func NewJobWrapper(
	resolver Resolver,
	serviceName string,
	jobName string,
	schedule string,
	timeout time.Duration,
	appCtx context.Context,
	logger *slog.Logger,
) *diJobWrapper {
	return &diJobWrapper{
		resolver:    resolver,
		serviceName: serviceName,
		jobName:     jobName,
		schedule:    schedule,
		timeout:     timeout,
		appCtx:      appCtx,
		logger:      logger.With("component", "cron", "job", jobName),
	}
}

// Run implements cronx.Job interface.
// This method is called by cronx scheduler on each scheduled execution.
func (w *diJobWrapper) Run() {
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.running = false
		w.lastRun = time.Now()
		w.mu.Unlock()
	}()

	w.runWithRecovery()
}

// runWithRecovery wraps executeJob with panic recovery.
// Following the pattern from worker/supervisor.go.
func (w *diJobWrapper) runWithRecovery() {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			w.logger.Error("job panicked",
				slog.Any("panic", r),
				slog.String("stack", string(stack)),
			)
			w.mu.Lock()
			w.lastErr = fmt.Errorf("panic: %v", r)
			w.mu.Unlock()
		}
	}()

	w.executeJob()
}

// executeJob resolves and runs the job.
func (w *diJobWrapper) executeJob() {
	// Resolve fresh instance from container (transient per execution)
	instance, err := w.resolver.ResolveByName(w.serviceName, nil)
	if err != nil {
		w.logger.Error("failed to resolve job",
			slog.String("error", err.Error()),
		)
		w.mu.Lock()
		w.lastErr = fmt.Errorf("resolve failed: %w", err)
		w.mu.Unlock()
		return
	}

	job, ok := instance.(CronJob)
	if !ok {
		w.logger.Error("resolved instance is not CronJob",
			slog.String("type", fmt.Sprintf("%T", instance)),
		)
		w.mu.Lock()
		w.lastErr = fmt.Errorf("type assertion failed: %T is not CronJob", instance)
		w.mu.Unlock()
		return
	}

	// Create context with timeout if specified
	ctx := w.appCtx
	if w.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(w.appCtx, w.timeout)
		defer cancel()
	}

	// Execute with logging
	start := time.Now()
	w.logger.Info("job started")

	err = job.Run(ctx)
	elapsed := time.Since(start)

	w.mu.Lock()
	w.lastErr = err
	w.mu.Unlock()

	if err != nil {
		w.logger.Error("job failed",
			slog.Duration("duration", elapsed),
			slog.String("error", err.Error()),
		)
	} else {
		w.logger.Info("job finished",
			slog.Duration("duration", elapsed),
		)
	}
}

// IsRunning returns true if the job is currently executing.
// Thread-safe for health check access.
func (w *diJobWrapper) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// LastRun returns the time of the last job execution.
// Thread-safe for health check access.
func (w *diJobWrapper) LastRun() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastRun
}

// LastError returns the error from the last job execution, if any.
// Thread-safe for health check access.
func (w *diJobWrapper) LastError() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastErr
}

// Name returns the job name for logging/debugging.
func (w *diJobWrapper) Name() string {
	return w.jobName
}

// Schedule returns the schedule expression for this job.
func (w *diJobWrapper) Schedule() string {
	return w.schedule
}
