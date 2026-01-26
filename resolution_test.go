package gaz

import (
	"errors"
	"strings"
	"testing"
)

// Test types for resolution tests
type testServiceA struct {
	value string
}

type testServiceB struct {
	value string
}

type testServiceC struct {
	value string
}

// Circular dependency test types
type cyclicA struct {
	b *cyclicB
}

type cyclicB struct {
	a *cyclicA
}

// Dependency chain test types
type depA struct {
	b *depB
}

type depB struct {
	c *depC
}

type depC struct {
	value string
}

func TestResolve_BasicResolution(t *testing.T) {
	c := New()

	// Register a simple service
	err := For[*testServiceA](c).ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "hello"}
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Resolve it
	svc, err := Resolve[*testServiceA](c)
	if err != nil {
		t.Fatalf("resolution failed: %v", err)
	}

	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	if svc.value != "hello" {
		t.Errorf("expected value 'hello', got %q", svc.value)
	}
}

func TestResolve_NotFound(t *testing.T) {
	c := New()

	// Try to resolve unregistered type
	_, err := Resolve[*testServiceA](c)

	if err == nil {
		t.Fatal("expected error for unregistered service")
	}

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}

	// Verify error message contains type name
	if !strings.Contains(err.Error(), "testServiceA") {
		t.Errorf("error should contain type name: %v", err)
	}
}

func TestResolve_Named(t *testing.T) {
	c := New()

	// Register two services with same type, different names
	err := For[*testServiceA](c).Named("first").ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "first-value"}
	})
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	err = For[*testServiceA](c).Named("second").ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "second-value"}
	})
	if err != nil {
		t.Fatalf("second registration failed: %v", err)
	}

	// Resolve each by name
	first, err := Resolve[*testServiceA](c, Named("first"))
	if err != nil {
		t.Fatalf("first resolution failed: %v", err)
	}

	second, err := Resolve[*testServiceA](c, Named("second"))
	if err != nil {
		t.Fatalf("second resolution failed: %v", err)
	}

	// Assert different instances
	if first == second {
		t.Error("expected different instances for different names")
	}

	if first.value != "first-value" {
		t.Errorf("expected first value 'first-value', got %q", first.value)
	}

	if second.value != "second-value" {
		t.Errorf("expected second value 'second-value', got %q", second.value)
	}
}

func TestResolve_CycleDetection(t *testing.T) {
	c := New()

	// A depends on B
	err := For[*cyclicA](c).Provider(func(c *Container) (*cyclicA, error) {
		b, err := Resolve[*cyclicB](c)
		if err != nil {
			return nil, err
		}
		return &cyclicA{b: b}, nil
	})
	if err != nil {
		t.Fatalf("registration of A failed: %v", err)
	}

	// B depends on A (creates cycle)
	err = For[*cyclicB](c).Provider(func(c *Container) (*cyclicB, error) {
		a, err := Resolve[*cyclicA](c)
		if err != nil {
			return nil, err
		}
		return &cyclicB{a: a}, nil
	})
	if err != nil {
		t.Fatalf("registration of B failed: %v", err)
	}

	// Attempt to resolve A should detect cycle
	_, resolveErr := Resolve[*cyclicA](c)

	if resolveErr == nil {
		t.Fatal("expected cycle detection error")
	}

	if !errors.Is(resolveErr, ErrCycle) {
		t.Fatalf("expected ErrCycle, got: %v", resolveErr)
	}

	// Verify chain is in error message
	errMsg := resolveErr.Error()
	if !strings.Contains(errMsg, "->") {
		t.Errorf("error should contain dependency chain: %v", resolveErr)
	}

	// Should contain both type names
	if !strings.Contains(errMsg, "cyclicA") {
		t.Errorf("error should contain cyclicA: %v", resolveErr)
	}
	if !strings.Contains(errMsg, "cyclicB") {
		t.Errorf("error should contain cyclicB: %v", resolveErr)
	}
}

