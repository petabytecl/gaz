package worker

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/mock"
)

// MockWorker is a testify mock implementing worker.Worker.
// Use NewMockWorker() or NewMockWorkerNamed() for pre-configured instances.
type MockWorker struct {
	mock.Mock
}

// NewMockWorker creates a MockWorker with default expectations.
// Name returns "mock-worker", OnStart/OnStop return nil.
func NewMockWorker() *MockWorker {
	m := &MockWorker{}
	m.On("Name").Return("mock-worker")
	m.On("OnStart", mock.Anything).Return(nil)
	m.On("OnStop", mock.Anything).Return(nil)
	return m
}

// NewMockWorkerNamed creates a MockWorker with a custom name.
// OnStart/OnStop return nil.
func NewMockWorkerNamed(name string) *MockWorker {
	m := &MockWorker{}
	m.On("Name").Return(name)
	m.On("OnStart", mock.Anything).Return(nil)
	m.On("OnStop", mock.Anything).Return(nil)
	return m
}

// Name returns the mock worker's name.
func (m *MockWorker) Name() string {
	args := m.Called()
	return args.String(0)
}

// OnStart records the start call and returns the mocked error.
func (m *MockWorker) OnStart(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// OnStop records the stop call and returns the mocked error.
func (m *MockWorker) OnStop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// SimpleWorker is a test worker that tracks OnStart/OnStop calls.
// Useful for cases where mock complexity isn't needed.
type SimpleWorker struct {
	name     string
	Started  atomic.Bool
	Stopped  atomic.Bool
	StartErr error
	StopErr  error
}

// NewSimpleWorker creates a SimpleWorker with the given name.
func NewSimpleWorker(name string) *SimpleWorker {
	return &SimpleWorker{name: name}
}

// Name returns the worker's name.
func (w *SimpleWorker) Name() string { return w.name }

// OnStart marks the worker as started and returns StartErr.
func (w *SimpleWorker) OnStart(_ context.Context) error {
	w.Started.Store(true)
	return w.StartErr
}

// OnStop marks the worker as stopped and returns StopErr.
func (w *SimpleWorker) OnStop(_ context.Context) error {
	w.Stopped.Store(true)
	return w.StopErr
}

// TestManager creates a worker.Manager suitable for testing.
// If logger is nil, a discard logger is used.
func TestManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return NewManager(logger)
}

// RequireWorkerStarted asserts the worker's OnStart was called.
func RequireWorkerStarted(tb testing.TB, w *SimpleWorker) {
	tb.Helper()
	if !w.Started.Load() {
		tb.Fatalf("expected worker %q to be started", w.Name())
	}
}

// RequireWorkerStopped asserts the worker's OnStop was called.
func RequireWorkerStopped(tb testing.TB, w *SimpleWorker) {
	tb.Helper()
	if !w.Stopped.Load() {
		tb.Fatalf("expected worker %q to be stopped", w.Name())
	}
}

// RequireWorkerNotStarted asserts the worker's OnStart was NOT called.
func RequireWorkerNotStarted(tb testing.TB, w *SimpleWorker) {
	tb.Helper()
	if w.Started.Load() {
		tb.Fatalf("expected worker %q to NOT be started", w.Name())
	}
}

// RequireWorkerNotStopped asserts the worker's OnStop was NOT called.
func RequireWorkerNotStopped(tb testing.TB, w *SimpleWorker) {
	tb.Helper()
	if w.Stopped.Load() {
		tb.Fatalf("expected worker %q to NOT be stopped", w.Name())
	}
}
