// Package runtime provides health checks based on Go runtime metrics.
// These checks are useful for liveness probes to detect resource exhaustion.
package runtime

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// pauseNsBufferSize is the size of the PauseNs circular buffer in runtime.MemStats.
const pauseNsBufferSize = 256

// GoroutineCount returns a check that fails if goroutine count exceeds threshold.
// Useful for detecting goroutine leaks which lead to resource exhaustion.
//
// Example threshold: 1000 for a typical web service.
func GoroutineCount(threshold int) func(context.Context) error {
	return func(_ context.Context) error {
		count := runtime.NumGoroutine()
		if count > threshold {
			return fmt.Errorf("runtime: too many goroutines (%d > %d)", count, threshold)
		}
		return nil
	}
}

// MemoryUsage returns a check that fails if heap allocation exceeds threshold.
// Threshold is in bytes. Useful for detecting memory leaks before OOM.
//
// Example threshold: 1<<30 (1GB) for a typical web service.
func MemoryUsage(threshold uint64) func(context.Context) error {
	return func(_ context.Context) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if m.Alloc > threshold {
			return fmt.Errorf("runtime: memory usage too high (%d bytes > %d bytes)",
				m.Alloc, threshold)
		}
		return nil
	}
}

// GCPause returns a check that fails if any recent GC pause exceeds threshold.
// Useful for detecting GC pressure that affects latency-sensitive applications.
//
// Example threshold: 100*time.Millisecond for latency-sensitive services.
func GCPause(threshold time.Duration) func(context.Context) error {
	// Safe conversion: time.Duration is int64, and nanoseconds of any reasonable
	// threshold (up to ~292 years) will always be positive and fit in uint64.
	thresholdNs := uint64(threshold.Nanoseconds()) //nolint:gosec // safe for reasonable thresholds
	return func(_ context.Context) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Check the most recent GC pause (PauseNs is a circular buffer)
		// NumGC is total GC cycles, PauseNs[(NumGC+255)%256] is most recent
		if m.NumGC > 0 {
			idx := (m.NumGC + pauseNsBufferSize - 1) % pauseNsBufferSize
			if m.PauseNs[idx] > thresholdNs {
				pauseDuration := time.Duration(m.PauseNs[idx]) //nolint:gosec // safe for GC pause durations
				return fmt.Errorf("runtime: GC pause too long (%s > %s)",
					pauseDuration, threshold)
			}
		}
		return nil
	}
}
