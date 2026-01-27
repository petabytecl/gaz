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
}

func (m *mockHandler) Enabled(context.Context, slog.Level) bool { return true }
func (m *mockHandler) Handle(_ context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		m.attrs = append(m.attrs, a)
		return true
	})
	return nil
}
func (m *mockHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return m }
func (m *mockHandler) WithGroup(name string) slog.Handler       { return m }

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
