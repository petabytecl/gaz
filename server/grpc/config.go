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

// Config holds configuration for the gRPC server.
type Config struct {
	// Port is the TCP port the gRPC server listens on.
	// Defaults to 50051 if not set.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// Reflection enables gRPC reflection for service discovery.
	// When enabled, tools like grpcurl can introspect available services.
	// Defaults to true.
	Reflection bool `json:"reflection" yaml:"reflection" mapstructure:"reflection"`

	// MaxRecvMsgSize is the maximum message size the server can receive.
	// Defaults to 4MB.
	MaxRecvMsgSize int `json:"max_recv_msg_size" yaml:"max_recv_msg_size" mapstructure:"max_recv_msg_size"`

	// MaxSendMsgSize is the maximum message size the server can send.
	// Defaults to 4MB.
	MaxSendMsgSize int `json:"max_send_msg_size" yaml:"max_send_msg_size" mapstructure:"max_send_msg_size"`

	// HealthEnabled enables the built-in gRPC health check service.
	// Defaults to true.
	HealthEnabled bool `json:"health_enabled" yaml:"health_enabled" mapstructure:"health_enabled"`

	// HealthCheckInterval is the interval for syncing health status.
	// Defaults to 5 seconds.
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval" mapstructure:"health_check_interval"`

	// DevMode enables development mode for verbose error messages.
	// Defaults to false.
	DevMode bool `json:"dev_mode" yaml:"dev_mode" mapstructure:"dev_mode"`
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() Config {
	return Config{
		Port:                DefaultPort,
		Reflection:          true,
		MaxRecvMsgSize:      DefaultMaxMsgSize,
		MaxSendMsgSize:      DefaultMaxMsgSize,
		HealthEnabled:       true,
		HealthCheckInterval: 5 * time.Second,
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
}

// SetDefaults applies default values to zero-value fields.
// Implements the config.Defaulter interface.
func (c *Config) SetDefaults() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	// Reflection defaults to true, but we can't distinguish between
	// explicitly set to false vs not set. Leave as-is since bool zero is false.
	// Wait, if it defaults to true, and user passes false, we need to respect it.
	// Usually SetDefaults is for zero values.
	// But bool zero value is false.
	// We handle this by setting defaults in DefaultConfig() and having flags overwrite.
	// Config loading usually starts with DefaultConfig().
	// So SetDefaults might strictly be for things that are logically invalid if zero.

	if c.MaxRecvMsgSize == 0 {
		c.MaxRecvMsgSize = DefaultMaxMsgSize
	}
	if c.MaxSendMsgSize == 0 {
		c.MaxSendMsgSize = DefaultMaxMsgSize
	}
	if c.HealthCheckInterval == 0 {
		c.HealthCheckInterval = 5 * time.Second
	}
	// HealthEnabled defaults to true. But if user explicitly sets false (zero value), we shouldn't overwrite it to true here.
	// However, if it's coming from a file/env where it was missing, it would be false.
	// The pattern usually is DefaultConfig() provides defaults, then config loading applies.
	// SetDefaults() is a safety net.
	// If HealthEnabled is false, we don't know if it's intentional or missing.
	// But since DefaultConfig sets it to true, we assume if we are here and it's false, it MIGHT be intentional if DefaultConfig wasn't used.
	// Let's assume DefaultConfig is used.
}

// Validate checks that the configuration is valid.
// Implements the config.Validator interface.
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
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
