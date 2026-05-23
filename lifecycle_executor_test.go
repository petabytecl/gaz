package gaz

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/di"
)

type executorServiceWrapper struct {
	name  string
	start func(context.Context) error
	stop  func(context.Context) error
}

func (s *executorServiceWrapper) Name() string      { return s.name }
func (s *executorServiceWrapper) TypeName() string  { return s.name }
func (s *executorServiceWrapper) IsEager() bool     { return false }
func (s *executorServiceWrapper) IsTransient() bool { return false }
func (s *executorServiceWrapper) GetInstance(
	_ *di.Container,
	_ []string,
) (any, error) {
	return nil, nil
}

func (s *executorServiceWrapper) Start(ctx context.Context) error {
	if s.start == nil {
		return nil
	}
	return s.start(ctx)
}

func (s *executorServiceWrapper) Stop(ctx context.Context) error {
	if s.stop == nil {
		return nil
	}
	return s.stop(ctx)
}
func (s *executorServiceWrapper) HasLifecycle() bool        { return true }
func (s *executorServiceWrapper) ServiceType() reflect.Type { return nil }
func (s *executorServiceWrapper) Groups() []string          { return nil }

func TestLifecycleExecutorStartRollsBackOnLayerFailure(t *testing.T) {
	startErr := errors.New("start failed")
	rollbackErr := errors.New("rollback failed")
	rollbackCalled := false

	executor := newLifecycleExecutor(lifecycleExecutorDeps{
		plan: &lifecyclePlan{
			services: map[string]di.ServiceWrapper{
				"A": &executorServiceWrapper{
					name:  "A",
					start: func(context.Context) error { return startErr },
				},
			},
			startupOrder:  [][]string{{"A"}},
			shutdownOrder: [][]string{{"A"}},
		},
		logger:          testLifecycleExecutorLogger(),
		shutdownTimeout: time.Second,
		perHookTimeout:  time.Second,
	})

	err := executor.Start(context.Background(), func(context.Context) error {
		rollbackCalled = true
		return rollbackErr
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "starting service A")
	require.ErrorIs(t, err, startErr)
	require.ErrorIs(t, err, rollbackErr)
	require.True(t, rollbackCalled)
}

func TestLifecycleExecutorStopUsesShutdownOrder(t *testing.T) {
	var stopped []string
	services := map[string]di.ServiceWrapper{
		"A": &executorServiceWrapper{
			name: "A",
			stop: func(context.Context) error {
				stopped = append(stopped, "A")
				return nil
			},
		},
		"B": &executorServiceWrapper{
			name: "B",
			stop: func(context.Context) error {
				stopped = append(stopped, "B")
				return nil
			},
		},
	}

	executor := newLifecycleExecutor(lifecycleExecutorDeps{
		plan: &lifecyclePlan{
			services:      services,
			shutdownOrder: [][]string{{"B"}, {"A"}},
		},
		logger:         testLifecycleExecutorLogger(),
		perHookTimeout: time.Second,
	})

	err := executor.Stop(context.Background())

	require.NoError(t, err)
	require.Equal(t, []string{"B", "A"}, stopped)
}

func testLifecycleExecutorLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
