package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1"

	"github.com/petabytecl/gaz/di"
)

// GRPCServerTestSuite tests the gRPC server lifecycle and functionality.
type GRPCServerTestSuite struct {
	suite.Suite
}

func TestGRPCServerTestSuite(t *testing.T) {
	suite.Run(t, new(GRPCServerTestSuite))
}

// setupTestContainer creates a DI container with built-in interceptor bundles registered.
func setupTestContainer(logger *slog.Logger) *di.Container {
	container := di.New()
	// Register built-in interceptor bundles.
	_ = di.For[*LoggingBundle](container).Instance(NewLoggingBundle(logger))
	_ = di.For[*RecoveryBundle](container).Instance(NewRecoveryBundle(logger, false))
	return container
}

func (s *GRPCServerTestSuite) TestGRPCServerStartStop() {
	// Setup.
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	logger := slog.Default()
	container := setupTestContainer(logger)

	server := NewServer(cfg, logger, container, nil)

	// Start.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	// Wait for server to bind.
	waitForPort(s.T(), cfg.Port)

	// Verify we can connect.
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	s.Require().NotNil(conn)
	defer func() { _ = conn.Close() }()

	// Stop.
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *GRPCServerTestSuite) TestGRPCServerReflection() {
	// Setup.
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = true
	logger := slog.Default()
	container := setupTestContainer(logger)

	server := NewServer(cfg, logger, container, nil)

	// Start.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	// Wait for server to bind.
	waitForPort(s.T(), cfg.Port)

	// Connect.
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer func() { _ = conn.Close() }()

	// Test reflection by listing services.
	client := rpb.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(ctx)
	s.Require().NoError(err)

	// Send request to list services.
	err = stream.Send(&rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	})
	s.Require().NoError(err)

	// Receive response.
	resp, err := stream.Recv()
	s.Require().NoError(err)
	s.Require().NotNil(resp.GetListServicesResponse())

	// Should have at least the reflection service itself.
	services := resp.GetListServicesResponse().GetService()
	s.Require().NotEmpty(services, "Reflection should return at least one service")

	// Look for reflection service (v1 or v1alpha).
	found := false
	for _, svc := range services {
		name := svc.GetName()
		if name == "grpc.reflection.v1.ServerReflection" || name == "grpc.reflection.v1alpha.ServerReflection" {
			found = true
			break
		}
	}
	s.True(found, "Reflection service should be discoverable")
}

func (s *GRPCServerTestSuite) TestGRPCServerServiceDiscovery() {
	// Setup.
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	logger := slog.Default()
	container := setupTestContainer(logger)

	// Register a mock service registrar by its concrete type.
	// ResolveAll[Registrar] finds it because *mockRegistrar implements Registrar.
	mockReg := &mockRegistrar{registered: false}
	err := di.For[*mockRegistrar](container).Instance(mockReg)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, nil)

	// Start.
	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	// Verify service was discovered and registered.
	s.True(mockReg.registered, "Service registrar should have been called")
}

func (s *GRPCServerTestSuite) TestGRPCServerPortBindingError() {
	// Bind a port first.
	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	defer func() { _ = lis.Close() }()

	port := lis.Addr().(*net.TCPAddr).Port

	// Try to start server on same port.
	cfg := DefaultConfig()
	cfg.Port = port
	logger := slog.Default()
	container := setupTestContainer(logger)

	server := NewServer(cfg, logger, container, nil)

	// Start should fail.
	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().Error(err)
	s.Contains(err.Error(), "bind port")
}

func (s *GRPCServerTestSuite) TestGRPCServerGracefulShutdown() {
	// Setup.
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	logger := slog.Default()
	container := setupTestContainer(logger)

	server := NewServer(cfg, logger, container, nil)

	// Start.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	// Wait for server to bind.
	waitForPort(s.T(), cfg.Port)

	// Connect and keep connection open.
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer func() { _ = conn.Close() }()

	// Start graceful shutdown with timeout.
	// Graceful shutdown should complete cleanly.
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *GRPCServerTestSuite) TestGRPCServerReflectionDisabled() {
	// Setup with reflection disabled.
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	cfg.Reflection = false
	logger := slog.Default()
	container := setupTestContainer(logger)

	server := NewServer(cfg, logger, container, nil)

	// Start.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	// Wait for server to bind.
	waitForPort(s.T(), cfg.Port)

	// Connect.
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer func() { _ = conn.Close() }()

	// Reflection should fail when disabled.
	client := rpb.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(ctx)
	s.Require().NoError(err) // Connection succeeds

	err = stream.Send(&rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	})
	if err != nil {
		// Send may fail with EOF if gRPC handshake isn't complete — still proves reflection is unavailable.
		return
	}

	// Receiving should fail or return error response.
	_, err = stream.Recv()
	s.Require().Error(err, "Reflection should not be available when disabled")
}

