package gateway

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/di"
	servergrpc "github.com/petabytecl/gaz/server/grpc"
)

type AutoConfigTestSuite struct {
	suite.Suite
}

func TestAutoConfigTestSuite(t *testing.T) {
	suite.Run(t, new(AutoConfigTestSuite))
}

func (s *AutoConfigTestSuite) TestGateway_AutoConfig_FromGRPC() {
	app := gaz.New()

	// Register grpc config with custom port (9090)
	customGrpcCfg := servergrpc.Config{
		Port:                9090,
		Reflection:          true,
		MaxRecvMsgSize:      4 * 1024 * 1024,
		MaxSendMsgSize:      4 * 1024 * 1024,
		HealthEnabled:       true,
		HealthCheckInterval: 5000000000,
		DevMode:             false,
	}

	// Register the config directly
	err := gaz.For[servergrpc.Config](app.Container()).Provider(func(_ *di.Container) (servergrpc.Config, error) {
		return customGrpcCfg, nil
	})
	s.Require().NoError(err)

	// Use Gateway module
	app.Use(NewModule())

	err = app.Build()
	s.Require().NoError(err)

	// Resolve Gateway
	gw, err := di.Resolve[*Gateway](app.Container())
	s.Require().NoError(err)

	// Check if GRPCTarget was updated to match grpc port
	s.Require().Equal("localhost:9090", gw.config.GRPCTarget)
}

func (s *AutoConfigTestSuite) TestGateway_AutoConfig_FromGRPC_DefaultPort() {
	app := gaz.New()

	// Register grpc config with default port (50051)
	customGrpcCfg := servergrpc.Config{
		Port:                50051,
		Reflection:          true,
		MaxRecvMsgSize:      4 * 1024 * 1024,
		MaxSendMsgSize:      4 * 1024 * 1024,
		HealthEnabled:       true,
		HealthCheckInterval: 5000000000,
		DevMode:             false,
	}

	err := gaz.For[servergrpc.Config](app.Container()).Provider(func(_ *di.Container) (servergrpc.Config, error) {
		return customGrpcCfg, nil
	})
	s.Require().NoError(err)

	app.Use(NewModule())
	err = app.Build()
	s.Require().NoError(err)

	gw, err := di.Resolve[*Gateway](app.Container())
	s.Require().NoError(err)

	// Should remain default
	s.Require().Equal("localhost:50051", gw.config.GRPCTarget)
}

func (s *AutoConfigTestSuite) TestGateway_AutoConfig_NoGRPC() {
	app := gaz.New()

	// Use Gateway module WITHOUT gRPC config registered
	app.Use(NewModule())

	err := app.Build()
	s.Require().NoError(err)

	// Resolve Gateway
	gw, err := di.Resolve[*Gateway](app.Container())
	s.Require().NoError(err)

	// Check if GRPCTarget remains default
	s.Require().Equal("localhost:50051", gw.config.GRPCTarget)
}
