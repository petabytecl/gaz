// Package sql provides a health check for SQL databases using database/sql.
package sql

import (
	"context"
	"database/sql"
	"fmt"
)

// Config configures the SQL database health check.
type Config struct {
	// DB is the database connection pool to check. Required.
	DB *sql.DB
}

// New creates a new SQL database health check.
// Uses PingContext which is optimized for connection testing and respects
// the context deadline.
//
// Returns nil if healthy (ping succeeds), error if unhealthy.
func New(cfg Config) func(context.Context) error {
	return func(ctx context.Context) error {
		if cfg.DB == nil {
			return fmt.Errorf("sql: database connection is nil")
		}
		if err := cfg.DB.PingContext(ctx); err != nil {
			return fmt.Errorf("sql: ping failed: %w", err)
		}
		return nil
	}
}
