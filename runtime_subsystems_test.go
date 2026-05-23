package gaz

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/eventbus"
)

func TestNewRuntimeSubsystems_RegistersEventBusInDI(t *testing.T) {
	container := NewContainer()

	subs, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.NoError(t, err)
	require.NotNil(t, subs.eventBus)

	require.NoError(t, container.Build())

	resolved, resolveErr := Resolve[*eventbus.EventBus](container)
	require.NoError(t, resolveErr)
	assert.Same(t, subs.eventBus, resolved)
}

func TestNewRuntimeSubsystems_NilLoggerFallback(t *testing.T) {
	container := NewContainer()

	subs, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          nil,
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.NoError(t, err)
	require.NotNil(t, subs.workerMgr)
	require.NotNil(t, subs.scheduler)
	require.NotNil(t, subs.eventBus)
}

func TestNewRuntimeSubsystems_SchedulerContextCancellable(t *testing.T) {
	container := NewContainer()

	subs, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.NoError(t, err)

	select {
	case <-subs.cronCtx.Done():
		t.Fatal("context should not be cancelled yet")
	default:
	}

	subs.cronCancel()

	select {
	case <-subs.cronCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("context should be cancelled after cronCancel()")
	}
}

func TestCriticalFailHandler_TriggersShutdown(t *testing.T) {
	var stopCalled atomic.Bool

	handler := criticalFailHandler(
		slog.Default(),
		5*time.Second,
		func(ctx context.Context) error {
			stopCalled.Store(true)
			deadline, ok := ctx.Deadline()
			if ok {
				assert.WithinDuration(t, time.Now().Add(5*time.Second), deadline, time.Second)
			}
			return nil
		},
	)

	handler()

	require.Eventually(t, stopCalled.Load, 2*time.Second, 10*time.Millisecond)
}

func TestNewRuntimeSubsystems_DuplicateEventBusReturnsError(t *testing.T) {
	container := NewContainer()
	require.NoError(t, For[*eventbus.EventBus](container).Instance(eventbus.New(slog.Default())))

	_, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.ErrorIs(t, err, ErrDIDuplicate)
}

func TestNewRuntimeSubsystems_NilContainerReturnsError(t *testing.T) {
	_, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       nil,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "container is required")
}

func TestNewRuntimeSubsystems_NilStopFuncReturnsError(t *testing.T) {
	_, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       NewContainer(),
		shutdownTimeout: 5 * time.Second,
		stopFunc:        nil,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "stop function is required")
}
