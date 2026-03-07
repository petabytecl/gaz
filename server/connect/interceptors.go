package connect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sort"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"

	"github.com/petabytecl/gaz/di"
)

// Priority constants for built-in Connect interceptors.
// Custom interceptors should use values between PriorityLogging and PriorityRecovery.
const (
	// PriorityLogging is the priority for the logging interceptor (runs first).
	PriorityLogging = 0
	// PriorityRateLimit is the priority for the rate limit interceptor (after logging, before auth).
	PriorityRateLimit = 25
	// PriorityAuth is the priority for the auth interceptor (after logging, before validation).
	PriorityAuth = 50
	// PriorityValidation is the priority for the validation interceptor.
	PriorityValidation = 100
	// PriorityRecovery is the priority for the recovery interceptor (runs last).
	PriorityRecovery = 1000
)

// InterceptorBundle provides Connect interceptors for auto-discovery.
// Implementations are automatically discovered and chained by the Vanguard server.
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
//	func (m *MetricsBundle) Name() string                      { return "metrics" }
//	func (m *MetricsBundle) Priority() int                     { return 500 }
//	func (m *MetricsBundle) Interceptors() []connect.Interceptor { return []connect.Interceptor{metricsInterceptor} }
//
//	// Register in DI:
//	gaz.For[*MetricsBundle](c).Provider(NewMetricsBundle)
type InterceptorBundle interface {
	// Name returns a unique identifier for logging and debugging.
	Name() string

	// Priority determines the order in the interceptor chain.
	// Lower values run earlier. Built-in interceptors use:
	//   - PriorityLogging (0): logging interceptor
	//   - PriorityRateLimit (25): rate limit interceptor
	//   - PriorityAuth (50): auth interceptor
	//   - PriorityValidation (100): validation interceptor
	//   - PriorityRecovery (1000): recovery interceptor
	// Custom interceptors should use values between 1 and 999.
	Priority() int

	// Interceptors returns Connect interceptors for this bundle.
	// Unlike gRPC bundles which return separate unary and stream interceptors,
	// Connect uses a single Interceptor type that handles both.
	// A bundle may return multiple interceptors.
	Interceptors() []connect.Interceptor
}

// CollectInterceptors discovers all InterceptorBundle implementations from
// the container, sorts them by priority, and returns the flattened interceptor slice.
func CollectInterceptors(container *di.Container, logger *slog.Logger) []connect.Interceptor {
	bundles, err := di.ResolveAll[InterceptorBundle](container)
	if err != nil {
		logger.Warn("failed to resolve connect interceptor bundles", slog.Any("error", err))
		return nil
	}

	// Sort by priority (lower = earlier in chain).
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].Priority() < bundles[j].Priority()
	})

	var interceptors []connect.Interceptor

	for _, b := range bundles {
		interceptors = append(interceptors, b.Interceptors()...)
		logger.Debug("registered connect interceptor bundle",
			slog.String("name", b.Name()),
			slog.Int("priority", b.Priority()),
		)
	}

	return interceptors
}

// AuthFunc validates Connect requests and returns an enriched context.
// Extract credentials from request headers (e.g., Authorization header).
// The function receives HTTP headers and the RPC spec, which are available
// in both unary and streaming handlers (unlike connect.AnyRequest which
// has unexported methods preventing external implementation).
//
// Register in DI to enable auth interceptor:
//
//	gaz.For[AuthFunc](c).Instance(myAuthFunc)
type AuthFunc func(ctx context.Context, header http.Header, spec connect.Spec) (context.Context, error)

// Limiter defines the interface for Connect rate limiting.
// Implementations should return nil to allow the request, or an error to reject it.
// The error should be a *connect.Error with an appropriate code (e.g., connect.CodeResourceExhausted).
// The function receives HTTP headers and the RPC spec for per-procedure or per-client decisions.
//
// Register a custom limiter in DI to override the default AlwaysPassLimiter:
//
//	gaz.For[Limiter](c).Instance(myLimiter)
type Limiter interface {
	Limit(ctx context.Context, header http.Header, spec connect.Spec) error
}

// AlwaysPassLimiter is a no-op limiter that allows all requests.
// This is the default limiter when no custom Limiter is registered in DI.
type AlwaysPassLimiter struct{}

// Limit always returns nil, allowing all requests.
func (l AlwaysPassLimiter) Limit(_ context.Context, _ http.Header, _ connect.Spec) error {
	return nil
}

// --- Logging Bundle ---

// LoggingBundle is the built-in logging interceptor bundle for Connect.
// It logs procedure name, duration, and error status for both unary and streaming RPCs.
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

// Interceptors returns the logging interceptor.
func (b *LoggingBundle) Interceptors() []connect.Interceptor {
	return []connect.Interceptor{&loggingInterceptor{logger: b.logger}}
}

// loggingInterceptor implements connect.Interceptor for request/response logging.
type loggingInterceptor struct {
	logger *slog.Logger
}

// WrapUnary logs procedure name, duration, and error status after the call completes.
func (l *loggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		resp, err := next(ctx, req)

		attrs := []any{
			slog.Duration("duration", time.Since(start)),
			slog.Bool("error", err != nil),
		}
		if req != nil {
			attrs = append(attrs, slog.String("procedure", req.Spec().Procedure))
		}

		l.logger.InfoContext(ctx, "connect rpc", attrs...)

		return resp, err
	}
}

