package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
)

// =============================================================================
// Test Get[T]
// =============================================================================

func TestGet_String_ReturnsValue(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("host", "localhost")
	mgr := config.NewWithBackend(backend)

	result := config.Get[string](mgr, "host")
	assert.Equal(t, "localhost", result)
}

func TestGet_Int_ReturnsValue(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("port", 8080)
	mgr := config.NewWithBackend(backend)

	result := config.Get[int](mgr, "port")
	assert.Equal(t, 8080, result)
}

func TestGet_Bool_ReturnsValue(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("debug", true)
	mgr := config.NewWithBackend(backend)

	result := config.Get[bool](mgr, "debug")
	assert.True(t, result)
}

func TestGet_Float64_ReturnsValue(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("rate", 1.5)
	mgr := config.NewWithBackend(backend)

	result := config.Get[float64](mgr, "rate")
	assert.Equal(t, 1.5, result)
}

func TestGet_MissingKey_ReturnsZeroValue(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend)

	strResult := config.Get[string](mgr, "missing")
	assert.Equal(t, "", strResult)

	intResult := config.Get[int](mgr, "missing")
	assert.Equal(t, 0, intResult)

	boolResult := config.Get[bool](mgr, "missing")
	assert.False(t, boolResult)
}

func TestGet_TypeMismatch_ReturnsZeroValue(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("port", "not-a-number") // String instead of int
	mgr := config.NewWithBackend(backend)

	result := config.Get[int](mgr, "port")
	assert.Equal(t, 0, result)
}

// =============================================================================
// Test GetOr[T]
// =============================================================================

func TestGetOr_MissingKey_ReturnsFallback(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend)

	result := config.GetOr(mgr, "missing", "default")
	assert.Equal(t, "default", result)

	intResult := config.GetOr(mgr, "missing", 8080)
	assert.Equal(t, 8080, intResult)
}

func TestGetOr_TypeMismatch_ReturnsFallback(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("port", "not-a-number")
	mgr := config.NewWithBackend(backend)

	result := config.GetOr(mgr, "port", 8080)
	assert.Equal(t, 8080, result)
}

func TestGetOr_ValuePresent_ReturnsValue(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("host", "prodhost")
	mgr := config.NewWithBackend(backend)

	result := config.GetOr(mgr, "host", "default")
	assert.Equal(t, "prodhost", result)
}

func TestGetOr_Duration_Works(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("timeout", 30*time.Second)
	mgr := config.NewWithBackend(backend)

	result := config.GetOr(mgr, "timeout", 10*time.Second)
	assert.Equal(t, 30*time.Second, result)

	// Missing key returns fallback
	result = config.GetOr(mgr, "other_timeout", 10*time.Second)
	assert.Equal(t, 10*time.Second, result)
}

// =============================================================================
// Test MustGet[T]
// =============================================================================

func TestMustGet_ValuePresent_ReturnsValue(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("host", "localhost")
	mgr := config.NewWithBackend(backend)

	result := config.MustGet[string](mgr, "host")
	assert.Equal(t, "localhost", result)
}

func TestMustGet_MissingKey_Panics(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend)

	assert.Panics(t, func() {
		config.MustGet[string](mgr, "missing")
	})
}

func TestMustGet_TypeMismatch_Panics(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("port", "not-a-number")
	mgr := config.NewWithBackend(backend)

	assert.Panics(t, func() {
		config.MustGet[int](mgr, "port")
	})
}

// =============================================================================
// Test with nested keys
// =============================================================================

func TestGet_NestedKey_Works(t *testing.T) {
	backend := cfgviper.New()
	backend.Set("database.host", "dbhost")
	backend.Set("database.port", 5432)
	mgr := config.NewWithBackend(backend)

	host := config.Get[string](mgr, "database.host")
	assert.Equal(t, "dbhost", host)

	port := config.Get[int](mgr, "database.port")
	assert.Equal(t, 5432, port)
}

func TestGetOr_NestedKey_Works(t *testing.T) {
	backend := cfgviper.New()
	mgr := config.NewWithBackend(backend)

	// Missing nested key returns fallback
	host := config.GetOr(mgr, "database.host", "localhost")
	assert.Equal(t, "localhost", host)
}
