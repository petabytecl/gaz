package gaz

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/worker"
)

// =============================================================================
// Worker Test Helpers
// =============================================================================

// testWorker is a minimal Worker implementation for App integration tests.
type testWorker struct {
	name    string
	started chan struct{}
	stopped chan struct{}
	stopCh  chan struct{}
	mu      sync.Mutex

	startCount int32
	stopCount  int32
}

func newTestWorker(name string) *testWorker {
	return &testWorker{
		name:    name,
		started: make(chan struct{}),
		stopped: make(chan struct{}),
		stopCh:  make(chan struct{}),
	}
}

func (w *testWorker) Start() {
	atomic.AddInt32(&w.startCount, 1)
	w.mu.Lock()
	select {
	case <-w.started:
	default:
		close(w.started)
	}
	w.mu.Unlock()
}

func (w *testWorker) Stop() {
	atomic.AddInt32(&w.stopCount, 1)
	w.mu.Lock()
	select {
	case <-w.stopped:
	default:
		close(w.stopped)
	}
	w.mu.Unlock()
}

func (w *testWorker) Name() string {
	return w.name
}

func (w *testWorker) getStartCount() int {
	return int(atomic.LoadInt32(&w.startCount))
}

func (w *testWorker) getStopCount() int {
	return int(atomic.LoadInt32(&w.stopCount))
}

// panicTestWorker panics on start for testing panic recovery.
type panicTestWorker struct {
	name       string
	startCount int32
}

func (w *panicTestWorker) Start() {
	atomic.AddInt32(&w.startCount, 1)
	panic("intentional panic for testing")
}

func (w *panicTestWorker) Stop() {}

func (w *panicTestWorker) Name() string { return w.name }

// =============================================================================
// App Worker Integration Tests
// =============================================================================

func (s *AppTestSuite) TestApp_WorkerAutoDiscovery() {
	app := New()

	// Register a worker via For[T]
	testW := newTestWorker("auto-discovery-worker")
	err := For[*testWorker](app.Container()).Named("test-worker").Instance(testW)
	s.Require().NoError(err)

	// Build should discover the worker
	err = app.Build()
	s.Require().NoError(err)

	// Run in goroutine
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for worker to start
	select {
	case <-testW.started:
	case <-time.After(2 * time.Second):
		s.Fail("worker did not start")
	}

	s.Equal(1, testW.getStartCount(), "worker should have been started once")

	// Stop the app
	err = app.Stop(context.Background())
	s.Require().NoError(err)

	// Wait for Run to return
	select {
	case runResult := <-runErr:
		s.Require().NoError(runResult)
	case <-time.After(2 * time.Second):
		s.Fail("Run did not return after Stop")
	}

	s.Equal(1, testW.getStopCount(), "worker should have been stopped once")
}

func (s *AppTestSuite) TestApp_WorkerStartsAfterServices() {
	app := New()

	// Track order
	var mu sync.Mutex
	var order []string

	// Register a service with OnStart hook
	type OrderService struct{}
	err := For[*OrderService](app.Container()).Named("order-service").Eager().
		OnStart(func(_ context.Context, _ *OrderService) error {
			mu.Lock()
			order = append(order, "service-started")
			mu.Unlock()
			return nil
		}).
		ProviderFunc(func(_ *Container) *OrderService {
			return &OrderService{}
		})
	s.Require().NoError(err)

	// Register a worker
	testW := newTestWorker("order-worker")
	// Use a custom wrapper to track when worker starts
	workerWrapper := &orderTrackingWorker{
		Worker: testW,
		onStart: func() {
			mu.Lock()
			order = append(order, "worker-started")
			mu.Unlock()
		},
	}
	err = For[*orderTrackingWorker](app.Container()).Named("order-worker").Instance(workerWrapper)
	s.Require().NoError(err)

	// Run in goroutine
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for worker to start
	select {
	case <-testW.started:
	case <-time.After(2 * time.Second):
		s.Fail("worker did not start")
	}

	// Stop the app
	s.Require().NoError(app.Stop(context.Background()))

	select {
	case <-runErr:
	case <-time.After(2 * time.Second):
		s.Fail("Run did not return")
	}

	// Verify order: services start before workers
	mu.Lock()
	defer mu.Unlock()
	s.Require().Len(order, 2)
	s.Equal("service-started", order[0], "service should start before worker")
	s.Equal("worker-started", order[1], "worker should start after service")
}

