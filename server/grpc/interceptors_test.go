package grpc

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewRecoveryInterceptor(t *testing.T) {
	logger := slog.Default()

	t.Run("recovers from panic in production mode", func(t *testing.T) {
		unary, _ := NewRecoveryInterceptor(logger, false)

		handler := func(_ context.Context, _ any) (any, error) {
			panic("test panic")
		}

		_, err := unary(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
		require.Error(t, err)

		// In production mode, error message should be generic.
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, st.Code())
		require.Equal(t, "internal server error", st.Message())
	})

	t.Run("recovers from panic in dev mode", func(t *testing.T) {
		unary, _ := NewRecoveryInterceptor(logger, true)

		handler := func(_ context.Context, _ any) (any, error) {
			panic("test panic in dev mode")
		}

		_, err := unary(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
		require.Error(t, err)

		// In dev mode, error message should contain panic details.
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, st.Code())
		require.Contains(t, st.Message(), "test panic in dev mode")
	})
}

func TestInterceptorLogger(t *testing.T) {
	logger := slog.Default()
	adapted := InterceptorLogger(logger)
	require.NotNil(t, adapted)

	// Just verify it doesn't panic when called.
	adapted.Log(context.Background(), 0, "test message", "key", "value")
}
