package vanguard

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/health"
)

// ServerTestSuite tests the Vanguard server lifecycle and functionality.
type ServerTestSuite struct {
	suite.Suite
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func (s *ServerTestSuite) TestOnStartAndStop() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	server := NewServer(cfg, logger, container, grpcServer)

	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	// Give server time to bind.
	time.Sleep(50 * time.Millisecond)

	// Verify we can connect.
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/nonexistent", cfg.Port))
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	// Should get some response (404 from transcoder or handler).
	s.NotNil(resp)

	// Stop.
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *ServerTestSuite) TestOnStartDiscoverConnectServices() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	// Register a mock connect registrar.
	mock := &mockConnectRegistrar{
		path:    "/test.Service/",
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("connect-ok")) }),
	}
	err := di.For[*mockConnectRegistrar](container).Instance(mock)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, grpcServer)

	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Request the Connect service path — should hit our mock handler via unknown handler.
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test.Service/Method", cfg.Port))
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	s.Equal("connect-ok", string(body))
}

func (s *ServerTestSuite) TestOnStartHealthAutoMount() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = true
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	// Register a health.Manager in DI.
	mgr := health.NewManager()
	err := di.For[*health.Manager](container).Instance(mgr)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, grpcServer)

	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Check health endpoints use health.DefaultConfig paths.
	for _, endpoint := range []string{"/live", "/ready", "/startup"} {
		resp, reqErr := http.Get(fmt.Sprintf("http://localhost:%d%s", cfg.Port, endpoint))
		s.Require().NoErrorf(reqErr, "GET %s should not error", endpoint)
		defer func() { _ = resp.Body.Close() }()
		s.Equalf(http.StatusOK, resp.StatusCode, "GET %s should return 200", endpoint)
	}
}

func (s *ServerTestSuite) TestOnStartNoHealthWithoutManager() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = true // Enabled, but no Manager registered.
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	server := NewServer(cfg, logger, container, grpcServer)
	// Health manager should be nil because none registered.
	s.Nil(server.healthManager)

	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()
}

func (s *ServerTestSuite) TestOnStopGracefulShutdown() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	server := NewServer(cfg, logger, container, grpcServer)

	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	time.Sleep(50 * time.Millisecond)

	// Graceful shutdown should complete cleanly.
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *ServerTestSuite) TestOnStopBeforeStart() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	server := NewServer(cfg, logger, container, nil)

	// OnStop before OnStart should not error.
	ctx := context.Background()
	err := server.OnStop(ctx)
	s.Require().NoError(err)
}

func (s *ServerTestSuite) TestSetUnknownHandler() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	server := NewServer(cfg, logger, container, grpcServer)
	server.SetUnknownHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("custom-handler"))
	}))

	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Request root path — should hit user unknown handler.
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/custom", cfg.Port))
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	s.Equal("custom-handler", string(body))
}

func (s *ServerTestSuite) TestOnStartNoServices() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	server := NewServer(cfg, logger, container, grpcServer)

	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err, "Server should start even with zero services")

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *ServerTestSuite) TestOnStartWithoutGRPCServer() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()

	// No gRPC server — should still work with Connect-only services.
	server := NewServer(cfg, logger, container, nil)

	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *ServerTestSuite) TestOnStartPortBindingError() {
	// Bind a port first.
	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	defer func() { _ = lis.Close() }()

	port := lis.Addr().(*net.TCPAddr).Port

	cfg := DefaultConfig()
	cfg.Port = port
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	server := NewServer(cfg, logger, container, grpcServer)

	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().Error(err)
	s.Contains(err.Error(), "bind port")
}

func (s *ServerTestSuite) TestServicePathToName() {
	tests := []struct {
		path     string
		expected string
	}{
		{"/package.Service/", "package.Service"},
		{"/package.Service", "package.Service"},
		{"package.Service/", "package.Service"},
		{"package.Service", "package.Service"},
		{"/", ""},
		{"", ""},
	}

	for _, tt := range tests {
		s.Run(tt.path, func() {
			s.Equal(tt.expected, servicePathToName(tt.path))
		})
	}
}

func (s *ServerTestSuite) TestOnStartWithReflection() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = true
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	// Register a mock Connect service so reflection has services to reflect.
	mock := &mockConnectRegistrar{
		path:    "/test.Reflectable/",
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	}
	err := di.For[*mockConnectRegistrar](container).Instance(mock)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, grpcServer)

	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Reflection v1 endpoint should respond.
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/grpc.reflection.v1.ServerReflection/ServerReflectionInfo", cfg.Port))
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	// We just verify the endpoint is mounted (non-404).
	s.NotEqual(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestOnStartConnectInterceptorCollection() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	// Register a Connect interceptor bundle with a real interceptor to cover the
	// `if len(connectInterceptors) > 0` branch in OnStart.
	bundle := &mockInterceptorBundle{interceptors: []connect.Interceptor{&noopInterceptor{}}}
	err := di.For[*mockInterceptorBundle](container).Instance(bundle)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, grpcServer)

	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *ServerTestSuite) TestNewServerNilLogger() {
	cfg := DefaultConfig()
	container := di.New()

	server := NewServer(cfg, nil, container, nil)
	s.NotNil(server.logger, "Should fall back to slog.Default()")
}

