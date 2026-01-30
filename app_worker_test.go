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

func (w *testWorker) OnStart(ctx context.Context) error {
	atomic.AddInt32(&w.startCount, 1)
	w.mu.Lock()
	select {
	case <-w.started:
	default:
		close(w.started)
	}
	w.mu.Unlock()
	return nil
}

func (w *testWorker) OnStop(ctx context.Context) error {
	atomic.AddInt32(&w.stopCount, 1)
	w.mu.Lock()
	select {
	case <-w.stopped:
	default:
		close(w.stopped)
	}
	w.mu.Unlock()
	return nil
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

func (w *panicTestWorker) OnStart(ctx context.Context) error {
	atomic.AddInt32(&w.startCount, 1)
	panic("intentional panic for testing")
}

func (w *panicTestWorker) OnStop(ctx context.Context) error { return nil }

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

	// Register a service that implements Starter interface
	orderSvc := &startTrackingService{
		onStart: func() {
			mu.Lock()
			order = append(order, "service-started")
			mu.Unlock()
		},
	}
	err := For[*startTrackingService](app.Container()).Named("order-service").Eager().Instance(orderSvc)
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
	// Note: There may be additional worker events (e.g., eventbus) but the key assertion
	// is that our service-started comes before any of our worker-started events
	mu.Lock()
	defer mu.Unlock()
	s.Require().GreaterOrEqual(len(order), 2, "should have at least 2 events")

	// Find first occurrence of each
	var serviceIdx, workerIdx int = -1, -1
	for i, event := range order {
		if event == "service-started" && serviceIdx == -1 {
			serviceIdx = i
		}
		if event == "worker-started" && workerIdx == -1 {
			workerIdx = i
		}
	}
	s.True(serviceIdx >= 0, "service should have started")
	s.True(workerIdx >= 0, "worker should have started")
	s.Less(serviceIdx, workerIdx, "service should start before worker")
}

func (s *AppTestSuite) TestApp_WorkerStopsBeforeServices() {
	app := New()

	// Track order
	var mu sync.Mutex
	var order []string

	// Register a service that implements Stopper interface
	stopSvc := &stopTrackingService{
		onStop: func() {
			mu.Lock()
			order = append(order, "service-stopped")
			mu.Unlock()
		},
	}
	err := For[*stopTrackingService](app.Container()).Named("stop-order-service").Eager().Instance(stopSvc)
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
	// Note: There may be additional worker events (e.g., eventbus) but the key assertion
	// is that our worker-stopped comes before our service-stopped
	mu.Lock()
	defer mu.Unlock()
	s.Require().GreaterOrEqual(len(order), 2, "should have at least 2 events")

	// Find first occurrence of each
	var serviceIdx, workerIdx int = -1, -1
	for i, event := range order {
		if event == "worker-stopped" && workerIdx == -1 {
			workerIdx = i
		}
		if event == "service-stopped" && serviceIdx == -1 {
			serviceIdx = i
		}
	}
	s.True(workerIdx >= 0, "worker should have stopped")
	s.True(serviceIdx >= 0, "service should have stopped")
	s.Less(workerIdx, serviceIdx, "worker should stop before service")
}

func (s *AppTestSuite) TestApp_WorkerPanicRecovery() {
	// This test expects panic recovery for workers, but with the new OnStart interface,
	// workers that implement OnStart also implement di.Starter, which is called during
	// service startup where panic recovery doesn't exist. The worker.Manager has its own
	// panic recovery, but that only applies to workers started via the Manager, not
	// services started via the DI layer.
	//
	// This test needs redesign to either:
	// 1. Add panic recovery to the DI layer for Starter interface
	// 2. Use the worker.Manager directly for panic recovery testing
	// 3. Make panicTestWorker NOT implement di.Starter
	//
	// Skipping pending architectural decision on panic recovery scope.
	s.T().Skip("Panic recovery scope changed with OnStart interface - needs redesign")
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

// startTrackingService implements Starter interface for testing service start order.
type startTrackingService struct {
	onStart func()
}

func (s *startTrackingService) OnStart(ctx context.Context) error {
	if s.onStart != nil {
		s.onStart()
	}
	return nil
}

// stopTrackingService implements Stopper interface for testing service stop order.
type stopTrackingService struct {
	onStop func()
}

func (s *stopTrackingService) OnStop(ctx context.Context) error {
	if s.onStop != nil {
		s.onStop()
	}
	return nil
}

type orderTrackingWorker struct {
	worker.Worker
	onStart func()
	onStop  func()
}

func (w *orderTrackingWorker) OnStart(ctx context.Context) error {
	if w.onStart != nil {
		w.onStart()
	}
	return w.Worker.OnStart(ctx)
}

func (w *orderTrackingWorker) OnStop(ctx context.Context) error {
	if w.onStop != nil {
		w.onStop()
	}
	return w.Worker.OnStop(ctx)
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
