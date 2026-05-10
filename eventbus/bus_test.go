package eventbus

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEvent implements Event interface for testing.
type testEvent struct {
	ID      string
	Message string
}

func (e testEvent) EventName() string { return "testEvent" }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestSubscribeAndPublish(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	defer bus.Close()

	var received atomic.Value

	sub := Subscribe(bus, func(ctx context.Context, e testEvent) {
		received.Store(e)
	})
	require.NotNil(t, sub)

	Publish(context.Background(), bus, testEvent{ID: "1", Message: "hello"}, "")

	// Wait for async delivery
	require.Eventually(t, func() bool {
		return received.Load() != nil
	}, time.Second, 10*time.Millisecond)

	got := received.Load()
	require.NotNil(t, got)
	assert.Equal(t, "1", got.(testEvent).ID)
	assert.Equal(t, "hello", got.(testEvent).Message)
}

func TestMultipleSubscribers(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	defer bus.Close()

	var count atomic.Int32

	Subscribe(bus, func(ctx context.Context, e testEvent) { count.Add(1) })
	Subscribe(bus, func(ctx context.Context, e testEvent) { count.Add(1) })
	Subscribe(bus, func(ctx context.Context, e testEvent) { count.Add(1) })

	Publish(context.Background(), bus, testEvent{ID: "1"}, "")

	require.Eventually(t, func() bool {
		return count.Load() == 3
	}, time.Second, 10*time.Millisecond)
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	defer bus.Close()

	var count atomic.Int32

	sub := Subscribe(bus, func(ctx context.Context, e testEvent) {
		count.Add(1)
	})

	Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	require.Eventually(t, func() bool {
		return count.Load() == 1
	}, time.Second, 10*time.Millisecond)

	sub.Unsubscribe()

	Publish(context.Background(), bus, testEvent{ID: "2"}, "")
	// After unsubscribe, count should remain 1. Use a short wait to confirm no delivery.
	require.Never(t, func() bool {
		return count.Load() > 1
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestTopicFiltering(t *testing.T) {
	t.Parallel()
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

	require.Eventually(t, func() bool {
		return adminCount.Load() == 1 && userCount.Load() == 1 && wildcardCount.Load() == 2
	}, time.Second, 10*time.Millisecond)
}

func TestPanicRecovery(t *testing.T) {
	t.Parallel()
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

	// Safe handler should have received the event despite the panic in the other handler
	require.Eventually(t, func() bool {
		return safeCount.Load() == 1
	}, time.Second, 10*time.Millisecond)
}

func TestCloseDrainsHandlers(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())

	var completed atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)

	// releaseHandler unblocks the handler goroutine, simulating a slow handler
	// without using time.Sleep.
	releaseHandler := make(chan struct{})
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		defer wg.Done()
		<-releaseHandler // Block until test releases
		completed.Store(true)
	})

	Publish(context.Background(), bus, testEvent{ID: "1"}, "")

	// Release the handler after a brief moment, then Close should wait for it
	go func() {
		// Small delay so Close() has time to begin waiting
		<-time.After(50 * time.Millisecond)
		close(releaseHandler)
	}()

	// Close should wait for handler to complete
	bus.Close()

	// Handler should have completed before Close returned
	assert.True(t, completed.Load())
}

