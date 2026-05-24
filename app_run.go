package gaz

import (
	"context"
	"os"
)

// Run executes the application lifecycle.
// It builds the container, starts services in order, and waits for a signal or stop call.
func (a *App) Run(ctx context.Context) error {
	return a.lifecycleSession().Run(ctx)
}

// Start initiates the application lifecycle.
// This is called automatically by WithCobra() or can be called manually.
func (a *App) Start(ctx context.Context) error {
	return a.lifecycleSession().Start(ctx)
}

// handleSignalShutdown is kept as a private compatibility wrapper for tests.
func (a *App) handleSignalShutdown(
	ctx context.Context,
	sig os.Signal,
	sigCh <-chan os.Signal,
) error {
	return a.lifecycleSession().handleSignalShutdown(ctx, sig, sigCh)
}
