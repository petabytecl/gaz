package grpc

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InterceptorLogger adapts slog.Logger to the go-grpc-middleware logging.Logger interface.
// This allows the gRPC logging interceptor to use gaz's standard slog-based logger.
func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

// NewLoggingInterceptor creates logging interceptors for gRPC requests.
// The interceptors log request start, completion, duration, and status.
//
// Returns both unary and stream server interceptors.
func NewLoggingInterceptor(logger *slog.Logger) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	loggerAdapter := InterceptorLogger(logger)

	opts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}

	return logging.UnaryServerInterceptor(loggerAdapter, opts...),
		logging.StreamServerInterceptor(loggerAdapter, opts...)
}

// NewRecoveryInterceptor creates panic recovery interceptors for gRPC handlers.
// When a panic occurs:
//   - Full stack trace is logged to the provided logger
//   - In dev mode, panic details are returned in the error message
//   - In production mode, a generic "internal server error" is returned
//
// Returns both unary and stream server interceptors.
func NewRecoveryInterceptor(logger *slog.Logger, devMode bool) (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	recoveryHandler := recovery.WithRecoveryHandlerContext(
		func(ctx context.Context, p any) error {
			// Log full stack trace.
			logger.ErrorContext(ctx, "panic recovered in gRPC handler",
				slog.Any("panic", p),
				slog.String("stack", string(debug.Stack())),
			)

			// Return error details only in dev mode.
			if devMode {
				return status.Errorf(codes.Internal, "panic: %v", p)
			}
			return status.Error(codes.Internal, "internal server error")
		},
	)

	return recovery.UnaryServerInterceptor(recoveryHandler),
		recovery.StreamServerInterceptor(recoveryHandler)
}
