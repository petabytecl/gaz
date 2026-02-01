package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMockWorker(t *testing.T) {
	m := NewMockWorker()

	assert.Equal(t, "mock-worker", m.Name())
	assert.NoError(t, m.OnStart(context.Background()))
	assert.NoError(t, m.OnStop(context.Background()))

	m.AssertCalled(t, "Name")
	m.AssertCalled(t, "OnStart", mock.Anything)
	m.AssertCalled(t, "OnStop", mock.Anything)
}

func TestMockWorkerNamed(t *testing.T) {
	m := NewMockWorkerNamed("custom-worker")

	assert.Equal(t, "custom-worker", m.Name())
	assert.NoError(t, m.OnStart(context.Background()))
	assert.NoError(t, m.OnStop(context.Background()))
}

func TestMockWorkerCustomExpectations(t *testing.T) {
	m := &MockWorker{}
	testErr := errors.New("start failed")

	m.On("Name").Return("failing-worker")
	m.On("OnStart", mock.Anything).Return(testErr)
	m.On("OnStop", mock.Anything).Return(nil)

	assert.Equal(t, "failing-worker", m.Name())
	assert.ErrorIs(t, m.OnStart(context.Background()), testErr)
}

func TestSimpleWorker(t *testing.T) {
	w := NewSimpleWorker("test-worker")

	assert.Equal(t, "test-worker", w.Name())
	assert.False(t, w.Started.Load(), "should not be started initially")
	assert.False(t, w.Stopped.Load(), "should not be stopped initially")

	require.NoError(t, w.OnStart(context.Background()))
	assert.True(t, w.Started.Load(), "should be started after OnStart")

	require.NoError(t, w.OnStop(context.Background()))
	assert.True(t, w.Stopped.Load(), "should be stopped after OnStop")
}

func TestSimpleWorkerWithErrors(t *testing.T) {
	w := NewSimpleWorker("error-worker")
	startErr := errors.New("start error")
	stopErr := errors.New("stop error")

	w.StartErr = startErr
	w.StopErr = stopErr

	assert.ErrorIs(t, w.OnStart(context.Background()), startErr)
	assert.ErrorIs(t, w.OnStop(context.Background()), stopErr)
}

func TestTestManager(t *testing.T) {
	// Test with nil logger
	mgr := TestManager(nil)
	require.NotNil(t, mgr)

	// Should be able to register workers
	w := NewSimpleWorker("test")
	require.NoError(t, mgr.Register(w))
}

func TestRequireWorkerStarted(t *testing.T) {
	w := NewSimpleWorker("test")
	w.Started.Store(true)
	RequireWorkerStarted(t, w) // Should not fail
}

func TestRequireWorkerStopped(t *testing.T) {
	w := NewSimpleWorker("test")
	w.Stopped.Store(true)
	RequireWorkerStopped(t, w) // Should not fail
}

func TestRequireWorkerNotStarted(t *testing.T) {
	w := NewSimpleWorker("test")
	RequireWorkerNotStarted(t, w) // Should not fail - worker not started
}

func TestRequireWorkerNotStopped(t *testing.T) {
	w := NewSimpleWorker("test")
	RequireWorkerNotStopped(t, w) // Should not fail - worker not stopped
}
