package backoff

import (
	"math/rand/v2"
	"time"
)

// Default values for ExponentialBackOff.
// These defaults are tuned for worker/supervisor use cases.
const (
	DefaultInitialInterval     = 1 * time.Second // Worker-appropriate (not 100ms)
	DefaultRandomizationFactor = 0.5             // ±50% jitter
	DefaultMultiplier          = 2.0             // Double each retry (not 1.5)
	DefaultMaxInterval         = 5 * time.Minute // Upper bound on retry delay
	DefaultMaxElapsedTime      = 0               // Disabled - worker controls via circuit breaker
)

// Clock is an interface for getting the current time.
// This allows for testing without real time delays.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (t systemClock) Now() time.Time { return time.Now() }

// SystemClock is the default [Clock] implementation using [time.Now].
var SystemClock Clock = systemClock{}

// ExponentialBackOff implements a [BackOff] that increases the delay
// exponentially for each retry attempt with optional randomization (jitter).
//
// The formula for computing the next delay is:
//
//	randomized_interval = current_interval * (1 ± randomization_factor)
//
// After each call to [NextBackOff], the current_interval is multiplied by
// the [Multiplier] until it reaches [MaxInterval].
//
// If [MaxElapsedTime] > 0 and the total elapsed time exceeds it, [NextBackOff]
// returns [Stop] to signal that retries should cease.
//
// The jitter calculation uses [math/rand/v2] top-level functions which are
// thread-safe and auto-seeded.
type ExponentialBackOff struct {
	// InitialInterval is the first retry delay.
	InitialInterval time.Duration
	// MaxInterval is the upper bound on the retry delay.
	MaxInterval time.Duration
	// Multiplier is applied to the current interval after each retry.
	Multiplier float64
	// RandomizationFactor adds jitter to the delay.
	// 0 means no randomization, 0.5 means ±50% jitter.
	RandomizationFactor float64
	// MaxElapsedTime limits total retry duration. 0 means no limit.
	MaxElapsedTime time.Duration

	// Clock provides the current time. Defaults to SystemClock.
	Clock Clock
	// Stop is the sentinel value returned when retries should cease.
	// This is a copy of the package-level Stop constant.
	Stop time.Duration

	startTime       time.Time
	currentInterval time.Duration
}

// Option configures an [ExponentialBackOff].
type Option func(*ExponentialBackOff)

// NewExponentialBackOff creates a new [ExponentialBackOff] with the given options.
// Call [Reset] is called internally; the backoff is ready to use immediately.
func NewExponentialBackOff(opts ...Option) *ExponentialBackOff {
	b := &ExponentialBackOff{
		InitialInterval:     DefaultInitialInterval,
		MaxInterval:         DefaultMaxInterval,
		Multiplier:          DefaultMultiplier,
		RandomizationFactor: DefaultRandomizationFactor,
		MaxElapsedTime:      DefaultMaxElapsedTime,
		Stop:                Stop,
		Clock:               SystemClock,
	}
	for _, opt := range opts {
		opt(b)
	}
	b.Reset()
	return b
}

// WithInitialInterval sets the initial retry interval.
// Values <= 0 are ignored.
func WithInitialInterval(d time.Duration) Option {
	return func(b *ExponentialBackOff) {
		if d > 0 {
			b.InitialInterval = d
		}
	}
}

// WithMaxInterval sets the maximum retry interval.
// Values <= 0 are ignored.
func WithMaxInterval(d time.Duration) Option {
	return func(b *ExponentialBackOff) {
		if d > 0 {
			b.MaxInterval = d
		}
	}
}

// WithMultiplier sets the interval multiplier.
// Values <= 0 are ignored.
func WithMultiplier(m float64) Option {
	return func(b *ExponentialBackOff) {
		if m > 0 {
			b.Multiplier = m
		}
	}
}

// WithRandomizationFactor sets the jitter factor.
// Valid range is [0, 1]. Values outside this range are ignored.
// 0 means no randomization, 0.5 means ±50% jitter.
func WithRandomizationFactor(f float64) Option {
	return func(b *ExponentialBackOff) {
		if f >= 0 && f <= 1 {
			b.RandomizationFactor = f
		}
	}
}

// WithMaxElapsedTime sets the maximum total elapsed time.
// 0 means no limit (default).
// Negative values are ignored.
func WithMaxElapsedTime(d time.Duration) Option {
	return func(b *ExponentialBackOff) {
		if d >= 0 {
			b.MaxElapsedTime = d
		}
	}
}

// WithClock sets a custom [Clock] implementation.
// nil values are ignored.
func WithClock(c Clock) Option {
	return func(b *ExponentialBackOff) {
		if c != nil {
			b.Clock = c
		}
	}
}

// Reset restores the backoff to its initial state.
// Call this after a successful operation.
func (b *ExponentialBackOff) Reset() {
	b.currentInterval = b.InitialInterval
	b.startTime = b.Clock.Now()
}

// NextBackOff calculates the next retry delay with randomization.
// Returns [Stop] if [MaxElapsedTime] is exceeded.
func (b *ExponentialBackOff) NextBackOff() time.Duration {
	elapsed := b.GetElapsedTime()
	next := getRandomValueFromInterval(b.RandomizationFactor, rand.Float64(), b.currentInterval)
	b.incrementCurrentInterval()

	if b.MaxElapsedTime != 0 && elapsed+next > b.MaxElapsedTime {
		return b.Stop
	}
	return next
}

// GetElapsedTime returns the time since [Reset] was called.
func (b *ExponentialBackOff) GetElapsedTime() time.Duration {
	return b.Clock.Now().Sub(b.startTime)
}

// incrementCurrentInterval multiplies the current interval by the multiplier,
// with overflow protection to ensure it doesn't exceed MaxInterval.
func (b *ExponentialBackOff) incrementCurrentInterval() {
	// Check for overflow: if current * multiplier would exceed max
	if float64(b.currentInterval) >= float64(b.MaxInterval)/b.Multiplier {
		b.currentInterval = b.MaxInterval
	} else {
		b.currentInterval = time.Duration(float64(b.currentInterval) * b.Multiplier)
	}
}

// getRandomValueFromInterval returns a random value from the interval:
// [currentInterval - delta, currentInterval + delta] where delta = randomizationFactor * currentInterval.
func getRandomValueFromInterval(randomizationFactor, random float64, currentInterval time.Duration) time.Duration {
	if randomizationFactor == 0 {
		return currentInterval // no randomness when factor is 0
	}
	delta := randomizationFactor * float64(currentInterval)
	minInterval := float64(currentInterval) - delta
	maxInterval := float64(currentInterval) + delta

	// Get a random value from the range [minInterval, maxInterval].
	return time.Duration(minInterval + (random * (maxInterval - minInterval + 1)))
}

// Compile-time interface check
var _ BackOff = (*ExponentialBackOff)(nil)
