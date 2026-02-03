package http

import (
	"errors"
	"time"
)

// Default configuration values.
const (
	// DefaultPort is the default HTTP port.
	DefaultPort = 8080

	// DefaultReadTimeout is the default timeout for reading the entire request.
	DefaultReadTimeout = 10 * time.Second

	// DefaultWriteTimeout is the default timeout for writing the response.
	DefaultWriteTimeout = 30 * time.Second

	// DefaultIdleTimeout is the default timeout for idle connections.
	DefaultIdleTimeout = 120 * time.Second

	// DefaultReadHeaderTimeout is the default timeout for reading request headers.
	// This is critical for preventing slow loris attacks.
	DefaultReadHeaderTimeout = 5 * time.Second
)

// Config holds configuration for the HTTP server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	// Defaults to 8080 if not set.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// ReadTimeout is the maximum duration for reading the entire request,
	// including the body. Defaults to 10 seconds.
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout"`

	// WriteTimeout is the maximum duration before timing out writes of the
	// response. Defaults to 30 seconds.
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout"`

	// IdleTimeout is the maximum amount of time to wait for the next request
	// when keep-alives are enabled. Defaults to 120 seconds.
	IdleTimeout time.Duration `json:"idle_timeout" yaml:"idle_timeout" mapstructure:"idle_timeout"`

	// ReadHeaderTimeout is the amount of time allowed to read request headers.
	// This is critical for preventing slow loris attacks.
	// Defaults to 5 seconds.
	ReadHeaderTimeout time.Duration `json:"read_header_timeout" yaml:"read_header_timeout" mapstructure:"read_header_timeout"`
}

// DefaultConfig returns a Config with safe defaults.
// The timeout values are chosen to balance responsiveness with protection
// against slow loris and similar attacks.
func DefaultConfig() Config {
	return Config{
		Port:              DefaultPort,
		ReadTimeout:       DefaultReadTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
	}
}

// SetDefaults applies default values to zero-value fields.
// Implements the config.Defaulter interface.
func (c *Config) SetDefaults() {
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultWriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = DefaultIdleTimeout
	}
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = DefaultReadHeaderTimeout
	}
}

// Validate checks that the configuration is valid.
// Implements the config.Validator interface.
func (c *Config) Validate() error {
	if c.Port <= 0 {
		return errors.New("http: port must be greater than 0")
	}
	if c.Port > 65535 {
		return errors.New("http: port must be less than or equal to 65535")
	}
	if c.ReadTimeout <= 0 {
		return errors.New("http: read_timeout must be greater than 0")
	}
	if c.WriteTimeout <= 0 {
		return errors.New("http: write_timeout must be greater than 0")
	}
	if c.IdleTimeout <= 0 {
		return errors.New("http: idle_timeout must be greater than 0")
	}
	if c.ReadHeaderTimeout <= 0 {
		return errors.New("http: read_header_timeout must be greater than 0")
	}
	return nil
}
