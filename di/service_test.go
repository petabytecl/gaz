package di

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

	svc := newLazySingleton("test", "*gaz.testService", provider)
	c := New()

	instance1, err := svc.GetInstance(c, nil)
	s.Require().NoError(err, "getInstance 1")

	instance2, err := svc.GetInstance(c, nil)
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

	svc := newLazySingleton("test", "*gaz.testService", provider)
	c := New()

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	instances := make([]any, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()
			inst, err := svc.GetInstance(c, nil)
			instances[idx] = inst
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Check no errors
	for i, err := range errors {
		s.Require().NoError(err, "goroutine %d got error", i)
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

	instance1, err := svc.GetInstance(c, nil)
	s.Require().NoError(err, "getInstance 1")

	instance2, err := svc.GetInstance(c, nil)
	s.Require().NoError(err, "getInstance 2")

	s.Equal(2, callCount, "provider should be called twice")

	// Instances should be different
	ts1, ok := instance1.(*testService)
	s.Require().True(ok, "instance1 should be *testService")
	ts2, ok := instance2.(*testService)
	s.Require().True(ok, "instance2 should be *testService")
	s.NotEqual(ts1.id, ts2.id, "transient instances should be different")
}

func (s *ServiceSuite) TestEagerSingleton_IsEagerTrue() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider)

	s.True(svc.IsEager(), "eagerSingleton.IsEager() should return true")

	// Verify lazy and transient are NOT eager
	lazy := newLazySingleton("test", "*gaz.testService", provider)
	s.False(lazy.IsEager(), "lazySingleton.IsEager() should return false")

	transient := newTransient("test", "*gaz.testService", provider)
	s.False(transient.IsEager(), "transientService.IsEager() should return false")
}

func (s *ServiceSuite) TestInstanceService_ReturnsValue() {
	original := &testService{id: 42}
	svc := newInstanceService("test", "*gaz.testService", original)
	c := New()

	instance, err := svc.GetInstance(c, nil)
	s.Require().NoError(err, "getInstance")

	// Verify we got the exact same value (pointer equality)
	s.Same(original, instance, "instanceService should return the exact value provided")

	// Call again to confirm same value
	instance2, err := svc.GetInstance(c, nil)
	s.Require().NoError(err, "getInstance 2")

	s.Same(original, instance2, "instanceService should always return the same value")
}

func (s *ServiceSuite) TestInstanceService_IsNotEager() {
	original := &testService{id: 1}
	svc := newInstanceService("test", "*gaz.testService", original)

	// Instance service is already instantiated, so isEager() should be false
	// (no need to instantiate at Build() time)
	s.False(svc.IsEager(), "instanceService.IsEager() should return false")
}

func (s *ServiceSuite) TestServiceWrapper_NameAndTypeName() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	tests := []struct {
		name    string
		wrapper ServiceWrapper
		expName string
		expType string
	}{
		{
			name:    "lazySingleton",
			wrapper: newLazySingleton("myService", "*app.MyService", provider),
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
			wrapper: newEagerSingleton("eager", "*app.Eager", provider),
			expName: "eager",
			expType: "*app.Eager",
		},
		{
			name:    "instanceService",
			wrapper: newInstanceService("instance", "*app.Instance", &testService{id: 1}),
			expName: "instance",
			expType: "*app.Instance",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expName, tt.wrapper.Name())
			s.Equal(tt.expType, tt.wrapper.TypeName())
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

	svc := newEagerSingleton("test", "*gaz.testService", provider)
	c := New()

	instance1, err := svc.GetInstance(c, nil)
	s.Require().NoError(err, "getInstance 1")

	instance2, err := svc.GetInstance(c, nil)
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
	err := svc.Start(context.Background())
	s.Require().NoError(err, "transient start should succeed")

	err = svc.Stop(context.Background())
	s.Require().NoError(err, "transient stop should succeed")

	// hasLifecycle should always return false
	s.False(svc.HasLifecycle(), "transient should not have lifecycle")
}

