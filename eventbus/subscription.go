package eventbus

import "reflect"

// Subscription represents an active event subscription.
//
// Each subscription has a unique ID and tracks the event type and optional
// topic filter. Use [Unsubscribe] to remove the subscription from the EventBus.
//
// Subscriptions are returned by Subscribe and should be stored if you need
// to unsubscribe later. For subscriptions that should last the lifetime of
// the application, you can ignore the return value.
type Subscription struct {
	id        uint64       // Atomic counter ID, not UUID
	eventType reflect.Type // Type key for handler lookup
	topic     string       // Optional topic filter
	bus       unsubscriber // Interface to avoid circular dependency
}

// unsubscriber is an internal interface for the EventBus unsubscribe method.
//
// This interface breaks the circular dependency between Subscription and
// EventBus. The EventBus implements this interface, allowing Subscription
// to call back into the bus without importing the full EventBus type.
type unsubscriber interface {
	unsubscribe(eventType reflect.Type, topic string, id uint64)
}

// Unsubscribe removes this subscription from the EventBus.
//
// After calling Unsubscribe, the handler will no longer receive events.
// Any events already queued for this handler will still be delivered.
//
// Safe to call multiple times (idempotent). Calling Unsubscribe on a
// subscription that was already unsubscribed or on a closed bus is a no-op.
// Calling Unsubscribe on a nil Subscription is also a no-op.
func (s *Subscription) Unsubscribe() {
	if s == nil || s.bus == nil {
		return
	}
	s.bus.unsubscribe(s.eventType, s.topic, s.id)
}

// newSubscription creates a new Subscription with the given parameters.
//
// This is an internal constructor used by the EventBus Subscribe method.
func newSubscription(id uint64, eventType reflect.Type, topic string, bus unsubscriber) *Subscription {
	return &Subscription{
		id:        id,
		eventType: eventType,
		topic:     topic,
		bus:       bus,
	}
}