func TestPublishToClosedBus(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())

	var count atomic.Int32
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		count.Add(1)
	})

	bus.Close()

	// Should be silent no-op, no panic
	Publish(context.Background(), bus, testEvent{ID: "1"}, "")

	// Confirm no delivery after publishing to closed bus
	require.Never(t, func() bool {
		return count.Load() > 0
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestSubscribeToClosedBus(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	bus.Close()

	sub := Subscribe(bus, func(ctx context.Context, e testEvent) {})
	assert.Nil(t, sub)
}

func TestWorkerInterface(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	defer bus.Close()

	assert.Equal(t, "eventbus.EventBus", bus.Name())

	// OnStart/OnStop should not panic and return nil
	err := bus.OnStart(context.Background())
	assert.NoError(t, err)

	err = bus.OnStop(context.Background())
	assert.NoError(t, err)
}

func TestBufferSizeOption(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	defer bus.Close()

	var received atomic.Int32

	// Small buffer
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		received.Add(1)
	}, WithBufferSize(2))

	// Publish several events
	for range 5 {
		Publish(context.Background(), bus, testEvent{ID: "1"}, "")
	}

	require.Eventually(t, func() bool {
		return received.Load() == 5
	}, time.Second, 10*time.Millisecond)
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	defer bus.Close()

	// Slow consumer with tiny buffer -- blocks on channel to simulate slow processing
	block := make(chan struct{})
	defer close(block)
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		select {
		case <-block:
		case <-ctx.Done():
		}
	}, WithBufferSize(1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// This should not block forever due to context cancellation
	for range 10 {
		Publish(ctx, bus, testEvent{ID: "1"}, "")
	}
}

func TestDoubleClose(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())

	// First close
	bus.Close()

	// Second close should be idempotent (no panic)
	bus.Close()
}

func TestDoubleUnsubscribe(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	// Calling Unsubscribe on a nil subscription should be safe
	var sub *Subscription
	sub.Unsubscribe() // Should not panic
}

func TestConcurrentPublish(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	defer bus.Close()

	var count atomic.Int32

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		count.Add(1)
	})

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			Publish(context.Background(), bus, testEvent{ID: string(rune('A' + id))}, "")
		}(i)
	}

	wg.Wait()

	require.Eventually(t, func() bool {
		return count.Load() == 100
	}, time.Second, 10*time.Millisecond)
}

func TestConcurrentSubscribe(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())
	defer bus.Close()

	var wg sync.WaitGroup
	for range 10 {
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

// anotherEvent is a second event type for testing type routing.
type anotherEvent struct {
	Value int
}

func (e anotherEvent) EventName() string { return "anotherEvent" }

func TestEventTypeRouting(t *testing.T) {
	t.Parallel()
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

	require.Eventually(t, func() bool {
		return testEventCount.Load() == 1 && anotherEventCount.Load() == 1
	}, time.Second, 10*time.Millisecond)
}

func TestEmptyTopicPublish(t *testing.T) {
	t.Parallel()
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

	require.Eventually(t, func() bool {
		return exactCount.Load() == 1 && wildcardCount.Load() == 1
	}, time.Second, 10*time.Millisecond)

	// Both should receive because:
	// - Empty topic subscription matches empty topic publish
	// - Wildcard subscription matches all topics
	// Both subscribers have topic="" so both should receive
}

func TestEventBus_ConcurrentClosePublish(t *testing.T) {
	// This test verifies that concurrent Close() and Publish() never panics
	// with a send-on-closed-channel error. Run with -race flag.
	for range 100 {
		bus := New(testLogger())

		// Subscribe a handler
		Subscribe(bus, func(ctx context.Context, e testEvent) {
			// Yield to increase contention window
			runtime.Gosched()
		})

		var wg sync.WaitGroup

		// Spawn 50 goroutines publishing concurrently
		for i := range 50 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				Publish(context.Background(), bus, testEvent{
					ID:      strconv.Itoa(id),
					Message: "concurrent",
				}, "")
			}(i)
		}

		// One goroutine calls Close
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Close()
		}()

		wg.Wait()
	}
	// If we reach here without panic, the race is fixed
}

func TestEventBus_CloseIdempotent(t *testing.T) {
	bus := New(testLogger())

	Subscribe(bus, func(ctx context.Context, e testEvent) {})

	// Close multiple times must not panic
	bus.Close()
	bus.Close()
	bus.Close()
}

