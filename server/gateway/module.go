package gateway

import (
	"fmt"
	"log/slog"
	"net/http"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz"
	serverhttp "github.com/petabytecl/gaz/server/http"
)

// NewModule creates a Gateway module.
// Returns a gaz.Module that registers Gateway components.
//
// Components registered:
//   - gateway.Config (loaded from flags/config)
//   - *gateway.Gateway (eager, initializes on app start)
//   - http.Handler (via Gateway.Handler())
//   - Includes server/http module to provide the HTTP server
//
// Example:
//
//	app := gaz.New()
//	app.Use(gateway.NewModule())
func NewModule() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule("gateway").
		Flags(defaultCfg.Flags).
		Use(serverhttp.NewModule()). // Use http module to provide the server
		Provide(func(c *gaz.Container) error {
			// Register Config provider
			return gaz.For[Config](c).Provider(func(c *gaz.Container) (Config, error) {
				// Initialize with defaults including CORS defaults based on dev mode
				// Note: We can't fully know dev mode from flags yet, so we start with default
				cfg := DefaultConfig()

				// Resolve ProviderValues to load config
				if pv, err := gaz.Resolve[*gaz.ProviderValues](c); err == nil {
					if err := pv.UnmarshalKey(defaultCfg.Namespace(), &cfg); err != nil {
						// ignore error, use defaults
					}
				}

				// Re-apply CORS defaults if not set, respecting the loaded DevMode
				if len(cfg.CORS.AllowedOrigins) == 0 {
					cfg.CORS = DefaultCORSConfig(cfg.DevMode)
				}

				if err := cfg.Validate(); err != nil {
					return Config{}, fmt.Errorf("gateway config validate: %w", err)
				}

				return cfg, nil
			})
		}).
		Provide(func(c *gaz.Container) error {
			// Register Gateway
			return gaz.For[*Gateway](c).
				Eager().
				Provider(func(c *gaz.Container) (*Gateway, error) {
					cfg, err := gaz.Resolve[Config](c)
					if err != nil {
						return nil, fmt.Errorf("resolve gateway config: %w", err)
					}

					logger := slog.Default()
					if resolved, resolveErr := gaz.Resolve[*slog.Logger](c); resolveErr == nil {
						logger = resolved
					}

					// Try to resolve TracerProvider (optional).
					var tp *sdktrace.TracerProvider
					if resolved, resolveErr := gaz.Resolve[*sdktrace.TracerProvider](c); resolveErr == nil {
						tp = resolved
					}

					return NewGateway(cfg, logger, c, cfg.DevMode, tp), nil
				})
		}).
		Provide(func(c *gaz.Container) error {
			// Register Gateway as http.Handler so http.Server can use it
			return gaz.For[http.Handler](c).Provider(func(c *gaz.Container) (http.Handler, error) {
				gw, err := gaz.Resolve[*Gateway](c)
				if err != nil {
					return nil, fmt.Errorf("resolve gateway for handler: %w", err)
				}
				return gw.Handler(), nil
			})
		}).
		Build()
}
