// Package dns provides a health check for DNS hostname resolution.
package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// DefaultTimeout is the default timeout for DNS resolution.
const DefaultTimeout = 2 * time.Second

// ErrEmptyHost is returned when the hostname is empty.
var ErrEmptyHost = errors.New("dns: hostname is empty")

// Config configures the DNS resolution health check.
type Config struct {
	// Host is the hostname to resolve. Required.
	Host string
	// Timeout for the resolution. Optional, defaults to 2s.
	// The context deadline takes precedence if shorter.
	Timeout time.Duration
}

// New creates a new DNS resolution health check.
// Verifies DNS resolution by looking up the hostname.
//
// Returns nil if resolution succeeds with at least one address, error otherwise.
func New(cfg Config) func(context.Context) error {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	resolver := &net.Resolver{}

	return func(ctx context.Context) error {
		if cfg.Host == "" {
			return ErrEmptyHost
		}

		ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()

		addrs, err := resolver.LookupHost(ctx, cfg.Host)
		if err != nil {
			return fmt.Errorf("dns: lookup failed: %w", err)
		}
		if len(addrs) == 0 {
			return fmt.Errorf("dns: no addresses found for %s", cfg.Host)
		}
		return nil
	}
}
