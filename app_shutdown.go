package gaz

import (
	"context"
)

// Stop initiates graceful shutdown of the application.
// It executes OnStop hooks for all services in reverse dependency order.
// Safe to call even if Run() was not used (e.g., Cobra integration).
// Stop is idempotent - calling it multiple times returns the same result.
func (a *App) Stop(ctx context.Context) error {
	return a.lifecycleSession().Stop(ctx)
}
