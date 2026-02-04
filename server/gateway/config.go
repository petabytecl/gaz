package gateway

import (
	"fmt"

	"github.com/spf13/pflag"
)

// DefaultPort is the default port for the HTTP Gateway.
// Uses port 8080, distinct from gRPC (50051) and health (9090).
const DefaultPort = 8080

// DefaultGRPCTarget is the default gRPC server target for loopback connections.
const DefaultGRPCTarget = "localhost:50051"

// DefaultCORSMaxAge is the default max age for preflight request caching (24 hours in seconds).
const DefaultCORSMaxAge = 86400

// Config holds configuration for the Gateway.
type Config struct {
	// Port is the TCP port the Gateway listens on.
	// Defaults to 8080 if not set.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// GRPCTarget is the gRPC server target for loopback connections.
	// Defaults to "localhost:50051" if not set.
	GRPCTarget string `json:"grpc_target" yaml:"grpc_target" mapstructure:"grpc_target"`

	// CORS contains CORS configuration for the Gateway.
	CORS CORSConfig `json:"cors" yaml:"cors" mapstructure:"cors"`

	// DevMode enables development mode.
	// In dev mode, CORS is wide-open and error responses include details.
	// Defaults to false.
	DevMode bool `json:"dev_mode" yaml:"dev_mode" mapstructure:"dev_mode"`
}

// CORSConfig holds CORS configuration for the Gateway.
type CORSConfig struct {
	// AllowedOrigins is a list of allowed origins.
	// Use ["*"] to allow all origins (dev mode only, not with credentials).
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowed_origins" mapstructure:"allowed_origins"`

	// AllowedMethods is a list of allowed HTTP methods.
	AllowedMethods []string `json:"allowed_methods" yaml:"allowed_methods" mapstructure:"allowed_methods"`

	// AllowedHeaders is a list of allowed request headers.
	// Use ["*"] to allow all headers (dev mode only).
	AllowedHeaders []string `json:"allowed_headers" yaml:"allowed_headers" mapstructure:"allowed_headers"`

	// ExposedHeaders is a list of headers exposed to the browser.
	ExposedHeaders []string `json:"exposed_headers" yaml:"exposed_headers" mapstructure:"exposed_headers"`

	// AllowCredentials indicates whether credentials (cookies, auth headers) are allowed.
	// Cannot be used with AllowedOrigins ["*"].
	AllowCredentials bool `json:"allow_credentials" yaml:"allow_credentials" mapstructure:"allow_credentials"`

	// MaxAge is the maximum age (in seconds) for preflight request caching.
	MaxAge int `json:"max_age" yaml:"max_age" mapstructure:"max_age"`
}

// DefaultConfig returns a Config with safe defaults.
func DefaultConfig() Config {
	return Config{
		Port:       DefaultPort,
		GRPCTarget: DefaultGRPCTarget,
		CORS:       DefaultCORSConfig(false),
		DevMode:    false,
	}
}

// Namespace returns the config namespace.
func (c *Config) Namespace() string {
	return "gateway"
}

// Flags registers the config flags.
func (c *Config) Flags(fs *pflag.FlagSet) {
	fs.IntVar(&c.Port, "gateway-port", c.Port, "Gateway HTTP port")
	fs.StringVar(&c.GRPCTarget, "gateway-grpc-target", c.GRPCTarget, "gRPC server target (default: localhost:<grpc-port>)")
	fs.BoolVar(&c.DevMode, "gateway-dev-mode", c.DevMode, "Enable development mode")
}

// DefaultCORSConfig returns a CORSConfig with appropriate defaults.
// In dev mode, CORS is wide-open for convenience.
// In prod mode, origins must be explicitly configured.
func DefaultCORSConfig(devMode bool) CORSConfig {
	if devMode {
		return CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			ExposedHeaders:   []string{},
			AllowCredentials: false, // Cannot use * with credentials.
			MaxAge:           DefaultCORSMaxAge,
		}
	}
	return CORSConfig{
		AllowedOrigins:   []string{}, // Must be explicitly configured.
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           DefaultCORSMaxAge,
	}
}

// SetDefaults applies default values to zero-value fields.
// Implements the config.Defaulter interface.
func (c *Config) SetDefaults() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.GRPCTarget == "" {
		c.GRPCTarget = DefaultGRPCTarget
	}
	// Note: CORS defaults are not applied here since we can't distinguish
	// between intentionally empty and not set. Use DefaultCORSConfig
	// when creating the initial configuration.
}

// Validate checks that the configuration is valid.
// Implements the config.Validator interface.
func (c *Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("gateway: invalid port %d: must be between 1 and 65535", c.Port)
	}
	return nil
}
