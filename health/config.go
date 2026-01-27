package health

// Config holds configuration for the management server.
type Config struct {
	// Port is the TCP port the management server listens on.
	// Defaults to 9090 if not set.
	Port int

	// LivenessPath is the path for the liveness probe.
	// Defaults to "/live".
	LivenessPath string

	// ReadinessPath is the path for the readiness probe.
	// Defaults to "/ready".
	ReadinessPath string

	// StartupPath is the path for the startup probe.
	// Defaults to "/startup".
	StartupPath string
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() Config {
	return Config{
		Port:          9090,
		LivenessPath:  "/live",
		ReadinessPath: "/ready",
		StartupPath:   "/startup",
	}
}
