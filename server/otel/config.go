package otel

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
)

const (
	// DefaultSampleRatio is the default sampling ratio for root spans (10%).
	DefaultSampleRatio = 0.1
)

// Config holds OpenTelemetry configuration.
type Config struct {
	// Endpoint is the OTLP endpoint (e.g., "localhost:4317").
	// If empty, tracing is disabled.
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`

	// ServiceName is the service name for traces.
	// Default: "gaz".
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`

	// SampleRatio is the sampling ratio for root spans (0.0-1.0).
	// Only applies to spans without incoming trace context.
	// Default: 0.1 (10%).
	SampleRatio float64 `json:"sample_ratio" yaml:"sample_ratio" mapstructure:"sample_ratio"`

	// Insecure uses insecure connection to the collector.
	// Default: true for development.
	Insecure bool `json:"insecure" yaml:"insecure" mapstructure:"insecure"`
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

// Namespace returns the config namespace.
func (c *Config) Namespace() string {
	return "otel"
}

// Flags registers the config flags.
func (c *Config) Flags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Endpoint, "otel-endpoint", c.Endpoint, "OTLP endpoint (e.g. localhost:4317)")
	fs.StringVar(&c.ServiceName, "otel-service-name", c.ServiceName, "Service name for traces")
	fs.Float64Var(&c.SampleRatio, "otel-sample-ratio", c.SampleRatio, "Sampling ratio for root spans (0.0-1.0)")
	fs.BoolVar(&c.Insecure, "otel-insecure", c.Insecure, "Use insecure connection to collector")
}

// SetDefaults applies default values to zero-value fields.
// Implements the config.Defaulter interface.
func (c *Config) SetDefaults() {
	if c.ServiceName == "" {
		c.ServiceName = "gaz"
	}
	if c.SampleRatio <= 0 {
		c.SampleRatio = DefaultSampleRatio
	}
	// Insecure defaults to false (Go zero value is correct).
	// Endpoint empty means disabled (intentional, no default).
}

// Validate checks that the configuration is valid.
// Implements the config.Validator interface.
func (c *Config) Validate() error {
	if c.SampleRatio < 0 || c.SampleRatio > 1.0 {
		return fmt.Errorf("otel: invalid sample_ratio %f: must be between 0.0 and 1.0", c.SampleRatio)
	}
	if c.Endpoint != "" && c.ServiceName == "" {
		return errors.New("otel: service_name required when endpoint is set")
	}
	return nil
}
