package grpc

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

	gazhealth "github.com/petabytecl/gaz/health"
)

type HealthAdapterTestSuite struct {
	suite.Suite
	logger *slog.Logger
}

func TestHealthAdapterTestSuite(t *testing.T) {
	suite.Run(t, new(HealthAdapterTestSuite))
}

func (s *HealthAdapterTestSuite) SetupTest() {
	s.logger = slog.Default()
}

func (s *HealthAdapterTestSuite) TestNewHealthAdapter_DefaultInterval() {
	manager := gazhealth.NewManager()
	interval := 5 * time.Second
	adapter := newHealthAdapter(manager, interval, s.logger)

	s.Require().NotNil(adapter)
	s.Equal(interval, adapter.interval)
}

func (s *HealthAdapterTestSuite) TestRegister() {
	manager := gazhealth.NewManager()
	adapter := newHealthAdapter(manager, time.Second, s.logger)

	// Create a gRPC server and register health service.
	srv := grpc.NewServer()
	defer srv.Stop()

	// Register should not panic.
	s.NotPanics(func() {
		adapter.Register(srv)
	})
}

func (s *HealthAdapterTestSuite) TestHealthAdapter_Healthy() {
	// Create manager with a healthy check.
	manager := gazhealth.NewManager()
	manager.AddReadinessCheck("always-healthy", func(_ context.Context) error {
		return nil
	})

	// Create gRPC health adapter with short interval.
	adapter := newHealthAdapter(manager, 50*time.Millisecond, s.logger)

	// Create and start gRPC server.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer func() { _ = lis.Close() }()

	srv := grpc.NewServer()
	adapter.Register(srv)

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			s.T().Logf("Server error: %v", serveErr)
		}
	}()
	defer srv.Stop()

	// Start the health adapter.
	ctx := context.Background()
	adapter.Start(ctx)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = adapter.Stop(stopCtx)
	}()

	// Wait for initial check by polling gRPC health status.
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer func() { _ = conn.Close() }()

	client := healthpb.NewHealthClient(conn)

	s.Require().Eventually(func() bool {
		checkCtx, checkCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer checkCancel()
		resp, checkErr := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
		return checkErr == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	}, 2*time.Second, 20*time.Millisecond)
}

func (s *HealthAdapterTestSuite) TestHealthAdapter_Unhealthy() {
	// Create manager with a failing check.
	manager := gazhealth.NewManager()
	manager.AddReadinessCheck("always-failing", func(_ context.Context) error {
		return errors.New("unhealthy")
	})

	// Create gRPC health adapter with short interval.
	adapter := newHealthAdapter(manager, 50*time.Millisecond, s.logger)

	// Create and start gRPC server.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer func() { _ = lis.Close() }()

	srv := grpc.NewServer()
	adapter.Register(srv)

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			s.T().Logf("Server error: %v", serveErr)
		}
	}()
	defer srv.Stop()

	// Start the health adapter.
	ctx := context.Background()
	adapter.Start(ctx)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = adapter.Stop(stopCtx)
	}()

	// Wait for initial check by polling gRPC health status.
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer func() { _ = conn.Close() }()

	client := healthpb.NewHealthClient(conn)

	s.Require().Eventually(func() bool {
		checkCtx, checkCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer checkCancel()
		resp, checkErr := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
		return checkErr == nil && resp.GetStatus() == healthpb.HealthCheckResponse_NOT_SERVING
	}, 2*time.Second, 20*time.Millisecond)
}