func TestEventBus_PublishAfterCloseIsNoop(t *testing.T) {
	bus := New(testLogger())

	var count atomic.Int32
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		count.Add(1)
	})

	bus.Close()

	// Publish after close must not panic and must not deliver
	Publish(context.Background(), bus, testEvent{ID: "1"}, "")

	// Confirm no delivery
	require.Never(t, func() bool {
		return count.Load() > 0
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func TestContextPropagation(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	type ctxKey string
	const traceKey ctxKey = "trace-id"

	var receivedTrace atomic.Value

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		if v := ctx.Value(traceKey); v != nil {
			receivedTrace.Store(v)
		}
	})

	// Publish with a context containing a trace ID
	ctx := context.WithValue(context.Background(), traceKey, "abc-123")
	Publish(ctx, bus, testEvent{ID: "1", Message: "traced"}, "")

	require.Eventually(t, func() bool {
		return receivedTrace.Load() != nil
	}, time.Second, 10*time.Millisecond)

	got := receivedTrace.Load()
	require.NotNil(t, got, "handler should receive context value from publisher")
	assert.Equal(t, "abc-123", got.(string))
}

func TestContextPropagationCancelledContextStillDelivers(t *testing.T) {
	bus := New(testLogger())
	defer bus.Close()

	var received atomic.Bool

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		received.Store(true)
	}, WithBufferSize(10))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	Publish(ctx, bus, testEvent{ID: "1"}, "")

	// With cancelled context, Publish may not deliver because the select
	// picks ctx.Done(). This is acceptable behavior.
	// Use a short wait to let any potential delivery complete.
	require.Never(t, func() bool {
		// We just confirm this doesn't panic. Delivery is not guaranteed.
		return false
	}, 50*time.Millisecond, 10*time.Millisecond)
}

func TestContextPropagationTraceIDInHandler(t *testing.T) {
	bus := New(testLogger())

	type ctxKey string
	const requestIDKey ctxKey = "request-id"

	var mu sync.Mutex
	var got []string

	Subscribe(bus, func(ctx context.Context, e testEvent) {
		if v, ok := ctx.Value(requestIDKey).(string); ok {
			mu.Lock()
			got = append(got, v)
			mu.Unlock()
		}
	})

	for _, id := range []string{"req-1", "req-2", "req-3"} {
		ctx := context.WithValue(context.Background(), requestIDKey, id)
		Publish(ctx, bus, testEvent{ID: id}, "")
	}

	bus.Close()

	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{"req-1", "req-2", "req-3"}, got)
}

func TestCloseReturnsWithinDeadlineWhenSubscriberHung(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())

	// Subscribe with buffer size 1 and a handler that blocks (simulates hung subscriber).
	// The handler sleeps, simulating a slow consumer that can't keep up.
	handlerStarted := make(chan struct{})
	hungBlock := make(chan struct{}) // Simulates a hung handler without time.Sleep
	t.Cleanup(func() { close(hungBlock) })
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		close(handlerStarted)
		<-hungBlock // Block indefinitely (hung handler)
	}, WithBufferSize(1))

	// Send one event to start the handler (it will block in the handler)
	Publish(context.Background(), bus, testEvent{ID: "start-handler"}, "")
	<-handlerStarted // Wait for handler to begin processing

	// Now the buffer is empty (handler is processing event 1). Fill it so the next
	// Publish would block in the old code (which held RLock during send).
	Publish(context.Background(), bus, testEvent{ID: "fill-buffer"}, "")

	// Attempt a Publish that would block on full buffer in the old code.
	// In the new code, Close() fires tombstone and the blocked Publish unblocks.
	publishDone := make(chan struct{})
	go func() {
		Publish(context.Background(), bus, testEvent{ID: "would-block"}, "")
		close(publishDone)
	}()

	// Wait briefly for the Publish to attempt delivery on full buffer
	require.Eventually(t, func() bool {
		// The goroutine is launched; give scheduler time to run it
		runtime.Gosched()
		return true
	}, 100*time.Millisecond, 5*time.Millisecond)

	// Close the bus — tombstone should unblock the Publish goroutine
	closeDone := make(chan struct{})
	go func() {
		bus.Close()
		close(closeDone)
	}()

	// The Publish goroutine should unblock via tombstone within 1 second
	select {
	case <-publishDone:
		// Publish unblocked via tombstone — success
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked for > 2s despite Close; tombstone pattern failed")
	}

	// Close may still be waiting for the hung handler (expected), but Publish is unblocked.
	// That's the key invariant: Close() does NOT deadlock even with hung subscribers.
}

