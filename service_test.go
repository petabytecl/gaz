package gaz

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// testService is a simple service for testing.
type testService struct {
	id int
}

// ServiceSuite tests the service wrapper implementations.
type ServiceSuite struct {
	suite.Suite
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}

func (s *ServiceSuite) TestLazySingleton_InstantiatesOnce() {
	callCount := 0
	provider := func(_ *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newLazySingleton("test", "*gaz.testService", provider, nil, nil)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	s.Require().NoError(err, "getInstance 1")

	instance2, err := svc.getInstance(c, nil)
	s.Require().NoError(err, "getInstance 2")

	s.Equal(1, callCount, "provider should be called once")
	s.Same(instance1, instance2, "instances should be identical")
}

func (s *ServiceSuite) TestLazySingleton_ConcurrentAccess() {
	var callCount int32
	provider := func(_ *Container) (*testService, error) {
		atomic.AddInt32(&callCount, 1)
		// Small delay to increase chance of race
		time.Sleep(10 * time.Millisecond)
		return &testService{id: int(callCount)}, nil
	}

	svc := newLazySingleton("test", "*gaz.testService", provider, nil, nil)
	c := New()

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	instances := make([]any, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			inst, err := svc.getInstance(c, nil)
			instances[idx] = inst
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Check no errors
	for i, err := range errors {
		s.NoError(err, "goroutine %d got error", i)
	}

	// Check provider called exactly once
	s.Equal(int32(1), callCount, "provider should be called once")

	// Check all instances are the same
	first := instances[0]
	for i, inst := range instances {
		s.Same(first, inst, "goroutine %d got different instance", i)
	}
}

func (s *ServiceSuite) TestTransientService_NewInstanceEachTime() {
	callCount := 0
	provider := func(_ *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newTransient("test", "*gaz.testService", provider)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	s.Require().NoError(err, "getInstance 1")

	instance2, err := svc.getInstance(c, nil)
	s.Require().NoError(err, "getInstance 2")

	s.Equal(2, callCount, "provider should be called twice")

	// Instances should be different
	ts1 := instance1.(*testService)
	ts2 := instance2.(*testService)
	s.NotEqual(ts1.id, ts2.id, "transient instances should be different")
}

func (s *ServiceSuite) TestEagerSingleton_IsEagerTrue() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider, nil, nil)

	s.True(svc.isEager(), "eagerSingleton.isEager() should return true")

	// Verify lazy and transient are NOT eager
	lazy := newLazySingleton("test", "*gaz.testService", provider, nil, nil)
	s.False(lazy.isEager(), "lazySingleton.isEager() should return false")

	transient := newTransient("test", "*gaz.testService", provider)
	s.False(transient.isEager(), "transientService.isEager() should return false")
}

func (s *ServiceSuite) TestInstanceService_ReturnsValue() {
	original := &testService{id: 42}
	svc := newInstanceService("test", "*gaz.testService", original, nil, nil)
	c := New()

	instance, err := svc.getInstance(c, nil)
	s.Require().NoError(err, "getInstance")

	// Verify we got the exact same value (pointer equality)
	s.Same(original, instance, "instanceService should return the exact value provided")

	// Call again to confirm same value
	instance2, err := svc.getInstance(c, nil)
	s.Require().NoError(err, "getInstance 2")

	s.Same(original, instance2, "instanceService should always return the same value")
}

func (s *ServiceSuite) TestInstanceService_IsNotEager() {
	original := &testService{id: 1}
	svc := newInstanceService("test", "*gaz.testService", original, nil, nil)

	// Instance service is already instantiated, so isEager() should be false
	// (no need to instantiate at Build() time)
	s.False(svc.isEager(), "instanceService.isEager() should return false")
}

func (s *ServiceSuite) TestServiceWrapper_NameAndTypeName() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	tests := []struct {
		name    string
		wrapper serviceWrapper
		expName string
		expType string
	}{
		{
			name:    "lazySingleton",
			wrapper: newLazySingleton("myService", "*app.MyService", provider, nil, nil),
			expName: "myService",
			expType: "*app.MyService",
		},
		{
			name:    "transientService",
			wrapper: newTransient("transient", "*app.Transient", provider),
			expName: "transient",
			expType: "*app.Transient",
		},
		{
			name:    "eagerSingleton",
			wrapper: newEagerSingleton("eager", "*app.Eager", provider, nil, nil),
			expName: "eager",
			expType: "*app.Eager",
		},
		{
			name:    "instanceService",
			wrapper: newInstanceService("instance", "*app.Instance", &testService{id: 1}, nil, nil),
			expName: "instance",
			expType: "*app.Instance",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expName, tt.wrapper.name())
			s.Equal(tt.expType, tt.wrapper.typeName())
		})
	}
}

