package gaz

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/worker"
)

func TestNewRuntimeSubsystems_RegistersRuntimeParticipantsInDI(t *testing.T) {
	container := NewContainer()

	subs, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.NoError(t, err)
	require.NotNil(t, subs.workerMgr)
	require.NotNil(t, subs.scheduler)
	require.NotNil(t, subs.eventBus)

	require.NoError(t, container.Build())

	resolvedManager, resolveManagerErr := Resolve[*worker.Manager](container)
	require.NoError(t, resolveManagerErr)
	assert.Same(t, subs.workerMgr, resolvedManager)

	resolvedScheduler, resolveSchedulerErr := Resolve[*cron.Scheduler](container)
	require.NoError(t, resolveSchedulerErr)
	assert.Same(t, subs.scheduler, resolvedScheduler)

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

func TestNewRuntimeSubsystems_ReusesRegisteredWorkerManager(t *testing.T) {
	container := NewContainer()
	existing := worker.NewManager(slog.Default())
	require.NoError(t, For[*worker.Manager](container).Instance(existing))

	subs, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.NoError(t, err)
	require.Same(t, existing, subs.workerMgr)

	require.NoError(t, container.Build())

	resolved, resolveErr := Resolve[*worker.Manager](container)
	require.NoError(t, resolveErr)
	assert.Same(t, existing, resolved)
}

func TestNewRuntimeSubsystems_RegisteredWorkerManagerResolveError(t *testing.T) {
	container := NewContainer()
	resolveErr := errors.New("worker manager provider failed")
	require.NoError(t, For[*worker.Manager](container).Provider(
		func(*Container) (*worker.Manager, error) {
			return nil, resolveErr
		},
	))

	_, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.ErrorIs(t, err, resolveErr)
	require.Contains(t, err.Error(), "resolve worker manager")
}

func TestNewRuntimeSubsystems_ReplacesRegisteredCronScheduler(t *testing.T) {
	container := NewContainer()
	existing := cron.NewScheduler(container, context.Background(), slog.Default())
	require.NoError(t, For[*cron.Scheduler](container).Instance(existing))

	subs, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.NoError(t, err)
	require.NotSame(t, existing, subs.scheduler)

	require.NoError(t, container.Build())

	resolved, resolveErr := Resolve[*cron.Scheduler](container)
	require.NoError(t, resolveErr)
	assert.Same(t, subs.scheduler, resolved)
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

func TestNewRuntimeSubsystems_ReusesRegisteredEventBus(t *testing.T) {
	container := NewContainer()
	existing := eventbus.New(slog.Default())
	defer existing.Close()

	require.NoError(t, For[*eventbus.EventBus](container).Instance(existing))

	subs, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.NoError(t, err)
	require.Same(t, existing, subs.eventBus)

	require.NoError(t, container.Build())

	resolved, resolveErr := Resolve[*eventbus.EventBus](container)
	require.NoError(t, resolveErr)
	assert.Same(t, existing, resolved)
}

func TestNewRuntimeSubsystems_RegisteredEventBusResolveError(t *testing.T) {
	container := NewContainer()
	resolveErr := errors.New("eventbus provider failed")
	require.NoError(t, For[*eventbus.EventBus](container).Provider(
		func(*Container) (*eventbus.EventBus, error) {
			return nil, resolveErr
		},
	))

	_, err := newRuntimeSubsystems(runtimeSubsystemsDeps{
		logger:          slog.Default(),
		container:       container,
		shutdownTimeout: 5 * time.Second,
		stopFunc:        func(context.Context) error { return nil },
	})
	require.ErrorIs(t, err, resolveErr)
	require.Contains(t, err.Error(), "resolve eventbus")
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
