package vanguard

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	connectotel "connectrpc.com/otelconnect"
	"github.com/stretchr/testify/suite"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/petabytecl/gaz/di"
	connectpkg "github.com/petabytecl/gaz/server/connect"
)

// MiddlewareTestSuite tests the transport middleware and OTEL Connect bundle.
type MiddlewareTestSuite struct {
	suite.Suite
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

// --- Priority Constants ---

func (s *MiddlewareTestSuite) TestPriorityConstants() {
	s.Equal(0, PriorityCORS)
	s.Equal(100, PriorityOTEL)
	s.Less(PriorityCORS, PriorityOTEL, "CORS must run before OTEL")
}

// --- CORSMiddleware ---

func (s *MiddlewareTestSuite) TestCORSMiddleware_ImplementsTransportMiddleware() {
	cfg := DefaultCORSConfig(false)
	m := NewCORSMiddleware(cfg, false)

	var _ TransportMiddleware = m

	s.Equal("cors", m.Name())
	s.Equal(PriorityCORS, m.Priority())
}

func (s *MiddlewareTestSuite) TestCORSMiddleware_DevModeAllowsAll() {
	cfg := DefaultCORSConfig(true)
	m := NewCORSMiddleware(cfg, true)

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := m.Wrap(inner)

	// Send a preflight request from any origin.
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	s.Equal("*", rec.Header().Get("Access-Control-Allow-Origin"),
		"Dev mode should allow all origins")
}

func (s *MiddlewareTestSuite) TestCORSMiddleware_ProductionRespectsConfig() {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}
	m := NewCORSMiddleware(cfg, false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := m.Wrap(inner)

	// Allowed origin.
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	s.Equal("https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))

	// Disallowed origin.
	req2 := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req2.Header.Set("Origin", "https://evil.com")
	req2.Header.Set("Access-Control-Request-Method", "POST")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	s.Empty(rec2.Header().Get("Access-Control-Allow-Origin"),
		"Disallowed origin should not get CORS header")
}

// --- OTELMiddleware ---

func (s *MiddlewareTestSuite) TestOTELMiddleware_ImplementsTransportMiddleware() {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(s.T().Context()) }()

	m := NewOTELMiddleware(tp)

	var _ TransportMiddleware = m

	s.Equal("otel", m.Name())
	s.Equal(PriorityOTEL, m.Priority())
}

func (s *MiddlewareTestSuite) TestOTELMiddleware_WrapsHandler() {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(s.T().Context()) }()

	m := NewOTELMiddleware(tp)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Wrap(inner)
	s.NotNil(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	s.True(called, "Inner handler should be called through OTEL middleware")
	s.Equal(http.StatusOK, rec.Code)
}

// --- OTELConnectBundle ---

func (s *MiddlewareTestSuite) TestOTELConnectBundle_ImplementsConnectInterceptorBundle() {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(s.T().Context()) }()

	b := NewOTELConnectBundle(tp, slog.Default())

	var _ connectpkg.InterceptorBundle = b

	s.Equal("otelconnect", b.Name())
	s.Equal(connectpkg.PriorityValidation-1, b.Priority(),
		"OTEL Connect should run before validation")
}

func (s *MiddlewareTestSuite) TestOTELConnectBundle_ReturnsInterceptors() {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(s.T().Context()) }()

	b := NewOTELConnectBundle(tp, slog.Default())
	interceptors := b.Interceptors()

	s.Len(interceptors, 1, "Should return exactly one otelconnect interceptor")
}

// --- collectTransportMiddleware ---

func (s *MiddlewareTestSuite) TestCollectTransportMiddleware_EmptyContainer() {
	container := di.New()
	logger := slog.Default()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	result := collectTransportMiddleware(container, logger, inner)
	s.NotNil(result, "Should return original handler when no middleware registered")

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	result.ServeHTTP(rec, req)
	s.Equal(http.StatusOK, rec.Code)
}

// testMiddleware is a test helper that records wrap order.
type testMiddleware struct {
	name     string
	priority int
	order    *[]string
}

func (m *testMiddleware) Name() string  { return m.name }
func (m *testMiddleware) Priority() int { return m.priority }
func (m *testMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*m.order = append(*m.order, m.name)
		next.ServeHTTP(w, r)
	})
}

