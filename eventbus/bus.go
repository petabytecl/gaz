package eventbus

import (
	"context"
	"log/slog"
	"reflect"
	"runtime/debug"
	"sync"
)

// subscriptionKey uniquely identifies a subscription target.
type subscriptionKey struct {
	eventType reflect.Type
	topic     string // Empty = wildcard (all topics)
}

// asyncSubscription holds a subscription's channel and handler.
type asyncSubscription struct {
	id      uint64
	ch      chan any                   // Buffered channel for events
	done    chan struct{}              // Closed when handler goroutine exits
	handler func(context.Context, any) // Type-erased handler
}

// run processes events from the channel until it's closed.
func (s *asyncSubscription) run(logger *slog.Logger) {
	defer close(s.done)
	for event := range s.ch {
		s.safeInvoke(context.Background(), event, logger)
	}
}

// safeInvoke calls the handler with panic recovery.
func (s *asyncSubscription) safeInvoke(ctx context.Context, event any, logger *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("handler panic recovered",
				"error", r,
				"stack", string(debug.Stack()),
			)
		}
	}()
	s.handler(ctx, event)
}

// EventBus provides type-safe in-process pub/sub.
//
// EventBus routes events to subscribers based on Go types. It supports
// optional topic filtering for additional routing granularity.
//
// All event delivery is asynchronous - Publish returns immediately after
// queueing events. Each subscriber has its own buffered channel for delivery.
// When a buffer is full, Publish blocks (backpressure).
//
// EventBus implements worker.Worker for integration with gaz's lifecycle
// system. Call Close() to drain in-flight handlers before shutdown.
//
// # Thread Safety
//
// EventBus is safe for concurrent use. Multiple goroutines can subscribe,
// unsubscribe, and publish concurrently.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[subscriptionKey][]*asyncSubscription
	nextID   uint64
	closed   bool
	logger   *slog.Logger
}

// New creates a new EventBus.
//
// The logger is used for panic recovery logging. Pass slog.Default() if
// you don't have a custom logger.
func New(logger *slog.Logger) *EventBus {
	return &EventBus{
		handlers: make(map[subscriptionKey][]*asyncSubscription),
		logger:   logger.With("component", "eventbus.EventBus"),
	}
}

// Subscribe registers a handler for events of type T.
//
// Returns a Subscription that can be used to unsubscribe. The subscription
// starts receiving events immediately after Subscribe returns.
//
// If the bus is closed, Subscribe returns nil.
//
// Options:
//   - [WithTopic]: Filter to events with matching topic
//   - [WithBufferSize]: Configure async buffer size (default 100)
//
// # Example
//
//	sub := eventbus.Subscribe[UserCreated](bus, func(ctx context.Context, event UserCreated) {
//	    log.Printf("User created: %s", event.UserID)
//	})
//	defer sub.Unsubscribe()
func Subscribe[T Event](b *EventBus, handler Handler[T], opts ...SubscribeOption) *Subscription {
	options := applyOptions(opts)

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil // Can't subscribe to closed bus
	}

	eventType := reflect.TypeOf((*T)(nil)).Elem()
	key := subscriptionKey{eventType: eventType, topic: options.topic}

	b.nextID++
	id := b.nextID

	// Create async subscription with per-subscriber buffer
	sub := &asyncSubscription{
		id:   id,
		ch:   make(chan any, options.bufferSize),
		done: make(chan struct{}),
		handler: func(ctx context.Context, event any) {
			handler(ctx, event.(T))
		},
	}

	// Start handler goroutine
	go sub.run(b.logger)

	b.handlers[key] = append(b.handlers[key], sub)

	return newSubscription(id, eventType, options.topic, b)
}

// unsubscribe removes a subscription from the bus.
//
// This is called by Subscription.Unsubscribe() via the unsubscriber interface.
// It closes the subscription's channel, waits for the handler goroutine to exit,
// and removes the subscription from the handlers map.
func (b *EventBus) unsubscribe(eventType reflect.Type, topic string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := subscriptionKey{eventType: eventType, topic: topic}
	subs := b.handlers[key]

	for i, sub := range subs {
		if sub.id == id {
			close(sub.ch) // Signal handler to exit
			<-sub.done    // Wait for handler to finish
			b.handlers[key] = append(subs[:i], subs[i+1:]...)
			if len(b.handlers[key]) == 0 {
				delete(b.handlers, key)
			}
			return
		}
	}
}
