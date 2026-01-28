package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

// =============================================================================
// ResolutionSuite
// =============================================================================

type ResolutionSuite struct {
	suite.Suite
}

func TestResolutionSuite(t *testing.T) {
	suite.Run(t, new(ResolutionSuite))
}

// =============================================================================
// Test types for resolution tests
// =============================================================================

type testResolveServiceA struct {
	value string
}

type testResolveServiceB struct{}

// Circular dependency test types.
type testResolveCyclicA struct {
	b *testResolveCyclicB
}

type testResolveCyclicB struct {
	a *testResolveCyclicA
}

// Dependency chain test types.
type testResolveDepA struct {
	b *testResolveDepB
}

type testResolveDepB struct {
	c *testResolveDepC
}

type testResolveDepC struct {
	value string
}

// =============================================================================
// Resolve[T]() Tests
// =============================================================================

func (s *ResolutionSuite) TestResolve_BasicResolution() {
	c := New()

	// Register a simple service
	err := For[*testResolveServiceA](c).ProviderFunc(func(_ *Container) *testResolveServiceA {
		return &testResolveServiceA{value: "hello"}
	})
	s.Require().NoError(err, "registration failed")

	// Resolve it
	svc, err := Resolve[*testResolveServiceA](c)
	s.Require().NoError(err, "resolution failed")
	s.Require().NotNil(svc, "expected non-nil service")
	s.Equal("hello", svc.value)
}

func (s *ResolutionSuite) TestResolve_NotFound() {
	c := New()

	// Try to resolve unregistered type
	_, err := Resolve[*testResolveServiceA](c)

	s.Require().Error(err, "expected error for unregistered service")
	s.Require().ErrorIs(err, ErrNotFound)

	// Verify error message contains type name
	s.Contains(err.Error(), "testResolveServiceA", "error should contain type name")
}

func (s *ResolutionSuite) TestResolve_Named() {
	c := New()

	// Register two services with same type, different names
	err := For[*testResolveServiceA](c).Named("first").ProviderFunc(func(_ *Container) *testResolveServiceA {
		return &testResolveServiceA{value: "first-value"}
	})
	s.Require().NoError(err, "first registration failed")

	err = For[*testResolveServiceA](c).Named("second").ProviderFunc(func(_ *Container) *testResolveServiceA {
		return &testResolveServiceA{value: "second-value"}
	})
	s.Require().NoError(err, "second registration failed")

	// Resolve each by name
	first, err := Resolve[*testResolveServiceA](c, Named("first"))
	s.Require().NoError(err, "first resolution failed")

	second, err := Resolve[*testResolveServiceA](c, Named("second"))
	s.Require().NoError(err, "second resolution failed")

	// Assert different instances
	s.NotSame(first, second, "expected different instances for different names")
	s.Equal("first-value", first.value)
	s.Equal("second-value", second.value)
}