func (s *AppTestSuite) TestApp_WorkerStopsBeforeServices() {
	app := New()

	// Track order
	var mu sync.Mutex
	var order []string

	// Register a service with OnStop hook
	type StopOrderService struct{}
	err := For[*StopOrderService](app.Container()).Named("stop-order-service").Eager().
		OnStop(func(_ context.Context, _ *StopOrderService) error {
			mu.Lock()
			order = append(order, "service-stopped")
			mu.Unlock()
			return nil
		}).
		ProviderFunc(func(_ *Container) *StopOrderService {
			return &StopOrderService{}
		})
	s.Require().NoError(err)

	// Register a worker
	testW := newTestWorker("stop-order-worker")
	workerWrapper := &orderTrackingWorker{
		Worker: testW,
		onStop: func() {
			mu.Lock()
			order = append(order, "worker-stopped")
			mu.Unlock()
		},
	}
	err = For[*orderTrackingWorker](app.Container()).Named("stop-order-worker").Instance(workerWrapper)
	s.Require().NoError(err)

	// Run in goroutine
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for worker to start
	select {
	case <-testW.started:
	case <-time.After(2 * time.Second):
		s.Fail("worker did not start")
	}

	// Stop the app
	s.Require().NoError(app.Stop(context.Background()))

	select {
	case <-runErr:
	case <-time.After(2 * time.Second):
		s.Fail("Run did not return")
	}

	// Verify order: workers stop before services
	mu.Lock()
	defer mu.Unlock()
	s.Require().Len(order, 2)
	s.Equal("worker-stopped", order[0], "worker should stop before service")
	s.Equal("service-stopped", order[1], "service should stop after worker")
}

func (s *AppTestSuite) TestApp_WorkerPanicRecovery() {
	app := New()

	// Register a panicking worker
	panicW := &panicTestWorker{name: "panic-test-worker"}
	err := For[*panicTestWorker](app.Container()).Named("panic-worker").Instance(panicW)
	s.Require().NoError(err)

	// Run in goroutine
	runErr := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		runErr <- app.Run(ctx)
	}()

	// Wait for context timeout or stop
	select {
	case err := <-runErr:
		// App should complete without crashing
		s.NoError(err, "app should not crash due to worker panic")
	case <-time.After(5 * time.Second):
		s.Fail("Run did not return")
	}

	// Worker should have been started at least once (panic is recovered)
	s.GreaterOrEqual(int(atomic.LoadInt32(&panicW.startCount)), 1, "worker should have started")
}

func (s *AppTestSuite) TestApp_CriticalWorkerShutdown() {
	// This test verifies that when a critical worker fails, the app shuts down.
	// However, testing this is complex because we need to configure the worker
	// with WithCritical() option, which requires using the manager directly.
	// The current App integration doesn't expose this configuration.
	// This test is skipped pending future API enhancement.
	s.T().Skip("Critical worker API not exposed via App yet")
}

// =============================================================================
// Helper types for order tracking
// =============================================================================

type orderTrackingWorker struct {
	worker.Worker
	onStart func()
	onStop  func()
}

func (w *orderTrackingWorker) Start() {
	if w.onStart != nil {
		w.onStart()
	}
	w.Worker.Start()
}

func (w *orderTrackingWorker) Stop() {
	if w.onStop != nil {
		w.onStop()
	}
	w.Worker.Stop()
}

// =============================================================================
// Standalone Test Functions (not in suite)
// =============================================================================

func TestAppWorker_AutoDiscoveryDuringBuild(t *testing.T) {
	app := New()

	testW := newTestWorker("standalone-worker")
	err := For[*testWorker](app.Container()).Named("standalone").Instance(testW)
	require.NoError(t, err)

	// Build should discover the worker
	err = app.Build()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(ctx)
	}()

	// Wait for worker to start
	select {
	case <-testW.started:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not start")
	}

	assert.Equal(t, 1, testW.getStartCount())

	cancel()

	select {
	case <-runErr:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}
}

func TestAppWorker_MultipleWorkersStartConcurrently(t *testing.T) {
	app := New()

	workers := make([]*testWorker, 3)
	for i := range 3 {
		workers[i] = newTestWorker("multi-worker")
		err := For[*testWorker](app.Container()).Named("worker-" + string(rune('A'+i))).Instance(workers[i])
		require.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(ctx)
	}()

	// Wait for all workers to start
	for i, w := range workers {
		select {
		case <-w.started:
		case <-time.After(2 * time.Second):
			t.Fatalf("worker %d did not start", i)
		}
	}

	// All should have started
	for i, w := range workers {
		assert.Equal(t, 1, w.getStartCount(), "worker %d should have started", i)
	}

	cancel()

	select {
	case <-runErr:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}
}
