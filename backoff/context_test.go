package backoff

import (
	"context"
	"testing"
	"time"
)

func TestWithContext_PanicsOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil context")
		}
	}()
	WithContext(nil, &ZeroBackOff{})
}

func TestWithContext_ReturnsStopOnCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	b := WithContext(ctx, &ZeroBackOff{})
	result := b.NextBackOff()

	if result != Stop {
		t.Errorf("expected Stop (-1), got %v", result)
	}
}

func TestWithContext_DelegatesToBackOff(t *testing.T) {
	ctx := context.Background()
	delay := 100 * time.Millisecond
	b := WithContext(ctx, NewConstantBackOff(delay))

	result := b.NextBackOff()

	if result != delay {
		t.Errorf("expected %v, got %v", delay, result)
	}
}

func TestWithContext_AvoidsDoulbeWrap(t *testing.T) {
	ctx1 := context.Background()
	ctx2, cancel := context.WithCancel(context.Background())
	defer cancel()

	inner := NewConstantBackOff(100 * time.Millisecond)
	wrapped1 := WithContext(ctx1, inner)
	wrapped2 := WithContext(ctx2, wrapped1)

	// Should unwrap and use ctx2 with inner directly
	bc, ok := wrapped2.(*backOffContext)
	if !ok {
		t.Fatal("expected *backOffContext")
	}

	// The embedded BackOff should be inner, not wrapped1
	if _, isWrapped := bc.BackOff.(*backOffContext); isWrapped {
		t.Error("double-wrapping detected; expected unwrapping")
	}

	// The context should be ctx2
	if bc.ctx != ctx2 {
		t.Error("expected ctx2 after re-wrapping")
	}
}

func TestWithContext_ContextMethod(t *testing.T) {
	ctx := context.Background()
	b := WithContext(ctx, &ZeroBackOff{})

	if b.Context() != ctx {
		t.Error("Context() should return the wrapped context")
	}
}

func TestWithContext_Reset(t *testing.T) {
	ctx := context.Background()
	inner := NewExponentialBackOff(WithInitialInterval(10 * time.Millisecond))
	b := WithContext(ctx, inner)

	// Call NextBackOff to advance state
	b.NextBackOff()

	// Reset should delegate
	b.Reset()

	// After reset, the exponential backoff should be at initial interval
	if inner.currentInterval != inner.InitialInterval {
		t.Error("Reset did not delegate to wrapped backoff")
	}
}

func TestGetContext_ReturnsContextFromWrapped(t *testing.T) {
	ctx := context.Background()
	b := WithContext(ctx, &ZeroBackOff{})

	if getContext(b) != ctx {
		t.Error("getContext should extract context from wrapped BackOff")
	}
}

func TestGetContext_ReturnsBackgroundForUnwrapped(t *testing.T) {
	b := &ZeroBackOff{}

	if getContext(b) != context.Background() {
		t.Error("getContext should return context.Background() for unwrapped BackOff")
	}
}

func TestGetContext_ThroughTries(t *testing.T) {
	ctx := context.Background()
	inner := WithContext(ctx, &ZeroBackOff{})
	wrapped := WithMaxRetries(inner, 5)

	if getContext(wrapped) != ctx {
		t.Error("getContext should extract context through tries wrapper")
	}
}
