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
)

// AppOptions configuration for App.
type AppOptions struct {
	ShutdownTimeout time.Duration
}

// AppOption configures AppOptions.
type AppOption func(*AppOptions)

// WithShutdownTimeout sets the timeout for graceful shutdown.
func WithShutdownTimeout(d time.Duration) AppOption {
	return func(o *AppOptions) {
		o.ShutdownTimeout = d
	}
}

// App is the application runtime wrapper.
// It orchestrates dependency injection, lifecycle management, and signal handling.
type App struct {
	container *Container
	opts      AppOptions

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
}

// NewApp creates a new App with the given container and options.
func NewApp(c *Container, opts ...AppOption) *App {
	options := AppOptions{
		ShutdownTimeout: 30 * time.Second, // Default
	}
	for _, opt := range opts {
		opt(&options)
	}

	return &App{
		container: c,
		opts:      options,
	}
}

// Run executes the application lifecycle.
// It builds the container, starts services in order, and waits for a signal or stop call.
func (a *App) Run(ctx context.Context) error {
	if err := a.container.Build(); err != nil {
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
	graph := a.container.getGraph()
	services := make(map[string]serviceWrapper)
	a.container.mu.RLock()
	for k, v := range a.container.services {
		if w, ok := v.(serviceWrapper); ok {
			services[k] = w
		}
	}
	a.container.mu.RUnlock()

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		return err
	}

	// Start services layer by layer
	for _, layer := range startupOrder {
		var wg sync.WaitGroup
		errCh := make(chan error, len(layer))

		for _, name := range layer {

			svc := services[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := svc.start(ctx); err != nil {
					errCh <- fmt.Errorf("starting service %s: %w", name, err)
				}
			}()
		}
		wg.Wait()
		close(errCh)

		if err := <-errCh; err != nil {
			// Rollback: stop everything we started?
			// For simplicity, we call Stop() which attempts to stop everything.
			// Ideally we only stop what started, but Stop() is safe to call on everything.
			// Use background context for rollback as original ctx might be fine but we are failing.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
			defer cancel()
			_ = a.Stop(shutdownCtx)
			return err
		}
	}

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// Block until stopped
	select {
	case <-ctx.Done():
		// Context cancelled, initiate shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		return a.Stop(shutdownCtx)
	case <-sigCh:
		// Signal received, initiate shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()
		return a.Stop(shutdownCtx)
	case <-a.stopCh:
		// Stopped externally (Stop() called)
		return nil
	}
}

// Stop initiates graceful shutdown of the application.
func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	// We do NOT set running=false here. It happens in Run's defer.
	// We close stopCh to signal Run to exit, BUT we do work first.
	// Wait, if we do work first, Run is still waiting.
	// After work, we close stopCh, Run returns.
	a.mu.Unlock()

	// Compute shutdown order (reverse of startup)
	// We need to re-compute because we don't store it.
	graph := a.container.getGraph()
	services := make(map[string]serviceWrapper)
	a.container.mu.RLock()
	for k, v := range a.container.services {
		if w, ok := v.(serviceWrapper); ok {
			services[k] = w
		}
	}
	a.container.mu.RUnlock()

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		// Should not happen if Build passed, unless graph changed (impossible after Build)
		return err
	}
	shutdownOrder := ComputeShutdownOrder(startupOrder)

	var lastErr error

	// Stop services layer by layer
	for _, layer := range shutdownOrder {
		var wg sync.WaitGroup
		errCh := make(chan error, len(layer))

		for _, name := range layer {

			svc := services[name]
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := svc.stop(ctx); err != nil {
					errCh <- fmt.Errorf("stopping service %s: %w", name, err)
				}
			}()
		}
		wg.Wait()
		close(errCh)

		// Collect errors but continue shutdown
		for err := range errCh {
			if lastErr == nil {
				lastErr = err
			} else {
				lastErr = fmt.Errorf("%w; %w", lastErr, err)
			}
		}
	}

	// Signal Run to exit
	a.mu.Lock()
	select {
	case <-a.stopCh:
		// Already closed
	default:
		close(a.stopCh)
	}
	a.mu.Unlock()

	return lastErr
}
