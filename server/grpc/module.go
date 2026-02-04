package grpc

import (
	"fmt"
	"log/slog"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz"
)

// NewModule creates a gRPC module.
// Returns a gaz.Module that registers gRPC server components.
//
// Components registered:
//   - grpc.Config (loaded from flags/config)
//   - *grpc.Server (eager, starts on app start)
//
// Example:
//
//	app := gaz.New()
//	app.Use(grpc.NewModule())
func NewModule() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule("grpc").
		Flags(defaultCfg.Flags).
		Provide(func(c *gaz.Container) error {
			// Register Config provider
			return gaz.For[Config](c).Provider(func(c *gaz.Container) (Config, error) {
				// Start with the default configuration which has flags bound to it
				cfg := defaultCfg

				// Resolve ProviderValues to load config
				if pv, err := gaz.Resolve[*gaz.ProviderValues](c); err == nil {
					// We use the namespace from the default config to unmarshal
					if unmarshalErr := pv.UnmarshalKey(defaultCfg.Namespace(), &cfg); unmarshalErr != nil {
						// If key not found, we rely on defaults
						// Ideally we might want to log this but for now defaults are fine
						_ = unmarshalErr
					}
				}

				if err := cfg.Validate(); err != nil {
					return Config{}, fmt.Errorf("grpc config validate: %w", err)
				}

				return cfg, nil
			})
		}).
		Provide(func(c *gaz.Container) error {
			// Register Server
			return gaz.For[*Server](c).
				Eager().
				Provider(func(c *gaz.Container) (*Server, error) {
					cfg, err := gaz.Resolve[Config](c)
					if err != nil {
						return nil, fmt.Errorf("resolve grpc config: %w", err)
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

					return NewServer(cfg, logger, c, cfg.DevMode, tp), nil
				})
		}).
		Build()
}