func (s *ServiceSuite) TestEagerSingleton_BehavesLikeLazySingleton() {
	// Eager singleton should cache like lazy singleton
	callCount := 0
	provider := func(_ *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider, nil, nil)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	s.Require().NoError(err, "getInstance 1")

	instance2, err := svc.getInstance(c, nil)
	s.Require().NoError(err, "getInstance 2")

	s.Equal(1, callCount, "provider should be called once")
	s.Same(instance1, instance2, "eager singleton instances should be identical")
}

// Test transient service lifecycle methods (no-op but need coverage).
func (s *ServiceSuite) TestTransientService_LifecycleMethods() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newTransient("test", "*gaz.testService", provider)

	// start and stop should be no-ops
	err := svc.start(context.Background())
	s.NoError(err, "transient start should succeed")

	err = svc.stop(context.Background())
	s.NoError(err, "transient stop should succeed")

	// hasLifecycle should always return false
	s.False(svc.hasLifecycle(), "transient should not have lifecycle")
}

// Test lazy singleton lifecycle edge cases.
func (s *ServiceSuite) TestLazySingleton_LifecycleNotBuilt() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	startCalled := false
	stopCalled := false
	startHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			startCalled = true
			return nil
		},
	}
	stopHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			stopCalled = true
			return nil
		},
	}

	svc := newLazySingleton("test", "*gaz.testService", provider, startHooks, stopHooks)

	// When not built, start and stop should be no-ops
	err := svc.start(context.Background())
	s.NoError(err)
	s.False(startCalled, "start hook should not be called when not built")

	err = svc.stop(context.Background())
	s.NoError(err)
	s.False(stopCalled, "stop hook should not be called when not built")
}

func (s *ServiceSuite) TestLazySingleton_LifecycleWithHooks() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	startCalled := false
	stopCalled := false
	startHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			startCalled = true
			return nil
		},
	}
	stopHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			stopCalled = true
			return nil
		},
	}

	svc := newLazySingleton("test", "*gaz.testService", provider, startHooks, stopHooks)
	c := New()

	// Build the instance
	_, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// Now hooks should be called
	err = svc.start(context.Background())
	s.NoError(err)
	s.True(startCalled, "start hook should be called")

	err = svc.stop(context.Background())
	s.NoError(err)
	s.True(stopCalled, "stop hook should be called")
}

func (s *ServiceSuite) TestLazySingleton_StartError() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	startHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			return errors.New("start failed")
		},
	}

	svc := newLazySingleton("test", "*gaz.testService", provider, startHooks, nil)
	c := New()

	// Build the instance
	_, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// start should return the error
	err = svc.start(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "start failed")
}

func (s *ServiceSuite) TestLazySingleton_StopError() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	stopHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			return errors.New("stop failed")
		},
	}

	svc := newLazySingleton("test", "*gaz.testService", provider, nil, stopHooks)
	c := New()

	// Build the instance
	_, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// stop should return the error
	err = svc.stop(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "stop failed")
}

func (s *ServiceSuite) TestLazySingleton_HasLifecycle() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	// No hooks
	svc := newLazySingleton("test", "*gaz.testService", provider, nil, nil)
	s.False(svc.hasLifecycle())

	// With start hook only
	svc2 := newLazySingleton(
		"test",
		"*gaz.testService",
		provider,
		[]func(context.Context, any) error{
			func(_ context.Context, _ any) error { return nil },
		},
		nil,
	)
	s.True(svc2.hasLifecycle())

	// With stop hook only
	svc3 := newLazySingleton(
		"test",
		"*gaz.testService",
		provider,
		nil,
		[]func(context.Context, any) error{
			func(_ context.Context, _ any) error { return nil },
		},
	)
	s.True(svc3.hasLifecycle())
}

// Test eager singleton lifecycle edge cases.
func (s *ServiceSuite) TestEagerSingleton_LifecycleNotBuilt() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider, nil, nil)

	// When not built, start should be no-op
	err := svc.start(context.Background())
	s.NoError(err)

	// stop when not built should also be no-op
	err = svc.stop(context.Background())
	s.NoError(err)
}

func (s *ServiceSuite) TestEagerSingleton_StartError() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	startHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			return errors.New("start failed")
		},
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider, startHooks, nil)
	c := New()

	// Build the instance
	_, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// start should return the error
	err = svc.start(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "start failed")
}

func (s *ServiceSuite) TestEagerSingleton_StopError() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	stopHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			return errors.New("stop failed")
		},
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider, nil, stopHooks)
	c := New()

	// Build the instance
	_, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// stop should return the error
	err = svc.stop(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "stop failed")
}

// Test instance service lifecycle.
func (s *ServiceSuite) TestInstanceService_LifecycleHooks() {
	original := &testService{id: 42}

	startCalled := false
	stopCalled := false
	startHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			startCalled = true
			return nil
		},
	}
	stopHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			stopCalled = true
			return nil
		},
	}

	svc := newInstanceService("test", "*gaz.testService", original, startHooks, stopHooks)

	err := svc.start(context.Background())
	s.NoError(err)
	s.True(startCalled)

	err = svc.stop(context.Background())
	s.NoError(err)
	s.True(stopCalled)
}