func (s *ServiceSuite) TestLazySingleton_HasLifecycle() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	// No lifecycle interfaces
	svc := newLazySingleton("test", "*gaz.testService", provider)
	s.False(svc.HasLifecycle())
}

// Test eager singleton lifecycle edge cases.
func (s *ServiceSuite) TestEagerSingleton_LifecycleNotBuilt() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider)

	// When not built, start should be no-op
	err := svc.Start(context.Background())
	s.Require().NoError(err)

	// stop when not built should also be no-op
	err = svc.Stop(context.Background())
	s.Require().NoError(err)
}

func (s *ServiceSuite) TestInstanceService_HasLifecycle() {
	original := &testService{id: 42}

	// testService does not implement Starter/Stopper, so no lifecycle
	svc := newInstanceService("test", "*gaz.testService", original)
	s.False(svc.HasLifecycle())
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

	svc := newEagerSingleton("test", "*gaz.starterService", provider)
	c := New()

	// Build the instance
	instance, err := svc.GetInstance(c, nil)
	s.Require().NoError(err)

	// start should call OnStart
	err = svc.Start(context.Background())
	s.Require().NoError(err)
	starterSvc, ok := instance.(*starterService)
	s.Require().True(ok, "instance should be *starterService")
	s.True(starterSvc.started)
}

func (s *ServiceSuite) TestEagerSingleton_StopperInterface() {
	provider := func(_ *Container) (*stopperService, error) {
		return &stopperService{}, nil
	}

	svc := newEagerSingleton("test", "*gaz.stopperService", provider)
	c := New()

	// Build the instance
	instance, err := svc.GetInstance(c, nil)
	s.Require().NoError(err)

	// stop should call OnStop
	err = svc.Stop(context.Background())
	s.Require().NoError(err)
	stopperSvc, ok := instance.(*stopperService)
	s.Require().True(ok, "instance should be *stopperService")
	s.True(stopperSvc.stopped)
}

func (s *ServiceSuite) TestEagerSingleton_StarterInterfaceError() {
	provider := func(_ *Container) (*failingStarterService, error) {
		return &failingStarterService{}, nil
	}

	svc := newEagerSingleton("test", "*gaz.failingStarterService", provider)
	c := New()

	// Build the instance
	_, err := svc.GetInstance(c, nil)
	s.Require().NoError(err)

	// start should return the error
	err = svc.Start(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "starter failed")
}

func (s *ServiceSuite) TestEagerSingleton_StopperInterfaceError() {
	provider := func(_ *Container) (*failingStopperService, error) {
		return &failingStopperService{}, nil
	}

	svc := newEagerSingleton("test", "*gaz.failingStopperService", provider)
	c := New()

	// Build the instance
	_, err := svc.GetInstance(c, nil)
	s.Require().NoError(err)

	// stop should return the error
	err = svc.Stop(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "stopper failed")
}

// =============================================================================
// Tests for internal instance service helper
// =============================================================================

func (s *ServiceSuite) TestInstanceServiceAny_Lifecycle() {
	// Instance Service Any - used internally by registerInstance()
	inst := &starterService{}
	svc := NewInstanceServiceAny("instance", "any", inst)

	s.Require().NoError(svc.Start(context.Background()))
	s.True(inst.started)

	s.Require().NoError(svc.Stop(context.Background()))

	// Verify hasLifecycle override
	s.True(svc.HasLifecycle())
}

func (s *ServiceSuite) TestInstanceService_StopperInterface() {
	original := &stopperService{}

	svc := newInstanceService("test", "*gaz.stopperService", original)

	err := svc.Stop(context.Background())
	s.Require().NoError(err)
	s.True(original.stopped)
}

func (s *ServiceSuite) TestInstanceService_StarterInterfaceError() {
	original := &failingStarterService{}

	svc := newInstanceService("test", "*gaz.failingStarterService", original)

	err := svc.Start(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "starter failed")
}

func (s *ServiceSuite) TestInstanceService_StopperInterfaceError() {
	original := &failingStopperService{}

	svc := newInstanceService("test", "*gaz.failingStopperService", original)

	err := svc.Stop(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "stopper failed")
}

