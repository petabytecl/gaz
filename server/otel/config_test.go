package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "", cfg.Endpoint, "endpoint should be empty by default (disabled)")
	assert.Equal(t, "gaz", cfg.ServiceName, "service name should default to 'gaz'")
	assert.InDelta(t, 0.1, cfg.SampleRatio, 0.001, "sample ratio should be 0.1 (10%)")
	assert.True(t, cfg.Insecure, "insecure should default to true for development")
}

func TestDefaultSampleRatio(t *testing.T) {
	assert.Equal(t, 0.1, DefaultSampleRatio)
}

func TestConfig_ZeroValue(t *testing.T) {
	var cfg Config

	// Zero value should have empty/zero fields
	assert.Empty(t, cfg.Endpoint)
	assert.Empty(t, cfg.ServiceName)
	assert.Zero(t, cfg.SampleRatio)
	assert.False(t, cfg.Insecure)
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		Endpoint:    "localhost:4317",
		ServiceName: "my-service",
		SampleRatio: 0.5,
		Insecure:    true,
	}

	assert.Equal(t, "localhost:4317", cfg.Endpoint)
	assert.Equal(t, "my-service", cfg.ServiceName)
	assert.Equal(t, 0.5, cfg.SampleRatio)
	assert.True(t, cfg.Insecure)
}

func TestConfig_SetDefaults(t *testing.T) {
	t.Run("applies defaults to zero values", func(t *testing.T) {
		var cfg Config
		cfg.SetDefaults()

		assert.Equal(t, "gaz", cfg.ServiceName)
		assert.InDelta(t, DefaultSampleRatio, cfg.SampleRatio, 0.001)
		// Endpoint stays empty (disabled by default).
		assert.Empty(t, cfg.Endpoint)
		// Insecure stays false (Go zero value).
		assert.False(t, cfg.Insecure)
	})

	t.Run("preserves existing values", func(t *testing.T) {
		cfg := Config{
			Endpoint:    "collector:4317",
			ServiceName: "custom-service",
			SampleRatio: 0.5,
			Insecure:    true,
		}
		cfg.SetDefaults()

		// Existing values should be preserved.
		assert.Equal(t, "collector:4317", cfg.Endpoint)
		assert.Equal(t, "custom-service", cfg.ServiceName)
		assert.InDelta(t, 0.5, cfg.SampleRatio, 0.001)
		assert.True(t, cfg.Insecure)
	})

	t.Run("handles negative sample ratio", func(t *testing.T) {
		cfg := Config{SampleRatio: -1}
		cfg.SetDefaults()

		assert.InDelta(t, DefaultSampleRatio, cfg.SampleRatio, 0.001)
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		cfg := DefaultConfig()
		assert.NoError(t, cfg.Validate())
	})

	t.Run("valid config with endpoint passes", func(t *testing.T) {
		cfg := Config{
			Endpoint:    "localhost:4317",
			ServiceName: "test-service",
			SampleRatio: 0.5,
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("sample ratio below 0 fails", func(t *testing.T) {
		cfg := Config{SampleRatio: -0.1}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sample_ratio")
		assert.Contains(t, err.Error(), "must be between 0.0 and 1.0")
	})

	t.Run("sample ratio above 1 fails", func(t *testing.T) {
		cfg := Config{SampleRatio: 1.5}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sample_ratio")
	})

	t.Run("sample ratio at boundaries passes", func(t *testing.T) {
		// 0.0 is valid (never sample).
		cfg := Config{SampleRatio: 0.0}
		assert.NoError(t, cfg.Validate())

		// 1.0 is valid (always sample).
		cfg = Config{SampleRatio: 1.0}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("endpoint without service name fails", func(t *testing.T) {
		cfg := Config{
			Endpoint:    "localhost:4317",
			ServiceName: "", // Empty.
			SampleRatio: 0.1,
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service_name required")
	})

	t.Run("disabled endpoint with empty service name passes", func(t *testing.T) {
		cfg := Config{
			Endpoint:    "", // Disabled.
			ServiceName: "",
			SampleRatio: 0.1,
		}
		assert.NoError(t, cfg.Validate())
	})
}
