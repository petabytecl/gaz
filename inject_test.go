package gaz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Test types for injection tests
type Database struct {
	connStr string
}

type Logger struct {
	level string
}

type Cache struct {
	addr string
}

type Handler struct {
	DB     *Database `gaz:"inject"`
	Logger *Logger   `gaz:"inject"`
}

// InjectionSuite tests field injection functionality.
type InjectionSuite struct {
	suite.Suite
}

func TestInjectionSuite(t *testing.T) {
	suite.Run(t, new(InjectionSuite))
}

func (s *InjectionSuite) TestBasicInjection() {
	c := New()

	db := &Database{connStr: "postgres://localhost"}
	logger := &Logger{level: "debug"}

	require.NoError(s.T(), For[*Database](c).Instance(db))
	require.NoError(s.T(), For[*Logger](c).Instance(logger))
	require.NoError(s.T(), For[*Handler](c).ProviderFunc(func(c *Container) *Handler {
		return &Handler{} // Fields auto-injected after provider returns
	}))

	h, err := Resolve[*Handler](c)
	require.NoError(s.T(), err)

	assert.NotNil(s.T(), h.DB, "DB field should be injected")
	assert.NotNil(s.T(), h.Logger, "Logger field should be injected")
	assert.Same(s.T(), db, h.DB, "DB should be the same instance")
	assert.Same(s.T(), logger, h.Logger, "Logger should be the same instance")
}

// DB type for named injection tests
type DB struct {
	name string
}

type ServiceWithNamedDeps struct {
	Primary *DB `gaz:"inject,name=primary"`
	Replica *DB `gaz:"inject,name=replica"`
}

func (s *InjectionSuite) TestNamed() {
	c := New()

	primary := &DB{name: "primary"}
	replica := &DB{name: "replica"}

	require.NoError(s.T(), For[*DB](c).Named("primary").Instance(primary))
	require.NoError(s.T(), For[*DB](c).Named("replica").Instance(replica))
	require.NoError(s.T(), For[*ServiceWithNamedDeps](c).ProviderFunc(func(c *Container) *ServiceWithNamedDeps {
		return &ServiceWithNamedDeps{}
	}))

	svc, err := Resolve[*ServiceWithNamedDeps](c)
	require.NoError(s.T(), err)

	require.NotNil(s.T(), svc.Primary, "Primary should be injected")
	assert.Equal(s.T(), "primary", svc.Primary.name)
	require.NotNil(s.T(), svc.Replica, "Replica should be injected")
	assert.Equal(s.T(), "replica", svc.Replica.name)
}

type HandlerWithOptionalCache struct {
	Cache *Cache `gaz:"inject,optional"`
}

func (s *InjectionSuite) TestOptionalNotRegistered() {
	c := New()

	// Don't register Cache
	require.NoError(s.T(), For[*HandlerWithOptionalCache](c).ProviderFunc(func(c *Container) *HandlerWithOptionalCache {
		return &HandlerWithOptionalCache{}
	}))

	h, err := Resolve[*HandlerWithOptionalCache](c)
	require.NoError(s.T(), err)

	assert.Nil(s.T(), h.Cache, "Cache should be nil when not registered")
}

func (s *InjectionSuite) TestOptionalRegistered() {
	c := New()

	cache := &Cache{addr: "localhost:6379"}
	require.NoError(s.T(), For[*Cache](c).Instance(cache))
	require.NoError(s.T(), For[*HandlerWithOptionalCache](c).ProviderFunc(func(c *Container) *HandlerWithOptionalCache {
		return &HandlerWithOptionalCache{}
	}))

	h, err := Resolve[*HandlerWithOptionalCache](c)
	require.NoError(s.T(), err)

	assert.NotNil(s.T(), h.Cache, "Cache should be populated when registered")
	assert.Same(s.T(), cache, h.Cache, "Cache should be the same instance")
}

type BadHandler struct {
	db *Database `gaz:"inject"` // unexported field!
}

func (s *InjectionSuite) TestUnexportedFieldReturnsError() {
	c := New()

	require.NoError(s.T(), For[*Database](c).Instance(&Database{}))
	require.NoError(s.T(), For[*BadHandler](c).ProviderFunc(func(c *Container) *BadHandler {
		return &BadHandler{}
	}))

	_, err := Resolve[*BadHandler](c)
	assert.Error(s.T(), err, "expected error for unexported field")
	assert.ErrorIs(s.T(), err, ErrNotSettable)
}

type HandlerWithMissingDep struct {
	DB *Database `gaz:"inject"` // not registered, not optional
}