func (s *ServerTestSuite) TestBuildTranscoderNoGRPCServer() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()

	// No gRPC server — exercises the plain Vanguard transcoder path.
	server := NewServer(cfg, logger, container, nil)

	// Register a mock Connect registrar.
	mock := &mockConnectRegistrar{
		path:    "/test.PlainConnect/",
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("plain-ok")) }),
	}
	err := di.For[*mockConnectRegistrar](container).Instance(mock)
	s.Require().NoError(err)

	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Connect service should be reachable even without gRPC.
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/test.PlainConnect/Method", cfg.Port))
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	s.Equal("plain-ok", string(body))
}

func (s *ServerTestSuite) TestOnStartReflectionWithGRPCServices() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = true
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()

	// Register gRPC services on the grpc.Server so reflection has gRPC services to enumerate.
	grpcServer := grpc.NewServer()
	// Register a connect registrar too to have both service types.
	mock := &mockConnectRegistrar{
		path:    "/test.Both/",
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	}
	err := di.For[*mockConnectRegistrar](container).Instance(mock)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, grpcServer)
	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *ServerTestSuite) TestOnStartReflectionDisabledNoMounting() {
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	cfg.HealthEnabled = false
	logger := slog.Default()
	container := di.New()
	grpcServer := grpc.NewServer()

	// Register a connect service so there ARE services, but reflection disabled.
	mock := &mockConnectRegistrar{
		path:    "/test.NoReflect/",
		handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	}
	err := di.For[*mockConnectRegistrar](container).Instance(mock)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, grpcServer)
	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *ServerTestSuite) TestBuildHealthMux() {
	mgr := health.NewManager()
	mux := buildHealthMux(mgr, nil)
	s.NotNil(mux)

	// Nil manager should return nil.
	nilMux := buildHealthMux(nil, nil)
	s.Nil(nilMux)
}

func (s *ServerTestSuite) TestBuildHealthMuxDefaultPaths() {
	mgr := health.NewManager()
	mux := buildHealthMux(mgr, nil)
	s.Require().NotNil(mux)

	// Default health.Config paths: /live, /ready, /startup
	for _, path := range []string{"/live", "/ready", "/startup"} {
		req, err := http.NewRequest(http.MethodGet, path, nil)
		s.Require().NoError(err)
		_, pattern := mux.Handler(req)
		s.NotEmptyf(pattern, "handler should be registered at %s", path)
	}
}

func (s *ServerTestSuite) TestBuildHealthMuxCustomPaths() {
	mgr := health.NewManager()
	cfg := &health.Config{
		LivenessPath:  "/custom-live",
		ReadinessPath: "/custom-ready",
		StartupPath:   "/custom-startup",
	}
	mux := buildHealthMux(mgr, cfg)
	s.Require().NotNil(mux)

	for _, path := range []string{"/custom-live", "/custom-ready", "/custom-startup"} {
		req, err := http.NewRequest(http.MethodGet, path, nil)
		s.Require().NoError(err)
		_, pattern := mux.Handler(req)
		s.NotEmptyf(pattern, "handler should be registered at custom path %s", path)
	}
}

// mockConnectRegistrar is a test double for connect.Registrar.
type mockConnectRegistrar struct {
	path    string
	handler http.Handler
}

func (m *mockConnectRegistrar) RegisterConnect(_ ...connect.HandlerOption) (string, http.Handler) {
	return m.path, m.handler
}

// mockInterceptorBundle is a test double for connect.InterceptorBundle.
type mockInterceptorBundle struct {
	interceptors []connect.Interceptor
}

func (m *mockInterceptorBundle) Name() string                        { return "mock" }
func (m *mockInterceptorBundle) Priority() int                       { return 0 }
func (m *mockInterceptorBundle) Interceptors() []connect.Interceptor { return m.interceptors }

// noopInterceptor is a test interceptor that does nothing.
type noopInterceptor struct{}

func (n *noopInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc { return next }
func (n *noopInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (n *noopInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// getFreePort finds an available port for testing.
func getFreePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer func() { _ = lis.Close() }()
	return lis.Addr().(*net.TCPAddr).Port
}
