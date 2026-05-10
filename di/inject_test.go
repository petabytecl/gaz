package di

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

// =============================================================================
// InjectSuite - Tests for parseTag and injectStruct
// =============================================================================

type InjectSuite struct {
	suite.Suite
}

func TestInjectSuite(t *testing.T) {
	suite.Run(t, new(InjectSuite))
}

// =============================================================================
// parseTag Tests
// =============================================================================

func (s *InjectSuite) TestParseTag_Empty() {
	opts := parseTag("")
	s.False(opts.inject, "empty tag should not set inject")
	s.Empty(opts.name, "empty tag should not set name")
	s.False(opts.optional, "empty tag should not set optional")
}

func (s *InjectSuite) TestParseTag_InjectOnly() {
	opts := parseTag("inject")
	s.True(opts.inject, "should set inject=true")
	s.Empty(opts.name, "should not set name")
	s.False(opts.optional, "should not set optional")
}

func (s *InjectSuite) TestParseTag_InjectWithOptional() {
	opts := parseTag("inject,optional")
	s.True(opts.inject, "should set inject=true")
	s.Empty(opts.name, "should not set name")
	s.True(opts.optional, "should set optional=true")
}

func (s *InjectSuite) TestParseTag_InjectWithName() {
	opts := parseTag("inject,name=myService")
	s.True(opts.inject, "should set inject=true")
	s.Equal("myService", opts.name, "should set name")
	s.False(opts.optional, "should not set optional")
}

func (s *InjectSuite) TestParseTag_InjectWithNameAndOptional() {
	opts := parseTag("inject,name=primary,optional")
	s.True(opts.inject, "should set inject=true")
	s.Equal("primary", opts.name, "should set name")
	s.True(opts.optional, "should set optional=true")
}

func (s *InjectSuite) TestParseTag_OptionalWithoutInject() {
	opts := parseTag("optional")
	s.False(opts.inject, "optional alone should not set inject")
	s.True(opts.optional, "should set optional=true")
}

func (s *InjectSuite) TestParseTag_WhitespaceHandling() {
	opts := parseTag("inject, name=foo")
	s.True(opts.inject, "should set inject=true")
	s.Equal("foo", opts.name, "should handle whitespace before name=")
}

func (s *InjectSuite) TestParseTag_NameOnly() {
	opts := parseTag("name=bar")
	s.False(opts.inject, "name alone should not set inject")
	s.Equal("bar", opts.name, "should set name")
}

// =============================================================================
// injectStruct Tests
// =============================================================================

func (s *InjectSuite) TestInjectStruct_NonPointer() {
	c := New()
	type MyStruct struct {
		Dep string `gaz:"inject"`
	}
	target := MyStruct{} // Not a pointer

	err := injectStruct(c, target, nil)
	s.NoError(err, "non-pointer should return nil error")
}

func (s *InjectSuite) TestInjectStruct_NonStructPointer() {
	c := New()
	err := injectStruct(c, new("hello"), nil)
	s.NoError(err, "pointer to non-struct should return nil error")
}

func (s *InjectSuite) TestInjectStruct_UnexportedField() {
	c := New()
	type myTarget struct {
		unexportedField string `gaz:"inject"`
	}
	target := &myTarget{}

	err := injectStruct(c, target, nil)
	s.ErrorIs(err, ErrNotSettable, "unexported field with gaz tag should return ErrNotSettable")
}

func (s *InjectSuite) TestInjectStruct_OptionalMissingService() {
	c := New()
	type myTarget struct {
		OptDep *testOptionalDep `gaz:"inject,optional"`
	}
	target := &myTarget{}

	err := injectStruct(c, target, nil)
	s.NoError(err, "optional field with missing service should not error")
	s.Nil(target.OptDep, "optional field should remain zero value")
}

func (s *InjectSuite) TestInjectStruct_RequiredMissingService() {
	c := New()
	type myTarget struct {
		Dep *testRequiredDep `gaz:"inject"`
	}
	target := &myTarget{}

	err := injectStruct(c, target, nil)
	s.Error(err, "required field with missing service should error")
	s.ErrorIs(err, ErrNotFound, "should be ErrNotFound")
}

func (s *InjectSuite) TestInjectStruct_TypeMismatch() {
	c := New()
	// Register a string but try to inject into an int field
	s.Require().NoError(For[string](c).Named("myValue").Instance("hello"))

	type myTarget struct {
		Value int `gaz:"inject,name=myValue"`
	}
	target := &myTarget{}

	err := injectStruct(c, target, nil)
	s.ErrorIs(err, ErrTypeMismatch, "type mismatch should return ErrTypeMismatch")
}

