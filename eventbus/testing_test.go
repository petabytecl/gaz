package eventbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/eventbus"
)

// =============================================================================
// TestBus tests
// =============================================================================

func TestTestBus_Creates(t *testing.T) {
	bus := eventbus.TestBus()
	require.NotNil(t, bus)
	defer bus.Close()
}

func TestTestBus_CanPublishAndSubscribe(t *testing.T) {
	bus := eventbus.TestBus()
	defer bus.Close()

	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](1)
	eventbus.Subscribe(bus, ts.Handler())

	eventbus.Publish(context.Background(), bus, eventbus.TestEvent{
		ID:      "123",
		Message: "hello",
	}, "")

	require.True(t, ts.WaitFor(time.Second))

	events := ts.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "123", events[0].ID)
	assert.Equal(t, "hello", events[0].Message)
}

// =============================================================================
// TestSubscriber tests
// =============================================================================

func TestTestSubscriber_NewWithExpectedCount(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](3)
	require.NotNil(t, ts)
	assert.Equal(t, 0, ts.Count())
}

func TestTestSubscriber_NewWithZeroCount(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](0)
	require.NotNil(t, ts)
	assert.Equal(t, 0, ts.Count())
}

func TestTestSubscriber_Handler_CollectsEvents(t *testing.T) {
	bus := eventbus.TestBus()
	defer bus.Close()

	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](2)
	eventbus.Subscribe(bus, ts.Handler())

	eventbus.Publish(context.Background(), bus, eventbus.TestEvent{ID: "1"}, "")
	eventbus.Publish(context.Background(), bus, eventbus.TestEvent{ID: "2"}, "")

	require.True(t, ts.WaitFor(time.Second))

	events := ts.Events()
	assert.Len(t, events, 2)
	assert.Equal(t, "1", events[0].ID)
	assert.Equal(t, "2", events[1].ID)
}

func TestTestSubscriber_Events_ReturnsCopy(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](0)

	// Manually invoke handler
	handler := ts.Handler()
	handler(context.Background(), eventbus.TestEvent{ID: "1"})

	events1 := ts.Events()
	events2 := ts.Events()

	// Should be separate slices
	events1[0] = eventbus.TestEvent{ID: "modified"}
	assert.Equal(t, "1", events2[0].ID)
}

func TestTestSubscriber_WaitFor_ReturnsTrue(t *testing.T) {
	bus := eventbus.TestBus()
	defer bus.Close()

	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](1)
	eventbus.Subscribe(bus, ts.Handler())

	eventbus.Publish(context.Background(), bus, eventbus.TestEvent{ID: "1"}, "")

	result := ts.WaitFor(time.Second)
	assert.True(t, result)
}

func TestTestSubscriber_WaitFor_ReturnsFalseOnTimeout(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](5)
	// No events published

	result := ts.WaitFor(10 * time.Millisecond)
	assert.False(t, result)
}

func TestTestSubscriber_WaitFor_ReturnsFalseWithZeroCount(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](0)

	result := ts.WaitFor(10 * time.Millisecond)
	assert.False(t, result)
}

func TestTestSubscriber_Count(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](0)

	handler := ts.Handler()
	assert.Equal(t, 0, ts.Count())

	handler(context.Background(), eventbus.TestEvent{ID: "1"})
	assert.Equal(t, 1, ts.Count())

	handler(context.Background(), eventbus.TestEvent{ID: "2"})
	assert.Equal(t, 2, ts.Count())
}

func TestTestSubscriber_Reset(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](0)

	handler := ts.Handler()
	handler(context.Background(), eventbus.TestEvent{ID: "1"})
	handler(context.Background(), eventbus.TestEvent{ID: "2"})
	assert.Equal(t, 2, ts.Count())

	ts.Reset(1)
	assert.Equal(t, 0, ts.Count())
}

// =============================================================================
// TestEvent tests
// =============================================================================

func TestTestEvent_EventName(t *testing.T) {
	e := eventbus.TestEvent{ID: "1", Message: "test"}
	assert.Equal(t, "TestEvent", e.EventName())
}

// =============================================================================
// Require* helper tests
// =============================================================================

func TestRequireEventReceived(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](0)
	handler := ts.Handler()
	handler(context.Background(), eventbus.TestEvent{ID: "1"})

	// Should not panic
	eventbus.RequireEventReceived(t, ts)
}

func TestRequireEventCount(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](0)
	handler := ts.Handler()
	handler(context.Background(), eventbus.TestEvent{ID: "1"})
	handler(context.Background(), eventbus.TestEvent{ID: "2"})
	handler(context.Background(), eventbus.TestEvent{ID: "3"})

	// Should not panic
	eventbus.RequireEventCount(t, ts, 3)
}

func TestRequireEventsReceived(t *testing.T) {
	bus := eventbus.TestBus()
	defer bus.Close()

	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](2)
	eventbus.Subscribe(bus, ts.Handler())

	eventbus.Publish(context.Background(), bus, eventbus.TestEvent{ID: "1"}, "")
	eventbus.Publish(context.Background(), bus, eventbus.TestEvent{ID: "2"}, "")

	// Should not panic
	eventbus.RequireEventsReceived(t, ts, time.Second)
}

func TestRequireNoEvents(t *testing.T) {
	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](0)

	// Should not panic - no events received
	eventbus.RequireNoEvents(t, ts)
}

// =============================================================================
// Integration tests
// =============================================================================

func TestTestSubscriber_WithTopicFiltering(t *testing.T) {
	bus := eventbus.TestBus()
	defer bus.Close()

	adminTS := eventbus.NewTestSubscriber[eventbus.TestEvent](1)
	userTS := eventbus.NewTestSubscriber[eventbus.TestEvent](1)

	eventbus.Subscribe(bus, adminTS.Handler(), eventbus.WithTopic("admin"))
	eventbus.Subscribe(bus, userTS.Handler(), eventbus.WithTopic("user"))

	// Publish to admin topic only
	eventbus.Publish(context.Background(), bus, eventbus.TestEvent{ID: "admin-1"}, "admin")

	require.True(t, adminTS.WaitFor(time.Second))
	assert.Equal(t, 1, adminTS.Count())
	assert.Equal(t, 0, userTS.Count())
}

func TestTestSubscriber_ThreadSafety(t *testing.T) {
	bus := eventbus.TestBus()
	defer bus.Close()

	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](100)
	eventbus.Subscribe(bus, ts.Handler())

	// Publish concurrently
	for i := 0; i < 100; i++ {
		go func(id int) {
			eventbus.Publish(context.Background(), bus, eventbus.TestEvent{
				ID: string(rune('0' + id%10)),
			}, "")
		}(i)
	}

	require.True(t, ts.WaitFor(5*time.Second))
	assert.Equal(t, 100, ts.Count())
}
