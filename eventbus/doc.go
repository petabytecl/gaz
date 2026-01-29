// Package eventbus provides type-safe in-process pub/sub for Go applications.
//
// This package implements a generic event bus that enables loosely coupled
// communication between application components. Events are published to the bus
// and delivered asynchronously to all matching subscribers.
//
// # Core Concepts
//
// Events must implement the [Event] interface, which requires an EventName() method
// for logging and debugging. Subscribers receive events via [Handler] functions
// that accept context and the strongly-typed event.
//
// # Type Safety
//
// The eventbus uses Go generics for type-safe subscriptions:
//
//   - Subscribe[T] ensures handlers only receive events of type T
//   - Publish[T] ensures only valid events can be published
//   - No runtime type assertions needed in handlers
//
// # Async Fire-and-Forget
//
// All event delivery is asynchronous by default. Publish returns immediately
// after queueing the event. Handlers run concurrently in separate goroutines
// and do not return errors - they are fire-and-forget. Handlers should log
// errors internally if needed.
//
// # Buffer Configuration
//
// Each subscription has a configurable buffer for async delivery. When the
// buffer is full, Publish blocks (backpressure). The default buffer size is 100.
// Configure per subscription with [WithBufferSize].
//
// # Topic Filtering
//
// Events can be published with an optional topic string. Subscribers can filter
// to receive only events matching a specific topic using [WithTopic]. Omitting
// the topic option subscribes to all events of that type.
//
// # Lifecycle Integration
//
// The [EventBus] implements worker.Worker for integration with gaz's lifecycle
// system. It starts automatically with app.Run() and stops gracefully on shutdown,
// draining in-flight events before returning.
//
// # Usage Example
//
//	// Define an event
//	type UserCreated struct {
//	    UserID string
//	}
//
//	func (e UserCreated) EventName() string { return "UserCreated" }
//
//	// Subscribe to events
//	sub := eventbus.Subscribe[UserCreated](bus, func(ctx context.Context, event UserCreated) {
//	    log.Printf("User created: %s", event.UserID)
//	})
//
//	// Publish an event
//	eventbus.Publish(bus, UserCreated{UserID: "123"})
//
//	// Unsubscribe when done
//	sub.Unsubscribe()
//
// # Subscription Management
//
// Subscribe returns a [Subscription] handle that can be used to unsubscribe.
// Calling Unsubscribe() is safe to call multiple times (idempotent).
// When the bus stops, all subscriptions are automatically cleaned up.
package eventbus