func TestResolve_ProviderError_Propagates(t *testing.T) {
	c := New()

	providerErr := errors.New("provider failed")

	// Register service with provider that returns error
	err := For[*testServiceA](c).Provider(func(c *Container) (*testServiceA, error) {
		return nil, providerErr
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Resolve should propagate the error
	_, resolveErr := Resolve[*testServiceA](c)

	if resolveErr == nil {
		t.Fatal("expected error from provider")
	}

	if !errors.Is(resolveErr, providerErr) {
		t.Fatalf("expected provider error to be wrapped, got: %v", resolveErr)
	}

	// Error should have resolution context
	if !strings.Contains(resolveErr.Error(), "resolving") {
		t.Errorf("error should contain resolution context: %v", resolveErr)
	}
}

func TestResolve_DependencyChain(t *testing.T) {
	c := New()

	// Register C (leaf dependency)
	err := For[*depC](c).ProviderFunc(func(c *Container) *depC {
		return &depC{value: "leaf"}
	})
	if err != nil {
		t.Fatalf("registration of C failed: %v", err)
	}

	// Register B depending on C
	err = For[*depB](c).Provider(func(c *Container) (*depB, error) {
		depC, err := Resolve[*depC](c)
		if err != nil {
			return nil, err
		}
		return &depB{c: depC}, nil
	})
	if err != nil {
		t.Fatalf("registration of B failed: %v", err)
	}

	// Register A depending on B
	err = For[*depA](c).Provider(func(c *Container) (*depA, error) {
		depB, err := Resolve[*depB](c)
		if err != nil {
			return nil, err
		}
		return &depA{b: depB}, nil
	})
	if err != nil {
		t.Fatalf("registration of A failed: %v", err)
	}

	// Resolve A (should build entire chain)
	a, err := Resolve[*depA](c)
	if err != nil {
		t.Fatalf("resolution failed: %v", err)
	}

	// Verify all three instantiated correctly
	if a == nil {
		t.Fatal("expected non-nil A")
	}
	if a.b == nil {
		t.Fatal("expected non-nil B")
	}
	if a.b.c == nil {
		t.Fatal("expected non-nil C")
	}
	if a.b.c.value != "leaf" {
		t.Errorf("expected C value 'leaf', got %q", a.b.c.value)
	}
}

func TestResolve_Transient_NewInstanceEachTime(t *testing.T) {
	c := New()

	callCount := 0

	// Register transient service
	err := For[*testServiceA](c).Transient().ProviderFunc(func(c *Container) *testServiceA {
		callCount++
		return &testServiceA{value: "transient"}
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Resolve twice
	first, err := Resolve[*testServiceA](c)
	if err != nil {
		t.Fatalf("first resolution failed: %v", err)
	}

	second, err := Resolve[*testServiceA](c)
	if err != nil {
		t.Fatalf("second resolution failed: %v", err)
	}

	// Assert different instances (pointer comparison)
	if first == second {
		t.Error("expected different instances for transient service")
	}

	// Assert provider called twice
	if callCount != 2 {
		t.Errorf("expected provider called 2 times, got %d", callCount)
	}
}

func TestResolve_Singleton_SameInstance(t *testing.T) {
	c := New()

	callCount := 0

	// Register singleton service (default)
	err := For[*testServiceA](c).ProviderFunc(func(c *Container) *testServiceA {
		callCount++
		return &testServiceA{value: "singleton"}
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Resolve twice
	first, err := Resolve[*testServiceA](c)
	if err != nil {
		t.Fatalf("first resolution failed: %v", err)
	}

	second, err := Resolve[*testServiceA](c)
	if err != nil {
		t.Fatalf("second resolution failed: %v", err)
	}

	// Assert same instance (pointer comparison)
	if first != second {
		t.Error("expected same instance for singleton service")
	}

	// Assert provider called only once
	if callCount != 1 {
		t.Errorf("expected provider called 1 time, got %d", callCount)
	}
}

func TestResolve_TypeMismatch(t *testing.T) {
	c := New()

	// Register service A
	err := For[*testServiceA](c).ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "a"}
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Try to resolve as B using A's name - this should cause type mismatch
	// We need to use Named() with the wrong type to force mismatch
	aTypeName := TypeName[*testServiceA]()
	_, resolveErr := Resolve[*testServiceB](c, Named(aTypeName))

	if resolveErr == nil {
		t.Fatal("expected type mismatch error")
	}

	if !errors.Is(resolveErr, ErrTypeMismatch) {
		t.Fatalf("expected ErrTypeMismatch, got: %v", resolveErr)
	}
}

func TestResolve_Instance_DirectValue(t *testing.T) {
	c := New()

	original := &testServiceA{value: "pre-built"}

	// Register pre-built instance
	err := For[*testServiceA](c).Instance(original)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Resolve
	resolved, err := Resolve[*testServiceA](c)
	if err != nil {
		t.Fatalf("resolution failed: %v", err)
	}

	// Should be the exact same instance
	if resolved != original {
		t.Error("expected exact same instance as registered")
	}
}

func TestResolve_NamedNotFound(t *testing.T) {
	c := New()

	// Register with type name (default)
	err := For[*testServiceA](c).ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "default"}
	})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	// Try to resolve with different name
	_, resolveErr := Resolve[*testServiceA](c, Named("nonexistent"))

	if resolveErr == nil {
		t.Fatal("expected error for non-existent name")
	}

	if !errors.Is(resolveErr, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", resolveErr)
	}

	// Error should contain the name we searched for
	if !strings.Contains(resolveErr.Error(), "nonexistent") {
		t.Errorf("error should contain searched name: %v", resolveErr)
	}
}
