package gaz

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/worker"
)

// lifecycleExecutor owns service and worker execution for a resolved
// lifecycle plan. App remains responsible for build state, signal handling,
// and one-shot shutdown orchestration.
type lifecycleExecutor struct {
	plan            *lifecyclePlan
	logger          *slog.Logger
	workerMgr       *worker.Manager
	shutdownTimeout time.Duration
	perHookTimeout  time.Duration
}

type lifecycleExecutorDeps struct {
	plan            *lifecyclePlan
	logger          *slog.Logger
	workerMgr       *worker.Manager
	shutdownTimeout time.Duration
	perHookTimeout  time.Duration
}

func newLifecycleExecutor(deps lifecycleExecutorDeps) *lifecycleExecutor {
	log := deps.logger
	if log == nil {
		log = slog.Default()
	}

	return &lifecycleExecutor{
		plan:            deps.plan,
		logger:          log,
		workerMgr:       deps.workerMgr,
		shutdownTimeout: deps.shutdownTimeout,
		perHookTimeout:  deps.perHookTimeout,
	}
}

func (a *App) lifecycleExecutor(plan *lifecyclePlan) *lifecycleExecutor {
	return newLifecycleExecutor(lifecycleExecutorDeps{
		plan:            plan,
		logger:          a.getLogger(),
		workerMgr:       a.workerMgr,
		shutdownTimeout: a.opts.ShutdownTimeout,
		perHookTimeout:  a.opts.PerHookTimeout,
	})
}

// Start starts services layer by layer in parallel, then starts workers.
// On failure at any stage it rolls back through the supplied stop function.
func (e *lifecycleExecutor) Start(
	ctx context.Context,
	rollback func(context.Context) error,
) error {
	e.logger.InfoContext(ctx, "starting application", "services_count", len(e.plan.services))

	for _, layer := range e.plan.startupOrder {
		if err := e.startLayer(ctx, layer); err != nil {
			return e.withRollback(err, rollback)
		}
	}

	e.logger.InfoContext(ctx, "starting workers")
	if e.workerMgr == nil {
		return nil
	}
	if err := e.workerMgr.Start(ctx); err != nil {
		return e.withRollback(fmt.Errorf("starting workers: %w", err), rollback)
	}

	return nil
}

func (e *lifecycleExecutor) startLayer(ctx context.Context, layer []string) error {
	errCh := make(chan error, len(layer))
	doneCh := make(chan struct{})

	for _, name := range layer {
		svc := e.plan.services[name]
		go func() {
			defer func() { doneCh <- struct{}{} }()
			start := time.Now()
			if err := svc.Start(ctx); err != nil {
				e.logger.ErrorContext(
					ctx,
					"failed to start service",
					"name", name,
					"error", err,
				)
				errCh <- fmt.Errorf("starting service %s: %w", name, err)
				return
			}
			e.logger.InfoContext(
				ctx,
				"service started",
				"name", name,
				"duration", time.Since(start),
			)
		}()
	}

	for range layer {
		<-doneCh
	}
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (e *lifecycleExecutor) withRollback(
	startupErr error,
	rollback func(context.Context) error,
) error {
	if rollback == nil {
		return startupErr
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), e.shutdownTimeout)
	defer cancel()
	return errors.Join(startupErr, rollback(shutdownCtx))
}

// Stop stops workers first, then lifecycle services in shutdown order.
func (e *lifecycleExecutor) Stop(ctx context.Context) error {
	var errs []error

	e.logger.InfoContext(ctx, "stopping workers")
	if e.workerMgr != nil {
		if err := e.workerMgr.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stopping workers: %w", err))
		}
	}

	if err := e.stopServices(ctx, e.plan.shutdownOrder, e.plan.services); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (e *lifecycleExecutor) stopServices(
	ctx context.Context,
	order [][]string,
	services map[string]di.ServiceWrapper,
) error {
	var errs []error

	for _, layer := range order {
		for _, name := range layer {
			if err := e.stopService(ctx, name, services[name]); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errors.Join(errs...)
}

func (e *lifecycleExecutor) stopService(
	ctx context.Context,
	name string,
	svc di.ServiceWrapper,
) error {
	timeout := e.perHookTimeout
	hookCtx, cancel := context.WithTimeout(ctx, timeout)

	start := time.Now()
	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.Stop(hookCtx)
	}()

	select {
	case err := <-errCh:
		cancel()
		elapsed := time.Since(start)
		if err != nil {
			e.logger.ErrorContext(
				ctx,
				"failed to stop service",
				"name", name,
				"error", err,
				"elapsed", elapsed,
			)
			return fmt.Errorf("stopping service %s: %w", name, err)
		}
		e.logger.InfoContext(
			ctx,
			"service stopped",
			"name", name,
			"duration", elapsed,
		)
		return nil
	case <-hookCtx.Done():
		cancel()
		elapsed := time.Since(start)
		e.logBlame(name, timeout, elapsed)
		return fmt.Errorf("stopping service %s: %w", name, context.DeadlineExceeded)
	}
}

func (e *lifecycleExecutor) logBlame(hookName string, timeout, elapsed time.Duration) {
	msg := fmt.Sprintf("shutdown: %s exceeded %s timeout (elapsed: %s)", hookName, timeout, elapsed)

	e.logger.Error(msg, "hook", hookName, "timeout", timeout, "elapsed", elapsed)
	_, _ = fmt.Fprintln(os.Stderr, msg)
}
