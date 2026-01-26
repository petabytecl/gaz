package gaz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLifecycleHooks(t *testing.T) {
	c := New()

	var startCalled, stopCalled bool

	type MyService struct{}

	err := For[*MyService](c).
		OnStart(func(ctx context.Context, s *MyService) error {
			startCalled = true
			return nil
		}).
		OnStop(func(ctx context.Context, s *MyService) error {
			stopCalled = true
			return nil
		}).
		ProviderFunc(func(_ *Container) *MyService { return &MyService{} })

	require.NoError(t, err)

	// Manually resolve to ensure it's built
	_, err = Resolve[*MyService](c)
	require.NoError(t, err)

	// Access the service wrapper internally
	// Since we are in package gaz, we can access private fields
	svcName := TypeName[*MyService]()
	svc, ok := c.services[svcName]
	require.True(t, ok)

	wrapper, ok := svc.(serviceWrapper)
	require.True(t, ok)

	// Verify hasLifecycle
	assert.True(t, wrapper.hasLifecycle())

	// Invoke start
	err = wrapper.start(context.Background())
	require.NoError(t, err)
	assert.True(t, startCalled, "OnStart hook should have been called")

	// Invoke stop
	err = wrapper.stop(context.Background())
	require.NoError(t, err)
	assert.True(t, stopCalled, "OnStop hook should have been called")
}

type lifecycleService struct {
	started bool
	stopped bool
}

func (s *lifecycleService) OnStart(ctx context.Context) error {
	s.started = true
	return nil
}

func (s *lifecycleService) OnStop(ctx context.Context) error {
	s.stopped = true
	return nil
}

func TestInterfaceLifecycle(t *testing.T) {
	c := New()

	svc := &lifecycleService{}

	err := For[*lifecycleService](c).
		Instance(svc)
	require.NoError(t, err)

	wrapper, ok := c.services[TypeName[*lifecycleService]()].(serviceWrapper)
	require.True(t, ok)

	// Instance service start() should call OnStart interface method
	err = wrapper.start(context.Background())
	require.NoError(t, err)
	assert.True(t, svc.started)

	err = wrapper.stop(context.Background())
	require.NoError(t, err)
	assert.True(t, svc.stopped)
}
