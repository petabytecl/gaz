package worker

import (
	"time"

	"github.com/petabytecl/gaz/backoff"
)

// BackoffConfig holds configuration for exponential backoff during worker restarts.
//
// This configuration provides sensible defaults for worker supervision.
// The backoff algorithm doubles the delay after each failure, with optional
// jitter to prevent thundering herd problems.
type BackoffConfig struct {
	// Min is the minimum delay before the first retry.
	// Default: 1 second
	Min time.Duration

	// Max is the maximum delay cap. The backoff will not exceed this value.
	// Default: 5 minutes
	Max time.Duration

	// Factor is the multiplier for each successive backoff.
	// Default: 2 (delays double: 1s, 2s, 4s, 8s, ...)
	Factor float64

	// Jitter adds randomization to delays to prevent thundering herd.
	// When true, delays are randomized within a range (±50%).
	// Default: true
	Jitter bool
}

// BackoffOption configures BackoffConfig.
type BackoffOption func(*BackoffConfig)

// NewBackoffConfig returns a BackoffConfig with sensible defaults.
//
// Default values (per RESEARCH.md recommendations):
//   - Min: 1 second
//   - Max: 5 minutes
//   - Factor: 2
//   - Jitter: true
func NewBackoffConfig() *BackoffConfig {
	return &BackoffConfig{
		Min:    1 * time.Second,
		Max:    5 * time.Minute,
		Factor: 2,
		Jitter: true,
	}
}

// Apply applies the given options to the BackoffConfig.
func (c *BackoffConfig) Apply(opts ...BackoffOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// NewBackoff creates an ExponentialBackOff instance from this config.
//
// The returned BackOff can be used to calculate successive delays:
//
//	b := cfg.NewBackoff()
//	delay := b.NextBackOff() // Get next delay
//	// ... wait ...
//	b.Reset() // Reset after stable run period
func (c *BackoffConfig) NewBackoff() *backoff.ExponentialBackOff {
	// Jitter: true = 0.5 (default), false = 0 (no randomization)
	randomizationFactor := 0.5
	if !c.Jitter {
		randomizationFactor = 0
	}

	return backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(c.Min),
		backoff.WithMaxInterval(c.Max),
		backoff.WithMultiplier(c.Factor),
		backoff.WithRandomizationFactor(randomizationFactor),
	)
}

// WithBackoffMin sets the minimum delay before the first retry.
func WithBackoffMin(d time.Duration) BackoffOption {
	return func(c *BackoffConfig) {
		if d > 0 {
			c.Min = d
		}
	}
}

// WithBackoffMax sets the maximum delay cap.
func WithBackoffMax(d time.Duration) BackoffOption {
	return func(c *BackoffConfig) {
		if d > 0 {
			c.Max = d
		}
	}
}

// WithBackoffFactor sets the multiplier for successive backoffs.
func WithBackoffFactor(f float64) BackoffOption {
	return func(c *BackoffConfig) {
		if f > 0 {
			c.Factor = f
		}
	}
}

// WithBackoffJitter enables or disables jitter.
// Jitter randomizes delays to prevent thundering herd.
func WithBackoffJitter(enabled bool) BackoffOption {
	return func(c *BackoffConfig) {
		c.Jitter = enabled
	}
}
