package cron

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps robfig/cron with DI-aware job execution and lifecycle management.
// It implements worker.Worker interface for automatic discovery and lifecycle integration.
//
// Key features:
//   - Wraps robfig/cron with slog logging adapter
//   - SkipIfStillRunning by default (CRN-08)
//   - Custom panic recovery for stack trace logging
//   - Graceful shutdown waits for running jobs (CRN-05)
//   - Health check support (CRN-09)
type Scheduler struct {
	cron     *cron.Cron
	logger   *slog.Logger
	resolver Resolver
	appCtx   context.Context

	mu      sync.Mutex
	jobs    []*diJobWrapper
	running bool
}

// NewScheduler creates a new Scheduler wrapping robfig/cron.
//
// The scheduler is configured with:
//   - slog logging adapter for structured logging
//   - SkipIfStillRunning to prevent overlapping job executions
//   - Custom panic recovery (not cron.Recover) for stack traces via slog
//
// Parameters:
//   - resolver: Container interface for resolving job instances
//   - appCtx: Application context (cancelled on shutdown)
//   - logger: Logger for structured logging
func NewScheduler(resolver Resolver, appCtx context.Context, logger *slog.Logger) *Scheduler {
	// Create slog adapter for cron logger
	adapter := NewSlogAdapter(logger)

	// Create cron instance with options
	// Note: We use custom panic recovery in diJobWrapper, not cron.Recover()
	// This gives us stack traces via slog
	c := cron.New(
		cron.WithLogger(adapter),
		cron.WithChain(cron.SkipIfStillRunning(adapter)),
	)

	return &Scheduler{
		cron:     c,
		logger:   logger.With("component", "cron.Scheduler"),
		resolver: resolver,
		appCtx:   appCtx,
		jobs:     make([]*diJobWrapper, 0),
	}
}

// Name implements worker.Worker interface.
// Returns the scheduler's identifier for logging and debugging.
func (s *Scheduler) Name() string {
	return "cron.Scheduler"
}

// OnStart implements worker.Worker interface.
// Starts the cron scheduler, beginning job execution.
//
// The context is stored for future job context propagation if needed.
// This method always returns nil as scheduler startup doesn't fail.
func (s *Scheduler) OnStart(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}
	s.running = true

	s.logger.InfoContext(ctx, "starting cron scheduler", slog.Int("jobs", len(s.jobs)))
	s.cron.Start()
	return nil
}

// OnStop implements worker.Worker interface.
// Stops the cron scheduler and waits for all running jobs to complete.
// This provides graceful shutdown per CRN-05.
//
// The context can be used for shutdown deadline enforcement (optional enhancement).
// This method always returns nil as scheduler stop doesn't fail.
func (s *Scheduler) OnStop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}
	s.running = false

	s.logger.InfoContext(ctx, "stopping cron scheduler, waiting for running jobs")

	// Stop() returns context that completes when running jobs finish
	cronCtx := s.cron.Stop()
	<-cronCtx.Done()

	s.logger.InfoContext(ctx, "cron scheduler stopped")
	return nil
}

// RegisterJob registers a job with the scheduler.
//
// Parameters:
//   - serviceName: Type name for container resolution (e.g., "*MyJob")
//   - jobName: Human-readable name for logging
//   - schedule: Cron schedule expression (empty string disables job)
//   - timeout: Job execution timeout (0 for no timeout)
//
// Returns error if schedule expression is invalid.
// Empty schedule is not an error - the job is simply not scheduled (soft disable).
func (s *Scheduler) RegisterJob(serviceName, jobName, schedule string, timeout time.Duration) error {
	// Empty schedule disables the job (per CONTEXT.md)
	if schedule == "" {
		s.logger.Info("job schedule disabled", slog.String("job", jobName))
		return nil
	}

	// Create DI-aware job wrapper
	wrapper := NewJobWrapper(
		s.resolver,
		serviceName,
		jobName,
		schedule,
		timeout,
		s.appCtx,
		s.logger,
	)

	// Register with robfig/cron
	// AddJob validates the schedule expression and returns error if invalid
	_, err := s.cron.AddJob(schedule, wrapper)
	if err != nil {
		return fmt.Errorf("invalid schedule for job %s: %w", jobName, err)
	}

	s.mu.Lock()
	s.jobs = append(s.jobs, wrapper)
	s.mu.Unlock()

	s.logger.Info("job registered",
		slog.String("job", jobName),
		slog.String("schedule", schedule),
	)

	return nil
}

// HealthCheck checks if the scheduler is running.
// Implements basic health check for CRN-09.
func (s *Scheduler) HealthCheck(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrNotRunning
	}

	return nil
}

// JobCount returns the number of registered jobs.
// Useful for testing and logging.
func (s *Scheduler) JobCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.jobs)
}

// IsRunning returns true if the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Jobs returns a copy of the registered job wrappers.
// Useful for health check introspection.
func (s *Scheduler) Jobs() []*diJobWrapper {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]*diJobWrapper, len(s.jobs))
	copy(result, s.jobs)
	return result
}