func (s *InjectSuite) TestInjectStruct_SuccessfulInjectionByType() {
	c := New()
	dep := &testInjectableDep{value: "injected"}
	s.Require().NoError(For[*testInjectableDep](c).Instance(dep))

	type myTarget struct {
		Dep *testInjectableDep `gaz:"inject"`
	}
	target := &myTarget{}

	err := injectStruct(c, target, nil)
	s.NoError(err, "injection should succeed")
	s.Same(dep, target.Dep, "should inject the registered instance")
}

func (s *InjectSuite) TestInjectStruct_SuccessfulInjectionByName() {
	c := New()
	primary := &testNamedDep{name: "primary"}
	secondary := &testNamedDep{name: "secondary"}
	s.Require().NoError(For[*testNamedDep](c).Named("primary").Instance(primary))
	s.Require().NoError(For[*testNamedDep](c).Named("secondary").Instance(secondary))

	type myTarget struct {
		Primary   *testNamedDep `gaz:"inject,name=primary"`
		Secondary *testNamedDep `gaz:"inject,name=secondary"`
	}
	target := &myTarget{}

	err := injectStruct(c, target, nil)
	s.NoError(err, "injection should succeed")
	s.Same(primary, target.Primary, "should inject primary")
	s.Same(secondary, target.Secondary, "should inject secondary")
}

func (s *InjectSuite) TestInjectStruct_FieldWithoutGazTag() {
	c := New()
	dep := &testInjectableDep{value: "injected"}
	s.Require().NoError(For[*testInjectableDep](c).Instance(dep))

	type myTarget struct {
		Dep    *testInjectableDep `gaz:"inject"`
		NoDep  *testInjectableDep // No gaz tag - should be skipped
		OtherF string             `json:"other"` // Different tag - should be skipped
	}
	target := &myTarget{}

	err := injectStruct(c, target, nil)
	s.NoError(err, "injection should succeed")
	s.Same(dep, target.Dep, "tagged field should be injected")
	s.Nil(target.NoDep, "untagged field should remain nil")
}

func (s *InjectSuite) TestInjectStruct_GazTagWithoutInject() {
	c := New()
	dep := &testInjectableDep{value: "injected"}
	s.Require().NoError(For[*testInjectableDep](c).Instance(dep))

	type myTarget struct {
		NotInjected *testInjectableDep `gaz:"name=something"` // Has gaz tag but no inject
	}
	target := &myTarget{}

	err := injectStruct(c, target, nil)
	s.NoError(err, "should succeed without injecting")
	s.Nil(target.NotInjected, "field with gaz tag but no inject should remain nil")
}

func (s *InjectSuite) TestInjectStruct_DependencyResolutionError() {
	c := New()
	// Register a service that will fail during resolution
	err := For[*testFailingDep](c).Provider(func(_ *Container) (*testFailingDep, error) {
		return nil, errors.New("provider failed")
	})
	s.Require().NoError(err)

	type myTarget struct {
		Dep *testFailingDep `gaz:"inject"`
	}
	target := &myTarget{}

	err = injectStruct(c, target, nil)
	s.Error(err, "should propagate resolution error")
	s.Contains(err.Error(), "provider failed", "should contain original error")
}

// =============================================================================
// Inject Field Memoization Tests
// =============================================================================

func (s *InjectSuite) TestInjectFieldMemoization_CacheHit() {
	c := New()
	dep := &testInjectableDep{value: "cached"}
	s.Require().NoError(For[*testInjectableDep](c).Instance(dep))

	type cachedTarget struct {
		Dep *testInjectableDep `gaz:"inject"`
	}

	// First injection parses and caches
	target1 := &cachedTarget{}
	err := injectStruct(c, target1, nil)
	s.NoError(err, "first injection should succeed")
	s.Same(dep, target1.Dep, "first injection should set the field")

	// Second injection should use cache
	target2 := &cachedTarget{}
	err = injectStruct(c, target2, nil)
	s.NoError(err, "second injection should succeed (cache hit)")
	s.Same(dep, target2.Dep, "second injection should set the field from cache")
}

func (s *InjectSuite) TestGetInjectFields_Idempotent() {
	type idempotentTarget struct {
		Dep *testInjectableDep `gaz:"inject"`
	}

	t := reflect.TypeOf(idempotentTarget{})

	fields1, err := getInjectFields(t)
	s.Require().NoError(err)
	s.Len(fields1, 1, "should find one inject field")
	s.NotEmpty(fields1[0].serviceName, "service name should be set")

	// Second call returns same result (from cache)
	fields2, err := getInjectFields(t)
	s.Require().NoError(err)
	s.Len(fields2, 1, "cached result should match")
	s.Equal(fields1[0].serviceName, fields2[0].serviceName)
	s.Equal(fields1[0].index, fields2[0].index)
	s.Equal(fields1[0].optional, fields2[0].optional)
}

// =============================================================================
// Test Helper Types
// =============================================================================

type (
	testOptionalDep   struct{}
	testRequiredDep   struct{}
	testInjectableDep struct{ value string }
	testNamedDep      struct{ name string }
	testFailingDep    struct{}
)
