package worker

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simpleWorker is a minimal Worker implementation for manager tests.
type simpleWorker struct {
	name       string
	startCount int32
	stopCount  int32
	started    chan struct{}
	stopped    chan struct{}
	mu         sync.Mutex
}

func newSimpleWorker(name string) *simpleWorker {
	return &simpleWorker{
		name:    name,
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (w *simpleWorker) Start() {
	atomic.AddInt32(&w.startCount, 1)
	w.mu.Lock()
	select {
	case <-w.started:
	default:
		close(w.started)
	}
	w.mu.Unlock()
}

func (w *simpleWorker) Stop() {
	atomic.AddInt32(&w.stopCount, 1)
	w.mu.Lock()
	select {
	case <-w.stopped:
	default:
		close(w.stopped)
	}
	w.mu.Unlock()
}

func (w *simpleWorker) Name() string {
	return w.name
}

func (w *simpleWorker) getStartCount() int {
	return int(atomic.LoadInt32(&w.startCount))
}

func (w *simpleWorker) getStopCount() int {
	return int(atomic.LoadInt32(&w.stopCount))
}

func TestManager_RegisterAndStart(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	worker := newSimpleWorker("test-worker")
	err := mgr.Register(worker)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = mgr.Start(ctx)
	require.NoError(t, err)

	// Wait for worker to start
	select {
	case <-worker.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}

	assert.Equal(t, 1, worker.getStartCount())

	// Stop the manager
	err = mgr.Stop()
	require.NoError(t, err)
}

func TestManager_Stop(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	worker1 := newSimpleWorker("worker-1")
	worker2 := newSimpleWorker("worker-2")

	require.NoError(t, mgr.Register(worker1))
	require.NoError(t, mgr.Register(worker2))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, mgr.Start(ctx))

	// Wait for workers to start
	select {
	case <-worker1.started:
	case <-time.After(time.Second):
		t.Fatal("worker1 did not start")
	}
	select {
	case <-worker2.started:
	case <-time.After(time.Second):
		t.Fatal("worker2 did not start")
	}

	// Stop all workers
	err := mgr.Stop()
	require.NoError(t, err)

	// Wait for done channel
	select {
	case <-mgr.Done():
	case <-time.After(time.Second):
		t.Fatal("manager done channel did not close")
	}

	assert.Equal(t, 1, worker1.getStopCount())
	assert.Equal(t, 1, worker2.getStopCount())
}

func TestManager_PoolWorkers(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	worker := newSimpleWorker("pool-worker")
	err := mgr.Register(worker, WithPoolSize(3))
	require.NoError(t, err)

	// Manager should have 3 supervisors now
	assert.Len(t, mgr.supervisors, 3)

	// Check names are indexed
	names := []string{
		mgr.supervisors[0].worker.Name(),
		mgr.supervisors[1].worker.Name(),
		mgr.supervisors[2].worker.Name(),
	}
	assert.Contains(t, names, "pool-worker-1")
	assert.Contains(t, names, "pool-worker-2")
	assert.Contains(t, names, "pool-worker-3")
}

func TestManager_Done(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	worker := newSimpleWorker("done-worker")
	require.NoError(t, mgr.Register(worker))

	ctx, cancel := context.WithCancel(context.Background())

	require.NoError(t, mgr.Start(ctx))

	// Wait for worker to start
	select {
	case <-worker.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}

	// Cancel context to stop
	cancel()

	// Done channel should close
	select {
	case <-mgr.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done channel did not close after context cancel")
	}
}

func TestManager_ConcurrentStart(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	// Register multiple workers
	workers := make([]*simpleWorker, 5)
	for i := 0; i < 5; i++ {
		workers[i] = newSimpleWorker("concurrent-" + string(rune('A'+i)))
		require.NoError(t, mgr.Register(workers[i]))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	require.NoError(t, mgr.Start(ctx))

	// Wait for all workers to start
	for _, w := range workers {
		select {
		case <-w.started:
		case <-time.After(time.Second):
			t.Fatalf("worker %s did not start", w.name)
		}
	}

	// All workers should have started very quickly (concurrently)
	elapsed := time.Since(start)
	assert.Less(t, elapsed, 500*time.Millisecond, "workers should start concurrently")

	mgr.Stop()
}

func TestManager_DoubleStart(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	worker := newSimpleWorker("double-start-worker")
	require.NoError(t, mgr.Register(worker))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// First start
	require.NoError(t, mgr.Start(ctx))

	// Wait for worker to start
	select {
	case <-worker.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}

	// Second start should be idempotent (no error)
	err := mgr.Start(ctx)
	assert.NoError(t, err, "second Start() should be idempotent")

	// Worker should only have been started once
	assert.Equal(t, 1, worker.getStartCount())

	mgr.Stop()
}

func TestManager_RegisterWhileRunning(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	worker1 := newSimpleWorker("first-worker")
	require.NoError(t, mgr.Register(worker1))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, mgr.Start(ctx))

	// Wait for first worker to start
	select {
	case <-worker1.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}

	// Try to register while running
	worker2 := newSimpleWorker("second-worker")
	err := mgr.Register(worker2)
	assert.Error(t, err, "registering while running should error")
	assert.ErrorIs(t, err, ErrManagerAlreadyRunning)

	mgr.Stop()
}

func TestManager_StopNotRunning(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	// Stop without starting should be fine
	err := mgr.Stop()
	assert.NoError(t, err)
}

func TestManager_SetCriticalFailHandler(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	called := false
	mgr.SetCriticalFailHandler(func() {
		called = true
	})

	// Trigger the handler manually (normally supervisor does this)
	mgr.handleCriticalFail()

	assert.True(t, called, "critical fail handler should have been called")
}

func TestManager_EmptyStart(t *testing.T) {
	logger := slog.Default()
	mgr := NewManager(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start with no workers should be fine
	err := mgr.Start(ctx)
	assert.NoError(t, err)

	// Stop should also be fine
	err = mgr.Stop()
	assert.NoError(t, err)
}
