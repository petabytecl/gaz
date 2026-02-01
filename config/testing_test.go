package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/config"
)

// =============================================================================
// MapBackend tests
// =============================================================================

func TestMapBackend_NewWithNil(t *testing.T) {
	backend := config.NewMapBackend(nil)
	require.NotNil(t, backend)

	// Should return zero values for missing keys
	assert.Equal(t, "", backend.GetString("missing"))
	assert.Equal(t, 0, backend.GetInt("missing"))
	assert.Equal(t, false, backend.GetBool("missing"))
}

func TestMapBackend_NewWithValues(t *testing.T) {
	backend := config.NewMapBackend(map[string]any{
		"host":    "localhost",
		"port":    8080,
		"debug":   true,
		"timeout": 30 * time.Second,
		"rate":    3.14,
	})

	assert.Equal(t, "localhost", backend.GetString("host"))
	assert.Equal(t, 8080, backend.GetInt("port"))
	assert.Equal(t, true, backend.GetBool("debug"))
	assert.Equal(t, 30*time.Second, backend.GetDuration("timeout"))
	assert.Equal(t, 3.14, backend.GetFloat64("rate"))
}

func TestMapBackend_Set(t *testing.T) {
	backend := config.NewMapBackend(nil)

	backend.Set("host", "example.com")
	backend.Set("port", 9000)

	assert.Equal(t, "example.com", backend.GetString("host"))
	assert.Equal(t, 9000, backend.GetInt("port"))
}

func TestMapBackend_SetDefault(t *testing.T) {
	backend := config.NewMapBackend(nil)

	backend.SetDefault("host", "default-host")
	assert.Equal(t, "default-host", backend.GetString("host"))

	// Explicit value should override default
	backend.Set("host", "explicit-host")
	assert.Equal(t, "explicit-host", backend.GetString("host"))
}

func TestMapBackend_IsSet(t *testing.T) {
	backend := config.NewMapBackend(nil)

	assert.False(t, backend.IsSet("missing"))

	backend.Set("explicit", "value")
	assert.True(t, backend.IsSet("explicit"))

	backend.SetDefault("default", "value")
	assert.True(t, backend.IsSet("default"))
}

func TestMapBackend_Get(t *testing.T) {
	backend := config.NewMapBackend(map[string]any{
		"key": "value",
	})

	// Get returns any type
	assert.Equal(t, "value", backend.Get("key"))
	assert.Nil(t, backend.Get("missing"))
}

func TestMapBackend_TypeMismatch(t *testing.T) {
	backend := config.NewMapBackend(map[string]any{
		"string-key": "not an int",
	})

	// Type mismatch should return zero value
	assert.Equal(t, 0, backend.GetInt("string-key"))
	assert.Equal(t, false, backend.GetBool("string-key"))
}

func TestMapBackend_ThreadSafety(t *testing.T) {
	backend := config.NewMapBackend(nil)

	// Concurrent writes
	done := make(chan struct{})
	go func() {
		for i := range 100 {
			backend.Set("key", i)
		}
		close(done)
	}()

	// Concurrent reads
	for range 100 {
		_ = backend.GetInt("key")
	}

	<-done
}

// =============================================================================
// TestManager tests
// =============================================================================

func TestTestManager_EmptyConfig(t *testing.T) {
	mgr := config.TestManager(nil)
	require.NotNil(t, mgr)

	backend := mgr.Backend()
	assert.Equal(t, "", backend.GetString("missing"))
}

func TestTestManager_WithValues(t *testing.T) {
	mgr := config.TestManager(map[string]any{
		"server.host": "localhost",
		"server.port": 3000,
	})

	backend := mgr.Backend()
	assert.Equal(t, "localhost", backend.GetString("server.host"))
	assert.Equal(t, 3000, backend.GetInt("server.port"))
}

// =============================================================================
// SampleConfig tests
// =============================================================================

func TestSampleConfig_Default(t *testing.T) {
	cfg := &config.SampleConfig{}
	cfg.Default()

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
	assert.False(t, cfg.Debug)
}

func TestSampleConfig_DefaultPreservesValues(t *testing.T) {
	cfg := &config.SampleConfig{
		Host:  "custom",
		Port:  9000,
		Debug: true,
	}
	cfg.Default()

	// Default should not override existing values
	assert.Equal(t, "custom", cfg.Host)
	assert.Equal(t, 9000, cfg.Port)
	assert.True(t, cfg.Debug)
}

// =============================================================================
// Require* helper tests
// =============================================================================

func TestRequireConfigValue(t *testing.T) {
	backend := config.NewMapBackend(map[string]any{
		"key": "value",
	})

	// Should not panic for matching values
	config.RequireConfigValue(t, backend, "key", "value")
}

func TestRequireConfigString(t *testing.T) {
	backend := config.NewMapBackend(map[string]any{
		"host": "localhost",
	})

	config.RequireConfigString(t, backend, "host", "localhost")
}

func TestRequireConfigInt(t *testing.T) {
	backend := config.NewMapBackend(map[string]any{
		"port": 8080,
	})

	config.RequireConfigInt(t, backend, "port", 8080)
}

func TestRequireConfigBool(t *testing.T) {
	backend := config.NewMapBackend(map[string]any{
		"debug": true,
	})

	config.RequireConfigBool(t, backend, "debug", true)
}

func TestRequireConfigIsSet(t *testing.T) {
	backend := config.NewMapBackend(map[string]any{
		"key": "value",
	})

	config.RequireConfigIsSet(t, backend, "key")
}
