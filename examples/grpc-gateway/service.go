package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/examples/grpc-gateway/proto"
	"github.com/petabytecl/gaz/server/gateway"
	servergrpc "github.com/petabytecl/gaz/server/grpc"
)

// GreeterService implements the Greeter gRPC service and
// handles registration for both gRPC server and Gateway.
type GreeterService struct {
	hello.UnimplementedGreeterServer
	logger *slog.Logger
}

// NewGreeterService creates a new GreeterService.
func NewGreeterService(c *gaz.Container) (*GreeterService, error) {
	// Logger is optional, but good practice
	logger := slog.Default()
	if gaz.Has[*slog.Logger](c) {
		var err error
		logger, err = gaz.Resolve[*slog.Logger](c)
		if err != nil {
			return nil, err
		}
	}

	return &GreeterService{
		logger: logger,
	}, nil
}

// SayHello implements hello.GreeterServer.
func (s *GreeterService) SayHello(ctx context.Context, req *hello.HelloRequest) (*hello.HelloReply, error) {
	name := req.GetName()
	if name == "" {
		name = "World"
	}
	s.logger.InfoContext(ctx, "Handling SayHello", "name", name)
	return &hello.HelloReply{
		Message: fmt.Sprintf("Hello, %s!", name),
	}, nil
}

// RegisterService registers the service with the gRPC server.
// Implements server/grpc.Registrar.
func (s *GreeterService) RegisterService(registrar grpc.ServiceRegistrar) {
	hello.RegisterGreeterServer(registrar, s)
}

// RegisterGateway registers the service with the HTTP Gateway.
// Implements server/gateway.Registrar.
func (s *GreeterService) RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return hello.RegisterGreeterHandler(ctx, mux, conn)
}

// Ensure interface compliance
var _ servergrpc.Registrar = (*GreeterService)(nil)
var _ gateway.Registrar = (*GreeterService)(nil)
