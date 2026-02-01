package backoff

import (
	"testing"
	"time"
)

func TestWithMaxRetries_StopsAfterMaxAttempts(t *testing.T) {
	maxTries := uint64(3)
	b := WithMaxRetries(&ZeroBackOff{}, maxTries)

	// Should return zero for first 3 calls
	for i := uint64(0); i < maxTries; i++ {
		result := b.NextBackOff()
		if result != 0 {
			t.Errorf("call %d: expected 0, got %v", i+1, result)
		}
	}

	// 4th call should return Stop
	result := b.NextBackOff()
	if result != Stop {
		t.Errorf("call 4: expected Stop (-1), got %v", result)
	}
}

func TestWithMaxRetries_ZeroReturnsStopImmediately(t *testing.T) {
	b := WithMaxRetries(&ZeroBackOff{}, 0)

	result := b.NextBackOff()
	if result != Stop {
		t.Errorf("expected Stop (-1) for maxTries=0, got %v", result)
	}
}

func TestWithMaxRetries_ResetResetsCounter(t *testing.T) {
	maxTries := uint64(2)
	b := WithMaxRetries(&ZeroBackOff{}, maxTries)

	// Exhaust retries
	for i := uint64(0); i < maxTries; i++ {
		b.NextBackOff()
	}
	if b.NextBackOff() != Stop {
		t.Fatal("expected Stop after exhausting retries")
	}

	// Reset
	b.Reset()

	// Should work again
	result := b.NextBackOff()
	if result != 0 {
		t.Errorf("after Reset: expected 0, got %v", result)
	}
}

func TestWithMaxRetries_DelegatesToBackOff(t *testing.T) {
	delay := 100 * time.Millisecond
	b := WithMaxRetries(NewConstantBackOff(delay), 5)

	result := b.NextBackOff()
	if result != delay {
		t.Errorf("expected %v, got %v", delay, result)
	}
}

func TestWithMaxRetries_SingleAttempt(t *testing.T) {
	b := WithMaxRetries(&ZeroBackOff{}, 1)

	// First call should succeed
	result := b.NextBackOff()
	if result != 0 {
		t.Errorf("first call: expected 0, got %v", result)
	}

	// Second call should return Stop
	result = b.NextBackOff()
	if result != Stop {
		t.Errorf("second call: expected Stop (-1), got %v", result)
	}
}

func TestWithMaxRetries_ResetDelegates(t *testing.T) {
	inner := NewExponentialBackOff(WithInitialInterval(10 * time.Millisecond))
	b := WithMaxRetries(inner, 5)

	// Advance the exponential backoff
	b.NextBackOff()

	// Reset should delegate
	b.Reset()

	// Inner backoff should be reset
	if inner.currentInterval != inner.InitialInterval {
		t.Error("Reset did not delegate to wrapped backoff")
	}
}
