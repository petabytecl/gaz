package otel

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/internal/configuredmodule"
)

// tracerRuntime owns the active or disabled OTEL runtime state.
type tracerRuntime struct {
	tp *sdktrace.TracerProvider
}

func newTracerRuntime(ctx context.Context, cfg Config, logger *slog.Logger) (*tracerRuntime, error) {
	tp, err := InitTracer(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	return &tracerRuntime{tp: tp}, nil
}

// TracerProvider returns the active TracerProvider, or nil when tracing is disabled.
func (r *tracerRuntime) TracerProvider() *sdktrace.TracerProvider {
	if r == nil {
		return nil
	}
	return r.tp
}

// OnStop shuts down the active TracerProvider. Disabled tracing is a no-op.
func (r *tracerRuntime) OnStop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	return ShutdownTracer(ctx, r.tp)
}

// NewModule creates an OTEL module.
// Returns a gaz.Module that registers TracerProvider components.
//
// If no endpoint is configured (via flags or OTEL_EXPORTER_OTLP_ENDPOINT env
// var), tracing is disabled and resolving *sdktrace.TracerProvider returns nil.
//
// Components registered:
//   - otel.Config
//   - *sdktrace.TracerProvider (may be nil if disabled)
//   - internal OTEL runtime participant for graceful shutdown
//
// Example:
//
//	app := gaz.New()
//	app.Use(otel.NewModule())
func NewModule() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule(ModuleName).
		Flags(defaultCfg.Flags).
		Provide(provideConfig(&defaultCfg)).
		Provide(registerTracerRuntime).
		Provide(registerTracerProvider).
		Build()
}

func provideConfig(defaultCfg *Config) func(*gaz.Container) error {
	return configuredmodule.ProvideDefaultedWithHook[Config, *Config](
		defaultCfg,
		"otel config",
		"otel config validate",
		func(cfg *Config) error {
			if cfg.Endpoint == "" {
				cfg.Endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			}
			return nil
		},
	)
}

// registerTracerRuntime registers the module that owns active and disabled OTEL state.
func registerTracerRuntime(c *gaz.Container) error {
	if err := gaz.For[*tracerRuntime](c).
		Eager().
		Provider(func(c *gaz.Container) (*tracerRuntime, error) {
			cfg, err := gaz.Resolve[Config](c)
			if err != nil {
				return nil, fmt.Errorf("resolve otel config: %w", err)
			}

			return newTracerRuntime(context.Background(), cfg, resolveLogger(c))
		}); err != nil {
		return fmt.Errorf("register tracer runtime: %w", err)
	}
	return nil
}

// registerTracerProvider registers the TracerProvider with the container.
func registerTracerProvider(c *gaz.Container) error {
	if err := gaz.For[*sdktrace.TracerProvider](c).
		Provider(func(c *gaz.Container) (*sdktrace.TracerProvider, error) {
			runtime, err := gaz.Resolve[*tracerRuntime](c)
			if err != nil {
				return nil, fmt.Errorf("resolve tracer runtime: %w", err)
			}
			return runtime.TracerProvider(), nil
		}); err != nil {
		return fmt.Errorf("register tracer provider: %w", err)
	}
	return nil
}

func resolveLogger(c *gaz.Container) *slog.Logger {
	if resolved, err := gaz.Resolve[*slog.Logger](c); err == nil {
		return resolved
	}
	return slog.Default()
}