func (s *GRPCServerTestSuite) TestGRPCServerGetGRPCServer() {
	// Setup.
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	logger := slog.Default()
	container := setupTestContainer(logger)

	server := NewServer(cfg, logger, container, nil)

	// Before OnStart, GRPCServer returns nil (deferred construction).
	s.Nil(server.GRPCServer(), "GRPCServer should be nil before OnStart")

	// Start the server to create the underlying grpc.Server.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	// After OnStart, GRPCServer should return the underlying grpc.Server.
	grpcServer := server.GRPCServer()
	s.Require().NotNil(grpcServer, "GRPCServer should be available after OnStart")
}

func (s *GRPCServerTestSuite) TestSkipListenerStartStop() {
	// Setup with skip-listener mode.
	cfg := DefaultConfig()
	cfg.SkipListener = true
	logger := slog.Default()
	container := setupTestContainer(logger)

	// Register a mock service registrar.
	mockReg := &mockRegistrar{registered: false}
	err := di.For[*mockRegistrar](container).Instance(mockReg)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, nil)

	// Start in skip-listener mode.
	ctx := context.Background()
	err = server.OnStart(ctx)
	s.Require().NoError(err)

	// Service should still be discovered and registered.
	s.True(mockReg.registered, "Service registrar should have been called in skip-listener mode")

	// No listener should have been created.
	s.Nil(server.listener, "Listener should be nil in skip-listener mode")

	// Stop should work without errors.
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *GRPCServerTestSuite) TestSkipListenerWithReflection() {
	// Setup with skip-listener mode and reflection enabled.
	cfg := DefaultConfig()
	cfg.SkipListener = true
	cfg.Reflection = true
	logger := slog.Default()
	container := setupTestContainer(logger)

	server := NewServer(cfg, logger, container, nil)

	// Start in skip-listener mode.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	// Verify reflection was registered on the grpc.Server.
	// When reflection is registered, GetServiceInfo should include the reflection service.
	serviceInfo := server.GRPCServer().GetServiceInfo()
	_, hasReflection := serviceInfo["grpc.reflection.v1.ServerReflection"]
	_, hasReflectionAlpha := serviceInfo["grpc.reflection.v1alpha.ServerReflection"]
	s.True(hasReflection || hasReflectionAlpha, "Reflection service should be registered in skip-listener mode")
}

func (s *GRPCServerTestSuite) TestSkipListenerNoPortBinding() {
	// Setup with skip-listener mode — port should be irrelevant.
	cfg := DefaultConfig()
	cfg.SkipListener = true
	cfg.Port = 0 // Invalid port, but should not matter.
	logger := slog.Default()
	container := setupTestContainer(logger)

	server := NewServer(cfg, logger, container, nil)

	// Start should succeed even with port=0 because no listener is created.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	// Stop.
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.OnStop(stopCtx)
	s.Require().NoError(err)
}

func (s *GRPCServerTestSuite) TestSkipListenerConfigValidation() {
	// SkipListener=true should skip port validation.
	cfg := DefaultConfig()
	cfg.SkipListener = true
	cfg.Port = 0 // Invalid port, but SkipListener=true should skip validation.
	err := cfg.Validate()
	s.Require().NoError(err, "Validate should skip port check when SkipListener is true")

	// SkipListener=false should still require valid port.
	cfg2 := DefaultConfig()
	cfg2.SkipListener = false
	cfg2.Port = 0
	err = cfg2.Validate()
	s.Require().Error(err, "Validate should require valid port when SkipListener is false")
}

func (s *GRPCServerTestSuite) TestSkipListenerConfigFlag() {
	// Verify SkipListener defaults to false.
	cfg := DefaultConfig()
	s.False(cfg.SkipListener, "SkipListener should default to false")
}

// mockRegistrar is a test double for Registrar.
type mockRegistrar struct {
	registered bool
}

func (m *mockRegistrar) RegisterService(_ grpc.ServiceRegistrar) {
	m.registered = true
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

// waitForPort waits until a TCP connection can be made to the given port.
func waitForPort(t *testing.T, port int) {
	t.Helper()
	addr := fmt.Sprintf("localhost:%d", port)
	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 2*time.Second, 10*time.Millisecond)
}
