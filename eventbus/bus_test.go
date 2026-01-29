package eventbus

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvent implements Event interface for testing
type testEvent struct {
	ID      string
	Message string
}

func (e testEvent) EventName() string { return "testEvent" }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestSubscribeAndPublish(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var received atomic.Value

	sub := Subscribe(bus, func(ctx context.Context, e testEvent) {
		received.Store(e)
	})
	require.NotNil(t, sub)

	Publish(context.Background(), bus, testEvent{ID: "1", Message: "hello"}, "")

	// Wait for async delivery
	time.Sleep(50 * time.Millisecond)

	got := received.Load()
	require.NotNil(t, got)
	assert.Equal(t, "1", got.(testEvent).ID)
	assert.Equal(t, "hello", got.(testEvent).Message)
}

func TestMultipleSubscribers(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var count atomic.Int32

	Subscribe(bus, func(ctx context.Context, e testEvent) { count.Add(1) })
	Subscribe(bus, func(ctx context.Context, e testEvent) { count.Add(1) })
	Subscribe(bus, func(ctx context.Context, e testEvent) { count.Add(1) })

	Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(3), count.Load())
}

func TestUnsubscribe(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var count atomic.Int32

	sub := Subscribe(bus, func(ctx context.Context, e testEvent) {
		count.Add(1)
	})

	Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load())

	sub.Unsubscribe()

	Publish(context.Background(), bus, testEvent{ID: "2"}, "")
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(1), count.Load()) // No change after unsubscribe
}

func TestTopicFiltering(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var adminCount, userCount, wildcardCount atomic.Int32

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		adminCount.Add(1)
	}, WithTopic("admin"))

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		userCount.Add(1)
	}, WithTopic("user"))

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		wildcardCount.Add(1)
	}) // No topic = wildcard

	Publish(context.Background(), bus, testEvent{ID: "1"}, "admin")
	Publish(context.Background(), bus, testEvent{ID: "2"}, "user")
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), adminCount.Load())
	assert.Equal(t, int32(1), userCount.Load())
	assert.Equal(t, int32(2), wildcardCount.Load()) // Wildcard receives all
}

func TestPanicRecovery(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var safeCount atomic.Int32

	// Handler that panics
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		panic("test panic")
	})

	// Handler that should still receive events
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		safeCount.Add(1)
	})

	Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	time.Sleep(100 * time.Millisecond)

	// Safe handler should have received the event
	assert.Equal(t, int32(1), safeCount.Load())
}

func TestCloseDrainsHandlers(t *testing.T) {
	bus := New(testLogger())

	var completed atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond) // Simulate slow handler
		completed.Store(true)
	})

	Publish(context.Background(), bus, testEvent{ID: "1"}, "")

	// Close should wait for handler to complete
	bus.Close()

	// Handler should have completed before Close returned
	assert.True(t, completed.Load())
}

func TestPublishToClosedBus(t *testing.T) {
	bus := New(testLogger())

	var count atomic.Int32
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		count.Add(1)
	})

	bus.Close()

	// Should be silent no-op, no panic
	Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int32(0), count.Load())
}

func TestSubscribeToClosedBus(t *testing.T) {
	bus := New(testLogger())
	bus.Close()

	sub := Subscribe(bus, func(ctx context.Context, e testEvent) {})
	assert.Nil(t, sub)
}

func TestWorkerInterface(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	assert.Equal(t, "eventbus.EventBus", bus.Name())

	// Start/Stop should not panic
	bus.Start()
	bus.Stop()
}

func TestBufferSizeOption(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var received atomic.Int32

	// Small buffer
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		time.Sleep(10 * time.Millisecond)
		received.Add(1)
	}, WithBufferSize(2))

	// Publish several events
	for i := 0; i < 5; i++ {
		Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	}

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(5), received.Load())
}

func TestContextCancellation(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	// Slow consumer with tiny buffer
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		time.Sleep(100 * time.Millisecond)
	}, WithBufferSize(1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// This should not block forever due to context cancellation
	for i := 0; i < 10; i++ {
		Publish(ctx, bus, testEvent{ID: "1"}, "")
	}
}

func TestDoubleClose(t *testing.T) {
	bus := New(testLogger())

	// First close
	bus.Close()

	// Second close should be idempotent (no panic)
	bus.Close()
}

func TestDoubleUnsubscribe(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	sub := Subscribe(bus, func(ctx context.Context, e testEvent) {})
	require.NotNil(t, sub)

	// First unsubscribe
	sub.Unsubscribe()

	// Second unsubscribe should be safe (no panic)
	sub.Unsubscribe()
}

func TestNilSubscription(t *testing.T) {
	// Calling Unsubscribe on a nil subscription should be safe
	var sub *Subscription
	sub.Unsubscribe() // Should not panic
}

func TestConcurrentPublish(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var count atomic.Int32

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		count.Add(1)
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			Publish(context.Background(), bus, testEvent{ID: string(rune('A' + id))}, "")
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(100), count.Load())
}

func TestConcurrentSubscribe(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub := Subscribe(bus, func(ctx context.Context, e testEvent) {})
			if sub != nil {
				sub.Unsubscribe()
			}
		}()
	}

	wg.Wait()
	// If we get here without race detector complaints, thread safety is good
}

// anotherEvent is a second event type for testing type routing
type anotherEvent struct {
	Value int
}

func (e anotherEvent) EventName() string { return "anotherEvent" }

func TestEventTypeRouting(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var testEventCount, anotherEventCount atomic.Int32

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		testEventCount.Add(1)
	})

	Subscribe(bus, func(ctx context.Context, e anotherEvent) {
		anotherEventCount.Add(1)
	})

	Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	Publish(context.Background(), bus, anotherEvent{Value: 42}, "")
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), testEventCount.Load())
	assert.Equal(t, int32(1), anotherEventCount.Load())
}

func TestEmptyTopicPublish(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var exactCount, wildcardCount atomic.Int32

	// Subscribe to empty topic (exact match)
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		exactCount.Add(1)
	}, WithTopic(""))

	// Subscribe without topic (wildcard)
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		wildcardCount.Add(1)
	})

	// Publish with empty topic
	Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	time.Sleep(50 * time.Millisecond)

	// Both should receive because:
	// - Empty topic subscription matches empty topic publish
	// - Wildcard subscription matches all topics
	// But wait, the code only adds wildcard handlers when topic != ""
	// So for empty topic publish, only exact match is found
	// Actually looking at the code, when topic == "", we don't add wildcard handlers
	// So only the exact match (empty topic) handler receives the event
	// But both subscribers have topic="" so both should receive
	assert.Equal(t, int32(1), exactCount.Load())
	assert.Equal(t, int32(1), wildcardCount.Load())
}

// Run: go test -coverprofile=coverage.out ./eventbus/...
// Target: 70%+ coverage
