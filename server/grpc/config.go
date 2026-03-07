package grpc

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

// DefaultPort is the default port for the gRPC server.
const DefaultPort = 50051

// DefaultMaxMsgSize is the default maximum message size (4MB).
const DefaultMaxMsgSize = 4 * 1024 * 1024

// DefaultHealthCheckInterval is the default interval for health checks.
const DefaultHealthCheckInterval = 5 * time.Second

// Config holds configuration for the gRPC server.
type Config struct {
	// Port is the TCP port the gRPC server listens on.
	// Defaults to 50051 if not set.
	Port int `json:"port" yaml:"port" mapstructure:"port" gaz:"port"`

	// Reflection enables gRPC reflection for service discovery.
	// When enabled, tools like grpcurl can introspect available services.
	// Defaults to true.
	Reflection bool `json:"reflection" yaml:"reflection" mapstructure:"reflection" gaz:"reflection"`

	// MaxRecvMsgSize is the maximum message size the server can receive.
	// Defaults to 4MB.
	MaxRecvMsgSize int `json:"max_recv_msg_size" yaml:"max_recv_msg_size" mapstructure:"max_recv_msg_size" gaz:"max_recv_msg_size"`

	// MaxSendMsgSize is the maximum message size the server can send.
	// Defaults to 4MB.
	MaxSendMsgSize int `json:"max_send_msg_size" yaml:"max_send_msg_size" mapstructure:"max_send_msg_size" gaz:"max_send_msg_size"`

	// HealthEnabled enables the built-in gRPC health check service.
	// Defaults to true.
	HealthEnabled bool `json:"health_enabled" yaml:"health_enabled" mapstructure:"health_enabled" gaz:"health_enabled"`

	// HealthCheckInterval is the interval for syncing health status.
	// Defaults to 5 seconds.
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval" mapstructure:"health_check_interval" gaz:"health_check_interval"`

	// DevMode enables development mode for verbose error messages.
	// Defaults to false.
	DevMode bool `json:"dev_mode" yaml:"dev_mode" mapstructure:"dev_mode" gaz:"dev_mode"`

	// SkipListener skips binding a listener and serving.
	// When true, the server still discovers registrars, registers services,
	// enables reflection, and wires health — but does not bind a port or
	// call server.Serve(). This is used when Vanguard handles connections.
	// Defaults to false.
	SkipListener bool `json:"skip_listener" yaml:"skip_listener" mapstructure:"skip_listener" gaz:"skip_listener"`
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() Config {
	return Config{
		Port:                DefaultPort,
		Reflection:          true,
		MaxRecvMsgSize:      DefaultMaxMsgSize,
		MaxSendMsgSize:      DefaultMaxMsgSize,
		HealthEnabled:       true,
		HealthCheckInterval: DefaultHealthCheckInterval,
		DevMode:             false,
	}
}

// Namespace returns the config namespace.
func (c *Config) Namespace() string {
	return "grpc"
}

// Flags registers the config flags.
func (c *Config) Flags(fs *pflag.FlagSet) {
	fs.IntVar(&c.Port, "grpc-port", c.Port, "gRPC server port")
	fs.BoolVar(&c.Reflection, "grpc-reflection", c.Reflection, "Enable gRPC reflection")
	fs.BoolVar(&c.HealthEnabled, "grpc-health-enabled", c.HealthEnabled, "Enable gRPC health check service")
	fs.DurationVar(&c.HealthCheckInterval, "grpc-health-interval", c.HealthCheckInterval, "Interval for syncing gRPC health status")
	fs.BoolVar(&c.DevMode, "grpc-dev-mode", c.DevMode, "Enable gRPC development mode")
	fs.BoolVar(&c.SkipListener, "grpc-skip-listener", c.SkipListener, "Skip binding a listener (used when Vanguard handles connections)")
}

// SetDefaults applies default values to zero-value fields.
// Boolean fields (Reflection, HealthEnabled, SkipListener) are not set here
// because their zero value (false) is indistinguishable from an explicit false.
// Use DefaultConfig() to get safe defaults before config loading.
// Implements the config.Defaulter interface.
func (c *Config) SetDefaults() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.MaxRecvMsgSize == 0 {
		c.MaxRecvMsgSize = DefaultMaxMsgSize
	}
	if c.MaxSendMsgSize == 0 {
		c.MaxSendMsgSize = DefaultMaxMsgSize
	}
	if c.HealthCheckInterval == 0 {
		c.HealthCheckInterval = DefaultHealthCheckInterval
	}
}

// Validate checks that the configuration is valid.
// Implements the config.Validator interface.
func (c *Config) Validate() error {
	if !c.SkipListener && (c.Port <= 0 || c.Port > 65535) {
		return fmt.Errorf("grpc: invalid port %d: must be between 1 and 65535", c.Port)
	}
	if c.MaxRecvMsgSize <= 0 {
		return fmt.Errorf("grpc: invalid max_recv_msg_size %d: must be positive", c.MaxRecvMsgSize)
	}
	if c.MaxSendMsgSize <= 0 {
		return fmt.Errorf("grpc: invalid max_send_msg_size %d: must be positive", c.MaxSendMsgSize)
	}
	if c.HealthEnabled && c.HealthCheckInterval <= 0 {
		return fmt.Errorf("grpc: invalid health_check_interval %s: must be positive", c.HealthCheckInterval)
	}
	return nil
}
