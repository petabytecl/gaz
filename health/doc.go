// Package health provides health check management for gaz applications.
//
// Health checks are used to determine if a service is ready to receive
// traffic (readiness), if it's still alive (liveness), and if it has
// completed initialization (startup). The package integrates with gaz's
// DI container and lifecycle management.
//
// # Quick Start
//
// Use [NewModule] to register health infrastructure with your app:
//
//	app := gaz.New()
//	app.UseDI(health.NewModule())
//	app.Build()
//
// This registers a [Manager] for check registration, a [ManagementServer]
// for HTTP endpoints, and a [ShutdownCheck] for graceful shutdown signaling.
//
// # Health Check Types
//
// The package supports three types of probes, aligned with Kubernetes:
//
//   - Liveness: Is the process running? Failures may trigger restart.
//   - Readiness: Can the service handle traffic? Failures stop traffic routing.
//   - Startup: Has initialization completed? Failures hold off other probes.
//
// # Registering Checks
//
// Use the [Registrar] interface to add custom health checks:
//
//	manager.AddReadinessCheck("database", func(ctx context.Context) error {
//	    return db.PingContext(ctx)
//	})
//
//	manager.AddLivenessCheck("memory", func(ctx context.Context) error {
//	    var m runtime.MemStats
//	    runtime.ReadMemStats(&m)
//	    if m.Alloc > 1<<30 { // 1GB
//	        return errors.New("memory usage too high")
//	    }
//	    return nil
//	})
//
// # HTTP Endpoints
//
// The [ManagementServer] exposes health endpoints on a dedicated port (default 9090):
//
//   - /live - Liveness probe (always returns 200 OK)
//   - /ready - Readiness probe (503 when unhealthy)
//   - /startup - Startup probe (503 when not ready)
//
// Configure paths and port via [NewModule] options:
//
//	health.NewModule(
//	    health.WithPort(8081),
//	    health.WithLivenessPath("/healthz"),
//	    health.WithReadinessPath("/ready"),
//	)
//
// # Graceful Shutdown
//
// The [ShutdownCheck] automatically fails readiness probes during shutdown,
// allowing load balancers to drain connections before the service stops.
// It is registered by default with [NewModule].
//
// # Testing
//
// The package provides test helpers for health check testing:
//
//   - [TestConfig] returns safe defaults (port 0 for random port)
//   - [MockRegistrar] is a testify/mock implementation of [Registrar]
//   - [TestManager] creates a manager suitable for testing
//   - [RequireHealthy] and [RequireUnhealthy] are assertion helpers
//
// Example test:
//
//	func TestDatabaseCheck(t *testing.T) {
//	    m := health.TestManager()
//	    m.AddReadinessCheck("db", func(ctx context.Context) error {
//	        return nil // healthy
//	    })
//	    health.RequireHealthy(t, m)
//	}
package health
