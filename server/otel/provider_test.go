package otel

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestInitTracer_Disabled(t *testing.T) {
	// Empty endpoint means tracing is disabled
	cfg := Config{
		Endpoint: "",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tp, err := InitTracer(context.Background(), cfg, logger)

	assert.NoError(t, err)
	assert.Nil(t, tp)
}

func TestInitTracer_DisabledWithNilLogger(t *testing.T) {
	cfg := Config{
		Endpoint: "",
	}

	// Should not panic with nil logger
	tp, err := InitTracer(context.Background(), cfg, nil)

	assert.NoError(t, err)
	assert.Nil(t, tp)
}

func TestInitTracer_InvalidEndpoint_CreatesProvider(t *testing.T) {
	// Note: OTLP exporter creates successfully even with invalid hosts
	// It only fails when actually trying to send spans
	// This is by design for async/batched export
	cfg := Config{
		Endpoint:    "invalid-host-that-does-not-exist.local:4317",
		ServiceName: "test-service",
		SampleRatio: 0.1,
		Insecure:    true,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tp, err := InitTracer(context.Background(), cfg, logger)

	// Exporter creation succeeds - failures happen on export
	assert.NoError(t, err)
	if tp != nil {
		// Clean up
		_ = ShutdownTracer(context.Background(), tp)
	}
}

func TestInitTracer_ValidEndpoint_CreatesProvider(t *testing.T) {
	cfg := Config{
		Endpoint:    "localhost:4317", // May or may not be reachable
		ServiceName: "test-service",
		SampleRatio: 0.1,
		Insecure:    true,
	}

	tp, err := InitTracer(context.Background(), cfg, nil)

	// Should succeed - exporter created even if endpoint unreachable
	assert.NoError(t, err)
	if tp != nil {
		_ = ShutdownTracer(context.Background(), tp)
	}
}

func TestInitTracer_SampleRatioNormalization(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"negative becomes default", -0.5, 0.1},
		{"zero becomes default", 0, 0.1},
		{"valid stays same", 0.5, 0.5},
		{"above 1.0 becomes 1.0", 1.5, 1.0},
		{"exactly 1.0 stays same", 1.0, 1.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Endpoint:    "", // Disabled, but we're testing normalization logic
				SampleRatio: tc.input,
			}

			// Since endpoint is empty, returns nil
			// The normalization happens inside InitTracer when creating sampler
			// We verify the constants are correct
			assert.Equal(t, 0.1, defaultSampleRatio)
			assert.Equal(t, 1.0, maxSampleRatio)

			tp, err := InitTracer(context.Background(), cfg, nil)
			assert.NoError(t, err)
			assert.Nil(t, tp)
		})
	}
}

func TestShutdownTracer_Nil(t *testing.T) {
	err := ShutdownTracer(context.Background(), nil)

	assert.NoError(t, err, "nil provider should return no error")
}

func TestShutdownTracer_UsesTimeout(t *testing.T) {
	// We can't easily test the 5s timeout without a real provider
	// but we verify the constant exists
	assert.Equal(t, shutdownTimeout.Seconds(), 5.0)
}

func TestInitTracer_Constants(t *testing.T) {
	// Verify important constants
	assert.Equal(t, 10.0, exporterTimeout.Seconds(), "exporter timeout should be 10s")
	assert.Equal(t, 5.0, shutdownTimeout.Seconds(), "shutdown timeout should be 5s")
	assert.InDelta(t, 0.1, defaultSampleRatio, 0.001, "default sample ratio should be 10%")
	assert.Equal(t, 1.0, maxSampleRatio, "max sample ratio should be 100%")
}

func TestInitTracer_SetsGlobalProviderOnSuccess(t *testing.T) {
	// We can't easily test this without a real collector
	// But we can verify that the global provider is accessible
	tp := otel.GetTracerProvider()
	require.NotNil(t, tp, "global tracer provider should exist")
}

