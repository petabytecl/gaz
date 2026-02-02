// Package redis provides a health check for Redis/Valkey using valkey-go.
package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/valkey-io/valkey-go"
)

// ErrNilClient is returned when the Valkey client is nil.
var ErrNilClient = errors.New("redis: client is nil")

// Config configures the Redis/Valkey health check.
type Config struct {
	// Client is the Valkey client to check. Required.
	// Use valkey.NewClient() to create one.
	Client valkey.Client
}

// New creates a new Redis/Valkey health check.
// Uses PING command to verify connectivity and response.
//
// Returns nil if PING returns "PONG", error otherwise.
func New(cfg Config) func(context.Context) error {
	return func(ctx context.Context) error {
		if cfg.Client == nil {
			return ErrNilClient
		}
		resp, err := cfg.Client.Do(ctx, cfg.Client.B().Ping().Build()).ToString()
		if err != nil {
			return fmt.Errorf("redis: ping failed: %w", err)
		}
		if resp != "PONG" {
			return fmt.Errorf("redis: unexpected ping response: %q", resp)
		}
		return nil
	}
}
