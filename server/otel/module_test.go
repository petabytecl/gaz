package otel

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
)

func TestNewModule_Registers(t *testing.T) {
	app := gaz.New()

	// Register module (without endpoint, tracing disabled)
	module := NewModule()
	err := module.Apply(app)
	require.NoError(t, err)

	err = app.Build()
	require.NoError(t, err)

	c := app.Container()

	// Verify Config is registered
	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)
	assert.Empty(t, cfg.Endpoint, "endpoint should be empty by default")
	assert.Equal(t, "gaz", cfg.ServiceName)

	// TracerProvider should be resolvable (may be nil if disabled)
	tp, err := di.Resolve[*sdktrace.TracerProvider](c)
	require.NoError(t, err)
	assert.Nil(t, tp, "should be nil when endpoint not configured")
}

func TestNewModule_EnvFallback(t *testing.T) {
	// Set environment variable
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	app := gaz.New()

	// NewModule without explicit endpoint should read from env
	module := NewModule()
	err := module.Apply(app)
	require.NoError(t, err)

	err = app.Build()
	require.NoError(t, err)

	c := app.Container()

	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)
	assert.Equal(t, "localhost:4317", cfg.Endpoint, "should use env var as fallback")
}

func TestNewModule_TracerStopper(t *testing.T) {
	app := gaz.New()

	// Register module (disabled - no endpoint)
	module := NewModule()
	err := module.Apply(app)
	require.NoError(t, err)

	err = app.Build()
	require.NoError(t, err)

	c := app.Container()

	// Stopper should be resolvable (may be nil if tracing disabled)
	stopper, err := di.Resolve[*tracerProviderStopper](c)
	require.NoError(t, err)
	assert.Nil(t, stopper, "stopper should be nil when tracing disabled")
}

func TestTracerProviderStopper_OnStop(t *testing.T) {
	// Test the stopper with nil provider
	stopper := &tracerProviderStopper{tp: nil}

	err := stopper.OnStop(context.Background())
	assert.NoError(t, err, "stopping nil provider should succeed")
}

func TestNewModule_SlogDefaultFallback(t *testing.T) {
	// gaz.New() registers logger by default, so fallback logic is hard to test via integration.
	// But we can test that resolution succeeds.
	app := gaz.New()

	module := NewModule()
	err := module.Apply(app)
	require.NoError(t, err)

	err = app.Build()
	require.NoError(t, err)

	c := app.Container()

	// Resolving TracerProvider should succeed
	tp, err := di.Resolve[*sdktrace.TracerProvider](c)
	require.NoError(t, err)
	// Note: tp is nil when no endpoint configured (default behavior)
	assert.Nil(t, tp, "should be nil when no endpoint configured")
}

func TestNewModule_ModuleName(t *testing.T) {
	module := NewModule()

	// Module should have a name for debugging/logging
	assert.Equal(t, "otel", module.Name())
}

// Verify that the stopper implements proper cleanup.
func TestTracerProviderStopper_Interface(t *testing.T) {
	// Verify tracerProviderStopper has OnStop method (di.Stopper)
	var _ interface {
		OnStop(context.Context) error
	} = &tracerProviderStopper{}
}

func TestNewModule_TracerProvider_WhenEnabled(t *testing.T) {
	// Set env var to enable tracing
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	app := gaz.New()

	module := NewModule()
	err := module.Apply(app)
	require.NoError(t, err)

	err = app.Build()
	require.NoError(t, err)

	c := app.Container()

	// TracerProvider should be non-nil when endpoint configured
	tp, err := di.Resolve[*sdktrace.TracerProvider](c)
	require.NoError(t, err)
	// May be non-nil even if endpoint unreachable
	if tp != nil {
		// Clean up
		stopper, _ := di.Resolve[*tracerProviderStopper](c)
		if stopper != nil {
			_ = stopper.OnStop(context.Background())
		}
	}
}

func TestRegisterTracerProvider_MissingConfig(t *testing.T) {
	c := di.New()

	// Register logger but NOT Config
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register tracer provider - registration should succeed
	err = registerTracerProvider(c)
	require.NoError(t, err)

	// Resolving should fail due to missing Config
	_, err = di.Resolve[*sdktrace.TracerProvider](c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config")
}

func TestRegisterTracerStopper_MissingProvider(t *testing.T) {
	c := di.New()

	// Don't register TracerProvider
	// Register stopper - registration should succeed
	err := registerTracerStopper(c)
	require.NoError(t, err)

	// Resolving should fail due to missing TracerProvider
	_, err = di.Resolve[*tracerProviderStopper](c)
	require.Error(t, err)
}
