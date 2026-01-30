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
		svc := newLazySingleton("test", "*di.simpleStarter", provider)

		assert.True(t, svc.HasLifecycle(), "HasLifecycle should return true for service implementing Starter")
	})

	t.Run("LazySingleton detects Stopper interface", func(t *testing.T) {
		provider := func(_ *Container) (*simpleStopper, error) {
			return &simpleStopper{}, nil
		}
		// No explicit hooks provided
		svc := newLazySingleton("test", "*di.simpleStopper", provider)

		assert.True(t, svc.HasLifecycle(), "HasLifecycle should return true for service implementing Stopper")
	})

	t.Run("LazySingleton returns false for no lifecycle", func(t *testing.T) {
		type noLifecycle struct{}
		provider := func(_ *Container) (*noLifecycle, error) {
			return &noLifecycle{}, nil
		}
		svc := newLazySingleton("test", "*di.noLifecycle", provider)

		assert.False(t, svc.HasLifecycle(), "HasLifecycle should return false for service with no lifecycle")
	})
}
