package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"google.golang.org/grpc"

	"github.com/petabytecl/gaz"
	hello "github.com/petabytecl/gaz/examples/vanguard/proto"
	"github.com/petabytecl/gaz/examples/vanguard/proto/helloconnect"
	connectpkg "github.com/petabytecl/gaz/server/connect"
	servergrpc "github.com/petabytecl/gaz/server/grpc"
)

// GreeterService implements both gRPC and Connect registration for the Greeter service.
// It is auto-discovered by the server module via the grpc.Registrar and connect.Registrar
// interfaces.
type GreeterService struct {
	hello.UnimplementedGreeterServer
	logger *slog.Logger
}

// Compile-time interface checks.
var (
	_ servergrpc.Registrar = (*GreeterService)(nil)
	_ connectpkg.Registrar = (*GreeterService)(nil)
)

// NewGreeterService creates a new GreeterService.
func NewGreeterService(c *gaz.Container) (*GreeterService, error) {
	logger := slog.Default()
	if gaz.Has[*slog.Logger](c) {
		resolved, err := gaz.Resolve[*slog.Logger](c)
		if err != nil {
			return nil, fmt.Errorf("resolve logger: %w", err)
		}
		logger = resolved
	}

	return &GreeterService{
		logger: logger,
	}, nil
}

// SayHello implements the gRPC GreeterServer interface.
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
// Implements server/grpc.Registrar for auto-discovery.
func (s *GreeterService) RegisterService(registrar grpc.ServiceRegistrar) {
	hello.RegisterGreeterServer(registrar, s)
}

// RegisterConnect registers the service with the Connect handler.
// Implements server/connect.Registrar for auto-discovery.
// Uses a greeterConnectAdapter to bridge the gRPC-style SayHello signature
// to the Connect GreeterHandler interface.
func (s *GreeterService) RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler) {
	return helloconnect.NewGreeterHandler(&greeterConnectAdapter{svc: s}, opts...)
}

// greeterConnectAdapter adapts GreeterService (gRPC signature) to the
// helloconnect.GreeterHandler interface (Connect signature).
type greeterConnectAdapter struct {
	svc *GreeterService
}

// SayHello implements helloconnect.GreeterHandler by unwrapping the Connect
// request, delegating to the gRPC-style handler, and wrapping the response.
func (a *greeterConnectAdapter) SayHello(
	ctx context.Context,
	req *connect.Request[hello.HelloRequest],
) (*connect.Response[hello.HelloReply], error) {
	reply, err := a.svc.SayHello(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(reply), nil
}
