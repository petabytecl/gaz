package vanguard

import (
	"errors"
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

// DefaultCORSMaxAge is the default max age for preflight request caching (24 hours in seconds).
const DefaultCORSMaxAge = 86400

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
	// When enabled and health.Manager is present, health endpoints are mounted
	// on the unknown handler using paths from health.Config.
	// Defaults to true.
	HealthEnabled bool `json:"health_enabled" yaml:"health_enabled" mapstructure:"health_enabled" gaz:"health_enabled"`

	// DevMode enables development mode for verbose error messages.
	// Defaults to false.
	DevMode bool `json:"dev_mode" yaml:"dev_mode" mapstructure:"dev_mode" gaz:"dev_mode"`

	// AllowZeroWriteTimeout explicitly opts in to zero write timeout.
	// When false (default), WriteTimeout=0 is rejected by Validate as a Slowloris risk.
	// Set to true only when streaming RPCs require no write timeout.
	AllowZeroWriteTimeout bool `json:"allow_zero_write_timeout" yaml:"allow_zero_write_timeout" mapstructure:"allow_zero_write_timeout" gaz:"allow_zero_write_timeout"`

	// CORS contains CORS configuration for the Vanguard server.
	CORS CORSConfig `json:"cors" yaml:"cors" mapstructure:"cors" gaz:"cors"`
}

// CORSConfig holds CORS configuration for the Vanguard server.
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
// ReadTimeout and WriteTimeout are intentionally zero for streaming safety.
func DefaultConfig() Config {
	return Config{
		Port:                  DefaultPort,
		ReadTimeout:           0,
		WriteTimeout:          0,
		ReadHeaderTimeout:     DefaultReadHeaderTimeout,
		IdleTimeout:           DefaultIdleTimeout,
		Reflection:            true,
		HealthEnabled:         true,
		DevMode:               false,
		AllowZeroWriteTimeout: true,
		CORS:                  DefaultCORSConfig(false),
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
	fs.StringSliceVar(&c.CORS.AllowedOrigins, "server-cors-origins", c.CORS.AllowedOrigins, "CORS allowed origins")
	fs.StringSliceVar(&c.CORS.AllowedMethods, "server-cors-methods", c.CORS.AllowedMethods, "CORS allowed HTTP methods")
	fs.StringSliceVar(&c.CORS.AllowedHeaders, "server-cors-headers", c.CORS.AllowedHeaders, "CORS allowed request headers")
	fs.StringSliceVar(&c.CORS.ExposedHeaders, "server-cors-exposed-headers", c.CORS.ExposedHeaders, "CORS exposed response headers")
	fs.BoolVar(&c.CORS.AllowCredentials, "server-cors-credentials", c.CORS.AllowCredentials, "CORS allow credentials")
	fs.IntVar(&c.CORS.MaxAge, "server-cors-max-age", c.CORS.MaxAge, "CORS preflight max age in seconds")
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
	// WriteTimeout=0 is a Slowloris risk unless explicitly opted in.
	if c.WriteTimeout == 0 && !c.AllowZeroWriteTimeout {
		return errors.New("vanguard: write_timeout=0 disables timeout protection (Slowloris risk); " +
			"set allow_zero_write_timeout=true to explicitly allow, or set a positive write_timeout")
	}
	return nil
}
