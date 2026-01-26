package gaz

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testService is a simple service for testing
type testService struct {
	id int
}

func TestLazySingleton_InstantiatesOnce(t *testing.T) {
	callCount := 0
	provider := func(c *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newLazySingleton("test", "*gaz.testService", provider)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	if err != nil {
		t.Fatalf("getInstance 1: %v", err)
	}

	instance2, err := svc.getInstance(c, nil)
	if err != nil {
		t.Fatalf("getInstance 2: %v", err)
	}

	if callCount != 1 {
		t.Errorf("provider called %d times, want 1", callCount)
	}

	if instance1 != instance2 {
		t.Error("instances should be identical")
	}
}

func TestLazySingleton_ConcurrentAccess(t *testing.T) {
	var callCount int32
	provider := func(c *Container) (*testService, error) {
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
		if err != nil {
			t.Errorf("goroutine %d got error: %v", i, err)
		}
	}

	// Check provider called exactly once
	if callCount != 1 {
		t.Errorf("provider called %d times, want 1", callCount)
	}

	// Check all instances are the same
	first := instances[0]
	for i, inst := range instances {
		if inst != first {
			t.Errorf("goroutine %d got different instance", i)
		}
	}
}

func TestTransientService_NewInstanceEachTime(t *testing.T) {
	callCount := 0
	provider := func(c *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newTransient("test", "*gaz.testService", provider)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	if err != nil {
		t.Fatalf("getInstance 1: %v", err)
	}

	instance2, err := svc.getInstance(c, nil)
	if err != nil {
		t.Fatalf("getInstance 2: %v", err)
	}

	if callCount != 2 {
		t.Errorf("provider called %d times, want 2", callCount)
	}

	// Instances should be different
	ts1 := instance1.(*testService)
	ts2 := instance2.(*testService)
	if ts1.id == ts2.id {
		t.Error("transient instances should be different")
	}
}

func TestEagerSingleton_IsEagerTrue(t *testing.T) {
	provider := func(c *Container) (*testService, error) {
		return &testService{id: 1}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider)

	if !svc.isEager() {
		t.Error("eagerSingleton.isEager() should return true")
	}

	// Verify lazy and transient are NOT eager
	lazy := newLazySingleton("test", "*gaz.testService", provider)
	if lazy.isEager() {
		t.Error("lazySingleton.isEager() should return false")
	}

	transient := newTransient("test", "*gaz.testService", provider)
	if transient.isEager() {
		t.Error("transientService.isEager() should return false")
	}
}

func TestInstanceService_ReturnsValue(t *testing.T) {
	original := &testService{id: 42}
	svc := newInstanceService("test", "*gaz.testService", original)
	c := New()

	instance, err := svc.getInstance(c, nil)
	if err != nil {
		t.Fatalf("getInstance: %v", err)
	}

	// Verify we got the exact same value (pointer equality)
	if instance != original {
		t.Error("instanceService should return the exact value provided")
	}

	// Call again to confirm same value
	instance2, err := svc.getInstance(c, nil)
	if err != nil {
		t.Fatalf("getInstance 2: %v", err)
	}

	if instance2 != original {
		t.Error("instanceService should always return the same value")
	}
}

func TestInstanceService_IsNotEager(t *testing.T) {
	original := &testService{id: 1}
	svc := newInstanceService("test", "*gaz.testService", original)

	// Instance service is already instantiated, so isEager() should be false
	// (no need to instantiate at Build() time)
	if svc.isEager() {
		t.Error("instanceService.isEager() should return false")
	}
}

func TestServiceWrapper_NameAndTypeName(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.wrapper.name(); got != tt.expName {
				t.Errorf("name() = %q, want %q", got, tt.expName)
			}
			if got := tt.wrapper.typeName(); got != tt.expType {
				t.Errorf("typeName() = %q, want %q", got, tt.expType)
			}
		})
	}
}

func TestEagerSingleton_BehavesLikeLazySingleton(t *testing.T) {
	// Eager singleton should cache like lazy singleton
	callCount := 0
	provider := func(c *Container) (*testService, error) {
		callCount++
		return &testService{id: callCount}, nil
	}

	svc := newEagerSingleton("test", "*gaz.testService", provider)
	c := New()

	instance1, err := svc.getInstance(c, nil)
	if err != nil {
		t.Fatalf("getInstance 1: %v", err)
	}

	instance2, err := svc.getInstance(c, nil)
	if err != nil {
		t.Fatalf("getInstance 2: %v", err)
	}

	if callCount != 1 {
		t.Errorf("provider called %d times, want 1", callCount)
	}

	if instance1 != instance2 {
		t.Error("eager singleton instances should be identical")
	}
}
