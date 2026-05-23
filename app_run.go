package gaz

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

// Run executes the application lifecycle.
// It builds the container, starts services in order, and waits for a signal or stop call.
func (a *App) Run(ctx context.Context) error {
	if err := a.Build(); err != nil {
		return err
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("app is already running")
	}
	a.stopCh = make(chan struct{})
	a.running = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}()

	if err := a.startServices(ctx); err != nil {
		return err
	}

	return a.waitForShutdownSignal(ctx)
}

// startServices starts services layer by layer in parallel, and then starts
// the worker manager. On failure at any stage it rolls back by stopping
// already-started services.
//
// The lifecycle plan is already resolved during Build(), so plan data
// (startupOrder, shutdownOrder, services) is immutable here.
func (a *App) startServices(ctx context.Context) error {
	plan, err := a.lifecyclePlan()
	if err != nil {
		return err
	}

	return a.lifecycleExecutor(plan).Start(ctx, a.Stop)
}

// waitForShutdownSignal blocks until a shutdown trigger (signal, context cancel, or Stop call).
// Returns the result of graceful shutdown.
func (a *App) waitForShutdownSignal(ctx context.Context) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
		// Context cancelled, treat like SIGTERM (graceful, no double-signal)
		a.Logger.InfoContext(ctx, "Shutting down gracefully...", "reason", "context cancelled")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		return a.Stop(shutdownCtx)

	case sig := <-sigCh:
		return a.handleSignalShutdown(ctx, sig, sigCh)

	case <-a.stopCh:
		// Stopped externally (Stop() called)
		return nil
	}
}

// handleSignalShutdown handles graceful shutdown triggered by a signal.
// For SIGINT, it spawns a force-exit watcher that exits immediately on second SIGINT.
// For SIGTERM, it performs graceful shutdown without double-signal behavior.
//
// The watcher goroutine is synchronized so it exits before this function returns,
// ensuring signal.Stop(sigCh) in the caller does not race with the watcher.
func (a *App) handleSignalShutdown(
	ctx context.Context,
	sig os.Signal,
	sigCh <-chan os.Signal,
) error {
	if sig == os.Interrupt {
		a.Logger.InfoContext(ctx, "Shutting down gracefully...", "hint", "Ctrl+C again to force")
	} else {
		a.Logger.InfoContext(ctx, "Shutting down gracefully...", "signal", sig.String())
	}

	// Create shutdown context
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
	defer cancel()

	shutdownErr := make(chan error, 1)
	shutdownDone := make(chan struct{})

	// Start graceful shutdown in goroutine so we can continue listening for signals
	go func() {
		defer close(shutdownDone)
		shutdownErr <- a.Stop(shutdownCtx)
	}()

	// If SIGINT, spawn force-exit watcher goroutine.
	// watcherDone is closed when the watcher exits, so we can wait for it
	// before returning (ensuring signal.Stop runs after the watcher).
	watcherDone := make(chan struct{})
	if sig == os.Interrupt {
		go func() {
			defer close(watcherDone)
			select {
			case <-sigCh:
				// Second SIGINT received - force exit immediately
				a.Logger.ErrorContext(ctx, "Received second interrupt, forcing exit")
				callExitFunc(1)
			case <-shutdownDone:
				// Normal completion, watcher exits
			}
		}()
	} else {
		close(watcherDone) // No watcher for non-SIGINT signals
	}

	// Wait for shutdown to complete
	err := <-shutdownErr
	// Ensure watcher goroutine exits before returning, so that
	// defer signal.Stop(sigCh) in the caller runs after the watcher.
	<-watcherDone
	return err
}
