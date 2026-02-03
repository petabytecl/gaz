package pgx

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNilPool is returned when the pool is nil.
var ErrNilPool = errors.New("pgx: pool is nil")

// Config configures the PGX database health check.
type Config struct {
	// Pool is the pgx connection pool to check. Required.
	Pool *pgxpool.Pool
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
