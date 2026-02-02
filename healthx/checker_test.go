package healthx

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewChecker_EmptyChecker(t *testing.T) {
	checker := NewChecker()
	result := checker.Check(context.Background())

	if result.Status != StatusUp {
		t.Errorf("expected StatusUp for empty checker (matches alexliesenfeld/health), got %v", result.Status)
	}
	if len(result.Details) != 0 {
		t.Errorf("expected empty details, got %d entries", len(result.Details))
	}
}

func TestNewChecker_SingleCheckPassing(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name: "test",
			Check: func(ctx context.Context) error {
				return nil
			},
		}),
	)

	result := checker.Check(context.Background())

	if result.Status != StatusUp {
		t.Errorf("expected StatusUp, got %v", result.Status)
	}
	if len(result.Details) != 1 {
		t.Errorf("expected 1 detail, got %d", len(result.Details))
	}
	if detail, ok := result.Details["test"]; !ok {
		t.Error("expected 'test' in details")
	} else {
		if detail.Status != StatusUp {
			t.Errorf("expected check StatusUp, got %v", detail.Status)
		}
		if detail.Error != nil {
			t.Errorf("expected nil error, got %v", detail.Error)
		}
		if detail.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}
	}
}

func TestNewChecker_SingleCheckFailing(t *testing.T) {
	testErr := errors.New("check failed")
	checker := NewChecker(
		WithCheck(Check{
			Name: "failing",
			Check: func(ctx context.Context) error {
				return testErr
			},
		}),
	)

	result := checker.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected StatusDown, got %v", result.Status)
	}
	if detail, ok := result.Details["failing"]; !ok {
		t.Error("expected 'failing' in details")
	} else {
		if detail.Status != StatusDown {
			t.Errorf("expected check StatusDown, got %v", detail.Status)
		}
		if detail.Error == nil {
			t.Error("expected error, got nil")
		}
	}
}

//nolint:gocognit // Test function with concurrent check verification
func TestNewChecker_MultipleChecksParallel(t *testing.T) {
	// Use atomic counter to verify concurrent execution
	var counter int32
	var maxConcurrent int32

	checker := NewChecker(
		WithCheck(Check{
			Name: "check1",
			Check: func(ctx context.Context) error {
				current := atomic.AddInt32(&counter, 1)
				// Track max concurrency
				for {
					currentMax := atomic.LoadInt32(&maxConcurrent)
					if current <= currentMax || atomic.CompareAndSwapInt32(&maxConcurrent, currentMax, current) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&counter, -1)
				return nil
			},
		}),
		WithCheck(Check{
			Name: "check2",
			Check: func(ctx context.Context) error {
				current := atomic.AddInt32(&counter, 1)
				for {
					currentMax := atomic.LoadInt32(&maxConcurrent)
					if current <= currentMax || atomic.CompareAndSwapInt32(&maxConcurrent, currentMax, current) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&counter, -1)
				return nil
			},
		}),
		WithCheck(Check{
			Name: "check3",
			Check: func(ctx context.Context) error {
				current := atomic.AddInt32(&counter, 1)
				for {
					currentMax := atomic.LoadInt32(&maxConcurrent)
					if current <= currentMax || atomic.CompareAndSwapInt32(&maxConcurrent, currentMax, current) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&counter, -1)
				return nil
			},
		}),
	)

	start := time.Now()
	result := checker.Check(context.Background())
	elapsed := time.Since(start)

	if result.Status != StatusUp {
		t.Errorf("expected StatusUp, got %v", result.Status)
	}
	if len(result.Details) != 3 {
		t.Errorf("expected 3 details, got %d", len(result.Details))
	}

	// If running in parallel, should take ~50ms, not ~150ms
	if elapsed > 100*time.Millisecond {
		t.Errorf("checks should run in parallel, took %v", elapsed)
	}

	// Verify at least 2 were running concurrently
	if maxConcurrent < 2 {
		t.Errorf("expected concurrent execution, max concurrent was %d", maxConcurrent)
	}
}

