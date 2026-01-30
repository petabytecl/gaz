package gaz

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// myTestService is a test service that tracks lifecycle callbacks.
type myTestService struct {
	started *bool
	stopped *bool
}

// OnStart implements di.Starter for myTestService.
func (s *myTestService) OnStart(_ context.Context) error {
	if s.started != nil {
		*s.started = true
	}
	return nil
}

// OnStop implements di.Stopper for myTestService.
func (s *myTestService) OnStop(_ context.Context) error {
	if s.stopped != nil {
		*s.stopped = true
	}
	return nil
}

func TestLifecycleHooks(t *testing.T) {
	c := NewContainer()

	var startCalled, stopCalled bool

	// Service implements di.Starter and di.Stopper interfaces - no fluent hooks needed
	err := For[*myTestService](c).
		ProviderFunc(func(_ *Container) *myTestService {
			return &myTestService{started: &startCalled, stopped: &stopCalled}
		})

	require.NoError(t, err)

	// Manually resolve to ensure it's built
	_, err = Resolve[*myTestService](c)
	require.NoError(t, err)

	// Access the service wrapper using public API
	svcName := TypeName[*myTestService]()
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

// =============================================================================
// WithHookTimeout tests
// =============================================================================

func TestWithHookTimeout(t *testing.T) {
	cfg := &HookConfig{}

	// Apply default - should be zero
	assert.Equal(t, time.Duration(0), cfg.Timeout)

	// Apply WithHookTimeout option
	opt := WithHookTimeout(5 * time.Second)
	opt(cfg)

	assert.Equal(t, 5*time.Second, cfg.Timeout)
}

func TestWithHookTimeout_MultipleApply(t *testing.T) {
	cfg := &HookConfig{}

	// Apply multiple options - last one wins
	opt1 := WithHookTimeout(5 * time.Second)
	opt2 := WithHookTimeout(30 * time.Second)

	opt1(cfg)
	assert.Equal(t, 5*time.Second, cfg.Timeout)

	opt2(cfg)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
}

func TestWithHookTimeout_ZeroDuration(t *testing.T) {
	cfg := &HookConfig{Timeout: 10 * time.Second}

	// Apply zero duration - should set to zero
	opt := WithHookTimeout(0)
	opt(cfg)

	assert.Equal(t, time.Duration(0), cfg.Timeout)
}
