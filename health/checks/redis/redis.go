// Package redis provides a health check for Redis using go-redis/v9.
package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ErrNilClient is returned when the Redis client is nil.
var ErrNilClient = errors.New("redis: client is nil")

// Pinger is an interface for Redis clients that can ping.
type Pinger interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

// Config configures the Redis health check.
type Config struct {
	// Client is the Redis client to check. Required.
	// Use redis.NewClient() to create one.
	Client Pinger
}

// New creates a new Redis health check.
// Uses PING command to verify connectivity and response.
//
// Returns nil if PING returns "PONG", error otherwise.
func New(cfg Config) func(context.Context) error {
	return func(ctx context.Context) error {
		if cfg.Client == nil {
			return ErrNilClient
		}
		pong, err := cfg.Client.Ping(ctx).Result()
		if err != nil {
			return fmt.Errorf("redis: ping failed: %w", err)
		}
		if pong != "PONG" {
			return fmt.Errorf("redis: unexpected ping response: %q", pong)
		}
		return nil
	}
}
