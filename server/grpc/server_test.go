package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/petabytecl/gaz/di"
)

// GRPCServerTestSuite tests the gRPC server lifecycle and functionality.
type GRPCServerTestSuite struct {
	suite.Suite
}

func TestGRPCServerTestSuite(t *testing.T) {
	suite.Run(t, new(GRPCServerTestSuite))
}

func (s *GRPCServerTestSuite) TestGRPCServerStartStop() {
	// Setup.
	cfg := DefaultConfig()
	cfg.Port = getFreePort(s.T())
	logger := slog.Default()
	container := di.New()

	server := NewServer(cfg, logger, container, false)

	// Start.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	// Give server time to bind.
	time.Sleep(50 * time.Millisecond)

	// Verify we can connect.
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	s.Require().NotNil(conn)
	defer conn.Close()

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
	container := di.New()

	server := NewServer(cfg, logger, container, false)

	// Start.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	// Give server time to bind.
	time.Sleep(50 * time.Millisecond)

	// Connect.
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer conn.Close()

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

	// Look for reflection service.
	found := false
	for _, svc := range services {
		if svc.Name == "grpc.reflection.v1alpha.ServerReflection" {
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
	container := di.New()

	// Register a mock service registrar by its concrete type.
	// ResolveAll[ServiceRegistrar] finds it because *mockServiceRegistrar implements ServiceRegistrar.
	mockRegistrar := &mockServiceRegistrar{registered: false}
	err := di.For[*mockServiceRegistrar](container).Instance(mockRegistrar)
	s.Require().NoError(err)

	server := NewServer(cfg, logger, container, false)

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
	s.True(mockRegistrar.registered, "Service registrar should have been called")
}

func (s *GRPCServerTestSuite) TestGRPCServerPortBindingError() {
	// Bind a port first.
	lis, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	defer lis.Close()

	port := lis.Addr().(*net.TCPAddr).Port

	// Try to start server on same port.
	cfg := DefaultConfig()
	cfg.Port = port
	logger := slog.Default()
	container := di.New()

	server := NewServer(cfg, logger, container, false)

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
	container := di.New()

	server := NewServer(cfg, logger, container, false)

	// Start.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	// Give server time to bind.
	time.Sleep(50 * time.Millisecond)

	// Connect and keep connection open.
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer conn.Close()

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
	container := di.New()

	server := NewServer(cfg, logger, container, false)

	// Start.
	ctx := context.Background()
	err := server.OnStart(ctx)
	s.Require().NoError(err)

	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.OnStop(stopCtx)
	}()

	// Give server time to bind.
	time.Sleep(50 * time.Millisecond)

	// Connect.
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", cfg.Port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer conn.Close()

	// Reflection should fail when disabled.
	client := rpb.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(ctx)
	s.Require().NoError(err) // Connection succeeds

	err = stream.Send(&rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	})
	s.Require().NoError(err)

	// Receiving should fail or return error response.
	_, err = stream.Recv()
	// When reflection is disabled, the RPC will fail.
	s.Require().Error(err, "Reflection should not be available when disabled")
}

// mockServiceRegistrar is a test double for ServiceRegistrar.
type mockServiceRegistrar struct {
	registered bool
}

func (m *mockServiceRegistrar) RegisterService(_ grpc.ServiceRegistrar) {
	m.registered = true
}

// getFreePort finds an available port for testing.
func getFreePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer lis.Close()
	return lis.Addr().(*net.TCPAddr).Port
}
