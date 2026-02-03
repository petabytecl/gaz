package health

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// ManagementServer serves health endpoints on a dedicated port.
type ManagementServer struct {
	config        Config
	server        *http.Server
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
// It returns immediately. Implements di.Starter interface.
func (s *ManagementServer) OnStart(ctx context.Context) error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.ErrorContext(ctx, "Management server error", "error", err)
		}
	}()

	return nil
}

// OnStop gracefully shuts down the management server.
// It first marks the application as shutting down to fail readiness probes.
// Implements di.Stopper interface.
func (s *ManagementServer) OnStop(ctx context.Context) error {
	// 1. Mark shutting down first
	if s.shutdownCheck != nil {
		s.shutdownCheck.MarkShuttingDown()
	}

	// 2. Stop the server
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown management server: %w", err)
	}
	return nil
}
