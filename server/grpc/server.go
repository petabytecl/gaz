package grpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/petabytecl/gaz/di"
)

// ServiceRegistrar is implemented by gRPC services that want to be
// auto-discovered and registered with the gRPC server.
//
// Implementations should register themselves with the provided server:
//
//	type GreeterService struct {
//	    pb.UnimplementedGreeterServer
//	}
//
//	func (s *GreeterService) RegisterService(server grpc.ServiceRegistrar) {
//	    pb.RegisterGreeterServer(server, s)
//	}
type ServiceRegistrar interface {
	RegisterService(server grpc.ServiceRegistrar)
}

// Server is a gRPC server with lifecycle management and auto-discovery.
// It implements di.Starter and di.Stopper for integration with gaz's lifecycle.
type Server struct {
	config    Config
	server    *grpc.Server
	listener  net.Listener
	container *di.Container
	logger    *slog.Logger
	devMode   bool
}

// NewServer creates a new gRPC server with the given configuration.
// The server is not started until OnStart is called.
//
// Parameters:
//   - cfg: Server configuration (port, reflection, message sizes)
//   - logger: Logger for request logging and error reporting
//   - container: DI container for service discovery
//   - devMode: If true, expose panic details in error responses
func NewServer(cfg Config, logger *slog.Logger, container *di.Container, devMode bool) *Server {
	// Create interceptors.
	loggingUnary, loggingStream := NewLoggingInterceptor(logger)
	recoveryUnary, recoveryStream := NewRecoveryInterceptor(logger, devMode)

	// Build server options.
	// Interceptor order: logging first (sees all requests), recovery last (catches panics).
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(loggingUnary, recoveryUnary),
		grpc.ChainStreamInterceptor(loggingStream, recoveryStream),
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
	}

	return &Server{
		config:    cfg,
		server:    grpc.NewServer(opts...),
		container: container,
		logger:    logger,
		devMode:   devMode,
	}
}

// OnStart starts the gRPC server.
// It binds to the configured port, discovers and registers services,
// enables reflection if configured, and starts serving in a goroutine.
// Implements di.Starter.
func (s *Server) OnStart(ctx context.Context) error {
	// Bind port first (fail fast if already in use).
	addr := fmt.Sprintf(":%d", s.config.Port)
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("grpc: bind port %d: %w", s.config.Port, err)
	}
	s.listener = lis

	// Auto-discover and register services.
	registrars, err := di.ResolveAll[ServiceRegistrar](s.container)
	if err != nil {
		// Close listener on error.
		_ = lis.Close()
		return fmt.Errorf("grpc: discover services: %w", err)
	}

	for _, r := range registrars {
		r.RegisterService(s.server)
	}

	// Enable reflection if configured.
	if s.config.Reflection {
		reflection.Register(s.server)
	}

	s.logger.InfoContext(ctx, "gRPC server starting",
		slog.Int("port", s.config.Port),
		slog.Bool("reflection", s.config.Reflection),
		slog.Int("services", len(registrars)),
	)

	// Spawn serve goroutine (non-blocking).
	go func() {
		if serveErr := s.server.Serve(lis); serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
			s.logger.Error("gRPC server error", slog.Any("error", serveErr))
		}
	}()

	return nil
}

// OnStop gracefully shuts down the gRPC server.
// It waits for active connections to complete or forces shutdown on context timeout.
// Implements di.Stopper.
func (s *Server) OnStop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "gRPC server stopping")

	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.InfoContext(ctx, "gRPC server stopped gracefully")
		return nil
	case <-ctx.Done():
		s.server.Stop()
		s.logger.WarnContext(ctx, "gRPC server force stopped")
		return fmt.Errorf("grpc: shutdown: %w", ctx.Err())
	}
}

// GRPCServer returns the underlying grpc.Server for direct access.
// This is useful for registering services manually if needed.
func (s *Server) GRPCServer() *grpc.Server {
	return s.server
}
