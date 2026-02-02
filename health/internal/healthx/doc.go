// Package healthx provides internal health check execution with parallel processing.
//
// This package replaces alexliesenfeld/health with a minimal API surface tailored
// to the gaz framework's needs. It provides:
//
//   - Check struct for configuring individual health checks
//   - Checker interface for executing checks and aggregating results
//   - Parallel execution with per-check timeouts (default 5s)
//   - Panic recovery to prevent individual check failures from crashing the handler
//   - Critical vs warning check distinction for graceful degradation
//
// # Basic Usage
//
//	checker := healthx.NewChecker(
//		healthx.WithCheck(healthx.Check{
//			Name:  "database",
//			Check: func(ctx context.Context) error {
//				return db.PingContext(ctx)
//			},
//		}),
//		healthx.WithCheck(healthx.Check{
//			Name:     "cache",
//			Check:    checkRedis,
//			Timeout:  2 * time.Second, // Custom timeout
//		}),
//	)
//
//	result := checker.Check(ctx)
//	// result.Status is StatusUp only if all critical checks pass
//
// # Check Criticality
//
// By default, all checks are critical - if any critical check fails, the overall
// status is StatusDown. To mark a check as non-critical (warning), the checker
// implementation treats it accordingly based on the Critical field. Warning checks
// are included in the result details but don't affect the aggregated status.
// This allows non-essential dependencies to degrade gracefully without marking
// the entire service as unavailable.
package healthx
