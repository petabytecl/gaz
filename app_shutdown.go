package gaz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
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

	plan, err := a.lifecyclePlan()
	if err != nil {
		close(done)
		// Should not happen if Build passed, unless graph changed (impossible after Build)
		return err
	}

	var errs []error
	if stopErr := a.lifecycleExecutor(plan).Stop(ctx); stopErr != nil {
		errs = append(errs, stopErr)
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
