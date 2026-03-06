package vanguard

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

// DefaultPort is the default port for the Vanguard server.
const DefaultPort = 8080

// DefaultReadHeaderTimeout is the default read header timeout.
// This protects against slowloris attacks.
const DefaultReadHeaderTimeout = 5 * time.Second

// DefaultIdleTimeout is the default idle timeout for keep-alive connections.
const DefaultIdleTimeout = 120 * time.Second

// Config holds configuration for the Vanguard server.
type Config struct {
	// Port is the TCP port the Vanguard server listens on.
	// Defaults to 8080 if not set.
	Port int `json:"port" yaml:"port" mapstructure:"port" gaz:"port"`

	// ReadTimeout is the maximum duration for reading the entire request.
	// Zero means no timeout, which is required for streaming RPCs.
	// Defaults to 0 (streaming-safe).
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout" gaz:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes of the response.
	// Zero means no timeout, which is required for streaming RPCs.
	// Defaults to 0 (streaming-safe).
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout" gaz:"write_timeout"`

	// IdleTimeout is the maximum duration an idle keep-alive connection will remain open.
	// Defaults to 120 seconds.
	IdleTimeout time.Duration `json:"idle_timeout" yaml:"idle_timeout" mapstructure:"idle_timeout" gaz:"idle_timeout"`

	// ReadHeaderTimeout is the maximum duration for reading request headers.
	// This protects against slowloris attacks.
	// Defaults to 5 seconds.
	ReadHeaderTimeout time.Duration `json:"read_header_timeout" yaml:"read_header_timeout" mapstructure:"read_header_timeout" gaz:"read_header_timeout"`

	// Reflection enables gRPC reflection via Connect handlers (v1 and v1alpha).
	// When enabled, tools like grpcurl can introspect available services.
	// Defaults to true.
	Reflection bool `json:"reflection" yaml:"reflection" mapstructure:"reflection" gaz:"reflection"`

	// HealthEnabled enables automatic health endpoint mounting.
	// When enabled and health.Manager is present, /healthz, /readyz, and /livez
	// endpoints are mounted on the unknown handler.
	// Defaults to true.
	HealthEnabled bool `json:"health_enabled" yaml:"health_enabled" mapstructure:"health_enabled" gaz:"health_enabled"`

	// DevMode enables development mode for verbose error messages.
	// Defaults to false.
	DevMode bool `json:"dev_mode" yaml:"dev_mode" mapstructure:"dev_mode" gaz:"dev_mode"`
}

// DefaultConfig returns a Config with safe defaults.
// ReadTimeout and WriteTimeout are intentionally zero for streaming safety.
func DefaultConfig() Config {
	return Config{
		Port:              DefaultPort,
		ReadTimeout:       0,
		WriteTimeout:      0,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		Reflection:        true,
		HealthEnabled:     true,
		DevMode:           false,
	}
}

// Namespace returns the config namespace.
// The Vanguard server uses "server" as its namespace, so flags are prefixed with server-.
func (c *Config) Namespace() string {
	return "server"
}

// Flags registers the config flags.
// ReadTimeout and WriteTimeout are not exposed as flags because zero is the
// correct default for streaming RPCs. Users should only change them via config
// file if they understand the streaming implications.
func (c *Config) Flags(fs *pflag.FlagSet) {
	fs.IntVar(&c.Port, "server-port", c.Port, "Vanguard server port")
	fs.DurationVar(&c.ReadHeaderTimeout, "server-read-header-timeout", c.ReadHeaderTimeout, "Maximum duration for reading request headers")
	fs.DurationVar(&c.IdleTimeout, "server-idle-timeout", c.IdleTimeout, "Maximum duration for idle keep-alive connections")
	fs.BoolVar(&c.Reflection, "server-reflection", c.Reflection, "Enable gRPC reflection via Connect handlers")
	fs.BoolVar(&c.HealthEnabled, "server-health-enabled", c.HealthEnabled, "Enable automatic health endpoint mounting")
	fs.BoolVar(&c.DevMode, "server-dev-mode", c.DevMode, "Enable development mode")
}

// SetDefaults applies default values to zero-value fields.
// ReadTimeout and WriteTimeout are NOT defaulted because zero is intentional
// for streaming safety. Only Port, ReadHeaderTimeout, and IdleTimeout are filled.
// Implements the config.Defaulter interface.
func (c *Config) SetDefaults() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = DefaultReadHeaderTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = DefaultIdleTimeout
	}
	// ReadTimeout and WriteTimeout are intentionally NOT defaulted.
	// Zero means no timeout, which is required for streaming RPCs.
}

// Validate checks that the configuration is valid.
// ReadTimeout=0 and WriteTimeout=0 are explicitly accepted as valid because
// streaming RPCs require no timeout on reads and writes.
// Implements the config.Validator interface.
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("vanguard: invalid port %d: must be between 1 and 65535", c.Port)
	}
	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("vanguard: invalid read_header_timeout %s: must be positive", c.ReadHeaderTimeout)
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("vanguard: invalid idle_timeout %s: must be positive", c.IdleTimeout)
	}
	// ReadTimeout and WriteTimeout accept zero values (streaming-safe).
	return nil
}
