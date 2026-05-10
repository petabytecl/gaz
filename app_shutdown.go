package gaz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/petabytecl/gaz/di"
)

// Stop initiates graceful shutdown of the application.
// It executes OnStop hooks for all services in reverse dependency order.
// Safe to call even if Run() was not used (e.g., Cobra integration).
// Stop is idempotent - calling it multiple times returns the same result.
func (a *App) Stop(ctx context.Context) error {
	a.stopOnce.Do(func() {
		a.stopErr = a.doStop(ctx)
	})
	return a.stopErr
}

// doStop performs the actual shutdown. Called only once via stopOnce.
func (a *App) doStop(ctx context.Context) error {
	a.mu.Lock()
	wasRunning := a.running
	wasBuilt := a.built
	a.mu.Unlock()

	// If app was never built, there's nothing to stop
	if !wasBuilt {
		return nil
	}

	// Cancel the cron scheduler context
	if a.cronCancel != nil {
		a.cronCancel()
	}

	// Start global timeout force-exit goroutine
	done := make(chan struct{})
	timer := time.NewTimer(a.opts.ShutdownTimeout)
	go func() {
		select {
		case <-done:
			timer.Stop()
			return
		case <-timer.C:
			// Re-check: shutdown may have finished between timer fire and this point
			select {
			case <-done:
				return // Clean exit won the race
			default:
				msg := fmt.Sprintf(
					"shutdown: global timeout %s exceeded, forcing exit",
					a.opts.ShutdownTimeout,
				)
				a.getLogger().Error(msg)
				_, _ = fmt.Fprintln(os.Stderr, msg)
				callExitFunc(1)
			}
		}
	}()

	// Use cached service set from startServices if available,
	// fallback to re-walking the container if Stop is called without Run/Start.
	a.mu.Lock()
	services := a.cachedNonWorkerServices
	a.mu.Unlock()
	if services == nil {
		services = a.collectNonWorkerServices()
	}

	// Compute shutdown order (reverse of startup)
	graph := a.container.GetGraph()

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		close(done)
		// Should not happen if Build passed, unless graph changed (impossible after Build)
		return err
	}
	shutdownOrder := ComputeShutdownOrder(startupOrder)

	var errs []error

	// Get logger safely (uses slog.Default() if nil)
	log := a.getLogger()

	// Stop workers first (they may depend on services)
	log.InfoContext(ctx, "stopping workers")
	if a.workerMgr != nil {
		if workerStopErr := a.workerMgr.Stop(ctx); workerStopErr != nil {
			errs = append(errs, fmt.Errorf("stopping workers: %w", workerStopErr))
		}
	}

	if serviceStopErr := a.stopServices(ctx, shutdownOrder, services); serviceStopErr != nil {
		errs = append(errs, serviceStopErr)
	}

	// Close logger file handle (if any) — after all services stopped, before exit
	if a.logCloser != nil {
		if closeErr := a.logCloser.Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("closing logger: %w", closeErr))
		}
	}

	// Cancel the force-exit goroutine
	close(done)

	// Signal Run to exit (only if Run() was used)
	if wasRunning {
		a.mu.Lock()
		select {
		case <-a.stopCh:
			// Already closed
		default:
			close(a.stopCh)
		}
		a.mu.Unlock()
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// stopServices stops services sequentially with per-hook timeout and blame logging.
func (a *App) stopServices(
	ctx context.Context,
	order [][]string,
	services map[string]di.ServiceWrapper,
) error {
	var errs []error

	// Stop services layer by layer, sequentially within each layer
	for _, layer := range order {
		for _, name := range layer {
			svc := services[name]

			// Create per-hook timeout context
			timeout := a.opts.PerHookTimeout
			hookCtx, cancel := context.WithTimeout(ctx, timeout)

			// Run hook in goroutine so we can detect timeout
			start := time.Now()
			errCh := make(chan error, 1)
			go func() {
				errCh <- svc.Stop(hookCtx)
			}()

			// Wait for hook completion or timeout
			select {
			case stopErr := <-errCh:
				cancel()
				elapsed := time.Since(start)
				if stopErr != nil {
					a.Logger.ErrorContext(
						ctx,
						"failed to stop service",
						"name", name,
						"error", stopErr,
						"elapsed", elapsed,
					)
					errs = append(errs, fmt.Errorf("stopping service %s: %w", name, stopErr))
				} else {
					a.Logger.InfoContext(
						ctx,
						"service stopped",
						"name", name,
						"duration", elapsed,
					)
				}
			case <-hookCtx.Done():
				cancel()
				elapsed := time.Since(start)
				// Blame logging: hook exceeded timeout
				a.logBlame(name, timeout, elapsed)
				errs = append(
					errs,
					fmt.Errorf("stopping service %s: %w", name, context.DeadlineExceeded),
				)
				// Continue to next hook (don't wait for the timed-out hook)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// logBlame logs blame information when a hook exceeds its timeout.
// Uses Logger first, falls back to stderr if Logger fails.
func (a *App) logBlame(hookName string, timeout, elapsed time.Duration) {
	msg := fmt.Sprintf("shutdown: %s exceeded %s timeout (elapsed: %s)", hookName, timeout, elapsed)

	// Try structured logger first
	if a.Logger != nil {
		a.Logger.Error(msg, "hook", hookName, "timeout", timeout, "elapsed", elapsed)
	}
	// Always write to stderr as fallback (guaranteed output even if logger is broken)
	_, _ = fmt.Fprintln(os.Stderr, msg)
}
