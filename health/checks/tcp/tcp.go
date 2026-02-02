// Package tcp provides a health check for TCP port connectivity.
package tcp

import (
	"context"
	"fmt"
	"net"
	"time"
)

// Config configures the TCP dial health check.
type Config struct {
	// Addr is the address to dial (host:port). Required.
	Addr string
	// Timeout for the dial operation. Optional, defaults to 2s.
	// The context deadline takes precedence if shorter.
	Timeout time.Duration
}

// New creates a new TCP dial health check.
// Verifies TCP connectivity by establishing and immediately closing a connection.
//
// Returns nil if connection succeeds, error if dial fails.
func New(cfg Config) func(context.Context) error {
	if cfg.Timeout == 0 {
		cfg.Timeout = 2 * time.Second
	}

	return func(ctx context.Context) error {
		if cfg.Addr == "" {
			return fmt.Errorf("tcp: address is empty")
		}

		var d net.Dialer
		d.Timeout = cfg.Timeout

		conn, err := d.DialContext(ctx, "tcp", cfg.Addr)
		if err != nil {
			return fmt.Errorf("tcp: dial failed: %w", err)
		}
		return conn.Close()
	}
}
