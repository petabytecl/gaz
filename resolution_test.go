package gaz

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

// ResolutionSuite tests service resolution functionality.
type ResolutionSuite struct {
	suite.Suite
}

func TestResolutionSuite(t *testing.T) {
	suite.Run(t, new(ResolutionSuite))
}

func (s *ResolutionSuite) TestBasicResolution() {
	c := New()

	// Register a simple service
	err := For[*testServiceA](c).ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "hello"}
	})
	require.NoError(s.T(), err, "registration failed")

	// Resolve it
	svc, err := Resolve[*testServiceA](c)
	require.NoError(s.T(), err, "resolution failed")
	require.NotNil(s.T(), svc, "expected non-nil service")
	assert.Equal(s.T(), "hello", svc.value)
}

func (s *ResolutionSuite) TestNotFound() {
	c := New()

	// Try to resolve unregistered type
	_, err := Resolve[*testServiceA](c)

	require.Error(s.T(), err, "expected error for unregistered service")
	assert.ErrorIs(s.T(), err, ErrNotFound)

	// Verify error message contains type name
	assert.Contains(s.T(), err.Error(), "testServiceA", "error should contain type name")
}

func (s *ResolutionSuite) TestNamed() {
	c := New()

	// Register two services with same type, different names
	err := For[*testServiceA](c).Named("first").ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "first-value"}
	})
	require.NoError(s.T(), err, "first registration failed")

	err = For[*testServiceA](c).Named("second").ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "second-value"}
	})
	require.NoError(s.T(), err, "second registration failed")

	// Resolve each by name
	first, err := Resolve[*testServiceA](c, Named("first"))
	require.NoError(s.T(), err, "first resolution failed")

	second, err := Resolve[*testServiceA](c, Named("second"))
	require.NoError(s.T(), err, "second resolution failed")

	// Assert different instances
	assert.NotSame(s.T(), first, second, "expected different instances for different names")
	assert.Equal(s.T(), "first-value", first.value)
	assert.Equal(s.T(), "second-value", second.value)
}

func (s *ResolutionSuite) TestCycleDetection() {
	c := New()

	// A depends on B
	err := For[*cyclicA](c).Provider(func(c *Container) (*cyclicA, error) {
		b, err := Resolve[*cyclicB](c)
		if err != nil {
			return nil, err
		}
		return &cyclicA{b: b}, nil
	})
	require.NoError(s.T(), err, "registration of A failed")

	// B depends on A (creates cycle)
	err = For[*cyclicB](c).Provider(func(c *Container) (*cyclicB, error) {
		a, err := Resolve[*cyclicA](c)
		if err != nil {
			return nil, err
		}
		return &cyclicB{a: a}, nil
	})
	require.NoError(s.T(), err, "registration of B failed")

	// Attempt to resolve A should detect cycle
	_, resolveErr := Resolve[*cyclicA](c)

	require.Error(s.T(), resolveErr, "expected cycle detection error")
	assert.ErrorIs(s.T(), resolveErr, ErrCycle)

	// Verify chain is in error message
	errMsg := resolveErr.Error()
	assert.Contains(s.T(), errMsg, "->", "error should contain dependency chain")

	// Should contain both type names
	assert.Contains(s.T(), errMsg, "cyclicA", "error should contain cyclicA")
	assert.Contains(s.T(), errMsg, "cyclicB", "error should contain cyclicB")
}

func (s *ResolutionSuite) TestProviderErrorPropagates() {
	c := New()

	providerErr := errors.New("provider failed")

	// Register service with provider that returns error
	err := For[*testServiceA](c).Provider(func(c *Container) (*testServiceA, error) {
		return nil, providerErr
	})
	require.NoError(s.T(), err, "registration failed")

	// Resolve should propagate the error
	_, resolveErr := Resolve[*testServiceA](c)

	require.Error(s.T(), resolveErr, "expected error from provider")
	assert.ErrorIs(s.T(), resolveErr, providerErr, "expected provider error to be wrapped")

	// Error should have resolution context
	assert.Contains(s.T(), resolveErr.Error(), "resolving", "error should contain resolution context")
}

