package logger

import (
	"context"
	"log/slog"
)

// ContextHandler propagates context values to log records.
// It wraps an underlying slog.Handler.
type ContextHandler struct {
	slog.Handler
}

// NewContextHandler returns a new ContextHandler wrapping the provided handler.
func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: h}
}

// Handle adds context values to the record before delegating to the embedded handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if traceID := GetTraceID(ctx); traceID != "" {
			r.AddAttrs(slog.String(TraceIDKey, traceID))
		}
		if requestID := GetRequestID(ctx); requestID != "" {
			r.AddAttrs(slog.String(RequestIDKey, requestID))
		}
	}

	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a new ContextHandler wrapping the result of calling
// WithAttrs on the underlying handler. This preserves context propagation
// while properly chaining attributes.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup returns a new ContextHandler wrapping the result of calling
// WithGroup on the underlying handler. This preserves context propagation
// while properly supporting grouped log attributes.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}
