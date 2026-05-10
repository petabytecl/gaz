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

	startOnce sync.Once
	stopOnce  sync.Once
	started   bool
	stopped   bool
}

// RequireStart starts the app or fails the test.
// It calls t.Helper() for proper test line reporting, creates a context with
// the configured timeout, and calls app.Start(ctx).
// If start fails, it calls t.Fatalf() to fail the test immediately.
//
// RequireStart is idempotent -- calling it multiple times is safe.
//
// RequireStart returns the App to support method chaining:
//
//	app.RequireStart().DoSomething()
func (a *App) RequireStart() *App {
	a.tb.Helper()

	a.startOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
		defer cancel()

		if err := a.app.Start(ctx); err != nil {
			a.tb.Fatalf("gaztest: app didn't start: %v", err)
		}

		a.started = true
	})

	return a
}

// RequireStop stops the app or fails the test.
// It calls t.Helper() for proper test line reporting, creates a context with
// the configured timeout, and calls app.Stop(ctx).
// If stop fails, it calls t.Fatalf() to fail the test immediately.
//
// RequireStop is idempotent -- calling it multiple times is safe.
// After the first successful stop, subsequent calls return immediately.
func (a *App) RequireStop() {
	a.tb.Helper()

	a.stopOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
		defer cancel()

		if err := a.app.Stop(ctx); err != nil {
			a.tb.Fatalf("gaztest: app didn't stop: %v", err)
		}

		a.stopped = true
	})
}

// cleanup is called by t.Cleanup() to ensure the app is stopped.
// It only stops the app if it was started and not already stopped.
// Unlike RequireStop, it logs errors instead of failing the test,
// since cleanup runs after the test function returns.
func (a *App) cleanup() {
	if !a.started {
		return // Never started, nothing to clean up
	}

	// Use stopOnce to ensure idempotency with RequireStop
	a.stopOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
		defer cancel()

		if err := a.app.Stop(ctx); err != nil {
			a.tb.Logf("gaztest cleanup: stop failed: %v", err)
		}

		a.stopped = true
	})
}

// Container returns the underlying DI container.
// This provides access to the container for resolving services in tests.
func (a *App) Container() *gaz.Container {
	return a.app.Container()
}
