package backoff

import "time"

// Stop indicates that the operation should not be retried.
const Stop time.Duration = -1

// Interface compliance assertions.
var (
	_ BackOff = (*ZeroBackOff)(nil)
	_ BackOff = (*StopBackOff)(nil)
	_ BackOff = (*ConstantBackOff)(nil)
)

// BackOff defines the interface for backoff algorithms.
// Implementations must be safe for concurrent use if used by multiple goroutines.
type BackOff interface {
	// NextBackOff returns the duration to wait before retrying the operation.
	// Returns [Stop] to signal that no more retries should be attempted.
	NextBackOff() time.Duration
	// Reset restores the backoff to its initial state.
	// Should be called after a successful operation.
	Reset()
}

// ZeroBackOff is a [BackOff] that always returns 0 (no delay between retries).
type ZeroBackOff struct{}

// Reset is a no-op for [ZeroBackOff].
func (z *ZeroBackOff) Reset() {}

// NextBackOff always returns 0.
func (z *ZeroBackOff) NextBackOff() time.Duration { return 0 }

// StopBackOff is a [BackOff] that always returns [Stop].
// Use this to signal that the operation should not be retried.
type StopBackOff struct{}

// Reset is a no-op for [StopBackOff].
func (s *StopBackOff) Reset() {}

// NextBackOff always returns [Stop].
func (s *StopBackOff) NextBackOff() time.Duration { return Stop }

// ConstantBackOff is a [BackOff] that always returns the same delay.
type ConstantBackOff struct {
	Delay time.Duration
}

// NewConstantBackOff creates a new [ConstantBackOff] with the given delay.
func NewConstantBackOff(delay time.Duration) *ConstantBackOff {
	return &ConstantBackOff{Delay: delay}
}

// Reset is a no-op for [ConstantBackOff].
func (c *ConstantBackOff) Reset() {}

// NextBackOff always returns the configured [Delay].
func (c *ConstantBackOff) NextBackOff() time.Duration { return c.Delay }
