package vanguard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"connectrpc.com/vanguard"
	"connectrpc.com/vanguard/vanguardgrpc"
	"google.golang.org/grpc"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/health"
	connectpkg "github.com/petabytecl/gaz/server/connect"
)

// Server is a unified server that composes gRPC, Connect, gRPC-Web, and REST
// protocols on a single port via Vanguard transcoder with h2c support.
// It implements di.Starter and di.Stopper for lifecycle management.
type Server struct {
	config             Config
	httpServer         *http.Server
	container          *di.Container
	grpcServer         *grpc.Server
	logger             *slog.Logger
	healthManager      *health.Manager
	userUnknownHandler http.Handler
}

// NewServer creates a new Vanguard server with the given configuration.
// The server is not started until OnStart is called.
//
// Parameters:
//   - cfg: Server configuration (port, timeouts, reflection, health)
//   - logger: Logger for server events (falls back to slog.Default)
//   - container: DI container for service discovery
//   - grpcServer: The raw *grpc.Server from the gRPC module (for vanguardgrpc bridge)
func NewServer(cfg Config, logger *slog.Logger, container *di.Container, grpcServer *grpc.Server) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	// Optionally resolve health.Manager from DI.
	var healthMgr *health.Manager
	if cfg.HealthEnabled {
		if mgr, err := di.Resolve[*health.Manager](container); err == nil {
			healthMgr = mgr
		}
	}

	return &Server{
		config:        cfg,
		container:     container,
		grpcServer:    grpcServer,
		logger:        logger,
		healthManager: healthMgr,
	}
}

// SetUnknownHandler sets a user-defined handler for non-RPC HTTP routes.
// Must be called before OnStart.
func (s *Server) SetUnknownHandler(h http.Handler) {
	s.userUnknownHandler = h
}

// OnStart starts the Vanguard server.
// It discovers Connect services, bridges gRPC services, registers reflection
// and health handlers, builds the Vanguard transcoder, and starts serving
// over h2c on the configured port.
// Implements di.Starter.
func (s *Server) OnStart(ctx context.Context) error {
	// 0. Collect Connect interceptors from DI.
	connectInterceptors := connectpkg.CollectConnectInterceptors(s.container, s.logger)
	var handlerOpts []connect.HandlerOption
	if len(connectInterceptors) > 0 {
		handlerOpts = append(handlerOpts, connect.WithInterceptors(connectInterceptors...))
	}

	// 1. Discover Connect services from DI.
	connectRegistrars, err := di.ResolveAll[connectpkg.Registrar](s.container)
	if err != nil {
		return fmt.Errorf("vanguard: discover connect services: %w", err)
	}

	// 2. Build the unknown handler mux that composes Connect services,
	// reflection, health, and user handlers.
	unknownMux := http.NewServeMux()
	serviceNames := make([]string, 0)

	// Mount Connect service handlers on the mux.
	for _, reg := range connectRegistrars {
		path, handler := reg.RegisterConnect(handlerOpts...)
		unknownMux.Handle(path, handler)
		serviceNames = append(serviceNames, servicePathToName(path))
	}

	// 3. Collect gRPC service names for reflection.
	if s.grpcServer != nil {
		for name := range s.grpcServer.GetServiceInfo() {
			serviceNames = append(serviceNames, name)
		}
	}

	// 4. Register reflection handlers (v1 and v1alpha) if enabled.
	if s.config.Reflection && len(serviceNames) > 0 {
		reflector := grpcreflect.NewStaticReflector(serviceNames...)
		v1Path, v1Handler := grpcreflect.NewHandlerV1(reflector)
		v1AlphaPath, v1AlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
		unknownMux.Handle(v1Path, v1Handler)
		unknownMux.Handle(v1AlphaPath, v1AlphaHandler)
		s.logger.DebugContext(ctx, "gRPC reflection registered",
			slog.Int("services", len(serviceNames)),
		)
	}

	// 5. Mount health endpoints if health.Manager is available.
	if s.healthManager != nil {
		healthMux := buildHealthMux(s.healthManager)
		if healthMux != nil {
			unknownMux.Handle("/healthz", healthMux)
			unknownMux.Handle("/readyz", healthMux)
			unknownMux.Handle("/livez", healthMux)
		}
	}

	// 6. Mount user-defined unknown handler as fallback.
	if s.userUnknownHandler != nil {
		unknownMux.Handle("/", s.userUnknownHandler)
	}

	// 7. Build Vanguard transcoder options.
	transcoderOpts := []vanguard.TranscoderOption{
		vanguard.WithUnknownHandler(unknownMux),
	}

	// 8. Build the transcoder.
	handler, transcoderErr := s.buildTranscoder(transcoderOpts)
	if transcoderErr != nil {
		return transcoderErr
	}

	// 8.5. Apply transport middleware chain (CORS, OTEL, custom middleware).
	handler = collectTransportMiddleware(s.container, s.logger, handler)

	// 9. Configure h2c via Go 1.26+ http.Protocols.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.Port),
		Handler:           handler,
		Protocols:         protocols,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		IdleTimeout:       s.config.IdleTimeout,
	}

	// 10. Verify port is available before spawning goroutine.
	addr := fmt.Sprintf(":%d", s.config.Port)
	var lc net.ListenConfig
	lis, listenErr := lc.Listen(ctx, "tcp", addr)
	if listenErr != nil {
		return fmt.Errorf("vanguard: bind port %d: %w", s.config.Port, listenErr)
	}

	s.logger.InfoContext(ctx, "vanguard server starting",
		slog.Int("port", s.config.Port),
		slog.Int("connect_services", len(connectRegistrars)),
		slog.Int("connect_interceptors", len(connectInterceptors)),
		slog.Bool("reflection", s.config.Reflection),
		slog.Bool("health", s.healthManager != nil),
		slog.Bool("grpc_bridge", s.grpcServer != nil),
	)

	// 11. Start serving in goroutine.
	go func() {
		if serveErr := s.httpServer.Serve(lis); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			s.logger.Error("vanguard server error", slog.Any("error", serveErr))
		}
	}()

	return nil
}

// buildTranscoder creates the Vanguard transcoder.
// Uses vanguardgrpc if a gRPC server is available, otherwise uses plain vanguard transcoder.
func (s *Server) buildTranscoder(opts []vanguard.TranscoderOption) (http.Handler, error) {
	if s.grpcServer != nil {
		transcoder, err := vanguardgrpc.NewTranscoder(s.grpcServer, opts...)
		if err != nil {
			return nil, fmt.Errorf("vanguard: build grpc transcoder: %w", err)
		}
		return transcoder, nil
	}
	// No gRPC server — build a plain transcoder with no services.
	// Connect services are handled by the unknownMux.
	transcoder, err := vanguard.NewTranscoder(nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("vanguard: build transcoder: %w", err)
	}
	return transcoder, nil
}

// OnStop gracefully shuts down the Vanguard server.
// It waits for active connections to drain or forces shutdown on context timeout.
// Implements di.Stopper.
func (s *Server) OnStop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.InfoContext(ctx, "vanguard server stopping")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("vanguard: shutdown: %w", err)
	}

	s.logger.InfoContext(ctx, "vanguard server stopped")
	return nil
}

// servicePathToName converts a Connect service path to a service name.
// Connect handlers return paths like "/package.Service/" which maps to "package.Service".
func servicePathToName(path string) string {
	// Trim leading and trailing slashes.
	name := path
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	if len(name) > 0 && name[len(name)-1] == '/' {
		name = name[:len(name)-1]
	}
	return name
}
