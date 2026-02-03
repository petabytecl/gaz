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
