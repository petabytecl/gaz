package logger

import (
	"context"
	"testing"
)

func TestContextHelpers(t *testing.T) {
	t.Run("TraceID", func(t *testing.T) {
		ctx := context.Background()
		traceID := "test-trace-id"

		ctx = WithTraceID(ctx, traceID)
		if got := GetTraceID(ctx); got != traceID {
			t.Errorf("GetTraceID() = %v, want %v", got, traceID)
		}
	})

	t.Run("RequestID", func(t *testing.T) {
		ctx := context.Background()
		requestID := "test-request-id"

		ctx = WithRequestID(ctx, requestID)
		if got := GetRequestID(ctx); got != requestID {
			t.Errorf("GetRequestID() = %v, want %v", got, requestID)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		ctx := context.Background()
		if got := GetTraceID(ctx); got != "" {
			t.Errorf("GetTraceID() = %v, want empty string", got)
		}
		if got := GetRequestID(ctx); got != "" {
			t.Errorf("GetRequestID() = %v, want empty string", got)
		}
	})
}