func (s *HealthAdapterTestSuite) TestHealthAdapter_StatusTransition() {
	// Create manager with a controllable check using atomic for thread safety.
	var healthy atomic.Bool
	healthy.Store(true)
	manager := gazhealth.NewManager()
	manager.AddReadinessCheck("toggle", func(_ context.Context) error {
		if healthy.Load() {
			return nil
		}
		return errors.New("unhealthy")
	})

	// Create gRPC health adapter with short interval.
	adapter := newHealthAdapter(manager, 50*time.Millisecond, s.logger)

	// Create and start gRPC server.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer func() { _ = lis.Close() }()

	srv := grpc.NewServer()
	adapter.Register(srv)

	go func() {
		if serveErr := srv.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			s.T().Logf("Server error: %v", serveErr)
		}
	}()
	defer srv.Stop()

	// Start the health adapter.
	ctx := context.Background()
	adapter.Start(ctx)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = adapter.Stop(stopCtx)
	}()

	// Create client.
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)
	defer func() { _ = conn.Close() }()

	client := healthpb.NewHealthClient(conn)

	// Wait for initial check (healthy) via polling.
	s.Require().Eventually(func() bool {
		checkCtx, checkCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer checkCancel()
		resp, checkErr := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
		return checkErr == nil && resp.GetStatus() == healthpb.HealthCheckResponse_SERVING
	}, 2*time.Second, 20*time.Millisecond)

	// Toggle to unhealthy.
	healthy.Store(false)

	// Wait for status transition to NOT_SERVING via polling.
	s.Require().Eventually(func() bool {
		checkCtx, checkCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer checkCancel()
		resp, checkErr := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
		return checkErr == nil && resp.GetStatus() == healthpb.HealthCheckResponse_NOT_SERVING
	}, 2*time.Second, 20*time.Millisecond)
}

func (s *HealthAdapterTestSuite) TestHealthAdapter_StopCleanly() {
	manager := gazhealth.NewManager()
	adapter := newHealthAdapter(manager, 50*time.Millisecond, s.logger)

	// Start the health adapter.
	ctx := context.Background()
	adapter.Start(ctx)

	// Stop should complete within timeout (the adapter starts its poll loop immediately).
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := adapter.Stop(stopCtx)
	s.NoError(err)
}

func (s *HealthAdapterTestSuite) TestStatusToString() {
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

func TestHealthAdapter_InitialUnknown(t *testing.T) {
	manager := gazhealth.NewManager()
	logger := slog.Default()
	adapter := newHealthAdapter(manager, time.Second, logger)

	// Before Start, status should be UNKNOWN.
	require.NotNil(t, adapter.health)
	assert.Equal(t, healthpb.HealthCheckResponse_UNKNOWN, adapter.lastStatus)
}

func TestHealthAdapter_DoubleClose(t *testing.T) {
	// Calling Stop twice must not panic (sync.Once guards close(stopCh)).
	manager := gazhealth.NewManager()
	logger := slog.Default()
	adapter := newHealthAdapter(manager, 50*time.Millisecond, logger)

	// Start the adapter so the poll loop is running.
	ctx := context.Background()
	adapter.Start(ctx)

	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// First stop should succeed.
	err := adapter.Stop(stopCtx)
	require.NoError(t, err)

	// Second stop must not panic.
	assert.NotPanics(t, func() {
		stopCtx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel2()
		_ = adapter.Stop(stopCtx2)
	}, "calling Stop twice should not panic")
}

func TestHealthAdapter_NoChecks_Unknown(t *testing.T) {
	// Manager with no checks reports NOT_SERVING (StatusUnknown maps to Down).
	manager := gazhealth.NewManager()
	logger := slog.Default()

	adapter := newHealthAdapter(manager, 50*time.Millisecond, logger)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = lis.Close() }()

	srv := grpc.NewServer()
	adapter.Register(srv)

	go func() {
		_ = srv.Serve(lis)
	}()
	defer srv.Stop()

	ctx := context.Background()
	adapter.Start(ctx)
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = adapter.Stop(stopCtx)
	}()

	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	client := healthpb.NewHealthClient(conn)

	// Wait for the health adapter to report NOT_SERVING via polling.
	require.Eventually(t, func() bool {
		checkCtx, checkCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer checkCancel()
		resp, checkErr := client.Check(checkCtx, &healthpb.HealthCheckRequest{Service: ""})
		return checkErr == nil && resp.GetStatus() == healthpb.HealthCheckResponse_NOT_SERVING
	}, 2*time.Second, 20*time.Millisecond)
}
