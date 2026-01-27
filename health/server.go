package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ManagementServer serves health endpoints on a dedicated port.
type ManagementServer struct {
	config        Config
	server        *http.Server
	shutdownCheck *ShutdownCheck
}

// NewManagementServer creates a new ManagementServer.
func NewManagementServer(config Config, manager *Manager, shutdownCheck *ShutdownCheck) *ManagementServer {
	mux := http.NewServeMux()
	mux.Handle(config.LivenessPath, manager.NewLivenessHandler())
	mux.Handle(config.ReadinessPath, manager.NewReadinessHandler())
	mux.Handle(config.StartupPath, manager.NewStartupHandler())

	return &ManagementServer{
		config: config,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", config.Port),
			Handler: mux,
		},
		shutdownCheck: shutdownCheck,
	}
}

// Start starts the management server in a background goroutine.
// It returns immediately.
func (s *ManagementServer) Start(ctx context.Context) error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// In a real application, we would want to log this error.
			// Since we don't have a configured logger yet, we just print it.
			fmt.Printf("Management server error: %v\n", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the management server.
// It first marks the application as shutting down to fail readiness probes.
func (s *ManagementServer) Stop(ctx context.Context) error {
	// 1. Mark shutting down first
	if s.shutdownCheck != nil {
		s.shutdownCheck.MarkShuttingDown()
	}

	// 2. Stop the server
	return s.server.Shutdown(ctx)
}
