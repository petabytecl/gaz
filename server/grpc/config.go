package grpc

import (
	"fmt"
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
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() Config {
	return Config{
		Port:           DefaultPort,
		Reflection:     true,
		MaxRecvMsgSize: DefaultMaxMsgSize,
		MaxSendMsgSize: DefaultMaxMsgSize,
	}
}

// SetDefaults applies default values to zero-value fields.
// Implements the config.Defaulter interface.
func (c *Config) SetDefaults() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	// Reflection defaults to true, but we can't distinguish between
	// explicitly set to false vs not set. Leave as-is since bool zero is false.
	if c.MaxRecvMsgSize == 0 {
		c.MaxRecvMsgSize = DefaultMaxMsgSize
	}
	if c.MaxSendMsgSize == 0 {
		c.MaxSendMsgSize = DefaultMaxMsgSize
	}
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
	return nil
}
