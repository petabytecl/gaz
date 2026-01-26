package gaz

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

// Test types for resolution tests.
type testServiceA struct {
	value string
}

type testServiceB struct{}

// Circular dependency test types.
type cyclicA struct {
	b *cyclicB
}

type cyclicB struct {
	a *cyclicA
}

// Dependency chain test types.
type depA struct {
	b *depB
}

type depB struct {
	c *depC
}

type depC struct {
	value string
}

// ResolutionSuite tests service resolution functionality.
type ResolutionSuite struct {
	suite.Suite
}

func TestResolutionSuite(t *testing.T) {
	suite.Run(t, new(ResolutionSuite))
}

func (s *ResolutionSuite) TestBasicResolution() {
	c := NewContainer()

	// Register a simple service
	err := For[*testServiceA](c).ProviderFunc(func(_ *Container) *testServiceA {
		return &testServiceA{value: "hello"}
	})
	s.Require().NoError(err, "registration failed")

	// Resolve it
	svc, err := Resolve[*testServiceA](c)
	s.Require().NoError(err, "resolution failed")
	s.Require().NotNil(svc, "expected non-nil service")
	s.Equal("hello", svc.value)
}

func (s *ResolutionSuite) TestNotFound() {
	c := NewContainer()

	// Try to resolve unregistered type
	_, err := Resolve[*testServiceA](c)

	s.Require().Error(err, "expected error for unregistered service")
	s.Require().ErrorIs(err, ErrNotFound)

	// Verify error message contains type name
	s.Contains(err.Error(), "testServiceA", "error should contain type name")
}

func (s *ResolutionSuite) TestNamed() {
	c := NewContainer()

	// Register two services with same type, different names
	err := For[*testServiceA](c).Named("first").ProviderFunc(func(_ *Container) *testServiceA {
		return &testServiceA{value: "first-value"}
	})
	s.Require().NoError(err, "first registration failed")

	err = For[*testServiceA](c).Named("second").ProviderFunc(func(_ *Container) *testServiceA {
		return &testServiceA{value: "second-value"}
	})
	s.Require().NoError(err, "second registration failed")

	// Resolve each by name
	first, err := Resolve[*testServiceA](c, Named("first"))
	s.Require().NoError(err, "first resolution failed")

	second, err := Resolve[*testServiceA](c, Named("second"))
	s.Require().NoError(err, "second resolution failed")

	// Assert different instances
	s.NotSame(first, second, "expected different instances for different names")
	s.Equal("first-value", first.value)
	s.Equal("second-value", second.value)
}

