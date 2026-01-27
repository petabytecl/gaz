package health

import (
	"github.com/petabytecl/gaz"
)

// WithHealthChecks enables the health module with the provided configuration.
// It registers the configuration instance and the health module.
//
// Usage:
//
//	gaz.New(
//	    health.WithHealthChecks(health.DefaultConfig()),
//	)
func WithHealthChecks(config Config) gaz.Option {
	return func(app *gaz.App) {
		app.ProvideInstance(config)
		app.Module("health", Module)
	}
}
