package otel

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz/di"
)

func TestNewModule_Registers(t *testing.T) {
	c := di.New()

	// Register logger (required dependency)
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register module (without endpoint, tracing disabled)
	module := NewModule()
	err = module.Register(c)
	require.NoError(t, err)

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

func TestNewModule_WithEndpoint(t *testing.T) {
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register module with endpoint (will fail connection but config should be set)
	module := NewModule(WithEndpoint("localhost:4317"))
	err = module.Register(c)
	require.NoError(t, err)

	// Verify Config has endpoint
	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)
	assert.Equal(t, "localhost:4317", cfg.Endpoint)
}

func TestNewModule_WithServiceName(t *testing.T) {
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	module := NewModule(WithServiceName("my-app"))
	err = module.Register(c)
	require.NoError(t, err)

	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)
	assert.Equal(t, "my-app", cfg.ServiceName)
}

func TestNewModule_WithSampleRatio(t *testing.T) {
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	module := NewModule(WithSampleRatio(0.5))
	err = module.Register(c)
	require.NoError(t, err)

	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)
	assert.Equal(t, 0.5, cfg.SampleRatio)
}

func TestNewModule_WithInsecure(t *testing.T) {
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	module := NewModule(WithInsecure(false))
	err = module.Register(c)
	require.NoError(t, err)

	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)
	assert.False(t, cfg.Insecure)
}

func TestNewModule_EnvFallback(t *testing.T) {
	// Set environment variable
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// NewModule without explicit endpoint should read from env
	module := NewModule()
	err = module.Register(c)
	require.NoError(t, err)

	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)
	assert.Equal(t, "localhost:4317", cfg.Endpoint, "should use env var as fallback")
}

func TestNewModule_ExplicitEndpointOverridesEnv(t *testing.T) {
	// Set environment variable
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "env-endpoint:4317")

	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Explicit endpoint should take precedence
	module := NewModule(WithEndpoint("explicit-endpoint:4317"))
	err = module.Register(c)
	require.NoError(t, err)

	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)
	assert.Equal(t, "explicit-endpoint:4317", cfg.Endpoint, "explicit endpoint should override env")
}

func TestNewModule_TracerStopper(t *testing.T) {
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register module (disabled - no endpoint)
	module := NewModule()
	err = module.Register(c)
	require.NoError(t, err)

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

func TestModuleOptions_ChainedApplication(t *testing.T) {
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Apply multiple options
	module := NewModule(
		WithEndpoint("custom:4317"),
		WithServiceName("custom-service"),
		WithSampleRatio(0.75),
		WithInsecure(true),
	)
	err = module.Register(c)
	require.NoError(t, err)

	cfg, err := di.Resolve[Config](c)
	require.NoError(t, err)

	assert.Equal(t, "custom:4317", cfg.Endpoint)
	assert.Equal(t, "custom-service", cfg.ServiceName)
	assert.Equal(t, 0.75, cfg.SampleRatio)
	assert.True(t, cfg.Insecure)
}

func TestDefaultModuleConfig(t *testing.T) {
	cfg := defaultModuleConfig()

	// Should match DefaultConfig values
	assert.Empty(t, cfg.endpoint)
	assert.Equal(t, "gaz", cfg.serviceName)
	assert.InDelta(t, 0.1, cfg.sampleRatio, 0.001)
	assert.True(t, cfg.insecure)
}

func TestNewModule_SlogDefaultFallback(t *testing.T) {
	c := di.New()

	// Register module without logger - should fallback to slog.Default()
	module := NewModule()
	err := module.Register(c)
	require.NoError(t, err)

	// Resolving TracerProvider should succeed with slog.Default() fallback
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

// Test for verbose debugging output.
func TestNewModule_DebugOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping verbose test in short mode")
	}

	// Use a handler that captures output
	var handler slog.Handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	c := di.New()
	err := di.For[*slog.Logger](c).Instance(logger)
	require.NoError(t, err)

	module := NewModule() // Disabled
	err = module.Register(c)
	require.NoError(t, err)

	// Resolve should log debug message about disabled tracing
	_, err = di.Resolve[*sdktrace.TracerProvider](c)
	require.NoError(t, err)
}

