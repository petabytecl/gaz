package eventbus

import (
	"context"
	"fmt"
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

// eventEnvelope wraps an event with its publisher's context for propagation.
type eventEnvelope struct {
	ctx   context.Context //nolint:containedctx // Envelope carries publisher context through channel.
	event any
}

// asyncSubscription holds a subscription's channel and handler.
type asyncSubscription struct {
	id        uint64
	ch        chan eventEnvelope         // Buffered channel for events with context
	done      chan struct{}              // Closed when handler goroutine exits
	tombstone chan struct{}              // Closed when subscription is cancelled; senders short-circuit
	handler   func(context.Context, any) // Type-erased handler
}

// run processes events from the channel until the tombstone fires or the channel is drained.
func (s *asyncSubscription) run(logger *slog.Logger) {
	defer close(s.done)
	for {
		select {
		case env, ok := <-s.ch:
			if !ok {
				return // Channel closed, drain complete
			}
			s.safeInvoke(env.ctx, env.event, logger)
		case <-s.tombstone:
			// Subscription cancelled. Drain remaining buffered events then exit.
			for {
				select {
				case env, ok := <-s.ch:
					if !ok {
						return
					}
					s.safeInvoke(env.ctx, env.event, logger)
				default:
					return
				}
			}
		}
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
		id:        id,
		ch:        make(chan eventEnvelope, options.bufferSize),
		done:      make(chan struct{}),
		tombstone: make(chan struct{}),
		handler: func(ctx context.Context, event any) {
			//nolint:errcheck // Type is guaranteed by generic Subscribe[T]
			handler(ctx, event.(T))
		},
	}

	// Start handler goroutine
	go sub.run(b.logger)

	b.handlers[key] = append(b.handlers[key], sub)

	return newSubscription(id, eventType, options.topic, b)
}

// Publish sends an event to all matching subscribers.
//
// Matching includes:
//   - Subscribers for exact (type, topic) pair
//   - Subscribers for (type, "") wildcard (subscribed to all topics)
//
// Publish returns immediately (fire-and-forget). Events are queued
// in each subscriber's buffer. Blocks if any subscriber's buffer is full.
//
// Publishing to a closed bus is a silent no-op (idempotent).
//
// # Example
//
//	eventbus.Publish(ctx, bus, UserCreated{UserID: "123"}, "")
//	eventbus.Publish(ctx, bus, UserCreated{UserID: "456"}, "admin")
func Publish[T Event](ctx context.Context, b *EventBus, event T, topic string) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return // Silent no-op
	}

	eventType := reflect.TypeOf(event)

	// Snapshot handlers under lock
	var handlers []*asyncSubscription

	// Exact topic match
	exactKey := subscriptionKey{eventType: eventType, topic: topic}
	handlers = append(handlers, b.handlers[exactKey]...)

	// Wildcard match (empty topic = all topics)
	if topic != "" {
		wildcardKey := subscriptionKey{eventType: eventType, topic: ""}
		handlers = append(handlers, b.handlers[wildcardKey]...)
	}
	b.mu.RUnlock()

	// Deliver OUTSIDE lock — no deadlock possible.
	// Tombstone channels prevent send-on-closed-channel panics: if a subscription
	// is cancelled while we are iterating, the tombstone fires and we skip it.
	env := eventEnvelope{ctx: ctx, event: event}
	for _, h := range handlers {
		select {
		case h.ch <- env:
			// Delivered
		case <-h.tombstone:
			// Subscription cancelled during send — skip
		case <-ctx.Done():
			return // Context cancelled, stop publishing
		}
	}
}

// Name implements worker.Worker interface.
func (b *EventBus) Name() string {
	return "eventbus.EventBus"
}

// OnStart implements worker.Worker interface.
//
// EventBus is always ready - no initialization needed beyond New().
// This method logs that the eventbus has started and returns nil.
func (b *EventBus) OnStart(ctx context.Context) error {
	b.logger.InfoContext(ctx, "eventbus started")
	return nil
}

// OnStop implements worker.Worker interface.
//
// Delegates to CloseWithContext so that in-flight handler draining respects
// the shutdown deadline. Returns ctx.Err() if the deadline is exceeded before
// all handlers finish.
func (b *EventBus) OnStop(ctx context.Context) error {
	return b.CloseWithContext(ctx)
}

// Close shuts down the EventBus and waits for in-flight handlers.
//
// After Close, Publish is a no-op and Subscribe returns nil.
// Safe to call multiple times (idempotent).
//
// Close waits indefinitely for all handler goroutines to finish processing
// their buffered events. Use CloseWithContext to bound the wait with a
// shutdown deadline.
func (b *EventBus) Close() {
	_ = b.CloseWithContext(context.Background())
}

// CloseWithContext shuts down the EventBus and waits for in-flight handlers,
// respecting the provided context deadline.
//
// After CloseWithContext, Publish is a no-op and Subscribe returns nil.
// Safe to call multiple times (idempotent).
//
// Returns ctx.Err() if the deadline expires before all handlers finish draining.
func (b *EventBus) CloseWithContext(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true

	// Collect all subscriptions
	var allSubs []*asyncSubscription
	for _, subs := range b.handlers {
		allSubs = append(allSubs, subs...)
	}

	// Signal tombstones — in-flight Publish senders (which released their
	// RLock before sending) will short-circuit via the tombstone select case.
	// Handler goroutines also watch tombstone and drain remaining buffered events.
	// We do NOT close ch here because a Publish goroutine that already captured
	// a handler reference might attempt a send-on-closed-channel panic.
	for _, sub := range allSubs {
		close(sub.tombstone)
	}
	b.mu.Unlock()

	// Wait for handler goroutines to drain and exit (outside lock)
	for _, sub := range allSubs {
		select {
		case <-sub.done:
			// Drained successfully
		case <-ctx.Done():
			b.logger.WarnContext(ctx, "shutdown deadline exceeded, some handlers not drained",
				"remaining", len(allSubs))
			return fmt.Errorf("eventbus close: %w", ctx.Err())
		}
	}

	b.logger.InfoContext(ctx, "eventbus stopped", "subscriptions_drained", len(allSubs))
	return nil
}

// unsubscribe removes a subscription from the bus.
//
// This is called by Subscription.Unsubscribe() via the unsubscriber interface.
// It closes the subscription's channel, waits for the handler goroutine to exit,
// and removes the subscription from the handlers map.
func (b *EventBus) unsubscribe(eventType reflect.Type, topic string, id uint64) {
	b.mu.Lock()

	if b.closed {
		b.mu.Unlock()
		return
	}

	key := subscriptionKey{eventType: eventType, topic: topic}
	subs := b.handlers[key]

	var target *asyncSubscription
	for i, sub := range subs {
		if sub.id == id {
			target = sub
			// Remove from map under lock
			b.handlers[key] = append(subs[:i], subs[i+1:]...)
			if len(b.handlers[key]) == 0 {
				delete(b.handlers, key)
			}
			break
		}
	}

	if target == nil {
		b.mu.Unlock()
		return
	}

	// Signal tombstone under lock so handler goroutine drains and exits.
	// We do NOT close target.ch — a concurrent Publish with a snapshot reference
	// might still attempt a send. The handler exits via tombstone drain.
	close(target.tombstone)
	b.mu.Unlock()

	// Wait for handler goroutine outside lock
	<-target.done
}
