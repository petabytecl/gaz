package backoff

import (
	"context"
	"testing"
	"time"
)

func TestTicker_SendsAtLeastOneTick(t *testing.T) {
	// Use StopBackOff which immediately signals stop after first tick
	ticker := NewTicker(&StopBackOff{})
	defer ticker.Stop()

	select {
	case <-ticker.C:
		// Got at least one tick - success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected at least one tick")
	}
}

func TestTicker_StopsOnStopCall(t *testing.T) {
	ticker := NewTicker(NewConstantBackOff(10 * time.Millisecond))

	// Receive first tick
	select {
	case <-ticker.C:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected first tick")
	}

	// Stop the ticker
	ticker.Stop()

	// Channel should be closed
	select {
	case _, ok := <-ticker.C:
		if ok {
			// Might get one more tick that was in flight
			select {
			case _, ok := <-ticker.C:
				if ok {
					t.Error("expected channel to close after stop")
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("channel not closed after stop")
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("channel not closed after stop")
	}
}

func TestTicker_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	backoff := WithContext(ctx, NewConstantBackOff(10*time.Millisecond))
	ticker := NewTicker(backoff)

	// Receive first tick
	select {
	case <-ticker.C:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected first tick")
	}

	// Cancel context
	cancel()

	// Channel should be closed
	select {
	case _, ok := <-ticker.C:
		if ok {
			// Might get one more tick
			select {
			case _, ok := <-ticker.C:
				if ok {
					t.Error("expected channel to close after context cancel")
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("channel not closed after context cancel")
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("channel not closed after context cancel")
	}
}

func TestTicker_StopsWhenBackOffStops(t *testing.T) {
	// Backoff that allows 2 ticks then stops
	ticker := NewTicker(WithMaxRetries(&ZeroBackOff{}, 2))

	// Should get exactly 3 ticks (initial + 2 from backoff)
	ticks := 0
	timeout := time.After(100 * time.Millisecond)

loop:
	for {
		select {
		case _, ok := <-ticker.C:
			if !ok {
				break loop
			}
			ticks++
		case <-timeout:
			t.Fatal("timeout waiting for ticker to stop")
		}
	}

	// Initial tick + 2 backoff ticks = 3 total
	if ticks != 3 {
		t.Errorf("expected 3 ticks, got %d", ticks)
	}
}

func TestTicker_StopIsIdempotent(t *testing.T) {
	ticker := NewTicker(&StopBackOff{})

	// Wait for the ticker to complete
	<-ticker.C

	// Multiple stops should not panic
	ticker.Stop()
	ticker.Stop()
	ticker.Stop()
}
