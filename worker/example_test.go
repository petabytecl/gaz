package worker_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/worker"
)

// EmailWorker demonstrates implementing the Worker interface.
// Workers manage their own goroutine lifecycle.
type EmailWorker struct {
	running bool
	stop    chan struct{}
}

func (w *EmailWorker) Name() string { return "email-worker" }

func (w *EmailWorker) OnStart(ctx context.Context) error {
	w.running = true
	w.stop = make(chan struct{})
	fmt.Println("email worker started")
	return nil
}

func (w *EmailWorker) OnStop(ctx context.Context) error {
	w.running = false
	close(w.stop)
	fmt.Println("email worker stopped")
	return nil
}

// Example_worker demonstrates implementing the Worker interface.
// Workers define OnStart, OnStop, and Name methods for lifecycle management.
// OnStart should be non-blocking; the worker spawns its own goroutine internally.
func Example_worker() {
	w := &EmailWorker{}
	_ = w.OnStart(context.Background())
	_ = w.OnStop(context.Background())
	// Output:
	// email worker started
	// email worker stopped
}

// ExampleNewManager demonstrates creating a worker manager.
// The manager coordinates multiple workers with unified startup and shutdown.
func ExampleNewManager() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	_ = worker.NewManager(logger)

	fmt.Println("manager created")
	// Output: manager created
}

// ExampleManager_Register demonstrates registering workers with the manager.
// Workers can be registered with optional configuration like pool size.
func ExampleManager_Register() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mgr := worker.NewManager(logger)

	w := &EmailWorker{}

	err := mgr.Register(w)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("worker registered")
	// Output: worker registered
}

// ExampleManager_Start demonstrates starting all registered workers.
// Start spawns supervisor goroutines for each worker and returns immediately.
// Note: This example cannot show output because Start is non-blocking
// and workers run in separate goroutines.
func ExampleManager_Start() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mgr := worker.NewManager(logger)

	w := &EmailWorker{}
	_ = mgr.Register(w)

	ctx := context.Background()
	err := mgr.Start(ctx)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Stop immediately to cleanup (in real code, this would wait for shutdown signal)
	_ = mgr.Stop()

	fmt.Println("manager started and stopped")
	// Output: manager started and stopped
}

// ExampleModule demonstrates using the worker module for direct DI usage.
// The Module function registers worker infrastructure into a DI container.
func ExampleModule() {
	c := di.New()

	// Register logger (normally done by gaz.New())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_ = di.For[*slog.Logger](c).Instance(logger)

	// Apply worker module
	if err := worker.Module(c); err != nil {
		fmt.Println("error:", err)
		return
	}

	// Build and resolve
	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	mgr, err := di.Resolve[*worker.Manager](c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("manager: %T\n", mgr)
	// Output: manager: *worker.Manager
}

// Example_restartPolicy demonstrates configuring worker restart behavior.
// Workers can be configured with options like max restarts and circuit window.
func Example_restartPolicy() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mgr := worker.NewManager(logger)

	w := &EmailWorker{}

	// Register with custom restart policy:
	// - Max 3 restarts within 5 minute window
	// - Worker must run 1 minute to be considered stable
	err := mgr.Register(w,
		worker.WithMaxRestarts(3),
		worker.WithCircuitWindow(5*time.Minute),
		worker.WithStableRunPeriod(time.Minute),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("worker registered with restart policy")
	// Output: worker registered with restart policy
}

// ExampleWithPoolSize demonstrates creating multiple worker instances.
// Pool workers run the same work function in parallel for increased throughput.
func ExampleWithPoolSize() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mgr := worker.NewManager(logger)

	w := &EmailWorker{}

	// Create 4 instances of the worker (email-worker-1, email-worker-2, etc.)
	err := mgr.Register(w, worker.WithPoolSize(4))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("pool worker registered")
	// Output: pool worker registered
}

// ExampleWithCritical demonstrates marking a worker as critical.
// Critical workers trigger application shutdown if their circuit breaker trips.
func ExampleWithCritical() {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mgr := worker.NewManager(logger)

	w := &EmailWorker{}

	// Critical workers cause app shutdown if they exhaust restart attempts
	err := mgr.Register(w, worker.WithCritical())
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("critical worker registered")
	// Output: critical worker registered
}

// ExampleSimpleWorker demonstrates using SimpleWorker for testing.
// SimpleWorker tracks OnStart/OnStop calls without mock complexity.
func ExampleSimpleWorker() {
	w := worker.NewSimpleWorker("test-worker")

	fmt.Println("name:", w.Name())
	fmt.Println("started before OnStart:", w.Started.Load())

	_ = w.OnStart(context.Background())
	fmt.Println("started after OnStart:", w.Started.Load())

	_ = w.OnStop(context.Background())
	fmt.Println("stopped after OnStop:", w.Stopped.Load())
	// Output:
	// name: test-worker
	// started before OnStart: false
	// started after OnStart: true
	// stopped after OnStop: true
}

// ExampleMockWorker demonstrates using MockWorker for testing.
// MockWorker uses testify/mock for flexible expectation setup.
func ExampleMockWorker() {
	m := worker.NewMockWorker()

	// MockWorker comes with default expectations:
	// - Name() returns "mock-worker"
	// - OnStart/OnStop return nil
	fmt.Println("name:", m.Name())
	// Output: name: mock-worker
}

// ExampleNewMockWorkerNamed demonstrates creating a named mock worker.
// Named mocks are useful when testing with multiple workers.
func ExampleNewMockWorkerNamed() {
	m := worker.NewMockWorkerNamed("custom-worker")

	fmt.Println("name:", m.Name())
	// Output: name: custom-worker
}

// ExampleTestManager demonstrates creating a test manager.
// TestManager creates a Manager with a discard logger for tests.
// Note: This example omits Output verification since workers run async.
func ExampleTestManager() {
	mgr := worker.TestManager(nil)

	w := worker.NewSimpleWorker("test-worker")
	_ = mgr.Register(w)

	ctx := context.Background()
	_ = mgr.Start(ctx)
	_ = mgr.Stop()

	// Note: In real tests, use RequireWorkerStarted/RequireWorkerStopped
	// for assertions. The workers run asynchronously so we just show
	// the pattern here.
}

// Example_moduleIntegration demonstrates using Module with di.Container.
// This shows how worker module integrates into the DI system.
func Example_moduleIntegration() {
	c := di.New()

	// Register logger (normally done by gaz.New())
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_ = di.For[*slog.Logger](c).Instance(logger)

	// Apply worker module
	if err := worker.Module(c); err != nil {
		fmt.Println("error:", err)
		return
	}

	// Build and resolve
	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	mgr, err := di.Resolve[*worker.Manager](c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("manager resolved: %T\n", mgr)
	// Output: manager resolved: *worker.Manager
}
