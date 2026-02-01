package gaztest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/gaztest"
	"github.com/petabytecl/gaz/health"
	"github.com/petabytecl/gaz/worker"
)

// =============================================================================
// V3 Pattern Examples - WithModules, RequireResolve, Subsystem Helpers
// =============================================================================

// Example_withModules demonstrates the v3 WithModules pattern for testing modules.
// WithModules registers modules with the test app during build,
// enabling clean integration tests.
//
// Note: This example is for documentation purposes. In a real test,
// use *testing.T instead of a mock.
func Example_withModules() {
	t := &testing.T{}

	// Create a simple module
	module := di.NewModuleFunc("example", func(c *di.Container) error {
		return di.For[string](c).Instance("hello from module")
	})

	// Build test app with module
	app, err := gaztest.New(t).
		WithModules(module).
		Build()
	if err != nil {
		fmt.Println("build failed:", err)
		return
	}

	app.RequireStart()
	defer app.RequireStop()

	// Module's service is available
	// In real code: msg := gaztest.RequireResolve[string](t, app)
	_ = app // Demonstrate pattern - avoid Output due to log noise
}

// Example_requireResolve demonstrates the v3 RequireResolve pattern.
// RequireResolve provides type-safe resolution that fails the test on error,
// eliminating manual error checking.
//
// Note: This example is for documentation purposes. In a real test,
// use *testing.T instead of a mock.
func Example_requireResolve() {
	t := &testing.T{}

	// Register a service
	type DatabaseConfig struct {
		Host string
		Port int
	}

	baseApp := gaz.New()
	_ = gaz.For[*DatabaseConfig](baseApp.Container()).Instance(&DatabaseConfig{Host: "localhost", Port: 5432})
	_ = baseApp.Build()

	app, err := gaztest.New(t).WithApp(baseApp).Build()
	if err != nil {
		fmt.Println("build failed:", err)
		return
	}

	app.RequireStart()
	defer app.RequireStop()

	// RequireResolve fails test if resolution fails - no error check needed
	// cfg := gaztest.RequireResolve[*DatabaseConfig](t, app)
	// fmt.Println(cfg.Host) // "localhost"
	_ = app // Demonstrate pattern - avoid Output due to log noise
}

// Example_subsystemHelpers demonstrates using subsystem test helpers.
// Each subsystem (health, worker, cron, config, eventbus) provides
// testing.go helpers for creating test instances.
func Example_subsystemHelpers() {
	// Health subsystem - safe defaults with port 0 for random available
	healthCfg := health.TestConfig()
	_ = healthCfg

	// Worker subsystem - mock and simple workers
	w := worker.NewSimpleWorker("background-task")
	_ = w

	// Cron subsystem - mock and simple jobs
	job := cron.NewSimpleJob("daily-report", "@daily")
	_ = job

	// EventBus subsystem - test bus with discard logger
	bus := eventbus.TestBus()
	defer bus.Close()

	// Config subsystem - in-memory config manager
	mgr := config.TestManager(map[string]any{
		"app.debug": true,
	})
	_ = mgr

	fmt.Println("All subsystem helpers available")
	// Output: All subsystem helpers available
}

// Example_withConfigMap demonstrates injecting test configuration values.
// WithConfigMap allows tests to override configuration without file I/O.
//
// Note: This example is for documentation purposes. In a real test,
// use *testing.T instead of a mock.
func Example_withConfigMap() {
	t := &testing.T{}

	app, err := gaztest.New(t).
		WithConfigMap(map[string]any{
			"worker.pool_size": 2,
			"health.port":      0,
		}).
		Build()
	if err != nil {
		fmt.Println("build failed:", err)
		return
	}

	app.RequireStart()
	defer app.RequireStop()

	_ = app // Demonstrate pattern - avoid Output due to log noise
}

// =============================================================================
// Full Integration Test Examples
// =============================================================================

// ExampleTestService is a sample service for integration test examples.
type ExampleTestService struct {
	Name string
}

