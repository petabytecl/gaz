// Package otel provides OpenTelemetry integration for the gaz framework.
//
// This package implements distributed tracing using OpenTelemetry with OTLP export.
// It provides a TracerProvider that can be used to instrument gRPC and HTTP servers
// for request tracing and observability.
//
// # Auto-Enable Behavior
//
// OpenTelemetry tracing is automatically enabled when an OTLP endpoint is configured.
// If no endpoint is set, the package gracefully degrades and returns a nil TracerProvider.
// This allows applications to run without a collector in development environments.
//
// # Configuration
//
// The package can be configured via:
//   - Module options (WithEndpoint, WithServiceName, WithSampleRatio)
//   - Environment variables (OTEL_EXPORTER_OTLP_ENDPOINT as fallback)
//
// # Usage
//
// Use NewModule to register the TracerProvider with the DI container:
//
//	app := gaz.New()
//	app.Use(otel.NewModule(
//	    otel.WithEndpoint("localhost:4317"),
//	    otel.WithServiceName("my-service"),
//	))
//
// The TracerProvider is automatically shut down when the application stops.
//
// # Instrumentation
//
// Once the TracerProvider is registered, other server packages (grpc, gateway)
// will automatically detect and use it for request tracing. Traces propagate
// across HTTP -> gRPC boundaries using W3C Trace Context headers.
//
// # Graceful Degradation
//
// If the OTLP collector is unreachable at startup, the package logs a warning
// and continues without tracing. This ensures applications start gracefully
// even when observability infrastructure is temporarily unavailable.
package otel