func (s *ResolutionSuite) TestCycleDetection() {
	c := NewContainer()

	// A depends on B
	err := For[*cyclicA](c).Provider(func(c *Container) (*cyclicA, error) {
		b, resolveErr := Resolve[*cyclicB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &cyclicA{b: b}, nil
	})
	s.Require().NoError(err, "registration of A failed")

	// B depends on A (creates cycle)
	err = For[*cyclicB](c).Provider(func(c *Container) (*cyclicB, error) {
		a, resolveErr := Resolve[*cyclicA](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &cyclicB{a: a}, nil
	})
	s.Require().NoError(err, "registration of B failed")

	// Attempt to resolve A should detect cycle
	_, resolveErr := Resolve[*cyclicA](c)

	s.Require().Error(resolveErr, "expected cycle detection error")
	s.Require().ErrorIs(resolveErr, ErrCycle)

	// Verify chain is in error message
	errMsg := resolveErr.Error()
	s.Contains(errMsg, "->", "error should contain dependency chain")

	// Should contain both type names
	s.Contains(errMsg, "cyclicA", "error should contain cyclicA")
	s.Contains(errMsg, "cyclicB", "error should contain cyclicB")
}

func (s *ResolutionSuite) TestProviderErrorPropagates() {
	c := NewContainer()

	providerErr := errors.New("provider failed")

	// Register service with provider that returns error
	err := For[*testServiceA](c).Provider(func(_ *Container) (*testServiceA, error) {
		return nil, providerErr
	})
	s.Require().NoError(err, "registration failed")

	// Resolve should propagate the error
	_, resolveErr := Resolve[*testServiceA](c)

	s.Require().Error(resolveErr, "expected error from provider")
	s.Require().ErrorIs(resolveErr, providerErr, "expected provider error to be wrapped")

	// Error should have resolution context
	s.Contains(
		resolveErr.Error(),
		"resolving",
		"error should contain resolution context",
	)
}

func (s *ResolutionSuite) TestDependencyChain() {
	c := NewContainer()

	// Register C (leaf dependency)
	err := For[*depC](c).ProviderFunc(func(_ *Container) *depC {
		return &depC{value: "leaf"}
	})
	s.Require().NoError(err, "registration of C failed")

	// Register B depending on C
	err = For[*depB](c).Provider(func(c *Container) (*depB, error) {
		resolvedC, resolveErr := Resolve[*depC](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &depB{c: resolvedC}, nil
	})
	s.Require().NoError(err, "registration of B failed")

	// Register A depending on B
	err = For[*depA](c).Provider(func(c *Container) (*depA, error) {
		resolvedB, resolveErr := Resolve[*depB](c)
		if resolveErr != nil {
			return nil, resolveErr
		}
		return &depA{b: resolvedB}, nil
	})
	s.Require().NoError(err, "registration of A failed")

	// Resolve A (should build entire chain)
	a, err := Resolve[*depA](c)
	s.Require().NoError(err, "resolution failed")

	// Verify all three instantiated correctly
	s.Require().NotNil(a, "expected non-nil A")
	s.Require().NotNil(a.b, "expected non-nil B")
	s.Require().NotNil(a.b.c, "expected non-nil C")
	s.Equal("leaf", a.b.c.value)
}

func (s *ResolutionSuite) TestTransientNewInstanceEachTime() {
	c := NewContainer()

	callCount := 0

	// Register transient service
	err := For[*testServiceA](c).Transient().ProviderFunc(func(_ *Container) *testServiceA {
		callCount++
		return &testServiceA{value: "transient"}
	})
	s.Require().NoError(err, "registration failed")

	// Resolve twice
	first, err := Resolve[*testServiceA](c)
	s.Require().NoError(err, "first resolution failed")

	second, err := Resolve[*testServiceA](c)
	s.Require().NoError(err, "second resolution failed")

	// Assert different instances (pointer comparison)
	s.NotSame(first, second, "expected different instances for transient service")

	// Assert provider called twice
	s.Equal(2, callCount, "expected provider called 2 times")
}

func (s *ResolutionSuite) TestSingletonSameInstance() {
	c := NewContainer()

	callCount := 0

	// Register singleton service (default)
	err := For[*testServiceA](c).ProviderFunc(func(_ *Container) *testServiceA {
		callCount++
		return &testServiceA{value: "singleton"}
	})
	s.Require().NoError(err, "registration failed")

	// Resolve twice
	first, err := Resolve[*testServiceA](c)
	s.Require().NoError(err, "first resolution failed")

	second, err := Resolve[*testServiceA](c)
	s.Require().NoError(err, "second resolution failed")

	// Assert same instance (pointer comparison)
	s.Same(first, second, "expected same instance for singleton service")

	// Assert provider called only once
	s.Equal(1, callCount, "expected provider called 1 time")
}

func (s *ResolutionSuite) TestTypeMismatch() {
	c := NewContainer()

	// Register service A
	err := For[*testServiceA](c).ProviderFunc(func(_ *Container) *testServiceA {
		return &testServiceA{value: "a"}
	})
	s.Require().NoError(err, "registration failed")

	// Try to resolve as B using A's name - this should cause type mismatch
	// We need to use Named() with the wrong type to force mismatch
	aTypeName := TypeName[*testServiceA]()
	_, resolveErr := Resolve[*testServiceB](c, Named(aTypeName))

	s.Require().Error(resolveErr, "expected type mismatch error")
	s.Require().ErrorIs(resolveErr, ErrTypeMismatch)
}

func (s *ResolutionSuite) TestInstanceDirectValue() {
	c := NewContainer()

	original := &testServiceA{value: "pre-built"}

	// Register pre-built instance
	err := For[*testServiceA](c).Instance(original)
	s.Require().NoError(err, "registration failed")

	// Resolve
	resolved, err := Resolve[*testServiceA](c)
	s.Require().NoError(err, "resolution failed")

	// Should be the exact same instance
	s.Same(original, resolved, "expected exact same instance as registered")
}

func (s *ResolutionSuite) TestNamedNotFound() {
	c := NewContainer()

	// Register with type name (default)
	err := For[*testServiceA](c).ProviderFunc(func(_ *Container) *testServiceA {
		return &testServiceA{value: "default"}
	})
	s.Require().NoError(err, "registration failed")

	// Try to resolve with different name
	_, resolveErr := Resolve[*testServiceA](c, Named("nonexistent"))

	s.Require().Error(resolveErr, "expected error for non-existent name")
	s.Require().ErrorIs(resolveErr, ErrNotFound)

	// Error should contain the name we searched for
	s.Contains(
		resolveErr.Error(), "nonexistent",
		"error should contain searched name",
	)
}
