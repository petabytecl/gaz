package otel

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the OTEL module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	endpoint    string
	serviceName string
	sampleRatio float64
	insecure    bool
}

func defaultModuleConfig() *moduleConfig {
	cfg := DefaultConfig()
	return &moduleConfig{
		endpoint:    cfg.Endpoint,
		serviceName: cfg.ServiceName,
		sampleRatio: cfg.SampleRatio,
		insecure:    cfg.Insecure,
	}
}

// WithEndpoint sets the OTLP endpoint.
// Example: "localhost:4317".
func WithEndpoint(endpoint string) ModuleOption {
	return func(c *moduleConfig) {
		c.endpoint = endpoint
	}
}

// WithServiceName sets the service name for traces.
// Default is "gaz".
func WithServiceName(name string) ModuleOption {
	return func(c *moduleConfig) {
		c.serviceName = name
	}
}

// WithSampleRatio sets the sampling ratio for root spans (0.0-1.0).
// Default is 0.1 (10%).
func WithSampleRatio(ratio float64) ModuleOption {
	return func(c *moduleConfig) {
		c.sampleRatio = ratio
	}
}

// WithInsecure sets whether to use insecure connection.
// Default is true.
func WithInsecure(insecure bool) ModuleOption {
	return func(c *moduleConfig) {
		c.insecure = insecure
	}
}

// tracerProviderStopper wraps TracerProvider to implement di.Stopper.
type tracerProviderStopper struct {
	tp *sdktrace.TracerProvider
}

// OnStop shuts down the TracerProvider.
func (t *tracerProviderStopper) OnStop(ctx context.Context) error {
	return ShutdownTracer(ctx, t.tp)
}

// NewModule creates an OTEL module with the given options.
// Returns a di.Module that registers TracerProvider components.
//
// If no endpoint is configured (via options or OTEL_EXPORTER_OTLP_ENDPOINT env var),
// tracing is disabled and a nil TracerProvider is registered.
//
// Components registered:
//   - *sdktrace.TracerProvider (may be nil if disabled)
//
// Example:
//
//	app := gaz.New()
//	app.Use(otel.NewModule())                               // uses env var
//	app.Use(otel.NewModule(otel.WithEndpoint("localhost:4317"))) // explicit endpoint
func NewModule(opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return di.NewModuleFunc("otel", func(c *di.Container) error {
		return registerOTELComponents(c, cfg)
	})
}

// registerOTELComponents registers all OTEL components with the container.
func registerOTELComponents(c *di.Container, cfg *moduleConfig) error {
	// Check environment variable fallback.
	endpoint := cfg.endpoint
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	// Build OTEL config.
	otelCfg := Config{
		Endpoint:    endpoint,
		ServiceName: cfg.serviceName,
		SampleRatio: cfg.sampleRatio,
		Insecure:    cfg.insecure,
	}

	// Register Config.
	if err := di.For[Config](c).Instance(otelCfg); err != nil {
		return fmt.Errorf("register otel config: %w", err)
	}

	// Register TracerProvider.
	if err := registerTracerProvider(c); err != nil {
		return err
	}

	// Register stopper for TracerProvider lifecycle management.
	return registerTracerStopper(c)
}

// registerTracerProvider registers the TracerProvider with the container.
func registerTracerProvider(c *di.Container) error {
	if err := di.For[*sdktrace.TracerProvider](c).
		Eager().
		Provider(func(c *di.Container) (*sdktrace.TracerProvider, error) {
			cfg, err := di.Resolve[Config](c)
			if err != nil {
				return nil, fmt.Errorf("resolve otel config: %w", err)
			}

			logger := slog.Default()
			if resolved, resolveErr := di.Resolve[*slog.Logger](c); resolveErr == nil {
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
func registerTracerStopper(c *di.Container) error {
	if err := di.For[*tracerProviderStopper](c).
		Provider(func(c *di.Container) (*tracerProviderStopper, error) {
			tp, err := di.Resolve[*sdktrace.TracerProvider](c)
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