func TestNewChecker_PerCheckTimeout(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name: "slow",
			Check: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Second):
					return nil
				}
			},
			Timeout: 50 * time.Millisecond,
		}),
	)

	start := time.Now()
	result := checker.Check(context.Background())
	elapsed := time.Since(start)

	if result.Status != StatusDown {
		t.Errorf("expected StatusDown due to timeout, got %v", result.Status)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("should have timed out quickly, took %v", elapsed)
	}
	if detail, ok := result.Details["slow"]; ok {
		if detail.Error == nil {
			t.Error("expected timeout error")
		}
	}
}

func TestNewChecker_DefaultTimeout(t *testing.T) {
	// Test that WithTimeout sets default timeout
	checker := NewChecker(
		WithTimeout(30*time.Millisecond),
		WithCheck(Check{
			Name: "slow",
			Check: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Second):
					return nil
				}
			},
			// No per-check timeout, should use default
		}),
	)

	start := time.Now()
	result := checker.Check(context.Background())
	elapsed := time.Since(start)

	if result.Status != StatusDown {
		t.Errorf("expected StatusDown due to timeout, got %v", result.Status)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("should have used default timeout (30ms), took %v", elapsed)
	}
}

func TestNewChecker_PanicRecovery(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name: "panicking",
			Check: func(ctx context.Context) error {
				panic("oops, something went wrong")
			},
		}),
	)

	// Should not panic
	result := checker.Check(context.Background())

	if result.Status != StatusDown {
		t.Errorf("expected StatusDown after panic, got %v", result.Status)
	}
	if detail, ok := result.Details["panicking"]; !ok {
		t.Error("expected 'panicking' in details")
	} else {
		if detail.Status != StatusDown {
			t.Errorf("expected check StatusDown, got %v", detail.Status)
		}
		if detail.Error == nil {
			t.Error("expected error from panic recovery")
		} else if detail.Error.Error() != "panic: oops, something went wrong" {
			t.Errorf("unexpected error message: %v", detail.Error)
		}
	}
}

func TestNewChecker_CriticalVsNonCritical(t *testing.T) {
	// Test that non-critical failing check doesn't affect overall status
	checker := NewChecker(
		WithCheck(Check{
			Name: "critical-ok",
			Check: func(ctx context.Context) error {
				return nil
			},
			Critical:    true,
			criticalSet: true,
		}),
		WithCheck(Check{
			Name: "warning-fail",
			Check: func(ctx context.Context) error {
				return errors.New("warning check failed")
			},
			Critical:    false,
			criticalSet: true,
		}),
	)

	result := checker.Check(context.Background())

	// Overall should be Up because only critical check passed
	if result.Status != StatusUp {
		t.Errorf("expected StatusUp (non-critical failure shouldn't affect status), got %v", result.Status)
	}

	// But both checks should be in details
	if len(result.Details) != 2 {
		t.Errorf("expected 2 details, got %d", len(result.Details))
	}

	if detail, ok := result.Details["critical-ok"]; ok {
		if detail.Status != StatusUp {
			t.Errorf("expected critical-ok StatusUp, got %v", detail.Status)
		}
	}
	if detail, ok := result.Details["warning-fail"]; ok {
		if detail.Status != StatusDown {
			t.Errorf("expected warning-fail StatusDown, got %v", detail.Status)
		}
	}
}

func TestNewChecker_CriticalFailing(t *testing.T) {
	// Test that critical failing check affects overall status
	checker := NewChecker(
		WithCheck(Check{
			Name: "critical-fail",
			Check: func(ctx context.Context) error {
				return errors.New("critical check failed")
			},
			Critical:    true,
			criticalSet: true,
		}),
		WithCheck(Check{
			Name: "warning-ok",
			Check: func(ctx context.Context) error {
				return nil
			},
			Critical:    false,
			criticalSet: true,
		}),
	)

	result := checker.Check(context.Background())

	// Overall should be Down because critical check failed
	if result.Status != StatusDown {
		t.Errorf("expected StatusDown (critical failure), got %v", result.Status)
	}
}