// WrapStreamingClient is a pass-through — server-side only.
func (l *loggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler logs procedure name, duration, and error status after the stream completes.
func (l *loggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		err := next(ctx, conn)

		attrs := []any{
			slog.Duration("duration", time.Since(start)),
			slog.Bool("error", err != nil),
		}
		if conn != nil {
			attrs = append(attrs, slog.String("procedure", conn.Spec().Procedure))
		}

		l.logger.InfoContext(ctx, "connect stream", attrs...)

		return err
	}
}

// --- Recovery Bundle ---

// RecoveryBundle is the built-in panic recovery interceptor bundle for Connect.
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

// Interceptors returns the recovery interceptor.
func (b *RecoveryBundle) Interceptors() []connect.Interceptor {
	return []connect.Interceptor{&recoveryInterceptor{logger: b.logger, devMode: b.devMode}}
}

// recoveryInterceptor implements connect.Interceptor for panic recovery.
type recoveryInterceptor struct {
	logger  *slog.Logger
	devMode bool
}

// WrapUnary wraps the handler with panic recovery.
func (r *recoveryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (_ connect.AnyResponse, err error) {
		defer func() {
			if p := recover(); p != nil {
				r.logger.ErrorContext(ctx, "panic recovered in connect handler",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())),
				)

				if r.devMode {
					err = connect.NewError(connect.CodeInternal, fmt.Errorf("panic: %v", p))
				} else {
					err = connect.NewError(connect.CodeInternal, errors.New("internal server error"))
				}
			}
		}()
		return next(ctx, req)
	}
}

// WrapStreamingClient is a pass-through — server-side only.
func (r *recoveryInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler wraps the streaming handler with panic recovery.
func (r *recoveryInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) (err error) {
		defer func() {
			if p := recover(); p != nil {
				r.logger.ErrorContext(ctx, "panic recovered in connect stream handler",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())),
				)

				if r.devMode {
					err = connect.NewError(connect.CodeInternal, fmt.Errorf("panic: %v", p))
				} else {
					err = connect.NewError(connect.CodeInternal, errors.New("internal server error"))
				}
			}
		}()
		return next(ctx, conn)
	}
}

// --- Auth Bundle ---

// AuthBundle is the built-in authentication interceptor bundle for Connect.
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

// Interceptors returns the auth interceptor.
func (b *AuthBundle) Interceptors() []connect.Interceptor {
	return []connect.Interceptor{&authInterceptor{authFunc: b.authFunc}}
}

// authInterceptor implements connect.Interceptor for authentication.
type authInterceptor struct {
	authFunc AuthFunc
}

// WrapUnary validates the request using the auth function.
func (a *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		newCtx, err := a.authFunc(ctx, req.Header(), req.Spec())
		if err != nil {
			return nil, err
		}
		return next(newCtx, req)
	}
}

// WrapStreamingClient is a pass-through — server-side only.
func (a *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler validates the streaming request using the auth function.
func (a *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		newCtx, err := a.authFunc(ctx, conn.RequestHeader(), conn.Spec())
		if err != nil {
			return err
		}
		return next(newCtx, conn)
	}
}

// --- Rate Limit Bundle ---

// RateLimitBundle is the built-in rate limiting interceptor bundle for Connect.
// It uses the registered Limiter to control request rates.
type RateLimitBundle struct {
	limiter Limiter
}

// NewRateLimitBundle creates a new rate limit interceptor bundle.
// If limiter is nil, AlwaysPassLimiter is used.
func NewRateLimitBundle(limiter Limiter) *RateLimitBundle {
	if limiter == nil {
		limiter = AlwaysPassLimiter{}
	}
	return &RateLimitBundle{limiter: limiter}
}

// Name returns the bundle identifier.
func (b *RateLimitBundle) Name() string {
	return "ratelimit"
}

// Priority returns the rate limit priority (after logging, before auth).
func (b *RateLimitBundle) Priority() int {
	return PriorityRateLimit
}

// Interceptors returns the rate limit interceptor.
func (b *RateLimitBundle) Interceptors() []connect.Interceptor {
	return []connect.Interceptor{&rateLimitInterceptor{limiter: b.limiter}}
}

// rateLimitInterceptor implements connect.Interceptor for rate limiting.
type rateLimitInterceptor struct {
	limiter Limiter
}

// WrapUnary checks the rate limit before processing the request.
func (r *rateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := r.limiter.Limit(ctx, req.Header(), req.Spec()); err != nil {
			return nil, fmt.Errorf("rate limit: %w", err)
		}
		return next(ctx, req)
	}
}

// WrapStreamingClient is a pass-through — server-side only.
func (r *rateLimitInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler checks the rate limit before processing the stream.
func (r *rateLimitInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := r.limiter.Limit(ctx, conn.RequestHeader(), conn.Spec()); err != nil {
			return fmt.Errorf("rate limit: %w", err)
		}
		return next(ctx, conn)
	}
}

// --- Validation Bundle ---

// ValidationBundle is the built-in proto validation interceptor bundle for Connect.
// It validates protobuf messages using connectrpc.com/validate rules.
type ValidationBundle struct {
	interceptor *validate.Interceptor
}

// NewValidationBundle creates a new validation interceptor bundle.
func NewValidationBundle() *ValidationBundle {
	return &ValidationBundle{interceptor: validate.NewInterceptor()}
}

// Name returns the bundle identifier.
func (b *ValidationBundle) Name() string {
	return "validation"
}

// Priority returns the validation priority.
func (b *ValidationBundle) Priority() int {
	return PriorityValidation
}

// Interceptors returns the validation interceptor.
func (b *ValidationBundle) Interceptors() []connect.Interceptor {
	return []connect.Interceptor{b.interceptor}
}