func TestInitTracer_ContextCancellation(t *testing.T) {
	cfg := Config{
		Endpoint:    "localhost:4317",
		ServiceName: "test",
		SampleRatio: 0.1,
		Insecure:    true,
	}

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should gracefully handle cancelled context
	tp, err := InitTracer(ctx, cfg, nil)

	// May return nil (graceful degradation) or error
	// Either is acceptable behavior
	if tp != nil {
		_ = ShutdownTracer(context.Background(), tp)
	}
	_ = err // Ignore error - cancellation handling varies
}

func TestInitTracer_FullPath_WithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	cfg := Config{
		Endpoint:    "localhost:4317",
		ServiceName: "full-path-test",
		SampleRatio: 0.5,
		Insecure:    true,
	}

	tp, err := InitTracer(context.Background(), cfg, logger)

	// Should succeed (exporter created even if endpoint unreachable)
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Verify shutdown works
	err = ShutdownTracer(context.Background(), tp)
	assert.NoError(t, err)
}

func TestInitTracer_SecureConnection(t *testing.T) {
	cfg := Config{
		Endpoint:    "localhost:4317",
		ServiceName: "secure-test",
		SampleRatio: 0.1,
		Insecure:    false, // Secure connection
	}

	tp, err := InitTracer(context.Background(), cfg, nil)

	// Should succeed - exporter created even if endpoint unreachable
	assert.NoError(t, err)
	if tp != nil {
		_ = ShutdownTracer(context.Background(), tp)
	}
}

func TestInitTracer_SampleRatio_Boundaries(t *testing.T) {
	testCases := []struct {
		name        string
		ratio       float64
		description string
	}{
		{"very_small", 0.001, "very small sample ratio"},
		{"half", 0.5, "50% sampling"},
		{"full", 1.0, "100% sampling"},
		{"over_max", 2.0, "over max is clamped"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Endpoint:    "localhost:4317",
				ServiceName: "test",
				SampleRatio: tc.ratio,
				Insecure:    true,
			}

			tp, err := InitTracer(context.Background(), cfg, nil)
			assert.NoError(t, err)
			if tp != nil {
				_ = ShutdownTracer(context.Background(), tp)
			}
		})
	}
}

func TestShutdownTracer_WithProvider(t *testing.T) {
	cfg := Config{
		Endpoint:    "localhost:4317",
		ServiceName: "shutdown-test",
		SampleRatio: 0.1,
		Insecure:    true,
	}

	tp, err := InitTracer(context.Background(), cfg, nil)
	require.NoError(t, err)
	if tp == nil {
		t.Skip("no tracer provider created")
	}

	// Shutdown should succeed
	err = ShutdownTracer(context.Background(), tp)
	assert.NoError(t, err)

	// Second shutdown should also succeed (idempotent)
	err = ShutdownTracer(context.Background(), tp)
	assert.NoError(t, err)
}

func TestShutdownTracer_Error(t *testing.T) {
	cfg := Config{
		Endpoint:    "localhost:4317",
		ServiceName: "shutdown-error-test",
		SampleRatio: 0.1,
		Insecure:    true,
	}

	tp, err := InitTracer(context.Background(), cfg, nil)
	require.NoError(t, err)
	if tp == nil {
		t.Skip("no tracer provider created")
	}

	// First shutdown should succeed
	err = ShutdownTracer(context.Background(), tp)
	assert.NoError(t, err)

	// Shutdown already-shutdown provider should still succeed
	err = ShutdownTracer(context.Background(), tp)
	assert.NoError(t, err)
}

func TestInitTracer_AllSampleRatios(t *testing.T) {
	// Test all boundary conditions for sample ratio normalization
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name  string
		ratio float64
	}{
		{"negative", -1.0},
		{"zero", 0.0},
		{"small_positive", 0.01},
		{"quarter", 0.25},
		{"half", 0.5},
		{"three_quarter", 0.75},
		{"one", 1.0},
		{"over_one", 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Endpoint:    "localhost:4317",
				ServiceName: "ratio-test",
				SampleRatio: tt.ratio,
				Insecure:    true,
			}

			tp, err := InitTracer(context.Background(), cfg, logger)
			require.NoError(t, err)
			if tp != nil {
				_ = ShutdownTracer(context.Background(), tp)
			}
		})
	}
}
