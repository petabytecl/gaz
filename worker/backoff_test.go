package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBackoffConfig_ReturnsCorrectDefaults(t *testing.T) {
	cfg := NewBackoffConfig()

	assert.Equal(t, 1*time.Second, cfg.Min)
	assert.Equal(t, 5*time.Minute, cfg.Max)
	assert.Equal(t, 2.0, cfg.Factor)
	assert.True(t, cfg.Jitter)
}

func TestNewBackoff_CreatesValidInstance(t *testing.T) {
	cfg := NewBackoffConfig()
	b := cfg.NewBackoff()

	assert.NotNil(t, b)
	assert.Equal(t, 1*time.Second, b.Min)
	assert.Equal(t, 5*time.Minute, b.Max)
	assert.Equal(t, 2.0, b.Factor)
	assert.True(t, b.Jitter)
}

func TestBackoffDuration_IncreasesExponentially(t *testing.T) {
	cfg := NewBackoffConfig()
	// Disable jitter for predictable testing
	cfg.Jitter = false
	b := cfg.NewBackoff()

	// First call should return Min (1s)
	d1 := b.Duration()
	assert.Equal(t, 1*time.Second, d1)

	// Second call should return Min * Factor (2s)
	d2 := b.Duration()
	assert.Equal(t, 2*time.Second, d2)

	// Third call should return 4s
	d3 := b.Duration()
	assert.Equal(t, 4*time.Second, d3)

	// Fourth call should return 8s
	d4 := b.Duration()
	assert.Equal(t, 8*time.Second, d4)
}

func TestBackoffReset_ResetsToMinimum(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Jitter = false
	b := cfg.NewBackoff()

	// Advance the backoff a few times
	_ = b.Duration() // 1s
	_ = b.Duration() // 2s
	_ = b.Duration() // 4s

	// Reset
	b.Reset()

	// Next duration should be back to Min
	d := b.Duration()
	assert.Equal(t, 1*time.Second, d)
}

func TestBackoffDuration_CappedAtMax(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Jitter = false
	cfg.Max = 10 * time.Second // Set low max for testing
	b := cfg.NewBackoff()

	// Keep calling until we hit the cap
	for range 10 {
		d := b.Duration()
		assert.LessOrEqual(t, d, 10*time.Second, "duration should never exceed Max")
	}
}

func TestWithBackoffMin_SetsMin(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Apply(WithBackoffMin(2 * time.Second))

	assert.Equal(t, 2*time.Second, cfg.Min)
}

func TestWithBackoffMin_IgnoresZero(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Apply(WithBackoffMin(0))

	assert.Equal(t, 1*time.Second, cfg.Min) // Default unchanged
}

func TestWithBackoffMax_SetsMax(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Apply(WithBackoffMax(10 * time.Minute))

	assert.Equal(t, 10*time.Minute, cfg.Max)
}

func TestWithBackoffMax_IgnoresZero(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Apply(WithBackoffMax(0))

	assert.Equal(t, 5*time.Minute, cfg.Max) // Default unchanged
}

func TestWithBackoffFactor_SetsFactor(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Apply(WithBackoffFactor(3))

	assert.Equal(t, 3.0, cfg.Factor)
}

func TestWithBackoffFactor_IgnoresZero(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Apply(WithBackoffFactor(0))

	assert.Equal(t, 2.0, cfg.Factor) // Default unchanged
}

func TestWithBackoffJitter_DisablesJitter(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Apply(WithBackoffJitter(false))

	assert.False(t, cfg.Jitter)
}

func TestWithBackoffJitter_EnablesJitter(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Jitter = false
	cfg.Apply(WithBackoffJitter(true))

	assert.True(t, cfg.Jitter)
}

func TestBackoffApply_ChainsMultipleOptions(t *testing.T) {
	cfg := NewBackoffConfig()
	cfg.Apply(
		WithBackoffMin(500*time.Millisecond),
		WithBackoffMax(1*time.Minute),
		WithBackoffFactor(1.5),
		WithBackoffJitter(false),
	)

	assert.Equal(t, 500*time.Millisecond, cfg.Min)
	assert.Equal(t, 1*time.Minute, cfg.Max)
	assert.Equal(t, 1.5, cfg.Factor)
	assert.False(t, cfg.Jitter)
}
