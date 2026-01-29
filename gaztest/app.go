package gaztest

import (
	"context"
	"sync"
	"time"

	"github.com/petabytecl/gaz"
)

// App wraps *gaz.App with test-friendly methods.
// It provides RequireStart and RequireStop methods that fail the test on error,
// and automatic cleanup via t.Cleanup().
type App struct {
	app     *gaz.App
	tb      TB
	timeout time.Duration

	mu      sync.Mutex
	stopped bool
	started bool
}

// RequireStart starts the app or fails the test.
// It calls t.Helper() for proper test line reporting, creates a context with
// the configured timeout, and calls app.Start(ctx).
// If start fails, it calls t.Fatalf() to fail the test immediately.
//
// RequireStart returns the App to support method chaining:
//
//	app.RequireStart().DoSomething()
func (a *App) RequireStart() *App {
	a.tb.Helper()

	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return a // Already started, idempotent
	}
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	if err := a.app.Start(ctx); err != nil {
		a.tb.Fatalf("gaztest: app didn't start: %v", err)
	}

	a.mu.Lock()
	a.started = true
	a.mu.Unlock()

	return a
}

// RequireStop stops the app or fails the test.
// It calls t.Helper() for proper test line reporting, creates a context with
// the configured timeout, and calls app.Stop(ctx).
// If stop fails, it calls t.Fatalf() to fail the test immediately.
//
// RequireStop is idempotent - calling it multiple times is safe.
// After the first successful stop, subsequent calls return immediately.
func (a *App) RequireStop() {
	a.tb.Helper()

	a.mu.Lock()
	if a.stopped {
		a.mu.Unlock()
		return // Already stopped, idempotent
	}
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	if err := a.app.Stop(ctx); err != nil {
		a.tb.Fatalf("gaztest: app didn't stop: %v", err)
	}

	a.mu.Lock()
	a.stopped = true
	a.mu.Unlock()
}

// cleanup is called by t.Cleanup() to ensure the app is stopped.
// It only stops the app if it was started and not already stopped.
// Unlike RequireStop, it logs errors instead of failing the test,
// since cleanup runs after the test function returns.
func (a *App) cleanup() {
	a.mu.Lock()
	if a.stopped || !a.started {
		a.mu.Unlock()
		return // Nothing to clean up
	}
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	if err := a.app.Stop(ctx); err != nil {
		a.tb.Logf("gaztest cleanup: stop failed: %v", err)
	}

	a.mu.Lock()
	a.stopped = true
	a.mu.Unlock()
}

// Container returns the underlying DI container.
// This provides access to the container for resolving services in tests.
func (a *App) Container() *gaz.Container {
	return a.app.Container()
}
