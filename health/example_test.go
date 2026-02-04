package health_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/petabytecl/gaz/health"
	"github.com/stretchr/testify/mock"
)

// ExampleManager_AddReadinessCheck demonstrates registering a readiness check.
func ExampleManager_AddReadinessCheck() {
	mgr := health.NewManager()

	// Register a readiness check for database connectivity
	mgr.AddReadinessCheck("database", func(ctx context.Context) error {
		// In real code, this would ping the database
		// return db.PingContext(ctx)
		return nil // healthy
	})

	// Build a checker and run checks
	checker := mgr.ReadinessChecker()
	result := checker.Check(context.Background())

	fmt.Printf("Status: %s\n", result.Status)
	// Output: Status: up
}

// ExampleManager_AddLivenessCheck demonstrates registering a liveness check.
func ExampleManager_AddLivenessCheck() {
	mgr := health.NewManager()

	// Register a liveness check for basic process health
	mgr.AddLivenessCheck("heartbeat", func(ctx context.Context) error {
		// Liveness checks should be simple and fast
		// Returning nil means the process is alive
		return nil
	})

	// Build a checker and run checks
	checker := mgr.LivenessChecker()
	result := checker.Check(context.Background())

	fmt.Printf("Status: %s\n", result.Status)
	// Output: Status: up
}

// ExampleManager_AddStartupCheck demonstrates registering a startup check.
func ExampleManager_AddStartupCheck() {
	mgr := health.NewManager()

	// Track initialization status
	initialized := true

	// Register a startup check
	mgr.AddStartupCheck("init", func(ctx context.Context) error {
		if !initialized {
			return errors.New("initialization in progress")
		}
		return nil
	})

	// Build a checker and run checks
	checker := mgr.StartupChecker()
	result := checker.Check(context.Background())

	fmt.Printf("Status: %s\n", result.Status)
	// Output: Status: up
}

// Example_customCheck demonstrates implementing a custom health check.
func Example_customCheck() {
	// Custom checks can be any function matching CheckFunc signature
	type DatabaseCheck struct {
		connectionOK bool
	}

	db := &DatabaseCheck{connectionOK: true}

	mgr := health.NewManager()

	// The Check method signature matches health.CheckFunc
	mgr.AddReadinessCheck("database", func(ctx context.Context) error {
		if !db.connectionOK {
			return errors.New("database connection lost")
		}
		return nil
	})

	// Verify the check passes
	checker := mgr.ReadinessChecker()
	result := checker.Check(context.Background())

	fmt.Printf("Status: %s\n", result.Status)
	// Output: Status: up
}

// Example_unhealthyCheck demonstrates a failing health check.
func Example_unhealthyCheck() {
	mgr := health.NewManager()

	// Register a check that always fails
	mgr.AddReadinessCheck("broken", func(ctx context.Context) error {
		return errors.New("service unavailable")
	})

	// Run the check
	checker := mgr.ReadinessChecker()
	result := checker.Check(context.Background())

	fmt.Printf("Status: %s\n", result.Status)
	// Output: Status: down
}

// ExampleTestConfig demonstrates using test configuration.
func ExampleTestConfig() {
	// TestConfig returns safe defaults for testing
	// Uses port 0 for random available port (avoids conflicts)
	cfg := health.TestConfig()

	fmt.Printf("Port: %d\n", cfg.Port)
	fmt.Printf("LivenessPath: %s\n", cfg.LivenessPath)
	// Output:
	// Port: 0
	// LivenessPath: /live
}

// ExampleNewTestConfig demonstrates customizing test config.
func ExampleNewTestConfig() {
	// NewTestConfig allows overriding specific fields
	cfg := health.NewTestConfig(func(c *health.Config) {
		c.Port = 8080
	})

	fmt.Printf("Port: %d\n", cfg.Port)
	fmt.Printf("ReadinessPath: %s\n", cfg.ReadinessPath)
	// Output:
	// Port: 8080
	// ReadinessPath: /ready
}

// ExampleTestManager demonstrates creating a manager for testing.
func ExampleTestManager() {
	// TestManager creates a clean manager for test isolation
	mgr := health.TestManager()

	// Add a test check
	mgr.AddReadinessCheck("test", func(ctx context.Context) error {
		return nil
	})

	// Verify it's healthy
	checker := mgr.ReadinessChecker()
	result := checker.Check(context.Background())

	fmt.Printf("Status: %s\n", result.Status)
	// Output: Status: up
}

// ExampleMockRegistrar demonstrates using the mock for testing.
func ExampleMockRegistrar() {
	// MockRegistrar lets you verify check registration
	mockReg := health.NewMockRegistrar()

	// Your service registers checks via the Registrar interface
	registerChecks(mockReg)

	// Verify expected checks were registered
	mockReg.AssertCalled(&testing.T{}, "AddReadinessCheck", "database", mock.Anything)
}

// Helper for mock example.
func registerChecks(r health.Registrar) {
	r.AddReadinessCheck("database", func(ctx context.Context) error {
		return nil
	})
}

// ExampleShutdownCheck demonstrates the shutdown check behavior.
func ExampleShutdownCheck() {
	// ShutdownCheck fails readiness when shutdown is signaled
	shutdownCheck := health.NewShutdownCheck()

	// Before shutdown, check passes
	err := shutdownCheck.Check(context.Background())
	fmt.Printf("Before shutdown: %v\n", err == nil)

	// Signal shutdown
	shutdownCheck.MarkShuttingDown()

	// After shutdown signal, check fails
	err = shutdownCheck.Check(context.Background())
	fmt.Printf("After shutdown: %v\n", err != nil)
	// Output:
	// Before shutdown: true
	// After shutdown: true
}

// ExampleDefaultConfig demonstrates getting default configuration.
func ExampleDefaultConfig() {
	cfg := health.DefaultConfig()

	fmt.Printf("Port: %d\n", cfg.Port)
	fmt.Printf("LivenessPath: %s\n", cfg.LivenessPath)
	fmt.Printf("ReadinessPath: %s\n", cfg.ReadinessPath)
	fmt.Printf("StartupPath: %s\n", cfg.StartupPath)
	// Output:
	// Port: 9090
	// LivenessPath: /live
	// ReadinessPath: /ready
	// StartupPath: /startup
}
