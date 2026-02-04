package otel

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz"
)

// tracerProviderStopper wraps TracerProvider to implement di.Stopper.
type tracerProviderStopper struct {
	tp *sdktrace.TracerProvider
}

// OnStop shuts down the TracerProvider.
func (t *tracerProviderStopper) OnStop(ctx context.Context) error {
	return ShutdownTracer(ctx, t.tp)
}

// NewModule creates an OTEL module.
// Returns a gaz.Module that registers TracerProvider components.
//
// If no endpoint is configured (via flags or OTEL_EXPORTER_OTLP_ENDPOINT env var),
// tracing is disabled and a nil TracerProvider is registered.
//
// Components registered:
//   - otel.Config
//   - *sdktrace.TracerProvider (may be nil if disabled)
//
// Example:
//
//	app := gaz.New()
//	app.Use(otel.NewModule())
func NewModule() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule("otel").
		Flags(defaultCfg.Flags).
		Provide(func(c *gaz.Container) error {
			// Register Config provider
			return gaz.For[Config](c).Provider(func(c *gaz.Container) (Config, error) {
				var cfg Config
				cfg.SetDefaults()

				// Resolve ProviderValues to load config
				if pv, err := gaz.Resolve[*gaz.ProviderValues](c); err == nil {
					if unmarshalErr := pv.UnmarshalKey(defaultCfg.Namespace(), &cfg); unmarshalErr != nil {
						// ignore error
						_ = unmarshalErr
					}
				}

				// Check environment variable fallback
				if cfg.Endpoint == "" {
					cfg.Endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
				}

				if err := cfg.Validate(); err != nil {
					return Config{}, fmt.Errorf("otel config validate: %w", err)
				}

				return cfg, nil
			})
		}).
		Provide(registerTracerProvider).
		Provide(registerTracerStopper).
		Build()
}

// registerTracerProvider registers the TracerProvider with the container.
func registerTracerProvider(c *gaz.Container) error {
	if err := gaz.For[*sdktrace.TracerProvider](c).
		Eager().
		Provider(func(c *gaz.Container) (*sdktrace.TracerProvider, error) {
			cfg, err := gaz.Resolve[Config](c)
			if err != nil {
				return nil, fmt.Errorf("resolve otel config: %w", err)
			}

			logger := slog.Default()
			if resolved, resolveErr := gaz.Resolve[*slog.Logger](c); resolveErr == nil {
				logger = resolved
			}

			return InitTracer(context.Background(), cfg, logger)
		}); err != nil {
		return fmt.Errorf("register tracer provider: %w", err)
	}
	return nil
}

// registerTracerStopper registers the TracerProvider stopper.
// This ensures TracerProvider is shut down when the app stops.
func registerTracerStopper(c *gaz.Container) error {
	if err := gaz.For[*tracerProviderStopper](c).
		Provider(func(c *gaz.Container) (*tracerProviderStopper, error) {
			tp, err := gaz.Resolve[*sdktrace.TracerProvider](c)
			if err != nil {
				return nil, fmt.Errorf("resolve tracer provider: %w", err)
			}
			if tp == nil {
				return nil, nil //nolint:nilnil // No stopper needed if tracing disabled.
			}
			return &tracerProviderStopper{tp: tp}, nil
		}); err != nil {
		return fmt.Errorf("register tracer stopper: %w", err)
	}
	return nil
}
