package di

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// simpleStarter implements Starter but has no other methods.
type simpleStarter struct {
	started bool
}

func (s *simpleStarter) OnStart(ctx context.Context) error {
	s.started = true
	return nil
}

// simpleStopper implements Stopper but has no other methods.
type simpleStopper struct {
	stopped bool
}

func (s *simpleStopper) OnStop(ctx context.Context) error {
	s.stopped = true
	return nil
}

// valueStarter implements Starter on value receiver.
type valueStarter struct {
	started bool
}

func (s valueStarter) OnStart(ctx context.Context) error {
	return nil
}

func TestHasLifecycle_AutoDetection(t *testing.T) {
	t.Run("LazySingleton detects Starter interface", func(t *testing.T) {
		provider := func(_ *Container) (*simpleStarter, error) {
			return &simpleStarter{}, nil
		}
		// No explicit hooks provided
		svc := newLazySingleton("test", "*di.simpleStarter", provider, nil, nil)

		assert.True(t, svc.HasLifecycle(), "HasLifecycle should return true for service implementing Starter")
	})

	t.Run("LazySingleton detects Stopper interface", func(t *testing.T) {
		provider := func(_ *Container) (*simpleStopper, error) {
			return &simpleStopper{}, nil
		}
		// No explicit hooks provided
		svc := newLazySingleton("test", "*di.simpleStopper", provider, nil, nil)

		assert.True(t, svc.HasLifecycle(), "HasLifecycle should return true for service implementing Stopper")
	})

	t.Run("LazySingleton returns false for no lifecycle", func(t *testing.T) {
		type noLifecycle struct{}
		provider := func(_ *Container) (*noLifecycle, error) {
			return &noLifecycle{}, nil
		}
		svc := newLazySingleton("test", "*di.noLifecycle", provider, nil, nil)

		assert.False(t, svc.HasLifecycle(), "HasLifecycle should return false for service with no lifecycle")
	})
}

// doubleLifecycle implements both Starter and Stopper.
type doubleLifecycle struct {
	started      bool
	stopped      bool
	startHookRun bool
	stopHookRun  bool
}

func (s *doubleLifecycle) OnStart(ctx context.Context) error {
	s.started = true
	return nil
}

func (s *doubleLifecycle) OnStop(ctx context.Context) error {
	s.stopped = true
	return nil
}

func TestLifecycle_ExplicitOverridesImplicit(t *testing.T) {
	instance := &doubleLifecycle{}

	// Explicit hooks that track execution
	startHook := func(ctx context.Context, i any) error {
		s := i.(*doubleLifecycle)
		s.startHookRun = true
		return nil
	}

	stopHook := func(ctx context.Context, i any) error {
		s := i.(*doubleLifecycle)
		s.stopHookRun = true
		return nil
	}

	// Register service with BOTH implicit interface and explicit hooks
	// We use newInstanceService here to simplify test setup, as baseService logic is shared
	svc := newInstanceService(
		"double",
		"*di.doubleLifecycle",
		instance,
		[]func(context.Context, any) error{startHook},
		[]func(context.Context, any) error{stopHook},
	)

	ctx := context.Background()

	// Test Start
	err := svc.Start(ctx)
	assert.NoError(t, err)

	assert.True(t, instance.startHookRun, "Explicit start hook should have run")
	assert.False(t, instance.started, "Implicit OnStart should NOT have run")

	// Test Stop
	err = svc.Stop(ctx)
	assert.NoError(t, err)

	assert.True(t, instance.stopHookRun, "Explicit stop hook should have run")
	assert.False(t, instance.stopped, "Implicit OnStop should NOT have run")
}
