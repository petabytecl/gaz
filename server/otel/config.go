package otel

const (
	// DefaultSampleRatio is the default sampling ratio for root spans (10%).
	DefaultSampleRatio = 0.1
)

// Config holds OpenTelemetry configuration.
type Config struct {
	// Endpoint is the OTLP endpoint (e.g., "localhost:4317").
	// If empty, tracing is disabled.
	Endpoint string

	// ServiceName is the service name for traces.
	// Default: "gaz".
	ServiceName string

	// SampleRatio is the sampling ratio for root spans (0.0-1.0).
	// Only applies to spans without incoming trace context.
	// Default: 0.1 (10%).
	SampleRatio float64

	// Insecure uses insecure connection to the collector.
	// Default: true for development.
	Insecure bool
}

// DefaultConfig returns the default OTEL configuration.
func DefaultConfig() Config {
	return Config{
		Endpoint:    "",                 // Disabled by default.
		ServiceName: "gaz",              // Default service name.
		SampleRatio: DefaultSampleRatio, // Sample 10% of root spans.
		Insecure:    true,               // Insecure for dev.
	}
}