func TestNewChecker_OnlyNonCriticalChecks(t *testing.T) {
	// Test that with only non-critical checks, status is Up (graceful degradation)
	checker := NewChecker(
		WithCheck(Check{
			Name: "warning1",
			Check: func(ctx context.Context) error {
				return nil
			},
			Critical:    false,
			criticalSet: true,
		}),
		WithCheck(Check{
			Name: "warning2",
			Check: func(ctx context.Context) error {
				return nil
			},
			Critical:    false,
			criticalSet: true,
		}),
	)

	result := checker.Check(context.Background())

	// No critical checks means status is Up (graceful degradation, non-critical don't affect overall status)
	if result.Status != StatusUp {
		t.Errorf("expected StatusUp (no critical checks, graceful degradation), got %v", result.Status)
	}
}

func TestNewChecker_DefaultCritical(t *testing.T) {
	// Test that checks without Critical set default to critical
	checker := NewChecker(
		WithCheck(Check{
			Name: "default",
			Check: func(ctx context.Context) error {
				return errors.New("failed")
			},
			// Critical not set - should default to true
		}),
	)

	result := checker.Check(context.Background())

	// Should be Down because check defaults to critical
	if result.Status != StatusDown {
		t.Errorf("expected StatusDown (default critical), got %v", result.Status)
	}
}

func TestNewChecker_CheckTimestamp(t *testing.T) {
	before := time.Now().UTC()

	checker := NewChecker(
		WithCheck(Check{
			Name: "timestamp-test",
			Check: func(ctx context.Context) error {
				return nil
			},
		}),
	)

	result := checker.Check(context.Background())

	after := time.Now().UTC()

	if detail, ok := result.Details["timestamp-test"]; ok {
		if detail.Timestamp.Before(before) {
			t.Errorf("timestamp %v should be after %v", detail.Timestamp, before)
		}
		if detail.Timestamp.After(after) {
			t.Errorf("timestamp %v should be before %v", detail.Timestamp, after)
		}
	} else {
		t.Error("expected 'timestamp-test' in details")
	}
}

func TestNewChecker_ContextCancellation(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name: "respects-context",
			Check: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Second):
					return nil
				}
			},
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	result := checker.Check(ctx)
	elapsed := time.Since(start)

	if result.Status != StatusDown {
		t.Errorf("expected StatusDown due to context cancellation, got %v", result.Status)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("should have respected context timeout, took %v", elapsed)
	}
}

func TestNewChecker_DuplicateCheckName(t *testing.T) {
	// Last check with same name should win
	checker := NewChecker(
		WithCheck(Check{
			Name: "duplicate",
			Check: func(ctx context.Context) error {
				return errors.New("first")
			},
		}),
		WithCheck(Check{
			Name: "duplicate",
			Check: func(ctx context.Context) error {
				return nil // second one passes
			},
		}),
	)

	result := checker.Check(context.Background())

	if result.Status != StatusUp {
		t.Errorf("expected StatusUp (second check wins), got %v", result.Status)
	}
}

func TestCheckerResult_DetailsContainsAllChecks(t *testing.T) {
	checker := NewChecker(
		WithCheck(Check{
			Name: "check1",
			Check: func(ctx context.Context) error {
				return nil
			},
		}),
		WithCheck(Check{
			Name: "check2",
			Check: func(ctx context.Context) error {
				return errors.New("failed")
			},
		}),
		WithCheck(Check{
			Name: "check3",
			Check: func(ctx context.Context) error {
				return nil
			},
		}),
	)

	result := checker.Check(context.Background())

	expectedNames := []string{"check1", "check2", "check3"}
	for _, name := range expectedNames {
		if _, ok := result.Details[name]; !ok {
			t.Errorf("expected %q in details", name)
		}
	}
}
