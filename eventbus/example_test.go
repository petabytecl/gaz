package eventbus_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/eventbus"
)

// UserCreated is a sample event for examples.
type UserCreated struct {
	UserID string
	Email  string
}

// EventName implements eventbus.Event interface.
func (e UserCreated) EventName() string { return "UserCreated" }

// OrderPlaced is another sample event for examples.
type OrderPlaced struct {
	OrderID string
	Amount  float64
}

// EventName implements eventbus.Event interface.
func (e OrderPlaced) EventName() string { return "OrderPlaced" }

// PaymentReceived is a sample event for typed event examples.
type PaymentReceived struct {
	PaymentID string
	Amount    float64
}

// EventName implements eventbus.Event interface.
func (e PaymentReceived) EventName() string { return "PaymentReceived" }

// ExampleNew demonstrates creating a new EventBus.
func ExampleNew() {
	// Create an EventBus with a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Suppress info logs for example
	}))
	bus := eventbus.New(logger)
	defer bus.Close()

	fmt.Println("EventBus created")
	// Output: EventBus created
}

// ExampleSubscribe demonstrates subscribing to events.
func ExampleSubscribe() {
	bus := eventbus.TestBus()
	defer bus.Close()

	// Subscribe to UserCreated events
	sub := eventbus.Subscribe(bus, func(ctx context.Context, event UserCreated) {
		fmt.Printf("User created: %s\n", event.UserID)
	})

	// Publish an event
	eventbus.Publish(context.Background(), bus, UserCreated{
		UserID: "user-123",
		Email:  "test@example.com",
	}, "")

	// Give async handler time to process
	time.Sleep(10 * time.Millisecond)

	// Unsubscribe when done
	sub.Unsubscribe()
	// Output: User created: user-123
}

// ExampleSubscribe_withTopic demonstrates topic-filtered subscriptions.
func ExampleSubscribe_withTopic() {
	bus := eventbus.TestBus()
	defer bus.Close()

	// Subscribe only to "admin" topic
	eventbus.Subscribe(bus, func(ctx context.Context, event UserCreated) {
		fmt.Printf("Admin user: %s\n", event.UserID)
	}, eventbus.WithTopic("admin"))

	// This won't be received (different topic)
	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "regular-user"}, "")

	// This will be received (matching topic)
	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "admin-user"}, "admin")

	time.Sleep(10 * time.Millisecond)
	// Output: Admin user: admin-user
}

// ExampleSubscribe_withBufferSize demonstrates configuring buffer size.
func ExampleSubscribe_withBufferSize() {
	bus := eventbus.TestBus()
	defer bus.Close()

	// High-throughput handler with large buffer
	eventbus.Subscribe(bus, func(ctx context.Context, event OrderPlaced) {
		fmt.Printf("Order: %s\n", event.OrderID)
	}, eventbus.WithBufferSize(1000))

	eventbus.Publish(context.Background(), bus, OrderPlaced{OrderID: "order-1"}, "")

	time.Sleep(10 * time.Millisecond)
	// Output: Order: order-1
}

// ExamplePublish demonstrates publishing events.
func ExamplePublish() {
	bus := eventbus.TestBus()
	defer bus.Close()

	// Subscribe to events
	eventbus.Subscribe(bus, func(ctx context.Context, event UserCreated) {
		fmt.Printf("Received: %s (%s)\n", event.UserID, event.Email)
	})

	// Publish without topic (empty string)
	eventbus.Publish(context.Background(), bus, UserCreated{
		UserID: "user-456",
		Email:  "hello@example.com",
	}, "")

	time.Sleep(10 * time.Millisecond)
	// Output: Received: user-456 (hello@example.com)
}

// ExamplePublish_withTopic demonstrates publishing with a topic.
func ExamplePublish_withTopic() {
	bus := eventbus.TestBus()
	defer bus.Close()

	// Wildcard subscriber (all topics)
	eventbus.Subscribe(bus, func(ctx context.Context, event OrderPlaced) {
		fmt.Printf("All orders: %s\n", event.OrderID)
	})

	// Topic-specific subscriber
	eventbus.Subscribe(bus, func(ctx context.Context, event OrderPlaced) {
		fmt.Printf("Priority orders: %s\n", event.OrderID)
	}, eventbus.WithTopic("priority"))

	// Publish with topic - both handlers receive it
	eventbus.Publish(context.Background(), bus, OrderPlaced{OrderID: "priority-1"}, "priority")

	time.Sleep(10 * time.Millisecond)
	// Unordered output:
	// All orders: priority-1
	// Priority orders: priority-1
}

