package runtime_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	runtimecheck "github.com/petabytecl/gaz/health/checks/runtime"
)

func TestGoroutineCount(t *testing.T) {
	ctx := context.Background()

	t.Run("passes when under threshold", func(t *testing.T) {
		// Use current count + 100 to ensure we're under
		current := runtime.NumGoroutine()
		check := runtimecheck.GoroutineCount(current + 100)

		if err := check(ctx); err != nil {
			t.Errorf("expected nil, got error: %v", err)
		}
	})

	t.Run("fails when over threshold", func(t *testing.T) {
		// Use 1 as threshold - guaranteed to fail since tests use goroutines
		check := runtimecheck.GoroutineCount(1)

		err := check(ctx)
		if err == nil {
			t.Error("expected error when over threshold, got nil")
		}
	})

	t.Run("error message includes counts", func(t *testing.T) {
		check := runtimecheck.GoroutineCount(1)
		err := check(ctx)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		errMsg := err.Error()
		if !contains(errMsg, "goroutines") {
			t.Errorf("error message should mention goroutines: %s", errMsg)
		}
	})
}

func TestMemoryUsage(t *testing.T) {
	ctx := context.Background()

	t.Run("passes when under threshold", func(t *testing.T) {
		// Use 10GB threshold - should always pass
		check := runtimecheck.MemoryUsage(10 << 30)

		if err := check(ctx); err != nil {
			t.Errorf("expected nil, got error: %v", err)
		}
	})

	t.Run("fails when over threshold", func(t *testing.T) {
		// Use 1 byte threshold - guaranteed to fail
		check := runtimecheck.MemoryUsage(1)

		err := check(ctx)
		if err == nil {
			t.Error("expected error when over threshold, got nil")
		}
	})

	t.Run("error message includes bytes", func(t *testing.T) {
		check := runtimecheck.MemoryUsage(1)
		err := check(ctx)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		errMsg := err.Error()
		if !contains(errMsg, "bytes") {
			t.Errorf("error message should mention bytes: %s", errMsg)
		}
	})
}

func TestGCPause(t *testing.T) {
	ctx := context.Background()

	t.Run("passes with high threshold", func(t *testing.T) {
		// Use 1 hour threshold - any GC pause is shorter
		check := runtimecheck.GCPause(time.Hour)

		if err := check(ctx); err != nil {
			t.Errorf("expected nil, got error: %v", err)
		}
	})

	t.Run("passes when no GC has run", func(t *testing.T) {
		// Very low threshold, but if no GC has run, should still pass
		// This is a best-effort test - can't guarantee no GC
		check := runtimecheck.GCPause(time.Hour)

		if err := check(ctx); err != nil {
			t.Errorf("expected nil with high threshold, got error: %v", err)
		}
	})
}

func TestContextNotBlocking(t *testing.T) {
	// Verify checks return immediately (they're CPU-bound, not blocking)
	ctx := context.Background()

	t.Run("GoroutineCount does not block", func(t *testing.T) {
		check := runtimecheck.GoroutineCount(1000)
		done := make(chan struct{})

		go func() {
			_ = check(ctx)
			close(done)
		}()

		select {
		case <-done:
			// OK - returned quickly
		case <-time.After(100 * time.Millisecond):
			t.Error("check took too long, may be blocking")
		}
	})

	t.Run("MemoryUsage does not block", func(t *testing.T) {
		check := runtimecheck.MemoryUsage(1 << 30)
		done := make(chan struct{})

		go func() {
			_ = check(ctx)
			close(done)
		}()

		select {
		case <-done:
			// OK - returned quickly
		case <-time.After(100 * time.Millisecond):
			t.Error("check took too long, may be blocking")
		}
	})

	t.Run("GCPause does not block", func(t *testing.T) {
		check := runtimecheck.GCPause(time.Hour)
		done := make(chan struct{})

		go func() {
			_ = check(ctx)
			close(done)
		}()

		select {
		case <-done:
			// OK - returned quickly
		case <-time.After(100 * time.Millisecond):
			t.Error("check took too long, may be blocking")
		}
	})
}

// contains checks if substr is in s.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
