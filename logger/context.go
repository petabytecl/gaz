package logger

import "context"

// ctxKey is an unexported type for context keys to prevent collisions.
type ctxKey string

// Context keys for internal storage.
const (
	ctxKeyTraceID   ctxKey = "trace_id"
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeySpanID    ctxKey = "span_id"
)

// Log attribute keys.
const (
	// TraceIDKey is the log attribute key for trace ID.
	TraceIDKey = "trace_id"
	// RequestIDKey is the log attribute key for request ID.
	RequestIDKey = "request_id"
	// SpanIDKey is the log attribute key for span ID.
	SpanIDKey = "span_id"
)

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID, traceID)
}

// GetTraceID extracts the trace ID from the context.
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTraceID).(string); ok {
		return v
	}
	return ""
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, requestID)
}

// GetRequestID extracts the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}
