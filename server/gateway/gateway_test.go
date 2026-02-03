package gateway

import (
	"context"
	"log/slog"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/petabytecl/gaz/di"
)

// GatewayTestSuite tests Gateway lifecycle and functionality.
type GatewayTestSuite struct {
	suite.Suite
}

func TestGatewayTestSuite(t *testing.T) {
	suite.Run(t, new(GatewayTestSuite))
}

func (s *GatewayTestSuite) TestNewGateway() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	gw := NewGateway(cfg, logger, container, false, nil)

	s.Require().NotNil(gw)
	s.Require().Equal(cfg.Port, gw.config.Port)
	s.Require().Equal(cfg.GRPCTarget, gw.config.GRPCTarget)
	s.Require().Equal(container, gw.container)
	s.Require().Equal(logger, gw.logger)
	s.Require().False(gw.devMode)
}

func (s *GatewayTestSuite) TestNewGateway_NilLogger() {
	cfg := DefaultConfig()
	container := di.New()

	gw := NewGateway(cfg, nil, container, false, nil)

	s.Require().NotNil(gw)
	s.Require().NotNil(gw.logger, "Nil logger should default to slog.Default()")
}

func (s *GatewayTestSuite) TestNewGateway_DevMode() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	gw := NewGateway(cfg, logger, container, true, nil)

	s.Require().True(gw.devMode)
}

func (s *GatewayTestSuite) TestGateway_OnStart_CreatesConnection() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	gw := NewGateway(cfg, logger, container, false, nil)

	// OnStart creates gRPC connection (note: connection creation succeeds
	// even if server is not running - it's lazy).
	err := gw.OnStart(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(gw.conn, "Connection should be created")
	s.Require().NotNil(gw.mux, "ServeMux should be created")
	s.Require().NotNil(gw.handler, "Handler should be created")

	// Cleanup.
	_ = gw.OnStop(context.Background())
}

func (s *GatewayTestSuite) TestGateway_OnStart_DiscoveryNoServices() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	gw := NewGateway(cfg, logger, container, false, nil)

	// Works with zero registrars.
	err := gw.OnStart(context.Background())
	s.Require().NoError(err)

	// Cleanup.
	_ = gw.OnStop(context.Background())
}

func (s *GatewayTestSuite) TestGateway_OnStart_DiscoveryWithServices() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	// Register a mock registrar.
	mockReg := &mockRegistrar{}
	err := di.For[*mockRegistrar](container).Instance(mockReg)
	s.Require().NoError(err)

	gw := NewGateway(cfg, logger, container, false, nil)

	err = gw.OnStart(context.Background())
	s.Require().NoError(err)
	s.Require().True(mockReg.called, "Registrar should have been called")

	// Cleanup.
	_ = gw.OnStop(context.Background())
}

func (s *GatewayTestSuite) TestGateway_OnStart_EmptyGRPCTarget() {
	cfg := Config{
		Port:       8080,
		GRPCTarget: "", // Empty - should use default.
	}
	logger := slog.Default()
	container := di.New()

	gw := NewGateway(cfg, logger, container, false, nil)

	err := gw.OnStart(context.Background())
	s.Require().NoError(err)

	// Cleanup.
	_ = gw.OnStop(context.Background())
}

func (s *GatewayTestSuite) TestGateway_OnStop() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	gw := NewGateway(cfg, logger, container, false, nil)

	// Start first.
	err := gw.OnStart(context.Background())
	s.Require().NoError(err)

	// Stop should close connection cleanly.
	err = gw.OnStop(context.Background())
	s.Require().NoError(err)
}

func (s *GatewayTestSuite) TestGateway_OnStop_NilConnection() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	gw := NewGateway(cfg, logger, container, false, nil)
	// Do NOT call OnStart.

	// OnStop should handle nil connection gracefully.
	err := gw.OnStop(context.Background())
	s.Require().NoError(err)
}

func (s *GatewayTestSuite) TestGateway_Handler() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	gw := NewGateway(cfg, logger, container, false, nil)

	// Before OnStart, handler is not nil (it's a DynamicHandler defaulting to 404).
	s.Require().NotNil(gw.Handler())

	// After OnStart, handler is still not nil.
	err := gw.OnStart(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(gw.Handler())

	// Cleanup.
	_ = gw.OnStop(context.Background())
}

func (s *GatewayTestSuite) TestGateway_OnStart_RegistrarError() {
	cfg := DefaultConfig()
	logger := slog.Default()
	container := di.New()

	// Register a mock registrar that returns an error.
	mockReg := &mockRegistrar{returnErr: true}
	err := di.For[*mockRegistrar](container).Instance(mockReg)
	s.Require().NoError(err)

	gw := NewGateway(cfg, logger, container, false, nil)

	err = gw.OnStart(context.Background())
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "register service")
}

// mockRegistrar implements Registrar for testing.
type mockRegistrar struct {
	called    bool
	returnErr bool
}

func (m *mockRegistrar) RegisterGateway(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	m.called = true
	if m.returnErr {
		return errMockRegistrar
	}
	return nil
}

var errMockRegistrar = &mockError{msg: "mock registrar error"}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