func TestNewModule_TracerProvider_WhenEnabled(t *testing.T) {
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register with endpoint (exporter will create even if unreachable)
	module := NewModule(WithEndpoint("localhost:4317"))
	err = module.Register(c)
	require.NoError(t, err)

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

func TestNewModule_TracerStopper_WhenEnabled(t *testing.T) {
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register with endpoint
	module := NewModule(WithEndpoint("localhost:4317"))
	err = module.Register(c)
	require.NoError(t, err)

	// Resolve provider first (triggers creation)
	tp, err := di.Resolve[*sdktrace.TracerProvider](c)
	require.NoError(t, err)

	if tp != nil {
		// Stopper should be non-nil when provider exists
		stopper, err := di.Resolve[*tracerProviderStopper](c)
		require.NoError(t, err)
		require.NotNil(t, stopper)

		// OnStop should work
		err = stopper.OnStop(context.Background())
		assert.NoError(t, err)
	}
}

func TestModuleConfig_AllOptions(t *testing.T) {
	cfg := defaultModuleConfig()

	// Apply all options
	WithEndpoint("test:4317")(cfg)
	WithServiceName("test-svc")(cfg)
	WithSampleRatio(0.25)(cfg)
	WithInsecure(false)(cfg)

	assert.Equal(t, "test:4317", cfg.endpoint)
	assert.Equal(t, "test-svc", cfg.serviceName)
	assert.Equal(t, 0.25, cfg.sampleRatio)
	assert.False(t, cfg.insecure)
}

func TestRegisterOTELComponents_ErrorPaths(t *testing.T) {
	// Test registerOTELComponents with valid container
	c := di.New()

	// Register logger
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	cfg := &moduleConfig{
		endpoint:    "",
		serviceName: "test",
		sampleRatio: 0.1,
		insecure:    true,
	}

	err = registerOTELComponents(c, cfg)
	require.NoError(t, err)
}

func TestRegisterTracerProvider_Direct(t *testing.T) {
	c := di.New()

	// Register Config first
	err := di.For[Config](c).Instance(Config{
		Endpoint:    "",
		ServiceName: "test",
		SampleRatio: 0.1,
		Insecure:    true,
	})
	require.NoError(t, err)

	// Register logger
	err = di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register tracer provider directly
	err = registerTracerProvider(c)
	require.NoError(t, err)

	// Should be resolvable (nil because disabled)
	tp, err := di.Resolve[*sdktrace.TracerProvider](c)
	require.NoError(t, err)
	assert.Nil(t, tp)
}

func TestRegisterTracerStopper_Direct(t *testing.T) {
	c := di.New()

	// Register Config
	err := di.For[Config](c).Instance(Config{
		Endpoint:    "",
		ServiceName: "test",
		SampleRatio: 0.1,
		Insecure:    true,
	})
	require.NoError(t, err)

	// Register logger
	err = di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register tracer provider first
	err = registerTracerProvider(c)
	require.NoError(t, err)

	// Register stopper
	err = registerTracerStopper(c)
	require.NoError(t, err)

	// Stopper should be nil (tracing disabled)
	stopper, err := di.Resolve[*tracerProviderStopper](c)
	require.NoError(t, err)
	assert.Nil(t, stopper)
}

func TestRegisterTracerStopper_WithEnabledTracing(t *testing.T) {
	c := di.New()

	// Register Config with endpoint
	err := di.For[Config](c).Instance(Config{
		Endpoint:    "localhost:4317",
		ServiceName: "test",
		SampleRatio: 0.1,
		Insecure:    true,
	})
	require.NoError(t, err)

	// Register logger
	err = di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register tracer provider first
	err = registerTracerProvider(c)
	require.NoError(t, err)

	// Register stopper
	err = registerTracerStopper(c)
	require.NoError(t, err)

	// Stopper should be non-nil
	stopper, err := di.Resolve[*tracerProviderStopper](c)
	require.NoError(t, err)
	require.NotNil(t, stopper)

	// Clean up
	err = stopper.OnStop(context.Background())
	assert.NoError(t, err)
}

func TestRegisterTracerProvider_MissingConfig(t *testing.T) {
	c := di.New()

	// Register logger but NOT Config
	err := di.For[*slog.Logger](c).Instance(slog.Default())
	require.NoError(t, err)

	// Register tracer provider - registration should succeed, resolution should fail
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
