package di

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// WithHookTimeout tests
// =============================================================================

func TestWithHookTimeout(t *testing.T) {
	cfg := &HookConfig{}

	// Apply default - should be zero
	assert.Equal(t, time.Duration(0), cfg.Timeout)

	// Apply WithHookTimeout option
	opt := WithHookTimeout(5 * time.Second)
	opt(cfg)

	assert.Equal(t, 5*time.Second, cfg.Timeout)
}

func TestWithHookTimeout_MultipleApply(t *testing.T) {
	cfg := &HookConfig{}

	// Apply multiple options - last one wins
	opt1 := WithHookTimeout(5 * time.Second)
	opt2 := WithHookTimeout(30 * time.Second)

	opt1(cfg)
	assert.Equal(t, 5*time.Second, cfg.Timeout)

	opt2(cfg)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
}

func TestWithHookTimeout_ZeroDuration(t *testing.T) {
	cfg := &HookConfig{Timeout: 10 * time.Second}

	// Apply zero duration - should set to zero
	opt := WithHookTimeout(0)
	opt(cfg)

	assert.Equal(t, time.Duration(0), cfg.Timeout)
}