func (s *InjectionSuite) TestMissingDependencyReturnsError() {
	c := New()

	// Don't register Database
	require.NoError(s.T(), For[*HandlerWithMissingDep](c).ProviderFunc(func(c *Container) *HandlerWithMissingDep {
		return &HandlerWithMissingDep{}
	}))

	_, err := Resolve[*HandlerWithMissingDep](c)
	assert.Error(s.T(), err, "expected error for missing dependency")
	assert.ErrorIs(s.T(), err, ErrNotFound)
}

// Types for cycle detection test
type ServiceA struct {
	B *ServiceB `gaz:"inject"`
}

type ServiceB struct {
	A *ServiceA `gaz:"inject"`
}

func (s *InjectionSuite) TestCycleViaInjection() {
	c := New()

	require.NoError(s.T(), For[*ServiceA](c).ProviderFunc(func(c *Container) *ServiceA {
		return &ServiceA{}
	}))
	require.NoError(s.T(), For[*ServiceB](c).ProviderFunc(func(c *Container) *ServiceB {
		return &ServiceB{}
	}))

	_, err := Resolve[*ServiceA](c)
	assert.Error(s.T(), err, "expected error for circular dependency")
	assert.ErrorIs(s.T(), err, ErrCycle)
}

func (s *InjectionSuite) TestNonStructPointerSkipped() {
	c := New()

	// Register a simple string - not a struct
	require.NoError(s.T(), For[string](c).ProviderFunc(func(c *Container) string {
		return "hello"
	}))

	// Should resolve without error - injection silently skipped for non-struct
	str, err := Resolve[string](c)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "hello", str)
}

func (s *InjectionSuite) TestParseTag() {
	tests := []struct {
		tag      string
		expected tagOptions
	}{
		{"inject", tagOptions{inject: true}},
		{"inject,optional", tagOptions{inject: true, optional: true}},
		{"inject,name=primary", tagOptions{inject: true, name: "primary"}},
		{"inject,name=primary,optional", tagOptions{inject: true, name: "primary", optional: true}},
		{"inject, name=foo , optional", tagOptions{inject: true, name: "foo", optional: true}}, // with spaces
		{"optional,inject", tagOptions{inject: true, optional: true}},                          // order doesn't matter
		{"name=foo", tagOptions{name: "foo", inject: false}},                                   // missing inject keyword
		{"", tagOptions{}}, // empty tag
	}

	for _, tt := range tests {
		s.Run(tt.tag, func() {
			opts := parseTag(tt.tag)
			assert.Equal(s.T(), tt.expected.inject, opts.inject, "inject")
			assert.Equal(s.T(), tt.expected.name, opts.name, "name")
			assert.Equal(s.T(), tt.expected.optional, opts.optional, "optional")
		})
	}
}

// Test that transient services also get injection
type TransientHandler struct {
	DB *Database `gaz:"inject"`
}

func (s *InjectionSuite) TestTransientService() {
	c := New()

	db := &Database{connStr: "test"}
	require.NoError(s.T(), For[*Database](c).Instance(db))
	require.NoError(s.T(), For[*TransientHandler](c).Transient().ProviderFunc(func(c *Container) *TransientHandler {
		return &TransientHandler{}
	}))

	h1, err := Resolve[*TransientHandler](c)
	require.NoError(s.T(), err)
	h2, err := Resolve[*TransientHandler](c)
	require.NoError(s.T(), err)

	// Different handler instances (transient)
	assert.NotSame(s.T(), h1, h2, "transient services should return different instances")
	// Both should have DB injected
	assert.NotNil(s.T(), h1.DB, "first handler should have DB injected")
	assert.NotNil(s.T(), h2.DB, "second handler should have DB injected")
	// Same DB instance (singleton)
	assert.Same(s.T(), h1.DB, h2.DB, "both handlers should have the same DB instance")
}

// Test injection doesn't happen on pre-built instances
type PreBuiltHandler struct {
	DB *Database `gaz:"inject"`
}

func (s *InjectionSuite) TestInstanceServiceNoInjection() {
	c := New()

	db := &Database{connStr: "test"}
	require.NoError(s.T(), For[*Database](c).Instance(db))

	// Pre-built instance with empty DB field
	handler := &PreBuiltHandler{}
	require.NoError(s.T(), For[*PreBuiltHandler](c).Instance(handler))

	h, err := Resolve[*PreBuiltHandler](c)
	require.NoError(s.T(), err)

	// Instance services don't get injection - DB should still be nil
	assert.Nil(s.T(), h.DB, "pre-built instances should not get injection")
}
