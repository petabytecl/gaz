package health

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

// ManagementServer serves health endpoints on a dedicated port.
type ManagementServer struct {
	config        Config
	server        *http.Server
	listener      net.Listener
	shutdownCheck *ShutdownCheck
	logger        *slog.Logger
}

// NewManagementServer creates a new ManagementServer.
func NewManagementServer(
	config Config,
	manager *Manager,
	shutdownCheck *ShutdownCheck,
	logger *slog.Logger,
) *ManagementServer {
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.Handle(config.LivenessPath, manager.NewLivenessHandler())
	mux.Handle(config.ReadinessPath, manager.NewReadinessHandler())
	mux.Handle(config.StartupPath, manager.NewStartupHandler())

	return &ManagementServer{
		config: config,
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", config.Port),
			Handler:           mux,
			ReadHeaderTimeout: DefaultReadHeaderTimeout,
		},
		shutdownCheck: shutdownCheck,
		logger:        logger,
	}
}

// OnStart starts the management server in a background goroutine.
// The listener is created synchronously so port-bind errors are returned
// immediately (and port 0 is resolved before the method returns).
// Implements di.Starter interface.
func (s *ManagementServer) OnStart(ctx context.Context) error {
	lc := net.ListenConfig{}

	lis, err := lc.Listen(ctx, "tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.server.Addr, err)
	}

	s.listener = lis

	s.logger.InfoContext(ctx, "Health server starting",
		slog.Int("port", s.Port()),
		slog.String("liveness-path", s.config.LivenessPath),
		slog.String("readiness-path", s.config.ReadinessPath),
		slog.String("startup-path", s.config.StartupPath),
	)

	go func() {
		if serveErr := s.server.Serve(lis); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			s.logger.ErrorContext(ctx, "Management server error", "error", serveErr)
		}
	}()

	return nil
}

// Port returns the actual port the server is listening on.
// After OnStart this reflects the real bound port (useful when configured with port 0).
// Before OnStart it returns the configured port.
func (s *ManagementServer) Port() int {
	if s.listener != nil {
		if addr, ok := s.listener.Addr().(*net.TCPAddr); ok {
			return addr.Port
		}
	}

	return s.config.Port
}

// OnStop gracefully shuts down the management server.
// It first marks the application as shutting down to fail readiness probes.
// Implements di.Stopper interface.
func (s *ManagementServer) OnStop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "Health server stopping")

	if s.shutdownCheck != nil {
		s.shutdownCheck.MarkShuttingDown()
	}

	// 2. Stop the server
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown management server: %w", err)
	}
	return nil
}