// =============================================================================
// IsTransient() tests for all service wrapper types
// =============================================================================

func (s *ServiceSuite) TestLazySingleton_IsTransient() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newLazySingleton("test", "*gaz.testService", provider)

	s.False(svc.IsTransient(), "lazySingleton.IsTransient() should return false")
}

func (s *ServiceSuite) TestTransientService_IsTransient() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newTransient("test", "*gaz.testService", provider)

	s.True(svc.IsTransient(), "transientService.IsTransient() should return true")
}

func (s *ServiceSuite) TestEagerSingleton_IsTransient() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider)

	s.False(svc.IsTransient(), "eagerSingleton.IsTransient() should return false")
}

func (s *ServiceSuite) TestInstanceService_IsTransient() {
	original := &testService{id: 42}

	svc := newInstanceService("test", "*gaz.testService", original)

	s.False(svc.IsTransient(), "instanceService.IsTransient() should return false")
}

func (s *ServiceSuite) TestInstanceServiceAny_IsTransient() {
	original := &testService{id: 42}

	svc := NewInstanceServiceAny("test", "*gaz.testService", original)

	s.False(svc.IsTransient(), "instanceServiceAny.IsTransient() should return false")
}

// =============================================================================
// Additional instanceServiceAny method tests
// =============================================================================

func (s *ServiceSuite) TestInstanceServiceAny_IsEager() {
	original := &testService{id: 42}

	svc := NewInstanceServiceAny("test", "*gaz.testService", original)

	// Instance service is already instantiated, so IsEager() should be false
	// (no need to instantiate at Build() time)
	s.False(svc.IsEager(), "instanceServiceAny.IsEager() should return false")
}

func (s *ServiceSuite) TestInstanceServiceAny_GetInstance() {
	original := &testService{id: 42}

	svc := NewInstanceServiceAny("test", "*gaz.testService", original)
	c := New()

	instance, err := svc.GetInstance(c, nil)
	s.Require().NoError(err, "getInstance")

	// Verify we got the exact same value (pointer equality)
	s.Same(original, instance, "instanceServiceAny should return the exact value provided")

	// Call again to confirm same value
	instance2, err := svc.GetInstance(c, nil)
	s.Require().NoError(err, "getInstance 2")

	s.Same(original, instance2, "instanceServiceAny should always return the same value")
}

func (s *ServiceSuite) TestInstanceServiceAny_NameAndTypeName() {
	original := &testService{id: 42}

	svc := NewInstanceServiceAny("myInstance", "*app.MyType", original)

	s.Equal("myInstance", svc.Name())
	s.Equal("*app.MyType", svc.TypeName())
}

func (s *ServiceSuite) TestInstanceServiceAny_StartError() {
	original := &failingStarterService{}

	svc := NewInstanceServiceAny("test", "*gaz.failingStarterService", original)

	err := svc.Start(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "starter failed")
}

func (s *ServiceSuite) TestInstanceServiceAny_StopError() {
	original := &failingStopperService{}

	svc := NewInstanceServiceAny("test", "*gaz.failingStopperService", original)

	err := svc.Stop(context.Background())
	s.Require().Error(err)
	s.Contains(err.Error(), "stopper failed")
}

// =============================================================================
// eagerSingleton HasLifecycle test
// =============================================================================

func (s *ServiceSuite) TestEagerSingleton_HasLifecycle() {
	provider := func(_ *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	// No lifecycle interfaces
	svc := newEagerSingleton("test", "*gaz.testService", provider)
	s.False(svc.HasLifecycle())

	// With Starter interface
	starterProvider := func(_ *Container) (*starterService, error) {
		return &starterService{}, nil
	}
	svc4 := newEagerSingleton("test", "*gaz.starterService", starterProvider)
	s.True(svc4.HasLifecycle())

	// With Stopper interface
	stopperProvider := func(_ *Container) (*stopperService, error) {
		return &stopperService{}, nil
	}
	svc5 := newEagerSingleton("test", "*gaz.stopperService", stopperProvider)
	s.True(svc5.HasLifecycle())
}
