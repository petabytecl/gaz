// Package backoff provides exponential backoff algorithms for retry logic.
//
// The package implements a [BackOff] interface that can be used to calculate
// delays between retry attempts. The primary implementation is [ExponentialBackOff],
// which increases the delay exponentially with configurable parameters.
//
// # Basic Usage
//
// Create an exponential backoff with default settings:
//
//	b := backoff.NewExponentialBackOff()
//	for {
//	    delay := b.NextBackOff()
//	    if delay == backoff.Stop {
//	        break // give up
//	    }
//	    time.Sleep(delay)
//	    // retry operation...
//	}
//
// # Custom Configuration
//
// Use functional options to customize the backoff behavior:
//
//	b := backoff.NewExponentialBackOff(
//	    backoff.WithInitialInterval(500 * time.Millisecond),
//	    backoff.WithMaxInterval(30 * time.Second),
//	    backoff.WithMultiplier(1.5),
//	    backoff.WithRandomizationFactor(0.3),
//	)
//
// # Simple BackOff Types
//
// For simple cases, use the provided implementations:
//   - [ZeroBackOff]: Always returns 0 (no delay)
//   - [StopBackOff]: Always returns [Stop] (no retries)
//   - [ConstantBackOff]: Always returns the same delay
//
// # Stop Sentinel
//
// The [Stop] constant (-1) signals that no more retries should be attempted.
// [ExponentialBackOff] returns this when [MaxElapsedTime] is exceeded.
package backoff
