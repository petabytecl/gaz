package health

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type GRPCServerTestSuite struct {
	suite.Suite
	logger *slog.Logger
}

func TestGRPCServerTestSuite(t *testing.T) {
	suite.Run(t, new(GRPCServerTestSuite))
}

func (s *GRPCServerTestSuite) SetupTest() {
	s.logger = slog.Default()
}

func (s *GRPCServerTestSuite) TestNewGRPCServer_DefaultInterval() {
	manager := NewManager()
	server := NewGRPCServer(manager, s.logger)

	s.Require().NotNil(server)
	s.Equal(DefaultGRPCCheckInterval, server.interval)
}

func (s *GRPCServerTestSuite) TestNewGRPCServer_WithCheckInterval() {
	manager := NewManager()
	interval := 10 * time.Second
	server := NewGRPCServer(manager, s.logger, WithCheckInterval(interval))

	s.Require().NotNil(server)
	s.Equal(interval, server.interval)
}

func (s *GRPCServerTestSuite) TestNewGRPCServer_WithZeroInterval_UsesDefault() {
	manager := NewManager()
	server := NewGRPCServer(manager, s.logger, WithCheckInterval(0))

	s.Require().NotNil(server)
	s.Equal(DefaultGRPCCheckInterval, server.interval)
}

func (s *GRPCServerTestSuite) TestRegister() {
	manager := NewManager()
	grpcServer := NewGRPCServer(manager, s.logger)

	// Create a gRPC server and register health service.
	srv := grpc.NewServer()
	defer srv.Stop()

	// Register should not panic.
	s.NotPanics(func() {
		grpcServer.Register(srv)
	})
}

func (s *GRPCServerTestSuite) TestGRPCServer_Healthy() {
	// Create manager with a healthy check.
	manager := NewManager()
	manager.AddReadinessCheck("always-healthy", func(_ context.Context) error {
		return nil
	})

	// Create gRPC health server with short interval.
	grpcHealthServer := NewGRPCServer(manager, s.logger, WithCheckInterval(50*time.Millisecond))

	// Create and start gRPC server.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer lis.Close()

	srv := grpc.NewServer()
	grpcHealthServer.Register(srv)

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			s.T().Logf("Server error: %v", serveErr)
		}
	}()
	defer srv.Stop()

	// Start the health server.
	ctx := context.Background()
	err = grpcHealthServer.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = grpcHealthServer.OnStop(stopCtx)
	}()

	// Wait for initial check.
	time.Sleep(100 * time.Millisecond)

	// Create client and check status.
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)

	checkCtx, checkCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer checkCancel()

	resp, err := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
	s.Require().NoError(err)
	s.Equal(healthpb.HealthCheckResponse_SERVING, resp.Status)
}

func (s *GRPCServerTestSuite) TestGRPCServer_Unhealthy() {
	// Create manager with a failing check.
	manager := NewManager()
	manager.AddReadinessCheck("always-failing", func(_ context.Context) error {
		return errors.New("unhealthy")
	})

	// Create gRPC health server with short interval.
	grpcHealthServer := NewGRPCServer(manager, s.logger, WithCheckInterval(50*time.Millisecond))

	// Create and start gRPC server.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer lis.Close()

	srv := grpc.NewServer()
	grpcHealthServer.Register(srv)

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			s.T().Logf("Server error: %v", serveErr)
		}
	}()
	defer srv.Stop()

	// Start the health server.
	ctx := context.Background()
	err = grpcHealthServer.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = grpcHealthServer.OnStop(stopCtx)
	}()

	// Wait for initial check.
	time.Sleep(100 * time.Millisecond)

	// Create client and check status.
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)

	checkCtx, checkCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer checkCancel()

	resp, err := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
	s.Require().NoError(err)
	s.Equal(healthpb.HealthCheckResponse_NOT_SERVING, resp.Status)
}

