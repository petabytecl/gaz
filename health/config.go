// Package health provides health check mechanisms for the application,
// including liveness, readiness, and startup probes.
package health

import "time"

// DefaultPort is the default port for the management server.
const DefaultPort = 9090

// DefaultReadHeaderTimeout is the default timeout for reading headers.
const DefaultReadHeaderTimeout = 5 * time.Second

// Config holds configuration for the management server.
type Config struct {
	// Port is the TCP port the management server listens on.
	// Defaults to 9090 if not set.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// LivenessPath is the path for the liveness probe.
	// Defaults to "/live".
	LivenessPath string `json:"liveness_path" yaml:"liveness_path" mapstructure:"liveness_path"`

	// ReadinessPath is the path for the readiness probe.
	// Defaults to "/ready".
	ReadinessPath string `json:"readiness_path" yaml:"readiness_path" mapstructure:"readiness_path"`

	// StartupPath is the path for the startup probe.
	// Defaults to "/startup".
	StartupPath string `json:"startup_path" yaml:"startup_path" mapstructure:"startup_path"`
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() Config {
	return Config{
		Port:          DefaultPort,
		LivenessPath:  "/live",
		ReadinessPath: "/ready",
		StartupPath:   "/startup",
	}
}
