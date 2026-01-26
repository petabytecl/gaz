package gaz

import (
	"errors"
	"testing"
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

func TestInject_BasicInjection(t *testing.T) {
	c := New()

	db := &Database{connStr: "postgres://localhost"}
	logger := &Logger{level: "debug"}

	if err := For[*Database](c).Instance(db); err != nil {
		t.Fatal(err)
	}
	if err := For[*Logger](c).Instance(logger); err != nil {
		t.Fatal(err)
	}
	if err := For[*Handler](c).ProviderFunc(func(c *Container) *Handler {
		return &Handler{} // Fields auto-injected after provider returns
	}); err != nil {
		t.Fatal(err)
	}

	h, err := Resolve[*Handler](c)
	if err != nil {
		t.Fatal(err)
	}

	if h.DB == nil {
		t.Error("DB field should be injected")
	}
	if h.Logger == nil {
		t.Error("Logger field should be injected")
	}
	if h.DB != db {
		t.Error("DB should be the same instance")
	}
	if h.Logger != logger {
		t.Error("Logger should be the same instance")
	}
}

// DB type for named injection tests
type DB struct {
	name string
}

type ServiceWithNamedDeps struct {
	Primary *DB `gaz:"inject,name=primary"`
	Replica *DB `gaz:"inject,name=replica"`
}

func TestInject_Named(t *testing.T) {
	c := New()

	primary := &DB{name: "primary"}
	replica := &DB{name: "replica"}

	if err := For[*DB](c).Named("primary").Instance(primary); err != nil {
		t.Fatal(err)
	}
	if err := For[*DB](c).Named("replica").Instance(replica); err != nil {
		t.Fatal(err)
	}
	if err := For[*ServiceWithNamedDeps](c).ProviderFunc(func(c *Container) *ServiceWithNamedDeps {
		return &ServiceWithNamedDeps{}
	}); err != nil {
		t.Fatal(err)
	}

	svc, err := Resolve[*ServiceWithNamedDeps](c)
	if err != nil {
		t.Fatal(err)
	}

	if svc.Primary == nil || svc.Primary.name != "primary" {
		t.Errorf("Primary should be primary DB, got %+v", svc.Primary)
	}
	if svc.Replica == nil || svc.Replica.name != "replica" {
		t.Errorf("Replica should be replica DB, got %+v", svc.Replica)
	}
}

type HandlerWithOptionalCache struct {
	Cache *Cache `gaz:"inject,optional"`
}

func TestInject_Optional_NotRegistered(t *testing.T) {
	c := New()

	// Don't register Cache
	if err := For[*HandlerWithOptionalCache](c).ProviderFunc(func(c *Container) *HandlerWithOptionalCache {
		return &HandlerWithOptionalCache{}
	}); err != nil {
		t.Fatal(err)
	}

	h, err := Resolve[*HandlerWithOptionalCache](c)
	if err != nil {
		t.Fatal(err)
	}

	if h.Cache != nil {
		t.Errorf("Cache should be nil when not registered, got %+v", h.Cache)
	}
}

func TestInject_Optional_Registered(t *testing.T) {
	c := New()

	cache := &Cache{addr: "localhost:6379"}
	if err := For[*Cache](c).Instance(cache); err != nil {
		t.Fatal(err)
	}
	if err := For[*HandlerWithOptionalCache](c).ProviderFunc(func(c *Container) *HandlerWithOptionalCache {
		return &HandlerWithOptionalCache{}
	}); err != nil {
		t.Fatal(err)
	}

	h, err := Resolve[*HandlerWithOptionalCache](c)
	if err != nil {
		t.Fatal(err)
	}

	if h.Cache == nil {
		t.Error("Cache should be populated when registered")
	}
	if h.Cache != cache {
		t.Error("Cache should be the same instance")
	}
}

type BadHandler struct {
	db *Database `gaz:"inject"` // unexported field!
}

func TestInject_UnexportedField_ReturnsError(t *testing.T) {
	c := New()

	if err := For[*Database](c).Instance(&Database{}); err != nil {
		t.Fatal(err)
	}
	if err := For[*BadHandler](c).ProviderFunc(func(c *Container) *BadHandler {
		return &BadHandler{}
	}); err != nil {
		t.Fatal(err)
	}

	_, err := Resolve[*BadHandler](c)
	if err == nil {
		t.Error("expected error for unexported field")
	}
	if !errors.Is(err, ErrNotSettable) {
		t.Errorf("expected ErrNotSettable, got %v", err)
	}
}

type HandlerWithMissingDep struct {
	DB *Database `gaz:"inject"` // not registered, not optional
}

func TestInject_MissingDependency_ReturnsError(t *testing.T) {
	c := New()

	// Don't register Database
	if err := For[*HandlerWithMissingDep](c).ProviderFunc(func(c *Container) *HandlerWithMissingDep {
		return &HandlerWithMissingDep{}
	}); err != nil {
		t.Fatal(err)
	}

	_, err := Resolve[*HandlerWithMissingDep](c)
	if err == nil {
		t.Error("expected error for missing dependency")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Types for cycle detection test
type ServiceA struct {
	B *ServiceB `gaz:"inject"`
}

type ServiceB struct {
	A *ServiceA `gaz:"inject"`
}

func TestInject_CycleViaInjection(t *testing.T) {
	c := New()

	if err := For[*ServiceA](c).ProviderFunc(func(c *Container) *ServiceA {
		return &ServiceA{}
	}); err != nil {
		t.Fatal(err)
	}
	if err := For[*ServiceB](c).ProviderFunc(func(c *Container) *ServiceB {
		return &ServiceB{}
	}); err != nil {
		t.Fatal(err)
	}

	_, err := Resolve[*ServiceA](c)
	if err == nil {
		t.Error("expected error for circular dependency")
	}
	if !errors.Is(err, ErrCycle) {
		t.Errorf("expected ErrCycle, got %v", err)
	}
}

func TestInject_NonStructPointer_Skipped(t *testing.T) {
	c := New()

	// Register a simple string - not a struct
	if err := For[string](c).ProviderFunc(func(c *Container) string {
		return "hello"
	}); err != nil {
		t.Fatal(err)
	}

	// Should resolve without error - injection silently skipped for non-struct
	s, err := Resolve[string](c)
	if err != nil {
		t.Fatal(err)
	}
	if s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}
}

func TestParseTag(t *testing.T) {
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
		t.Run(tt.tag, func(t *testing.T) {
			opts := parseTag(tt.tag)
			if opts.inject != tt.expected.inject {
				t.Errorf("inject: got %v, want %v", opts.inject, tt.expected.inject)
			}
			if opts.name != tt.expected.name {
				t.Errorf("name: got %q, want %q", opts.name, tt.expected.name)
			}
			if opts.optional != tt.expected.optional {
				t.Errorf("optional: got %v, want %v", opts.optional, tt.expected.optional)
			}
		})
	}
}

// Test that transient services also get injection
type TransientHandler struct {
	DB *Database `gaz:"inject"`
}

func TestInject_TransientService(t *testing.T) {
	c := New()

	db := &Database{connStr: "test"}
	if err := For[*Database](c).Instance(db); err != nil {
		t.Fatal(err)
	}
	if err := For[*TransientHandler](c).Transient().ProviderFunc(func(c *Container) *TransientHandler {
		return &TransientHandler{}
	}); err != nil {
		t.Fatal(err)
	}

	h1, err := Resolve[*TransientHandler](c)
	if err != nil {
		t.Fatal(err)
	}
	h2, err := Resolve[*TransientHandler](c)
	if err != nil {
		t.Fatal(err)
	}

	// Different handler instances (transient)
	if h1 == h2 {
		t.Error("transient services should return different instances")
	}
	// Both should have DB injected
	if h1.DB == nil || h2.DB == nil {
		t.Error("both handlers should have DB injected")
	}
	// Same DB instance (singleton)
	if h1.DB != h2.DB {
		t.Error("both handlers should have the same DB instance")
	}
}

// Test injection doesn't happen on pre-built instances
type PreBuiltHandler struct {
	DB *Database `gaz:"inject"`
}

func TestInject_InstanceService_NoInjection(t *testing.T) {
	c := New()

	db := &Database{connStr: "test"}
	if err := For[*Database](c).Instance(db); err != nil {
		t.Fatal(err)
	}

	// Pre-built instance with empty DB field
	handler := &PreBuiltHandler{}
	if err := For[*PreBuiltHandler](c).Instance(handler); err != nil {
		t.Fatal(err)
	}

	h, err := Resolve[*PreBuiltHandler](c)
	if err != nil {
		t.Fatal(err)
	}

	// Instance services don't get injection - DB should still be nil
	if h.DB != nil {
		t.Error("pre-built instances should not get injection")
	}
}
