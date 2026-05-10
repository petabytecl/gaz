package health

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

// Default configuration values.
const (
	// DefaultPort is the default port for the management server.
	DefaultPort = 9090

	// MaxPort is the maximum valid port number.
	MaxPort = 65535

	// DefaultReadHeaderTimeout is the default timeout for reading headers.
	DefaultReadHeaderTimeout = 5 * time.Second

	// DefaultReadTimeout is the default read timeout for the management server.
	DefaultReadTimeout = 10 * time.Second

	// DefaultWriteTimeout is the default write timeout for the management server.
	DefaultWriteTimeout = 10 * time.Second

	// DefaultIdleTimeout is the default idle timeout for the management server.
	DefaultIdleTimeout = 60 * time.Second

	// DefaultLivenessPath is the default path for the liveness probe.
	DefaultLivenessPath = "/live"

	// DefaultReadinessPath is the default path for the readiness probe.
	DefaultReadinessPath = "/ready"

	// DefaultStartupPath is the default path for the startup probe.
	DefaultStartupPath = "/startup"
)

// DefaultBindAddress is the default bind address for the management server.
// Binding to loopback prevents external access to health endpoints.
const DefaultBindAddress = "127.0.0.1"

// Config holds configuration for the management server.
type Config struct {
	// Port is the TCP port the management server listens on.
	// Defaults to 9090 if not set.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// BindAddress is the IP address the management server binds to.
	// Defaults to "127.0.0.1" (loopback only). Set to "0.0.0.0" to
	// allow external access (e.g., Kubernetes probes from kubelets).
	BindAddress string `json:"bind_address" yaml:"bind_address" mapstructure:"bind_address"`

	// ShowErrors controls whether error messages are included in health
	// check responses. Defaults to false to prevent information leakage.
	// Enable only in development or staging environments.
	ShowErrors bool `json:"show_errors" yaml:"show_errors" mapstructure:"show_errors"`

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
		BindAddress:   DefaultBindAddress,
		ShowErrors:    false,
		LivenessPath:  DefaultLivenessPath,
		ReadinessPath: DefaultReadinessPath,
		StartupPath:   DefaultStartupPath,
	}
}

// Namespace returns the config namespace.
func (c *Config) Namespace() string {
	return "health"
}

// Flags registers the config flags.
func (c *Config) Flags(fs *pflag.FlagSet) {
	fs.IntVar(&c.Port, "health-port", c.Port, "Health server port")
	fs.StringVar(&c.LivenessPath, "health-liveness-path", c.LivenessPath, "Liveness endpoint path")
	fs.StringVar(&c.ReadinessPath, "health-readiness-path", c.ReadinessPath, "Readiness endpoint path")
	fs.StringVar(&c.StartupPath, "health-startup-path", c.StartupPath, "Startup endpoint path")
}

// SetDefaults applies default values to zero-value fields.
// Implements the config.Defaulter interface.
func (c *Config) SetDefaults() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.BindAddress == "" {
		c.BindAddress = DefaultBindAddress
	}
	if c.LivenessPath == "" {
		c.LivenessPath = DefaultLivenessPath
	}
	if c.ReadinessPath == "" {
		c.ReadinessPath = DefaultReadinessPath
	}
	if c.StartupPath == "" {
		c.StartupPath = DefaultStartupPath
	}
}

// Validate checks that the configuration is valid.
// Implements the config.Validator interface.
func (c *Config) Validate() error {
	if c.Port <= 0 {
		return errors.New("health: port must be greater than 0")
	}
	if c.Port > MaxPort {
		return errors.New("health: port must be less than or equal to 65535")
	}
	return nil
}
