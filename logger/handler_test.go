package logger

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHandler struct {
	attrs []slog.Attr
	group string
}

func (m *mockHandler) Enabled(context.Context, slog.Level) bool { return true }
func (m *mockHandler) Handle(_ context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		m.attrs = append(m.attrs, a)
		return true
	})
	return nil
}

func (m *mockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Return a new mock with the attrs set to prove delegation happened
	newMock := &mockHandler{attrs: make([]slog.Attr, len(attrs))}
	copy(newMock.attrs, attrs)
	return newMock
}

func (m *mockHandler) WithGroup(name string) slog.Handler {
	return &mockHandler{group: name}
}

func TestContextHandler_Handle(t *testing.T) {
	mock := &mockHandler{}
	handler := NewContextHandler(mock)

	ctx := context.Background()
	ctx = WithTraceID(ctx, "test-trace-id")
	ctx = WithRequestID(ctx, "test-request-id")

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := handler.Handle(ctx, record)
	require.NoError(t, err)

	attrs := make(map[string]string)
	for _, a := range mock.attrs {
		attrs[a.Key] = a.Value.String()
	}

	assert.Equal(t, "test-trace-id", attrs[TraceIDKey])
	assert.Equal(t, "test-request-id", attrs[RequestIDKey])
}

func TestContextHandler_WithAttrs(t *testing.T) {
	mock := &mockHandler{}
	handler := NewContextHandler(mock)

	// WithAttrs should return a *ContextHandler wrapping the delegated handler
	result := handler.WithAttrs([]slog.Attr{slog.String("key", "value")})

	ctxHandler, ok := result.(*ContextHandler)
	require.True(t, ok, "WithAttrs should return *ContextHandler")

	// The inner handler should be the result of mock.WithAttrs(), not the original mock
	innerMock, ok := ctxHandler.Handler.(*mockHandler)
	require.True(t, ok, "inner handler should be *mockHandler")
	require.Len(t, innerMock.attrs, 1)
	assert.Equal(t, "key", innerMock.attrs[0].Key)
}

func TestContextHandler_WithGroup(t *testing.T) {
	mock := &mockHandler{}
	handler := NewContextHandler(mock)

	// WithGroup should return a *ContextHandler wrapping the delegated handler
	result := handler.WithGroup("mygroup")

	ctxHandler, ok := result.(*ContextHandler)
	require.True(t, ok, "WithGroup should return *ContextHandler")

	// The inner handler should be the result of mock.WithGroup()
	innerMock, ok := ctxHandler.Handler.(*mockHandler)
	require.True(t, ok, "inner handler should be *mockHandler")
	assert.Equal(t, "mygroup", innerMock.group)
}

func TestContextHandler_WithAttrs_PreservesContextPropagation(t *testing.T) {
	mock := &mockHandler{}
	handler := NewContextHandler(mock)

	// Chain: add attrs via WithAttrs, then handle with context values
	withAttrs := handler.WithAttrs([]slog.Attr{slog.String("extra", "data")})

	ctx := context.Background()
	ctx = WithTraceID(ctx, "trace-123")
	ctx = WithRequestID(ctx, "req-456")

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	err := withAttrs.Handle(ctx, record)
	require.NoError(t, err)

	// The inner mock should have received the context attrs via Handle
	innerMock := withAttrs.(*ContextHandler).Handler.(*mockHandler)
	attrs := make(map[string]string)
	for _, a := range innerMock.attrs {
		attrs[a.Key] = a.Value.String()
	}

	// Context propagation should still work
	assert.Equal(t, "trace-123", attrs[TraceIDKey])
	assert.Equal(t, "req-456", attrs[RequestIDKey])
}