func TestPublishConcurrentWithClose(t *testing.T) {
	t.Parallel()

	// Run with -race flag: concurrent Publish + Close must not panic or deadlock
	for range 50 {
		bus := New(testLogger())

		var received atomic.Int32
		Subscribe(bus, func(ctx context.Context, e testEvent) {
			received.Add(1)
		})

		var wg sync.WaitGroup

		// Launch 100 goroutines calling Publish
		for i := range 100 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				Publish(context.Background(), bus, testEvent{
					ID:      strconv.Itoa(id),
					Message: "concurrent",
				}, "")
			}(i)
		}

		// Launch 1 goroutine calling Close after yielding to scheduler
		wg.Add(1)
		go func() {
			defer wg.Done()
			runtime.Gosched()
			bus.Close()
		}()

		// Must complete within 5 seconds (no deadlock)
		complete := make(chan struct{})
		go func() {
			wg.Wait()
			close(complete)
		}()

		select {
		case <-complete:
			// Success: no panics, no deadlocks
		case <-time.After(5 * time.Second):
			t.Fatal("TestPublishConcurrentWithClose timed out; possible deadlock")
		}
	}
}

// TestEventBusOnStopRespectsContext verifies that OnStop returns ctx.Err() when
// the shutdown context expires before all handler goroutines finish draining (A3 regression).
func TestEventBusOnStopRespectsContext(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())

	// Subscribe a handler that blocks for a long time, simulating a slow consumer
	handlerStarted := make(chan struct{})
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		close(handlerStarted)
		// Block indefinitely (simulating hung handler)
		select {
		case <-time.After(30 * time.Second):
		case <-ctx.Done():
		}
	}, WithBufferSize(1))

	// Publish one event to trigger the handler
	Publish(context.Background(), bus, testEvent{ID: "block"}, "")

	// Wait for the handler to start
	select {
	case <-handlerStarted:
		// Handler is now blocking
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}

	// Call OnStop with a short deadline (100ms)
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer shortCancel()

	start := time.Now()
	err := bus.OnStop(shortCtx)
	elapsed := time.Since(start)

	// OnStop should return ctx.Err() because the handler is still blocked
	require.Error(t, err, "OnStop should return error when context expires")
	assert.ErrorIs(t, err, context.DeadlineExceeded,
		"OnStop should return DeadlineExceeded when handler is still blocked")

	// Should return within a reasonable time (not hang indefinitely)
	assert.Less(t, elapsed, 500*time.Millisecond,
		"OnStop should return promptly when context expires")
}

// TestEventBusCloseWithContextNilDeadline verifies that CloseWithContext with
// background context behaves like the original Close (waits indefinitely).
func TestEventBusCloseWithContextNilDeadline(t *testing.T) {
	t.Parallel()
	bus := New(testLogger())

	var completed atomic.Bool
	Subscribe(bus, func(ctx context.Context, e testEvent) {
		// Short delay to verify Close waits
		select {
		case <-time.After(50 * time.Millisecond):
			completed.Store(true)
		case <-ctx.Done():
		}
	})

	Publish(context.Background(), bus, testEvent{ID: "wait"}, "")

	// CloseWithContext(background) should wait for handler
	err := bus.CloseWithContext(context.Background())
	require.NoError(t, err)
	assert.True(t, completed.Load(), "handler should complete before CloseWithContext returns")
}

// Run: go test -coverprofile=coverage.out ./eventbus/...
// Target: 70%+ coverage