// ExampleSubscription_Unsubscribe demonstrates unsubscribing from events.
func ExampleSubscription_Unsubscribe() {
	bus := eventbus.TestBus()
	defer bus.Close()

	received := 0
	sub := eventbus.Subscribe(bus, func(ctx context.Context, event UserCreated) {
		received++
	})

	// Publish first event
	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "1"}, "")
	time.Sleep(10 * time.Millisecond)

	// Unsubscribe
	sub.Unsubscribe()

	// This event won't be received
	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "2"}, "")
	time.Sleep(10 * time.Millisecond)

	fmt.Printf("Received: %d events\n", received)
	// Output: Received: 1 events
}

// Example_typedEvents demonstrates using typed event structs.
func Example_typedEvents() {
	bus := eventbus.TestBus()
	defer bus.Close()

	// Subscribe using the example PaymentReceived type
	eventbus.Subscribe(bus, func(ctx context.Context, event PaymentReceived) {
		fmt.Printf("Payment: %.2f\n", event.Amount)
	})

	eventbus.Publish(context.Background(), bus, PaymentReceived{
		PaymentID: "pay-1",
		Amount:    99.99,
	}, "")

	time.Sleep(10 * time.Millisecond)
	// Output: Payment: 99.99
}

// ExampleModule demonstrates using the eventbus module for direct DI usage.
func ExampleModule() {
	// Create a DI container
	c := di.New()

	// Register logger (normally done by gaz.New())
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Suppress info logs for example
	}))
	_ = di.For[*slog.Logger](c).Instance(logger)

	// Apply eventbus module
	if err := eventbus.Module(c); err != nil {
		fmt.Println("error:", err)
		return
	}

	// Build and resolve
	if err := c.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	bus, err := di.Resolve[*eventbus.EventBus](c)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer bus.Close()

	fmt.Printf("EventBus: %T\n", bus)
	// Output: EventBus: *eventbus.EventBus
}

// ExampleTestBus demonstrates using the test bus factory.
func ExampleTestBus() {
	// TestBus creates a bus with a discard logger (no log noise in tests)
	bus := eventbus.TestBus()
	defer bus.Close()

	eventbus.Subscribe(bus, func(ctx context.Context, event UserCreated) {
		fmt.Println("Event received")
	})

	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "test"}, "")

	time.Sleep(10 * time.Millisecond)
	// Output: Event received
}

// ExampleNewTestSubscriber demonstrates using TestSubscriber for async testing.
func ExampleNewTestSubscriber() {
	bus := eventbus.TestBus()
	defer bus.Close()

	// Create subscriber expecting 2 events
	ts := eventbus.NewTestSubscriber[UserCreated](2)
	eventbus.Subscribe(bus, ts.Handler())

	// Publish events
	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "1"}, "")
	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "2"}, "")

	// Wait for events with timeout
	if ts.WaitFor(time.Second) {
		events := ts.Events()
		fmt.Printf("Received %d events\n", len(events))
		fmt.Printf("First user: %s\n", events[0].UserID)
	}
	// Output:
	// Received 2 events
	// First user: 1
}

// ExampleTestSubscriber_WaitFor demonstrates synchronization with WaitFor.
func ExampleTestSubscriber_WaitFor() {
	bus := eventbus.TestBus()
	defer bus.Close()

	ts := eventbus.NewTestSubscriber[OrderPlaced](1)
	eventbus.Subscribe(bus, ts.Handler())

	eventbus.Publish(context.Background(), bus, OrderPlaced{OrderID: "order-1", Amount: 50.0}, "")

	// WaitFor blocks until expected events received or timeout
	received := ts.WaitFor(time.Second)
	fmt.Printf("Received: %v\n", received)
	// Output: Received: true
}

// ExampleTestEvent demonstrates the built-in test event type.
func ExampleTestEvent() {
	bus := eventbus.TestBus()
	defer bus.Close()

	ts := eventbus.NewTestSubscriber[eventbus.TestEvent](1)
	eventbus.Subscribe(bus, ts.Handler())

	// TestEvent is provided for simple test scenarios
	eventbus.Publish(context.Background(), bus, eventbus.TestEvent{
		ID:      "test-1",
		Message: "hello",
	}, "")

	if ts.WaitFor(time.Second) {
		events := ts.Events()
		fmt.Printf("Event: %s - %s\n", events[0].ID, events[0].Message)
	}
	// Output: Event: test-1 - hello
}

// ExampleEventBus_Close demonstrates graceful shutdown.
func ExampleEventBus_Close() {
	bus := eventbus.TestBus()

	eventbus.Subscribe(bus, func(ctx context.Context, event UserCreated) {
		fmt.Println("Handler called")
	})

	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "1"}, "")

	// Close waits for all handlers to finish processing
	bus.Close()

	// After close, publish is a no-op
	eventbus.Publish(context.Background(), bus, UserCreated{UserID: "2"}, "")

	fmt.Println("Bus closed")
	// Output:
	// Handler called
	// Bus closed
}
