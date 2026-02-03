// Package pgx provides a health check for PostgreSQL databases using pgx/v5.
package pgx

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNilPool is returned when the pool is nil.
var ErrNilPool = errors.New("pgx: pool is nil")

// Pinger is an interface for types that can ping a database.
// This interface is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Compile-time check that pgxpool.Pool implements Pinger.
var _ Pinger = (*pgxpool.Pool)(nil)

// Config configures the PGX database health check.
type Config struct {
	// Pool is the pgx connection pool to check. Required.
	// This accepts *pgxpool.Pool or any type implementing Pinger.
	Pool Pinger
}

// New creates a new PGX database health check.
// Uses Ping which is optimized for connection testing and respects
// the context deadline.
//
// Returns nil if healthy (ping succeeds), error if unhealthy.
func New(cfg Config) func(context.Context) error {
	return func(ctx context.Context) error {
		if cfg.Pool == nil {
			return ErrNilPool
		}
		if err := cfg.Pool.Ping(ctx); err != nil {
			return fmt.Errorf("pgx: ping failed: %w", err)
		}
		return nil
	}
}
