package eventbus

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// TestBus - Factory for test EventBus
// =============================================================================

// TestBus creates an EventBus suitable for testing.
// Uses a discard logger to avoid log noise in tests.
//
// The returned EventBus is fully functional and should be
// closed after use (typically via defer in tests).
//
// # Example
//
//	bus := eventbus.TestBus()
//	defer bus.Close()
//
//	sub := eventbus.Subscribe(bus, func(ctx context.Context, e MyEvent) {
//	    // handle event
//	})
func TestBus() *EventBus {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(logger)
}

// =============================================================================
// TestSubscriber - Event collector with synchronization
// =============================================================================

// TestSubscriber collects events for testing and supports waiting for events.
// It is generic over the event type T and provides synchronization helpers
// for testing asynchronous event delivery.
//
// # Example
//
//	ts := eventbus.NewTestSubscriber[MyEvent](2)
//	eventbus.Subscribe(bus, ts.Handler())
//
//	eventbus.Publish(ctx, bus, MyEvent{ID: "1"}, "")
//	eventbus.Publish(ctx, bus, MyEvent{ID: "2"}, "")
//
//	if ts.WaitFor(time.Second) {
//	    events := ts.Events()
//	    // verify events
//	}
type TestSubscriber[T Event] struct {
	events []T
	mu     sync.Mutex
	wg     sync.WaitGroup
	count  int // Expected event count for WaitFor
}

// NewTestSubscriber creates a TestSubscriber expecting n events.
// Use n=0 if you don't know how many events to expect.
//
// When expectedCount > 0, the subscriber uses a WaitGroup internally
// to enable WaitFor() synchronization.
func NewTestSubscriber[T Event](expectedCount int) *TestSubscriber[T] {
	ts := &TestSubscriber[T]{count: expectedCount}
	if expectedCount > 0 {
		ts.wg.Add(expectedCount)
	}
	return ts
}

// Handler returns a handler function suitable for eventbus.Subscribe.
// The handler collects all received events and signals the WaitGroup
// when expected count is reached.
func (ts *TestSubscriber[T]) Handler() Handler[T] {
	return func(_ context.Context, event T) {
		ts.mu.Lock()
		ts.events = append(ts.events, event)
		if ts.count > 0 && len(ts.events) <= ts.count {
			ts.wg.Done()
		}
		ts.mu.Unlock()
	}
}

// Events returns a copy of received events.
// Safe to call concurrently with Handler.
func (ts *TestSubscriber[T]) Events() []T {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	result := make([]T, len(ts.events))
	copy(result, ts.events)
	return result
}

// WaitFor waits for the expected number of events with timeout.
// Returns true if all expected events were received, false on timeout.
//
// WaitFor only works when expectedCount > 0 was passed to NewTestSubscriber.
// If expectedCount was 0, WaitFor returns immediately with false.
//
// # Example
//
//	ts := eventbus.NewTestSubscriber[MyEvent](2)
//	// ... subscribe and publish ...
//	if !ts.WaitFor(time.Second) {
//	    t.Fatal("timeout waiting for events")
//	}
func (ts *TestSubscriber[T]) WaitFor(timeout time.Duration) bool {
	if ts.count <= 0 {
		return false
	}

	done := make(chan struct{})
	go func() {
		ts.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Count returns the number of events received so far.
// Safe to call concurrently with Handler.
func (ts *TestSubscriber[T]) Count() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return len(ts.events)
}

// Reset clears collected events and resets the expected count.
// Use this to reuse a TestSubscriber across multiple test cases.
func (ts *TestSubscriber[T]) Reset(expectedCount int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.events = nil
	ts.count = expectedCount
	if expectedCount > 0 {
		ts.wg = sync.WaitGroup{}
		ts.wg.Add(expectedCount)
	}
}

// =============================================================================
// TestEvent - Simple event for testing
// =============================================================================

// TestEvent is a simple event type for testing.
// It implements the Event interface and is suitable for basic tests.
type TestEvent struct {
	ID      string
	Message string
}

// EventName implements Event interface.
func (e TestEvent) EventName() string { return "TestEvent" }

// =============================================================================
// Require* assertion helpers
// =============================================================================

// RequireEventReceived asserts at least one event was received.
// Uses testing.TB for compatibility with both tests and benchmarks.
//
// # Example
//
//	ts := eventbus.NewTestSubscriber[MyEvent](1)
//	eventbus.Subscribe(bus, ts.Handler())
//	eventbus.Publish(ctx, bus, MyEvent{ID: "1"}, "")
//	time.Sleep(50 * time.Millisecond)
//	eventbus.RequireEventReceived(t, ts)
func RequireEventReceived[T Event](tb testing.TB, ts *TestSubscriber[T]) {
	tb.Helper()
	if ts.Count() == 0 {
		tb.Fatal("expected at least one event, got none")
	}
}

// RequireEventCount asserts exact number of events received.
//
// # Example
//
//	eventbus.RequireEventCount(t, ts, 3)
func RequireEventCount[T Event](tb testing.TB, ts *TestSubscriber[T], expected int) {
	tb.Helper()
	actual := ts.Count()
	if actual != expected {
		tb.Fatalf("expected %d events, got %d", expected, actual)
	}
}

// RequireEventsReceived waits for events with timeout and fails if not received.
// This is a convenience function that combines WaitFor and failure reporting.
//
// # Example
//
//	ts := eventbus.NewTestSubscriber[MyEvent](2)
//	eventbus.Subscribe(bus, ts.Handler())
//	eventbus.Publish(ctx, bus, MyEvent{}, "")
//	eventbus.Publish(ctx, bus, MyEvent{}, "")
//	eventbus.RequireEventsReceived(t, ts, time.Second)
func RequireEventsReceived[T Event](tb testing.TB, ts *TestSubscriber[T], timeout time.Duration) {
	tb.Helper()
	if !ts.WaitFor(timeout) {
		tb.Fatalf("timeout waiting for events after %v, received %d of %d expected",
			timeout, ts.Count(), ts.count)
	}
}

// RequireNoEvents asserts that no events were received.
// Useful for negative testing scenarios.
//
// # Example
//
//	// Verify filtering works - no events for wrong topic
//	eventbus.Publish(ctx, bus, MyEvent{}, "wrong-topic")
//	time.Sleep(50 * time.Millisecond)
//	eventbus.RequireNoEvents(t, ts)
func RequireNoEvents[T Event](tb testing.TB, ts *TestSubscriber[T]) {
	tb.Helper()
	if count := ts.Count(); count > 0 {
		tb.Fatalf("expected no events, got %d", count)
	}
}
