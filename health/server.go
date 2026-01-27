package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ManagementServer serves health endpoints on a dedicated port.
type ManagementServer struct {
	config Config
	server *http.Server
}

// NewManagementServer creates a new ManagementServer.
func NewManagementServer(config Config, manager *Manager) *ManagementServer {
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

	// Give the server a tiny bit of time to bind, just in case of immediate failure?
	// No, that's flaky. We just return.
	return nil
}

// Stop gracefully shuts down the management server.
func (s *ManagementServer) Stop(ctx context.Context) error {
	// Create a context with timeout if the parent context doesn't have one,
	// but usually the caller (App) provides a context with timeout.
	return s.server.Shutdown(ctx)
}
