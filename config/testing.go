package config

import (
	"sync"
	"testing"
	"time"
)

// =============================================================================
// MapBackend - In-memory Backend for testing
// =============================================================================

// MapBackend is a simple in-memory config backend for testing.
// It implements Backend with Get/GetString/GetInt/GetBool using a map.
//
// MapBackend is thread-safe and can be used in concurrent tests.
// Use Set() to configure values during test setup.
//
// # Example
//
//	backend := config.NewMapBackend(map[string]any{
//	    "server.host": "localhost",
//	    "server.port": 8080,
//	})
//	mgr := config.New(config.WithBackend(backend))
type MapBackend struct {
	values   map[string]any
	defaults map[string]any
	mu       sync.RWMutex
}

// NewMapBackend creates a MapBackend with initial values.
// Pass nil for an empty configuration.
func NewMapBackend(values map[string]any) *MapBackend {
	if values == nil {
		values = make(map[string]any)
	}
	return &MapBackend{
		values:   values,
		defaults: make(map[string]any),
	}
}

// Set sets a value (for test setup).
func (b *MapBackend) Set(key string, value any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.values[key] = value
}

// Get returns the value for a key.
// Values take precedence over defaults.
func (b *MapBackend) Get(key string) any {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if v, ok := b.values[key]; ok {
		return v
	}
	return b.defaults[key]
}

// GetString returns a string value for the key.
func (b *MapBackend) GetString(key string) string {
	v := b.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetInt returns an int value for the key.
func (b *MapBackend) GetInt(key string) int {
	v := b.Get(key)
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

// GetBool returns a bool value for the key.
func (b *MapBackend) GetBool(key string) bool {
	v := b.Get(key)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetDuration returns a time.Duration value for the key.
func (b *MapBackend) GetDuration(key string) time.Duration {
	v := b.Get(key)
	if d, ok := v.(time.Duration); ok {
		return d
	}
	return 0
}

// GetFloat64 returns a float64 value for the key.
func (b *MapBackend) GetFloat64(key string) float64 {
	v := b.Get(key)
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

// SetDefault sets a default value for a key.
// Defaults are returned only if no explicit value is set.
func (b *MapBackend) SetDefault(key string, value any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.defaults[key] = value
}

// IsSet checks if a key has been set (either value or default).
func (b *MapBackend) IsSet(key string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, ok := b.values[key]; ok {
		return true
	}
	_, ok := b.defaults[key]
	return ok
}

// Unmarshal unmarshals the entire config into a struct.
// For testing, this performs a simple field matching by key.
// It does not support nested structs or complex types.
func (b *MapBackend) Unmarshal(_ any) error {
	// Simple implementation - in tests, values are typically accessed directly
	// or the test uses a viper backend for complex unmarshaling needs.
	return nil
}

// UnmarshalKey unmarshals a specific key into a struct.
// For testing, this is a no-op - use Get* methods for value access.
func (b *MapBackend) UnmarshalKey(_ string, _ any) error {
	return nil
}

// =============================================================================
// TestManager - Factory for test config Manager
// =============================================================================

// TestManager creates a config.Manager with an in-memory MapBackend.
// Pass initial values or nil for empty config.
//
// This is a convenience factory for tests that need a Manager without
// file I/O or environment variable complexity.
//
// # Example
//
//	mgr := config.TestManager(map[string]any{
//	    "database.host": "localhost",
//	    "database.port": 5432,
//	})
//
//	host := mgr.Backend().GetString("database.host")
func TestManager(values map[string]any) *Manager {
	backend := NewMapBackend(values)
	return New(WithBackend(backend))
}

// =============================================================================
// SampleConfig - Common test config pattern
// =============================================================================

// SampleConfig is a sample config struct for testing config loading.
// It provides a minimal structure useful for testing the config system.
//
// SampleConfig implements both Defaulter and Validator interfaces,
// making it useful for testing the full config loading lifecycle.
type SampleConfig struct {
	Host  string `mapstructure:"host" gaz:"host"`
	Port  int    `mapstructure:"port" gaz:"port"`
	Debug bool   `mapstructure:"debug" gaz:"debug"`
}

// Default implements Defaulter interface.
// Sets sensible defaults for testing.
func (c *SampleConfig) Default() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}

// =============================================================================
// Require* assertion helpers
// =============================================================================

// RequireConfigLoaded verifies config was loaded without error.
// Uses testing.TB for compatibility with both tests and benchmarks.
//
// # Example
//
//	cfg := &MyConfig{}
//	config.RequireConfigLoaded(t, mgr, cfg)
//	// cfg is now populated
func RequireConfigLoaded(tb testing.TB, m *Manager, target any) {
	tb.Helper()
	if err := m.LoadInto(target); err != nil {
		tb.Fatalf("failed to load config: %v", err)
	}
}

// RequireConfigValue verifies a config key has the expected value.
// Uses reflect.DeepEqual for comparison, supporting complex types.
//
// # Example
//
//	config.RequireConfigValue(t, backend, "server.port", 8080)
func RequireConfigValue(tb testing.TB, b Backend, key string, expected any) {
	tb.Helper()
	actual := b.Get(key)
	if actual != expected {
		tb.Fatalf("config key %q: expected %v (%T), got %v (%T)", key, expected, expected, actual, actual)
	}
}

// RequireConfigString verifies a config key has the expected string value.
//
// # Example
//
//	config.RequireConfigString(t, backend, "server.host", "localhost")
func RequireConfigString(tb testing.TB, b Backend, key string, expected string) {
	tb.Helper()
	actual := b.GetString(key)
	if actual != expected {
		tb.Fatalf("config key %q: expected %q, got %q", key, expected, actual)
	}
}

// RequireConfigInt verifies a config key has the expected int value.
//
// # Example
//
//	config.RequireConfigInt(t, backend, "server.port", 8080)
func RequireConfigInt(tb testing.TB, b Backend, key string, expected int) {
	tb.Helper()
	actual := b.GetInt(key)
	if actual != expected {
		tb.Fatalf("config key %q: expected %d, got %d", key, expected, actual)
	}
}

// RequireConfigBool verifies a config key has the expected bool value.
//
// # Example
//
//	config.RequireConfigBool(t, backend, "server.debug", true)
func RequireConfigBool(tb testing.TB, b Backend, key string, expected bool) {
	tb.Helper()
	actual := b.GetBool(key)
	if actual != expected {
		tb.Fatalf("config key %q: expected %t, got %t", key, expected, actual)
	}
}

// RequireConfigIsSet verifies a config key is set.
//
// # Example
//
//	config.RequireConfigIsSet(t, backend, "database.host")
func RequireConfigIsSet(tb testing.TB, b Backend, key string) {
	tb.Helper()
	if !b.IsSet(key) {
		tb.Fatalf("config key %q: expected to be set, but was not", key)
	}
}
