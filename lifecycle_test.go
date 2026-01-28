package gaz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLifecycleHooks(t *testing.T) {
	c := NewContainer()

	var startCalled, stopCalled bool

	type MyService struct{}

	err := For[*MyService](c).
		OnStart(func(_ context.Context, _ *MyService) error {
			startCalled = true
			return nil
		}).
		OnStop(func(_ context.Context, _ *MyService) error {
			stopCalled = true
			return nil
		}).
		ProviderFunc(func(_ *Container) *MyService { return &MyService{} })

	require.NoError(t, err)

	// Manually resolve to ensure it's built
	_, err = Resolve[*MyService](c)
	require.NoError(t, err)

	// Access the service wrapper using public API
	svcName := TypeName[*MyService]()
	wrapper, ok := c.GetService(svcName)
	require.True(t, ok)

	// Verify HasLifecycle
	assert.True(t, wrapper.HasLifecycle())

	// Invoke Start
	err = wrapper.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, startCalled, "OnStart hook should have been called")

	// Invoke Stop
	err = wrapper.Stop(context.Background())
	require.NoError(t, err)
	assert.True(t, stopCalled, "OnStop hook should have been called")
}

type lifecycleService struct {
	started bool
	stopped bool
}

func (s *lifecycleService) OnStart(_ context.Context) error {
	s.started = true
	return nil
}

func (s *lifecycleService) OnStop(_ context.Context) error {
	s.stopped = true
	return nil
}

func TestInterfaceLifecycle(t *testing.T) {
	c := NewContainer()

	svc := &lifecycleService{}

	err := For[*lifecycleService](c).
		Instance(svc)
	require.NoError(t, err)

	wrapper, ok := c.GetService(TypeName[*lifecycleService]())
	require.True(t, ok)

	// Instance service Start() should call OnStart interface method
	err = wrapper.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, svc.started)

	err = wrapper.Stop(context.Background())
	require.NoError(t, err)
	assert.True(t, svc.stopped)
}
