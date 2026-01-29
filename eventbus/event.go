package eventbus

import "context"

// Event is the interface all publishable events must implement.
//
// This provides type-safe routing and logging/debugging support.
// Every event type in your application should implement this interface.
//
// # Convention
//
// EventName() should return the struct type name (e.g., "UserCreated").
// This name is used for logging, debugging, and metrics. It does not
// affect routing - routing is based on the Go type itself.
//
// # Example
//
//	type UserCreated struct {
//	    UserID    string
//	    Email     string
//	    CreatedAt time.Time
//	}
//
//	func (e UserCreated) EventName() string { return "UserCreated" }
type Event interface {
	// EventName returns a string identifier for logging and debugging.
	//
	// Convention: Use the struct type name (e.g., "UserCreated", "OrderPlaced").
	// This is for observability only - event routing uses the Go type.
	EventName() string
}

// Handler is a function that handles events of type T.
//
// Handlers receive context for cancellation awareness and the strongly-typed
// event. They do not return errors - handlers are fire-and-forget.
//
// # Error Handling
//
// If a handler needs to report errors, it should log them internally or
// publish a failure event. The eventbus recovers from panics and continues
// delivering to other subscribers.
//
// # Context
//
// The context passed to handlers is derived from the Publish call. Handlers
// should respect context cancellation for graceful shutdown. Long-running
// handlers should check ctx.Done() periodically.
//
// # Example
//
//	handler := func(ctx context.Context, event UserCreated) {
//	    if err := sendWelcomeEmail(ctx, event.Email); err != nil {
//	        slog.Error("failed to send welcome email",
//	            "error", err,
//	            "user_id", event.UserID,
//	        )
//	    }
//	}
type Handler[T Event] func(ctx context.Context, event T)