func (s *ResolutionSuite) TestResolve_CycleDetection() {
	c := New()

	// A depends on B
	err := For[*testResolveCyclicA](c).Provider(func(c *Container) (*testResolveCyclicA, error) {
		b, resolveErr := Resolve[*testResolveCyclicB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testResolveCyclicA{b: b}, nil
	})
	s.Require().NoError(err, "registration of A failed")

	// B depends on A (creates cycle)
	err = For[*testResolveCyclicB](c).Provider(func(c *Container) (*testResolveCyclicB, error) {
		a, resolveErr := Resolve[*testResolveCyclicA](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testResolveCyclicB{a: a}, nil
	})
	s.Require().NoError(err, "registration of B failed")

	// Attempt to resolve A should detect cycle
	_, resolveErr := Resolve[*testResolveCyclicA](c)

	s.Require().Error(resolveErr, "expected cycle detection error")
	s.Require().ErrorIs(resolveErr, ErrCycle)

	// Verify chain is in error message
	errMsg := resolveErr.Error()
	s.Contains(errMsg, "->", "error should contain dependency chain")

	// Should contain both type names
	s.Contains(errMsg, "testResolveCyclicA", "error should contain cyclicA")
	s.Contains(errMsg, "testResolveCyclicB", "error should contain cyclicB")
}

func (s *ResolutionSuite) TestResolve_ProviderErrorPropagates() {
	c := New()

	providerErr := errors.New("provider failed")

	// Register service with provider that returns error
	err := For[*testResolveServiceA](c).Provider(func(_ *Container) (*testResolveServiceA, error) {
		return nil, providerErr
	})
	s.Require().NoError(err, "registration failed")

	// Resolve should propagate the error
	_, resolveErr := Resolve[*testResolveServiceA](c)

	s.Require().Error(resolveErr, "expected error from provider")
	s.Require().ErrorIs(resolveErr, providerErr, "expected provider error to be wrapped")

	// Error should have resolution context
	s.Contains(
		resolveErr.Error(),
		"resolving",
		"error should contain resolution context",
	)
}

func (s *ResolutionSuite) TestResolve_DependencyChain() {
	c := New()

	// Register C (leaf dependency)
	err := For[*testResolveDepC](c).ProviderFunc(func(_ *Container) *testResolveDepC {
		return &testResolveDepC{value: "leaf"}
	})
	s.Require().NoError(err, "registration of C failed")

	// Register B depending on C
	err = For[*testResolveDepB](c).Provider(func(c *Container) (*testResolveDepB, error) {
		resolvedC, resolveErr := Resolve[*testResolveDepC](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testResolveDepB{c: resolvedC}, nil
	})
	s.Require().NoError(err, "registration of B failed")

	// Register A depending on B
	err = For[*testResolveDepA](c).Provider(func(c *Container) (*testResolveDepA, error) {
		resolvedB, resolveErr := Resolve[*testResolveDepB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &testResolveDepA{b: resolvedB}, nil
	})
	s.Require().NoError(err, "registration of A failed")

	// Resolve A (should build entire chain)
	a, err := Resolve[*testResolveDepA](c)
	s.Require().NoError(err, "resolution failed")

	// Verify all three instantiated correctly
	s.Require().NotNil(a, "expected non-nil A")
	s.Require().NotNil(a.b, "expected non-nil B")
	s.Require().NotNil(a.b.c, "expected non-nil C")
	s.Equal("leaf", a.b.c.value)
}

func (s *ResolutionSuite) TestResolve_TransientNewInstanceEachTime() {
	c := New()

	callCount := 0

	// Register transient service
	err := For[*testResolveServiceA](c).Transient().ProviderFunc(func(_ *Container) *testResolveServiceA {
		callCount++
		return &testResolveServiceA{value: "transient"}
	})
	s.Require().NoError(err, "registration failed")

	// Resolve twice
	first, err := Resolve[*testResolveServiceA](c)
	s.Require().NoError(err, "first resolution failed")

	second, err := Resolve[*testResolveServiceA](c)
	s.Require().NoError(err, "second resolution failed")

	// Assert different instances (pointer comparison)
	s.NotSame(first, second, "expected different instances for transient service")

	// Assert provider called twice
	s.Equal(2, callCount, "expected provider called 2 times")
}

func (s *ResolutionSuite) TestResolve_SingletonSameInstance() {
	c := New()

	callCount := 0

	// Register singleton service (default)
	err := For[*testResolveServiceA](c).ProviderFunc(func(_ *Container) *testResolveServiceA {
		callCount++
		return &testResolveServiceA{value: "singleton"}
	})
	s.Require().NoError(err, "registration failed")

	// Resolve twice
	first, err := Resolve[*testResolveServiceA](c)
	s.Require().NoError(err, "first resolution failed")

	second, err := Resolve[*testResolveServiceA](c)
	s.Require().NoError(err, "second resolution failed")

	// Assert same instance (pointer comparison)
	s.Same(first, second, "expected same instance for singleton service")

	// Assert provider called only once
	s.Equal(1, callCount, "expected provider called 1 time")
}

func (s *ResolutionSuite) TestResolve_TypeMismatch() {
	c := New()

	// Register service A
	err := For[*testResolveServiceA](c).ProviderFunc(func(_ *Container) *testResolveServiceA {
		return &testResolveServiceA{value: "a"}
	})
	s.Require().NoError(err, "registration failed")

	// Try to resolve as B using A's name - this should cause type mismatch
	aTypeName := TypeName[*testResolveServiceA]()
	_, resolveErr := Resolve[*testResolveServiceB](c, Named(aTypeName))

	s.Require().Error(resolveErr, "expected type mismatch error")
	s.Require().ErrorIs(resolveErr, ErrTypeMismatch)
}

func (s *ResolutionSuite) TestResolve_InstanceDirectValue() {
	c := New()

	original := &testResolveServiceA{value: "pre-built"}

	// Register pre-built instance
	err := For[*testResolveServiceA](c).Instance(original)
	s.Require().NoError(err, "registration failed")

	// Resolve
	resolved, err := Resolve[*testResolveServiceA](c)
	s.Require().NoError(err, "resolution failed")

	// Should be the exact same instance
	s.Same(original, resolved, "expected exact same instance as registered")
}

func (s *ResolutionSuite) TestResolve_NamedNotFound() {
	c := New()

	// Register with type name (default)
	err := For[*testResolveServiceA](c).ProviderFunc(func(_ *Container) *testResolveServiceA {
		return &testResolveServiceA{value: "default"}
	})
	s.Require().NoError(err, "registration failed")

	// Try to resolve with different name
	_, resolveErr := Resolve[*testResolveServiceA](c, Named("nonexistent"))

	s.Require().Error(resolveErr, "expected error for non-existent name")
	s.Require().ErrorIs(resolveErr, ErrNotFound)

	// Error should contain the name we searched for
	s.Contains(
		resolveErr.Error(), "nonexistent",
		"error should contain searched name",
	)
}

// =============================================================================
// MustResolve[T]() Tests
// =============================================================================

func (s *ResolutionSuite) TestMustResolve_Success() {
	c := New()

	err := For[*testResolveServiceA](c).Instance(&testResolveServiceA{value: "test"})
	s.Require().NoError(err)

	// MustResolve should return the instance
	svc := MustResolve[*testResolveServiceA](c)
	s.NotNil(svc)
	s.Equal("test", svc.value)
}

func (s *ResolutionSuite) TestMustResolve_PanicsOnNotFound() {
	c := New()

	// MustResolve should panic when service is not found
	s.Panics(func() {
		MustResolve[*testResolveServiceA](c)
	}, "MustResolve should panic when service not found")
}

func (s *ResolutionSuite) TestMustResolve_PanicsOnProviderError() {
	c := New()

	err := For[*testResolveServiceA](c).Provider(func(_ *Container) (*testResolveServiceA, error) {
		return nil, errors.New("provider error")
	})
	s.Require().NoError(err)

	// MustResolve should panic on provider error
	s.Panics(func() {
		MustResolve[*testResolveServiceA](c)
	}, "MustResolve should panic when provider returns error")
}

func (s *ResolutionSuite) TestMustResolve_PanicMessageContainsTypeName() {
	c := New()

	// Recover panic and check message
	defer func() {
		r := recover()
		s.Require().NotNil(r, "expected panic")
		panicMsg, ok := r.(string)
		s.Require().True(ok, "panic should be a string")
		s.Contains(panicMsg, "testResolveServiceA", "panic message should contain type name")
		s.Contains(panicMsg, "MustResolve", "panic message should mention MustResolve")
	}()

	MustResolve[*testResolveServiceA](c)
}

func (s *ResolutionSuite) TestMustResolve_WithNamed() {
	c := New()

	err := For[*testResolveServiceA](c).Named("special").Instance(&testResolveServiceA{value: "named"})
	s.Require().NoError(err)

	// MustResolve with Named option
	svc := MustResolve[*testResolveServiceA](c, Named("special"))
	s.NotNil(svc)
	s.Equal("named", svc.value)
}

// =============================================================================
// NewTestContainer() Tests
// =============================================================================

func (s *ResolutionSuite) TestNewTestContainer_ReturnsValidContainer() {
	c := NewTestContainer()
	s.NotNil(c, "NewTestContainer should return non-nil container")
	s.Equal(0, len(c.List()), "NewTestContainer should return empty container")
}

func (s *ResolutionSuite) TestNewTestContainer_CanRegisterAndResolve() {
	c := NewTestContainer()

	// Register a service
	err := For[*testResolveServiceA](c).Instance(&testResolveServiceA{value: "test"})
	s.Require().NoError(err)

	// Resolve it
	svc, err := Resolve[*testResolveServiceA](c)
	s.Require().NoError(err)
	s.Equal("test", svc.value)
}

func (s *ResolutionSuite) TestNewTestContainer_FunctionallyIdenticalToNew() {
	// Verify NewTestContainer behaves identically to New
	cNew := New()
	cTest := NewTestContainer()

	// Both should start empty
	s.Equal(cNew.List(), cTest.List())

	// Both should support same registration patterns
	s.Require().NoError(For[*testResolveServiceA](cNew).Instance(&testResolveServiceA{value: "a"}))
	s.Require().NoError(For[*testResolveServiceA](cTest).Instance(&testResolveServiceA{value: "a"}))

	// Both should support resolution
	_, err1 := Resolve[*testResolveServiceA](cNew)
	_, err2 := Resolve[*testResolveServiceA](cTest)
	s.Equal(err1 == nil, err2 == nil)
}
