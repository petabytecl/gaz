package gaz

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// Test types for injection tests.
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

	s.Require().NoError(For[*Database](c).Instance(db))
	s.Require().NoError(For[*Logger](c).Instance(logger))
	s.Require().NoError(For[*Handler](c).ProviderFunc(func(_ *Container) *Handler {
		return &Handler{} // Fields auto-injected after provider returns
	}))

	h, err := Resolve[*Handler](c)
	s.Require().NoError(err)

	s.NotNil(h.DB, "DB field should be injected")
	s.NotNil(h.Logger, "Logger field should be injected")
	s.Same(db, h.DB, "DB should be the same instance")
	s.Same(logger, h.Logger, "Logger should be the same instance")
}

// DB type for named injection tests.
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

	s.Require().NoError(For[*DB](c).Named("primary").Instance(primary))
	s.Require().NoError(For[*DB](c).Named("replica").Instance(replica))
	s.Require().NoError(
		For[*ServiceWithNamedDeps](c).ProviderFunc(func(_ *Container) *ServiceWithNamedDeps {
			return &ServiceWithNamedDeps{}
		}),
	)

	svc, err := Resolve[*ServiceWithNamedDeps](c)
	s.Require().NoError(err)

	s.Require().NotNil(svc.Primary, "Primary should be injected")
	s.Equal("primary", svc.Primary.name)
	s.Require().NotNil(svc.Replica, "Replica should be injected")
	s.Equal("replica", svc.Replica.name)
}

type HandlerWithOptionalCache struct {
	Cache *Cache `gaz:"inject,optional"`
}

func (s *InjectionSuite) TestOptionalNotRegistered() {
	c := New()

	// Don't register Cache
	s.Require().NoError(
		For[*HandlerWithOptionalCache](
			c,
		).ProviderFunc(func(_ *Container) *HandlerWithOptionalCache {
			return &HandlerWithOptionalCache{}
		}),
	)

	h, err := Resolve[*HandlerWithOptionalCache](c)
	s.Require().NoError(err)

	s.Nil(h.Cache, "Cache should be nil when not registered")
}

func (s *InjectionSuite) TestOptionalRegistered() {
	c := New()

	cache := &Cache{addr: "localhost:6379"}
	s.Require().NoError(For[*Cache](c).Instance(cache))
	s.Require().NoError(
		For[*HandlerWithOptionalCache](
			c,
		).ProviderFunc(func(_ *Container) *HandlerWithOptionalCache {
			return &HandlerWithOptionalCache{}
		}),
	)

	h, err := Resolve[*HandlerWithOptionalCache](c)
	s.Require().NoError(err)

	s.NotNil(h.Cache, "Cache should be populated when registered")
	s.Same(cache, h.Cache, "Cache should be the same instance")
}

type BadHandler struct {
	db *Database `gaz:"inject"` // unexported field!
}

func (s *InjectionSuite) TestUnexportedFieldReturnsError() {
	c := New()

	s.Require().NoError(For[*Database](c).Instance(&Database{}))
	s.Require().NoError(For[*BadHandler](c).ProviderFunc(func(_ *Container) *BadHandler {
		return &BadHandler{}
	}))

	_, err := Resolve[*BadHandler](c)
	s.Error(err, "expected error for unexported field")
	s.ErrorIs(err, ErrNotSettable)
}

type HandlerWithMissingDep struct {
	DB *Database `gaz:"inject"` // not registered, not optional
}

func (s *InjectionSuite) TestMissingDependencyReturnsError() {
	c := New()

	// Don't register Database
	s.Require().NoError(
		For[*HandlerWithMissingDep](c).ProviderFunc(func(_ *Container) *HandlerWithMissingDep {
			return &HandlerWithMissingDep{}
		}),
	)

	_, err := Resolve[*HandlerWithMissingDep](c)
	s.Error(err, "expected error for missing dependency")
	s.ErrorIs(err, ErrNotFound)
}

// Types for cycle detection test.
type ServiceA struct {
	B *ServiceB `gaz:"inject"`
}

type ServiceB struct {
	A *ServiceA `gaz:"inject"`
}

func (s *InjectionSuite) TestCycleViaInjection() {
	c := New()

	s.Require().NoError(For[*ServiceA](c).ProviderFunc(func(_ *Container) *ServiceA {
		return &ServiceA{}
	}))
	s.Require().NoError(For[*ServiceB](c).ProviderFunc(func(_ *Container) *ServiceB {
		return &ServiceB{}
	}))

	_, err := Resolve[*ServiceA](c)
	s.Error(err, "expected error for circular dependency")
	s.ErrorIs(err, ErrCycle)
}

func (s *InjectionSuite) TestNonStructPointerSkipped() {
	c := New()

	// Register a simple string - not a struct
	s.Require().NoError(For[string](c).ProviderFunc(func(_ *Container) string {
		return "hello"
	}))

	// Should resolve without error - injection silently skipped for non-struct
	str, err := Resolve[string](c)
	s.Require().NoError(err)
	s.Equal("hello", str)
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
		{
			"inject, name=foo , optional",
			tagOptions{inject: true, name: "foo", optional: true},
		}, // with spaces
		{
			"optional,inject",
			tagOptions{inject: true, optional: true},
		}, // order doesn't matter
		{
			"name=foo",
			tagOptions{name: "foo", inject: false},
		}, // missing inject keyword
		{"", tagOptions{}}, // empty tag
	}

	for _, tt := range tests {
		s.Run(tt.tag, func() {
			opts := parseTag(tt.tag)
			s.Equal(tt.expected.inject, opts.inject, "inject")
			s.Equal(tt.expected.name, opts.name, "name")
			s.Equal(tt.expected.optional, opts.optional, "optional")
		})
	}
}

// Test that transient services also get injection.
type TransientHandler struct {
	DB *Database `gaz:"inject"`
}

func (s *InjectionSuite) TestTransientService() {
	c := New()

	db := &Database{connStr: "test"}
	s.Require().NoError(For[*Database](c).Instance(db))
	s.Require().NoError(
		For[*TransientHandler](c).Transient().ProviderFunc(func(_ *Container) *TransientHandler {
			return &TransientHandler{}
		}),
	)

	h1, err := Resolve[*TransientHandler](c)
	s.Require().NoError(err)
	h2, err := Resolve[*TransientHandler](c)
	s.Require().NoError(err)

	// Different handler instances (transient)
	s.NotSame(h1, h2, "transient services should return different instances")
	// Both should have DB injected
	s.NotNil(h1.DB, "first handler should have DB injected")
	s.NotNil(h2.DB, "second handler should have DB injected")
	// Same DB instance (singleton)
	s.Same(h1.DB, h2.DB, "both handlers should have the same DB instance")
}

// Test injection doesn't happen on pre-built instances.
type PreBuiltHandler struct {
	DB *Database `gaz:"inject"`
}

func (s *InjectionSuite) TestInstanceServiceNoInjection() {
	c := New()

	db := &Database{connStr: "test"}
	s.Require().NoError(For[*Database](c).Instance(db))

	// Pre-built instance with empty DB field
	handler := &PreBuiltHandler{}
	s.Require().NoError(For[*PreBuiltHandler](c).Instance(handler))

	h, err := Resolve[*PreBuiltHandler](c)
	s.Require().NoError(err)

	// Instance services don't get injection - DB should still be nil
	s.Nil(h.DB, "pre-built instances should not get injection")
}
