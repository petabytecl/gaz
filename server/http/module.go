package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/internal/configuredmodule"
)

// NewModule creates an HTTP module.
// Returns a gaz.Module that registers HTTP server components.
//
// Components registered:
//   - http.Config (loaded from flags/config)
//   - *http.Server (eager, starts HTTP server)
//
// The server uses http.Handler resolved from the container if available.
// Otherwise, it defaults to http.NotFoundHandler().
//
// Example:
//
//	app := gaz.New()
//	app.Use(http.NewModule())
func NewModule() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule(ModuleName).
		Flags(defaultCfg.Flags).
		Provide(configuredmodule.ProvideValidated[Config, *Config](
			&defaultCfg,
			"http config",
			"http config validate",
		)).
		Provide(func(c *gaz.Container) error {
			// Register Server
			return gaz.For[*Server](c).
				Eager().
				Provider(func(c *gaz.Container) (*Server, error) {
					cfg, err := gaz.Resolve[Config](c)
					if err != nil {
						return nil, fmt.Errorf("resolve http config: %w", err)
					}

					// Resolve handler if available, otherwise use default
					var handler http.Handler
					if h, resolveErr := gaz.Resolve[http.Handler](c); resolveErr == nil {
						handler = h
					}

					// Try to resolve logger, use default if not available
					logger, err := gaz.Resolve[*slog.Logger](c)
					if err != nil {
						logger = slog.Default()
					}

					return NewServer(cfg, handler, logger), nil
				})
		}).
		Build()
}
