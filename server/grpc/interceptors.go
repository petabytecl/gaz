package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"

	"buf.build/go/protovalidate"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	protovalidateinterceptor "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/petabytecl/gaz/di"
)

// Priority constants for built-in interceptors.
// Custom interceptors should use values between PriorityLogging and PriorityRecovery.
const (
	// PriorityLogging is the priority for the logging interceptor (runs first).
	PriorityLogging = 0
	// PriorityAuth is the priority for the auth interceptor (after logging, before validation).
	PriorityAuth = 50
	// PriorityValidation is the priority for the validation interceptor.
	PriorityValidation = 100
	// PriorityRecovery is the priority for the recovery interceptor (runs last).
	PriorityRecovery = 1000
)

// InterceptorBundle provides gRPC interceptors for auto-discovery.
// Implementations are automatically discovered and chained by the gRPC server.
//
// To add custom interceptors:
//  1. Implement this interface
//  2. Register the implementation in the DI container
//  3. The server will auto-discover and chain it based on Priority()
//
// Example:
//
//	type MetricsBundle struct{}
//
//	func (m *MetricsBundle) Name() string { return "metrics" }
//	func (m *MetricsBundle) Priority() int { return 50 }
//	func (m *MetricsBundle) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
//	    return metricsUnary, metricsStream
//	}
//
//	// Register in DI:
//	gaz.For[*MetricsBundle](c).Provider(NewMetricsBundle)
type InterceptorBundle interface {
	// Name returns a unique identifier for logging and debugging.
	Name() string

	// Priority determines the order in the interceptor chain.
	// Lower values run earlier. Built-in interceptors use:
	//   - PriorityLogging (0): logging interceptor
	//   - PriorityRecovery (1000): recovery interceptor
	// Custom interceptors should use values between 1 and 999.
	Priority() int

	// Interceptors returns unary and stream server interceptors.
	// Either may be nil if the bundle only provides one type.
	Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor)
}

// collectInterceptors discovers all InterceptorBundles from the container,
// sorts them by priority, and returns the chained interceptors.
func collectInterceptors(container *di.Container, logger *slog.Logger) ([]grpc.UnaryServerInterceptor, []grpc.StreamServerInterceptor) {
	bundles, err := di.ResolveAll[InterceptorBundle](container)
	if err != nil {
		logger.Warn("failed to resolve interceptor bundles", slog.Any("error", err))
		return nil, nil
	}

	// Sort by priority (lower = earlier in chain).
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].Priority() < bundles[j].Priority()
	})

	var unary []grpc.UnaryServerInterceptor
	var stream []grpc.StreamServerInterceptor

	for _, b := range bundles {
		u, s := b.Interceptors()
		if u != nil {
			unary = append(unary, u)
		}
		if s != nil {
			stream = append(stream, s)
		}
		logger.Debug("registered interceptor bundle",
			slog.String("name", b.Name()),
			slog.Int("priority", b.Priority()),
		)
	}

	return unary, stream
}

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

// LoggingBundle is the built-in logging interceptor bundle.
// It logs request start, completion, duration, and status.
type LoggingBundle struct {
	logger *slog.Logger
}

// NewLoggingBundle creates a new logging interceptor bundle.
func NewLoggingBundle(logger *slog.Logger) *LoggingBundle {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggingBundle{logger: logger}
}

// Name returns the bundle identifier.
func (b *LoggingBundle) Name() string {
	return "logging"
}

// Priority returns the logging priority (runs first).
func (b *LoggingBundle) Priority() int {
	return PriorityLogging
}

// Interceptors returns the logging interceptors.
func (b *LoggingBundle) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return NewLoggingInterceptor(b.logger)
}

// AuthFunc is the authentication function type.
// It extracts and validates credentials from the context, returning an enriched context
// or an error if authentication fails.
//
// Use auth.AuthFromMD to extract tokens from metadata:
//
//	func myAuthFunc(ctx context.Context) (context.Context, error) {
//	    token, err := auth.AuthFromMD(ctx, "bearer")
//	    if err != nil {
//	        return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
//	    }
//	    // Validate token and enrich context...
//	    return ctx, nil
//	}
//
// Register in DI to enable auth interceptor:
//
//	gaz.For[grpc.AuthFunc](c).Instance(myAuthFunc)
type AuthFunc = auth.AuthFunc

// AuthBundle is the built-in authentication interceptor bundle.
// It validates requests using the registered AuthFunc.
type AuthBundle struct {
	authFunc AuthFunc
}

// NewAuthBundle creates a new auth interceptor bundle.
func NewAuthBundle(authFunc AuthFunc) *AuthBundle {
	return &AuthBundle{authFunc: authFunc}
}

// Name returns the bundle identifier.
func (b *AuthBundle) Name() string {
	return "auth"
}

// Priority returns the auth priority (after logging, before validation).
func (b *AuthBundle) Priority() int {
	return PriorityAuth
}

// Interceptors returns the auth interceptors.
func (b *AuthBundle) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return auth.UnaryServerInterceptor(b.authFunc),
		auth.StreamServerInterceptor(b.authFunc)
}

// RecoveryBundle is the built-in panic recovery interceptor bundle.
// It catches panics and returns appropriate error responses.
type RecoveryBundle struct {
	logger  *slog.Logger
	devMode bool
}

// NewRecoveryBundle creates a new recovery interceptor bundle.
func NewRecoveryBundle(logger *slog.Logger, devMode bool) *RecoveryBundle {
	if logger == nil {
		logger = slog.Default()
	}
	return &RecoveryBundle{logger: logger, devMode: devMode}
}

// Name returns the bundle identifier.
func (b *RecoveryBundle) Name() string {
	return "recovery"
}

// Priority returns the recovery priority (runs last).
func (b *RecoveryBundle) Priority() int {
	return PriorityRecovery
}

// Interceptors returns the recovery interceptors.
func (b *RecoveryBundle) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return NewRecoveryInterceptor(b.logger, b.devMode)
}

// ValidationBundle is the built-in protovalidate interceptor bundle.
// It validates protobuf messages using buf.build/go/protovalidate rules.
type ValidationBundle struct {
	validator protovalidate.Validator
}

// NewValidationBundle creates a new validation interceptor bundle.
// Returns an error if the validator cannot be created.
func NewValidationBundle() (*ValidationBundle, error) {
	v, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("create protovalidate validator: %w", err)
	}
	return &ValidationBundle{validator: v}, nil
}

// Name returns the bundle identifier.
func (b *ValidationBundle) Name() string {
	return "validation"
}

// Priority returns the validation priority.
func (b *ValidationBundle) Priority() int {
	return PriorityValidation
}

// Interceptors returns the validation interceptors.
func (b *ValidationBundle) Interceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	return protovalidateinterceptor.UnaryServerInterceptor(b.validator),
		protovalidateinterceptor.StreamServerInterceptor(b.validator)
}