func (s *GRPCServerTestSuite) TestGRPCServer_StatusTransition() {
	// Create manager with a controllable check using atomic for thread safety.
	var healthy atomic.Bool
	healthy.Store(true)
	manager := NewManager()
	manager.AddReadinessCheck("toggle", func(_ context.Context) error {
		if healthy.Load() {
			return nil
		}
		return errors.New("unhealthy")
	})

	// Create gRPC health server with short interval.
	grpcHealthServer := NewGRPCServer(manager, s.logger, WithCheckInterval(50*time.Millisecond))

	// Create and start gRPC server.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer lis.Close()

	srv := grpc.NewServer()
	grpcHealthServer.Register(srv)

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			s.T().Logf("Server error: %v", serveErr)
		}
	}()
	defer srv.Stop()

	// Start the health server.
	ctx := context.Background()
	err = grpcHealthServer.OnStart(ctx)
	s.Require().NoError(err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = grpcHealthServer.OnStop(stopCtx)
	}()

	// Wait for initial check (healthy).
	time.Sleep(100 * time.Millisecond)

	// Create client.
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)

	// Check initial status is SERVING.
	checkCtx, checkCancel := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
	checkCancel()
	s.Require().NoError(err)
	s.Equal(healthpb.HealthCheckResponse_SERVING, resp.Status)

	// Toggle to unhealthy.
	healthy.Store(false)

	// Wait for poll interval.
	time.Sleep(100 * time.Millisecond)

	// Check status is now NOT_SERVING.
	checkCtx2, checkCancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	resp, err = client.Check(checkCtx2, &healthpb.HealthCheckRequest{Service: ""})
	checkCancel2()
	s.Require().NoError(err)
	s.Equal(healthpb.HealthCheckResponse_NOT_SERVING, resp.Status)
}

func (s *GRPCServerTestSuite) TestGRPCServer_StopCleanly() {
	manager := NewManager()
	grpcHealthServer := NewGRPCServer(manager, s.logger, WithCheckInterval(50*time.Millisecond))

	// Start the health server.
	ctx := context.Background()
	err := grpcHealthServer.OnStart(ctx)
	s.Require().NoError(err)

	// Let it run for a bit.
	time.Sleep(100 * time.Millisecond)

	// Stop should complete within timeout.
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = grpcHealthServer.OnStop(stopCtx)
	s.NoError(err)
}

func (s *GRPCServerTestSuite) TestGRPCServer_HealthServerAccessor() {
	manager := NewManager()
	grpcHealthServer := NewGRPCServer(manager, s.logger)

	// Should be able to access the underlying health server.
	healthServer := grpcHealthServer.HealthServer()
	s.NotNil(healthServer)
}

func (s *GRPCServerTestSuite) TestStatusToString() {
	tests := []struct {
		status   healthpb.HealthCheckResponse_ServingStatus
		expected string
	}{
		{healthpb.HealthCheckResponse_UNKNOWN, "UNKNOWN"},
		{healthpb.HealthCheckResponse_SERVING, "SERVING"},
		{healthpb.HealthCheckResponse_NOT_SERVING, "NOT_SERVING"},
		{healthpb.HealthCheckResponse_SERVICE_UNKNOWN, "SERVICE_UNKNOWN"},
		{healthpb.HealthCheckResponse_ServingStatus(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		s.Run(tt.expected, func() {
			result := statusToString(tt.status)
			s.Equal(tt.expected, result)
		})
	}
}

// Standalone tests for additional coverage.

func TestGRPCServer_InitialUnknown(t *testing.T) {
	manager := NewManager()
	logger := slog.Default()
	grpcHealthServer := NewGRPCServer(manager, logger)

	// Before OnStart, status should be UNKNOWN.
	// The internal health.Server starts with UNKNOWN for "".
	require.NotNil(t, grpcHealthServer.health)
	assert.Equal(t, healthpb.HealthCheckResponse_UNKNOWN, grpcHealthServer.lastStatus)
}

func TestGRPCServer_WithNegativeInterval_UsesDefault(t *testing.T) {
	manager := NewManager()
	logger := slog.Default()
	server := NewGRPCServer(manager, logger, WithCheckInterval(-1*time.Second))

	require.NotNil(t, server)
	assert.Equal(t, DefaultGRPCCheckInterval, server.interval)
}

func TestGRPCServer_NoChecks_Healthy(t *testing.T) {
	// Manager with no checks should report as healthy.
	manager := NewManager()
	logger := slog.Default()

	grpcHealthServer := NewGRPCServer(manager, logger, WithCheckInterval(50*time.Millisecond))

	// Create and start gRPC server.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	srv := grpc.NewServer()
	grpcHealthServer.Register(srv)

	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	// Start the health server.
	ctx := context.Background()
	err = grpcHealthServer.OnStart(ctx)
	require.NoError(t, err)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = grpcHealthServer.OnStop(stopCtx)
	}()

	// Wait for initial check.
	time.Sleep(100 * time.Millisecond)

	// Create client and check status.
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)

	checkCtx, checkCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer checkCancel()

	resp, err := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
}