// Distinct concrete types for DI registration to enable ResolveAll discovery.
type testMiddlewareHigh struct{ testMiddleware }

type testMiddlewareMid struct{ testMiddleware }

type testMiddlewareLow struct{ testMiddleware }

func (s *MiddlewareTestSuite) TestCollectTransportMiddleware_AppliesInPriorityOrder() {
	container := di.New()
	logger := slog.Default()
	var order []string

	// Register middleware in reverse priority order to verify sorting.
	s.Require().NoError(di.For[*testMiddlewareHigh](container).ProviderFunc(
		func(_ *di.Container) *testMiddlewareHigh {
			return &testMiddlewareHigh{testMiddleware{name: "high", priority: 200, order: &order}}
		},
	))
	s.Require().NoError(di.For[*testMiddlewareLow](container).ProviderFunc(
		func(_ *di.Container) *testMiddlewareLow {
			return &testMiddlewareLow{testMiddleware{name: "low", priority: 10, order: &order}}
		},
	))
	s.Require().NoError(di.For[*testMiddlewareMid](container).ProviderFunc(
		func(_ *di.Container) *testMiddlewareMid {
			return &testMiddlewareMid{testMiddleware{name: "mid", priority: 100, order: &order}}
		},
	))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := collectTransportMiddleware(container, logger, inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Lowest priority wraps outermost — executes first on request.
	s.Equal([]string{"low", "mid", "high"}, order,
		"Middleware should execute in priority order (lowest first)")
}

// --- OTELMiddleware Wrap filter ---

func (s *MiddlewareTestSuite) TestOTELMiddleware_FiltersHealthEndpoints() {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(s.T().Context()) }()

	m := NewOTELMiddleware(tp)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Wrap(inner)

	// Health endpoints should still be served (just filtered from traces).
	for _, path := range []string{"/healthz", "/readyz", "/livez"} {
		called = false
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		s.True(called, "Inner handler should be called for %s", path)
	}

	// Reflection endpoints should also be filtered from traces.
	called = false
	req := httptest.NewRequest(http.MethodGet, "/grpc.reflection.v1.ServerReflection/ServerReflectionInfo", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	s.True(called, "Inner handler should be called for reflection endpoint")
}

// --- OTELConnectBundle Interceptors error handling ---

func (s *MiddlewareTestSuite) TestOTELConnectBundle_InterceptorsSuccess() {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(s.T().Context()) }()

	b := NewOTELConnectBundle(tp, slog.Default())
	interceptors := b.Interceptors()
	s.Len(interceptors, 1)
}

// --- DefaultCORSConfig ---

func (s *MiddlewareTestSuite) TestDefaultCORSConfig_DevMode() {
	cfg := DefaultCORSConfig(true)

	s.Equal([]string{"*"}, cfg.AllowedOrigins)
	s.Equal([]string{"*"}, cfg.AllowedHeaders)
	s.False(cfg.AllowCredentials, "Cannot use * with credentials")
	s.Equal(DefaultCORSMaxAge, cfg.MaxAge)
	s.Contains(cfg.AllowedMethods, "OPTIONS")
}

func (s *MiddlewareTestSuite) TestDefaultCORSConfig_Production() {
	cfg := DefaultCORSConfig(false)

	s.Empty(cfg.AllowedOrigins, "Production origins must be explicitly configured")
	s.True(cfg.AllowCredentials)
	s.Equal(DefaultCORSMaxAge, cfg.MaxAge)
	s.Contains(cfg.AllowedHeaders, "Authorization")
	s.Contains(cfg.AllowedHeaders, "Content-Type")
	s.Contains(cfg.ExposedHeaders, "X-Request-ID")
	s.NotContains(cfg.AllowedMethods, "OPTIONS",
		"Production methods should not include OPTIONS (handled by CORS handler)")
}

// --- Interface compliance compile checks ---

// Verify compile-time interface compliance.
var (
	_ TransportMiddleware          = (*CORSMiddleware)(nil)
	_ TransportMiddleware          = (*OTELMiddleware)(nil)
	_ connectpkg.InterceptorBundle = (*OTELConnectBundle)(nil)
	_ connect.Interceptor          = (*connectotel.Interceptor)(nil)
)
