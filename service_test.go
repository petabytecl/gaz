package gaz

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// testService is a simple service for testing
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
	provider := func(c *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newLazySingleton("test", "*gaz.testService", provider, nil, nil)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	require.NoError(s.T(), err, "getInstance 1")

	instance2, err := svc.getInstance(c, nil)
	require.NoError(s.T(), err, "getInstance 2")

	assert.Equal(s.T(), 1, callCount, "provider should be called once")
	assert.Same(s.T(), instance1, instance2, "instances should be identical")
}

func (s *ServiceSuite) TestLazySingleton_ConcurrentAccess() {
	var callCount int32
	provider := func(c *Container) (*testService, error) {
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

	for i := 0; i < numGoroutines; i++ {
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
		assert.NoError(s.T(), err, "goroutine %d got error", i)
	}

	// Check provider called exactly once
	assert.Equal(s.T(), int32(1), callCount, "provider should be called once")

	// Check all instances are the same
	first := instances[0]
	for i, inst := range instances {
		assert.Same(s.T(), first, inst, "goroutine %d got different instance", i)
	}
}

func (s *ServiceSuite) TestTransientService_NewInstanceEachTime() {
	callCount := 0
	provider := func(c *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newTransient("test", "*gaz.testService", provider)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	require.NoError(s.T(), err, "getInstance 1")

	instance2, err := svc.getInstance(c, nil)
	require.NoError(s.T(), err, "getInstance 2")

	assert.Equal(s.T(), 2, callCount, "provider should be called twice")

	// Instances should be different
	ts1 := instance1.(*testService)
	ts2 := instance2.(*testService)
	assert.NotEqual(s.T(), ts1.id, ts2.id, "transient instances should be different")
}

func (s *ServiceSuite) TestEagerSingleton_IsEagerTrue() {
	provider := func(c *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider, nil, nil)

	assert.True(s.T(), svc.isEager(), "eagerSingleton.isEager() should return true")

	// Verify lazy and transient are NOT eager
	lazy := newLazySingleton("test", "*gaz.testService", provider, nil, nil)
	assert.False(s.T(), lazy.isEager(), "lazySingleton.isEager() should return false")

	transient := newTransient("test", "*gaz.testService", provider)
	assert.False(s.T(), transient.isEager(), "transientService.isEager() should return false")
}

func (s *ServiceSuite) TestInstanceService_ReturnsValue() {
	original := &testService{id: 42}
	svc := newInstanceService("test", "*gaz.testService", original, nil, nil)
	c := New()

	instance, err := svc.getInstance(c, nil)
	require.NoError(s.T(), err, "getInstance")

	// Verify we got the exact same value (pointer equality)
	assert.Same(s.T(), original, instance, "instanceService should return the exact value provided")

	// Call again to confirm same value
	instance2, err := svc.getInstance(c, nil)
	require.NoError(s.T(), err, "getInstance 2")

	assert.Same(s.T(), original, instance2, "instanceService should always return the same value")
}

func (s *ServiceSuite) TestInstanceService_IsNotEager() {
	original := &testService{id: 1}
	svc := newInstanceService("test", "*gaz.testService", original, nil, nil)

	// Instance service is already instantiated, so isEager() should be false
	// (no need to instantiate at Build() time)
	assert.False(s.T(), svc.isEager(), "instanceService.isEager() should return false")
}

func (s *ServiceSuite) TestServiceWrapper_NameAndTypeName() {
	provider := func(c *Container) (*testService, error) {
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
			assert.Equal(s.T(), tt.expName, tt.wrapper.name())
			assert.Equal(s.T(), tt.expType, tt.wrapper.typeName())
		})
	}
}

func (s *ServiceSuite) TestEagerSingleton_BehavesLikeLazySingleton() {
	// Eager singleton should cache like lazy singleton
	callCount := 0
	provider := func(c *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider, nil, nil)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	require.NoError(s.T(), err, "getInstance 1")

	instance2, err := svc.getInstance(c, nil)
	require.NoError(s.T(), err, "getInstance 2")

	assert.Equal(s.T(), 1, callCount, "provider should be called once")
	assert.Same(s.T(), instance1, instance2, "eager singleton instances should be identical")
}