// TestExample_IntegrationTest demonstrates a complete integration test pattern.
// This pattern uses WithModules for module registration and RequireResolve
// for type-safe service resolution.
func TestExample_IntegrationTest(t *testing.T) {
	// Create module under test
	module := di.NewModuleFunc("test-module", func(c *di.Container) error {
		return di.For[*ExampleTestService](c).Provider(func(_ *di.Container) (*ExampleTestService, error) {
			return &ExampleTestService{Name: "integration"}, nil
		})
	})

	app, err := gaztest.New(t).
		WithModules(module).
		Build()
	require.NoError(t, err)

	app.RequireStart()
	defer app.RequireStop()

	// Use RequireResolve for type-safe resolution that fails on error
	svc := gaztest.RequireResolve[*ExampleTestService](t, app)
	require.Equal(t, "integration", svc.Name)
}

// TestExample_WorkerWithSimpleWorker demonstrates testing workers with SimpleWorker.
// SimpleWorker tracks OnStart/OnStop calls without mock complexity.
func TestExample_WorkerWithSimpleWorker(t *testing.T) {
	// Create a simple worker for testing
	w := worker.NewSimpleWorker("test-worker")

	// Create manager with discard logger
	mgr := worker.TestManager(nil)
	require.NoError(t, mgr.Register(w))

	// Start the manager (workers start asynchronously)
	ctx := context.Background()
	require.NoError(t, mgr.Start(ctx))

	// Wait for worker to start (polling since async)
	require.Eventually(t, func() bool {
		return w.Started.Load()
	}, time.Second, 10*time.Millisecond, "worker should start")

	// Stop and verify
	require.NoError(t, mgr.Stop())
	worker.RequireWorkerStopped(t, w)
}

// TestExample_CronWithSimpleJob demonstrates testing cron jobs with SimpleJob.
// SimpleJob tracks Run calls and can be executed manually for testing.
func TestExample_CronWithSimpleJob(t *testing.T) {
	// Create a simple job for testing
	job := cron.NewSimpleJob("test-job", "@every 1s")

	// Manually invoke Run to test job logic without waiting for schedule
	err := job.Run(context.Background())
	require.NoError(t, err)

	// Verify job ran
	cron.RequireJobRan(t, job)
	cron.RequireJobRunCount(t, job, 1)
}

// TestEvent is a sample event for eventbus testing.
type TestEvent struct {
	ID string
}

// EventName implements eventbus.Event.
func (e TestEvent) EventName() string { return "TestEvent" }

// TestExample_EventBusWithTestSubscriber demonstrates async event testing.
// TestSubscriber provides synchronization helpers for async event delivery.
func TestExample_EventBusWithTestSubscriber(t *testing.T) {
	bus := eventbus.TestBus()
	defer bus.Close()

	// Create subscriber expecting 2 events
	sub := eventbus.NewTestSubscriber[TestEvent](2)
	eventbus.Subscribe(bus, sub.Handler())

	// Publish events
	ctx := context.Background()
	eventbus.Publish(ctx, bus, TestEvent{ID: "1"}, "")
	eventbus.Publish(ctx, bus, TestEvent{ID: "2"}, "")

	// Wait for async delivery with timeout
	eventbus.RequireEventsReceived(t, sub, time.Second)
	eventbus.RequireEventCount(t, sub, 2)

	// Verify event content
	events := sub.Events()
	require.Equal(t, "1", events[0].ID)
	require.Equal(t, "2", events[1].ID)
}

// TestExample_ConfigWithMapBackend demonstrates testing configuration loading.
// TestManager creates an in-memory config manager for testing without file I/O.
func TestExample_ConfigWithMapBackend(t *testing.T) {
	mgr := config.TestManager(map[string]any{
		"server.host": "localhost",
		"server.port": 9090,
	})

	// Verify values
	config.RequireConfigString(t, mgr.Backend(), "server.host", "localhost")
	config.RequireConfigInt(t, mgr.Backend(), "server.port", 9090)
}

// TestExample_HealthWithTestConfig demonstrates testing health checks.
// TestConfig provides safe defaults (port 0) for parallel test execution.
func TestExample_HealthWithTestConfig(t *testing.T) {
	// TestConfig uses port 0 for random available port
	cfg := health.TestConfig()
	require.Equal(t, 0, cfg.Port)

	// Create manager for test assertions
	mgr := health.TestManager()
	mgr.AddReadinessCheck("test", func(ctx context.Context) error {
		return nil
	})

	// Verify health
	health.RequireHealthy(t, mgr)
	health.RequireReadinessCheckRegistered(t, mgr, "test")
}
