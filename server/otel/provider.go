package otel

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer initializes the OpenTelemetry TracerProvider.
//
// If cfg.Endpoint is empty, returns nil (OTEL disabled).
// If the exporter fails to connect, logs a warning and returns nil (graceful degradation).
//
// The function sets the global TracerProvider and TextMapPropagator.
func InitTracer(ctx context.Context, cfg Config, logger *slog.Logger) (*sdktrace.TracerProvider, error) {
	if cfg.Endpoint == "" {
		if logger != nil {
			logger.DebugContext(ctx, "OTEL tracing disabled, no endpoint configured")
		}
		return nil, nil
	}

	// Build exporter options.
	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}
	if cfg.Insecure {
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	}

	// Create exporter (with timeout to avoid blocking startup).
	exportCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(exportCtx, exporterOpts...)
	if err != nil {
		if logger != nil {
			logger.WarnContext(ctx, "failed to create OTLP exporter, tracing disabled",
				slog.Any("error", err),
				slog.String("endpoint", cfg.Endpoint),
			)
		}
		return nil, nil // Graceful degradation.
	}

	// Create resource with service name.
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		_ = exporter.Shutdown(ctx)
		return nil, fmt.Errorf("otel: create resource: %w", err)
	}

	// Configure sampler.
	// ParentBased: Respect incoming trace decisions.
	// TraceIDRatioBased: Sample root spans probabilistically.
	sampleRatio := cfg.SampleRatio
	if sampleRatio <= 0 {
		sampleRatio = 0.1 // Default 10%.
	} else if sampleRatio > 1 {
		sampleRatio = 1.0 // Cap at 100%.
	}
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))

	// Create TracerProvider.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global providers.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if logger != nil {
		logger.InfoContext(ctx, "OTEL tracing initialized",
			slog.String("endpoint", cfg.Endpoint),
			slog.String("service", cfg.ServiceName),
			slog.Float64("sample_ratio", sampleRatio),
		)
	}

	return tp, nil
}

// ShutdownTracer gracefully shuts down the TracerProvider.
// It flushes pending spans with a 5-second timeout.
// Returns nil if tp is nil.
func ShutdownTracer(ctx context.Context, tp *sdktrace.TracerProvider) error {
	if tp == nil {
		return nil
	}

	// Use a 5-second timeout for flush.
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tp.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("otel: shutdown tracer: %w", err)
	}

	return nil
}
