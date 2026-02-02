// Package redis provides a health check for Redis using go-redis/v9.
package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Config configures the Redis health check.
type Config struct {
	// Client is the Redis client to check. Required.
	// Accepts any client implementing redis.UniversalClient
	// (redis.Client, redis.ClusterClient, redis.Ring).
	Client redis.UniversalClient
}

// New creates a new Redis health check.
// Uses PING command to verify connectivity and response.
//
// Returns nil if PING returns "PONG", error otherwise.
func New(cfg Config) func(context.Context) error {
	return func(ctx context.Context) error {
		if cfg.Client == nil {
			return fmt.Errorf("redis: client is nil")
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