func (s *ResolutionSuite) TestDependencyChain() {
	c := New()

	// Register C (leaf dependency)
	err := For[*depC](c).ProviderFunc(func(c *Container) *depC {
		return &depC{value: "leaf"}
	})
	require.NoError(s.T(), err, "registration of C failed")

	// Register B depending on C
	err = For[*depB](c).Provider(func(c *Container) (*depB, error) {
		depC, err := Resolve[*depC](c)
		if err != nil {
			return nil, err
		}
		return &depB{c: depC}, nil
	})
	require.NoError(s.T(), err, "registration of B failed")

	// Register A depending on B
	err = For[*depA](c).Provider(func(c *Container) (*depA, error) {
		depB, err := Resolve[*depB](c)
		if err != nil {
			return nil, err
		}
		return &depA{b: depB}, nil
	})
	require.NoError(s.T(), err, "registration of A failed")

	// Resolve A (should build entire chain)
	a, err := Resolve[*depA](c)
	require.NoError(s.T(), err, "resolution failed")

	// Verify all three instantiated correctly
	require.NotNil(s.T(), a, "expected non-nil A")
	require.NotNil(s.T(), a.b, "expected non-nil B")
	require.NotNil(s.T(), a.b.c, "expected non-nil C")
	assert.Equal(s.T(), "leaf", a.b.c.value)
}

func (s *ResolutionSuite) TestTransientNewInstanceEachTime() {
	c := New()

	callCount := 0

	// Register transient service
	err := For[*testServiceA](c).Transient().ProviderFunc(func(c *Container) *testServiceA {
		callCount++
		return &testServiceA{value: "transient"}
	})
	require.NoError(s.T(), err, "registration failed")

	// Resolve twice
	first, err := Resolve[*testServiceA](c)
	require.NoError(s.T(), err, "first resolution failed")

	second, err := Resolve[*testServiceA](c)
	require.NoError(s.T(), err, "second resolution failed")

	// Assert different instances (pointer comparison)
	assert.NotSame(s.T(), first, second, "expected different instances for transient service")

	// Assert provider called twice
	assert.Equal(s.T(), 2, callCount, "expected provider called 2 times")
}

func (s *ResolutionSuite) TestSingletonSameInstance() {
	c := New()

	callCount := 0

	// Register singleton service (default)
	err := For[*testServiceA](c).ProviderFunc(func(c *Container) *testServiceA {
		callCount++
		return &testServiceA{value: "singleton"}
	})
	require.NoError(s.T(), err, "registration failed")

	// Resolve twice
	first, err := Resolve[*testServiceA](c)
	require.NoError(s.T(), err, "first resolution failed")

	second, err := Resolve[*testServiceA](c)
	require.NoError(s.T(), err, "second resolution failed")

	// Assert same instance (pointer comparison)
	assert.Same(s.T(), first, second, "expected same instance for singleton service")

	// Assert provider called only once
	assert.Equal(s.T(), 1, callCount, "expected provider called 1 time")
}

func (s *ResolutionSuite) TestTypeMismatch() {
	c := New()

	// Register service A
	err := For[*testServiceA](c).ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "a"}
	})
	require.NoError(s.T(), err, "registration failed")

	// Try to resolve as B using A's name - this should cause type mismatch
	// We need to use Named() with the wrong type to force mismatch
	aTypeName := TypeName[*testServiceA]()
	_, resolveErr := Resolve[*testServiceB](c, Named(aTypeName))

	require.Error(s.T(), resolveErr, "expected type mismatch error")
	assert.ErrorIs(s.T(), resolveErr, ErrTypeMismatch)
}

func (s *ResolutionSuite) TestInstanceDirectValue() {
	c := New()

	original := &testServiceA{value: "pre-built"}

	// Register pre-built instance
	err := For[*testServiceA](c).Instance(original)
	require.NoError(s.T(), err, "registration failed")

	// Resolve
	resolved, err := Resolve[*testServiceA](c)
	require.NoError(s.T(), err, "resolution failed")

	// Should be the exact same instance
	assert.Same(s.T(), original, resolved, "expected exact same instance as registered")
}

func (s *ResolutionSuite) TestNamedNotFound() {
	c := New()

	// Register with type name (default)
	err := For[*testServiceA](c).ProviderFunc(func(c *Container) *testServiceA {
		return &testServiceA{value: "default"}
	})
	require.NoError(s.T(), err, "registration failed")

	// Try to resolve with different name
	_, resolveErr := Resolve[*testServiceA](c, Named("nonexistent"))

	require.Error(s.T(), resolveErr, "expected error for non-existent name")
	assert.ErrorIs(s.T(), resolveErr, ErrNotFound)

	// Error should contain the name we searched for
	assert.True(s.T(), strings.Contains(resolveErr.Error(), "nonexistent"), "error should contain searched name")
}
