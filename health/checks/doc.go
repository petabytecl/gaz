// Package checks provides reusable health check implementations for common
// infrastructure dependencies.
//
// Each subpackage provides a Config struct and New() factory function that
// returns a health check function compatible with health.CheckFunc:
//
//	func(context.Context) error
//
// Example usage with the health package:
//
//	import (
//	    "github.com/petabytecl/gaz/health"
//	    checksql "github.com/petabytecl/gaz/health/checks/sql"
//	)
//
//	func main() {
//	    db, _ := sql.Open("postgres", dsn)
//
//	    registrar.AddReadinessCheck("database", checksql.New(checksql.Config{
//	        DB: db,
//	    }))
//	}
//
// Available check packages:
//   - sql: SQL database connectivity (database/sql)
//   - tcp: TCP port connectivity
//   - dns: DNS hostname resolution
//   - http: HTTP upstream availability
//   - runtime: Go runtime metrics (goroutines, memory, GC)
//   - redis: Redis connectivity (requires go-redis/v9)
//   - disk: Disk space monitoring (requires gopsutil/v4)
package checks
