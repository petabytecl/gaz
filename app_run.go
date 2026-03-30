package gaz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/worker"
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

	// Compute startup order
	graph := a.container.GetGraph()
	services := make(map[string]di.ServiceWrapper)
	a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
		// Skip workers - they have their own lifecycle via WorkerManager
		// Workers implement OnStart/OnStop which looks like di.Starter/di.Stopper,
		// but they should only be started/stopped by WorkerManager, not the DI layer.
		if !svc.IsTransient() {
			if instance, err := a.container.ResolveByName(name, nil); err == nil {
				if _, isWorker := instance.(worker.Worker); isWorker {
					return
				}
			}
		}
		services[name] = svc
	})

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		return err
	}

	a.Logger.InfoContext(ctx, "starting application", "services_count", len(services))

	// Start services layer by layer
	for _, layer := range startupOrder {
		var wg sync.WaitGroup
		errCh := make(chan error, len(layer))

		for _, name := range layer {
			svc := services[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				start := time.Now()
				if startErr := svc.Start(ctx); startErr != nil {
					a.Logger.ErrorContext(
						ctx,
						"failed to start service",
						"name", name,
						"error", startErr,
					)
					errCh <- fmt.Errorf("starting service %s: %w", name, startErr)
				} else {
					a.Logger.InfoContext(
						ctx,
						"service started",
						"name", name,
						"duration", time.Since(start),
					)
				}
			}()
		}
		wg.Wait()
		close(errCh)

		if startupErr := <-errCh; startupErr != nil {
			// Rollback: stop everything we started.
			// Use background context for rollback as original ctx might be fine but we are failing.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
			defer cancel()
			stopErr := a.Stop(shutdownCtx)
			return errors.Join(startupErr, stopErr)
		}
	}

	// Start workers after all services started
	a.Logger.InfoContext(ctx, "starting workers")
	if workerErr := a.workerMgr.Start(ctx); workerErr != nil {
		// Rollback
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		stopErr := a.Stop(shutdownCtx)
		return errors.Join(fmt.Errorf("starting workers: %w", workerErr), stopErr)
	}

	return a.waitForShutdownSignal(ctx)
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
func (a *App) handleSignalShutdown(
	ctx context.Context,
	sig os.Signal,
	sigCh <-chan os.Signal,
) error {
	// Log hint message about force exit option
	a.Logger.InfoContext(ctx, "Shutting down gracefully...", "hint", "Ctrl+C again to force")

	// Create shutdown context
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
	defer cancel()

	// Channel to receive shutdown result
	shutdownDone := make(chan error, 1)

	// Start graceful shutdown in goroutine so we can continue listening for signals
	go func() {
		shutdownDone <- a.Stop(shutdownCtx)
	}()

	// If SIGINT, spawn force-exit watcher goroutine
	if sig == os.Interrupt {
		go func() {
			select {
			case <-sigCh:
				// Second SIGINT received - force exit immediately
				a.Logger.ErrorContext(ctx, "Received second interrupt, forcing exit")
				callExitFunc(1)
			case <-shutdownDone:
				// Normal completion, watcher exits
			}
		}()
	}

	// Wait for shutdown to complete
	return <-shutdownDone
}