func (s *ServiceSuite) TestInstanceService_HasLifecycle() {
	original := &testService{id: 42}

	// No hooks
	svc := newInstanceService("test", "*gaz.testService", original, nil, nil)
	s.False(svc.hasLifecycle())

	// With hooks
	svc2 := newInstanceService(
		"test",
		"*gaz.testService",
		original,
		[]func(context.Context, any) error{
			func(_ context.Context, _ any) error { return nil },
		},
		nil,
	)
	s.True(svc2.hasLifecycle())
}

func (s *ServiceSuite) TestInstanceService_StartError() {
	original := &testService{id: 42}

	startHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			return errors.New("start failed")
		},
	}

	svc := newInstanceService("test", "*gaz.testService", original, startHooks, nil)

	err := svc.start(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "start failed")
}

func (s *ServiceSuite) TestInstanceService_StopError() {
	original := &testService{id: 42}

	stopHooks := []func(context.Context, any) error{
		func(_ context.Context, _ any) error {
			return errors.New("stop failed")
		},
	}

	svc := newInstanceService("test", "*gaz.testService", original, nil, stopHooks)

	err := svc.stop(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "stop failed")
}

// Test Starter/Stopper interfaces for eager singleton.
type starterService struct{ started bool }

func (s *starterService) OnStart(_ context.Context) error {
	s.started = true
	return nil
}

type stopperService struct{ stopped bool }

func (s *stopperService) OnStop(_ context.Context) error {
	s.stopped = true
	return nil
}

type failingStarterService struct{}

func (s *failingStarterService) OnStart(_ context.Context) error {
	return errors.New("starter failed")
}

type failingStopperService struct{}

func (s *failingStopperService) OnStop(_ context.Context) error {
	return errors.New("stopper failed")
}

func (s *ServiceSuite) TestEagerSingleton_StarterInterface() {
	provider := func(_ *Container) (*starterService, error) {
		return &starterService{}, nil
	}

	svc := newEagerSingleton("test", "*gaz.starterService", provider, nil, nil)
	c := New()

	// Build the instance
	instance, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// start should call OnStart
	err = svc.start(context.Background())
	s.NoError(err)
	s.True(instance.(*starterService).started)
}

func (s *ServiceSuite) TestEagerSingleton_StopperInterface() {
	provider := func(_ *Container) (*stopperService, error) {
		return &stopperService{}, nil
	}

	svc := newEagerSingleton("test", "*gaz.stopperService", provider, nil, nil)
	c := New()

	// Build the instance
	instance, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// stop should call OnStop
	err = svc.stop(context.Background())
	s.NoError(err)
	s.True(instance.(*stopperService).stopped)
}

func (s *ServiceSuite) TestEagerSingleton_StarterInterfaceError() {
	provider := func(_ *Container) (*failingStarterService, error) {
		return &failingStarterService{}, nil
	}

	svc := newEagerSingleton("test", "*gaz.failingStarterService", provider, nil, nil)
	c := New()

	// Build the instance
	_, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// start should return the error
	err = svc.start(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "starter failed")
}

func (s *ServiceSuite) TestEagerSingleton_StopperInterfaceError() {
	provider := func(_ *Container) (*failingStopperService, error) {
		return &failingStopperService{}, nil
	}

	svc := newEagerSingleton("test", "*gaz.failingStopperService", provider, nil, nil)
	c := New()

	// Build the instance
	_, err := svc.getInstance(c, nil)
	s.Require().NoError(err)

	// stop should return the error
	err = svc.stop(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "stopper failed")
}

// Test instance service with Starter/Stopper interfaces.
func (s *ServiceSuite) TestInstanceService_StarterInterface() {
	original := &starterService{}

	svc := newInstanceService("test", "*gaz.starterService", original, nil, nil)

	err := svc.start(context.Background())
	s.NoError(err)
	s.True(original.started)
}

func (s *ServiceSuite) TestInstanceService_StopperInterface() {
	original := &stopperService{}

	svc := newInstanceService("test", "*gaz.stopperService", original, nil, nil)

	err := svc.stop(context.Background())
	s.NoError(err)
	s.True(original.stopped)
}

func (s *ServiceSuite) TestInstanceService_StarterInterfaceError() {
	original := &failingStarterService{}

	svc := newInstanceService("test", "*gaz.failingStarterService", original, nil, nil)

	err := svc.start(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "starter failed")
}

func (s *ServiceSuite) TestInstanceService_StopperInterfaceError() {
	original := &failingStopperService{}

	svc := newInstanceService("test", "*gaz.failingStopperService", original, nil, nil)

	err := svc.stop(context.Background())
	s.Error(err)
	s.Contains(err.Error(), "stopper failed")
}
