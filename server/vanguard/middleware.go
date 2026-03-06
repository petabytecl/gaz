package vanguard

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz/di"
	connectpkg "github.com/petabytecl/gaz/server/connect"
)

// Transport middleware priority constants.
// Lower values run first (outermost in the handler chain).
const (
	// PriorityCORS is the priority for the CORS middleware (runs first).
	PriorityCORS = 0
	// PriorityOTEL is the priority for the OTEL middleware (runs after CORS).
	PriorityOTEL = 100
)

// TransportMiddleware wraps an http.Handler with cross-cutting HTTP concerns.
// Implementations are automatically discovered from the DI container and
// applied in priority order around the Vanguard handler.
type TransportMiddleware interface {
	// Name returns a unique identifier for logging and debugging.
	Name() string

	// Priority determines the order in the middleware chain.
	// Lower values wrap outermost (run first on request, last on response).
	Priority() int

	// Wrap applies the middleware to the given handler.
	Wrap(http.Handler) http.Handler
}

// collectTransportMiddleware discovers all TransportMiddleware from the DI container,
// sorts them by priority, and wraps the handler in reverse order so that the
// lowest-priority middleware is outermost.
func collectTransportMiddleware(container *di.Container, logger *slog.Logger, handler http.Handler) http.Handler {
	middlewares, err := di.ResolveAll[TransportMiddleware](container)
	if err != nil {
		logger.Warn("failed to resolve transport middleware", slog.Any("error", err))
		return handler
	}

	// Sort by priority ascending.
	sort.Slice(middlewares, func(i, j int) bool {
		return middlewares[i].Priority() < middlewares[j].Priority()
	})

	// Apply in reverse order so lowest priority wraps outermost.
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i].Wrap(handler)
		logger.Debug("applied transport middleware",
			slog.String("name", middlewares[i].Name()),
			slog.Int("priority", middlewares[i].Priority()),
		)
	}

	return handler
}

// --- CORS Middleware ---

// CORSMiddleware implements TransportMiddleware for CORS handling.
// In dev mode, it allows all origins. In production, it applies
// configured CORS restrictions.
type CORSMiddleware struct {
	corsHandler *cors.Cors
}

// NewCORSMiddleware creates a new CORS transport middleware.
// In dev mode, all origins are allowed. In production, the configured
// CORSConfig origins, methods, and headers are enforced.
func NewCORSMiddleware(cfg CORSConfig, devMode bool) *CORSMiddleware {
	var corsHandler *cors.Cors
	if devMode {
		corsHandler = cors.AllowAll()
	} else {
		corsHandler = cors.New(cors.Options{
			AllowedOrigins:   cfg.AllowedOrigins,
			AllowedMethods:   cfg.AllowedMethods,
			AllowedHeaders:   cfg.AllowedHeaders,
			ExposedHeaders:   cfg.ExposedHeaders,
			AllowCredentials: cfg.AllowCredentials,
			MaxAge:           cfg.MaxAge,
		})
	}
	return &CORSMiddleware{corsHandler: corsHandler}
}

// Name returns the middleware identifier.
func (m *CORSMiddleware) Name() string {
	return "cors"
}

// Priority returns the CORS priority (outermost handler).
func (m *CORSMiddleware) Priority() int {
	return PriorityCORS
}

// Wrap applies CORS handling to the given handler.
func (m *CORSMiddleware) Wrap(next http.Handler) http.Handler {
	return m.corsHandler.Handler(next)
}

// --- OTEL Transport Middleware ---

// OTELMiddleware implements TransportMiddleware for OpenTelemetry HTTP tracing.
// It wraps the handler with otelhttp instrumentation, filtering out
// health and reflection endpoints.
type OTELMiddleware struct {
	tp *sdktrace.TracerProvider
}

// NewOTELMiddleware creates a new OTEL transport middleware with the given TracerProvider.
func NewOTELMiddleware(tp *sdktrace.TracerProvider) *OTELMiddleware {
	return &OTELMiddleware{tp: tp}
}

// Name returns the middleware identifier.
func (m *OTELMiddleware) Name() string {
	return "otel"
}

// Priority returns the OTEL priority (after CORS).
func (m *OTELMiddleware) Priority() int {
	return PriorityOTEL
}

// Wrap applies OpenTelemetry HTTP instrumentation to the given handler.
// Health endpoints (/healthz, /readyz, /livez) and gRPC reflection
// endpoints are filtered from traces to reduce noise.
func (m *OTELMiddleware) Wrap(next http.Handler) http.Handler {
	mw := otelhttp.NewMiddleware("vanguard",
		otelhttp.WithTracerProvider(m.tp),
		otelhttp.WithFilter(func(r *http.Request) bool {
			path := r.URL.Path
			// Filter out health check endpoints.
			if path == "/healthz" || path == "/readyz" || path == "/livez" {
				return false
			}
			// Filter out gRPC reflection endpoints.
			if strings.HasPrefix(path, "/grpc.reflection.") {
				return false
			}
			return true
		}),
	)
	return mw(next)
}

// --- OTEL Connect Bundle ---

// OTELConnectBundle implements connect.ConnectInterceptorBundle for OpenTelemetry
// Connect RPC tracing. It provides otelconnect interceptors when a TracerProvider
// is available.
type OTELConnectBundle struct {
	tp     *sdktrace.TracerProvider
	logger *slog.Logger
}

// NewOTELConnectBundle creates a new OTEL Connect interceptor bundle.
func NewOTELConnectBundle(tp *sdktrace.TracerProvider, logger *slog.Logger) *OTELConnectBundle {
	return &OTELConnectBundle{tp: tp, logger: logger}
}

// Name returns the bundle identifier.
func (b *OTELConnectBundle) Name() string {
	return "otelconnect"
}

// Priority returns the OTEL Connect priority (before validation, after auth).
func (b *OTELConnectBundle) Priority() int {
	return connectpkg.PriorityValidation - 1
}

// Interceptors returns the otelconnect interceptor.
// Health check procedures are filtered from traces.
// If interceptor creation fails, a warning is logged and an empty slice is returned.
func (b *OTELConnectBundle) Interceptors() []connect.Interceptor {
	interceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithTracerProvider(b.tp),
		otelconnect.WithFilter(func(_ context.Context, spec connect.Spec) bool {
			// Filter out health check procedures.
			return !strings.HasPrefix(spec.Procedure, "/grpc.health.v1.Health/")
		}),
	)
	if err != nil {
		b.logger.Warn("failed to create otelconnect interceptor", slog.Any("error", err))
		return nil
	}
	return []connect.Interceptor{interceptor}
}
