// Package main demonstrates an HTTP server with graceful shutdown and health checks.
//
// This example shows:
//   - Custom HTTP server with lifecycle hooks
//   - Graceful shutdown using context timeout
//   - Health check integration via health.WithHealthChecks
//   - Proper server.Shutdown() for connection draining
//
// Run with: go run .
// Test with: curl http://localhost:8080/hello
// Health: curl http://localhost:9090/ready
// Stop with: Ctrl+C (graceful) or Ctrl+C twice (force)
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/health"
)

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port            int           `json:"port" yaml:"port"`
	ReadTimeout     time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout" yaml:"write_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout"`
}

// DefaultServerConfig returns sensible defaults for the HTTP server.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:            8080,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// Server wraps http.Server with lifecycle management.
type Server struct {
	httpServer *http.Server
	config     ServerConfig
}

// NewServer creates a new HTTP server with the given configuration and handler.
func NewServer(config ServerConfig, handler http.Handler) *Server {
	return &Server{
		config: config,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", config.Port),
			Handler:      handler,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		},
	}
}

// OnStart starts the HTTP server in a goroutine.
// It returns immediately, allowing other services to start.
func (s *Server) OnStart(_ context.Context) error {
	go func() {
		log.Printf("HTTP server listening on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	return nil
}

// OnStop gracefully shuts down the HTTP server.
// It waits for active connections to complete within the context deadline.
func (s *Server) OnStop(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

// Handler creates the HTTP request multiplexer.
type Handler struct {
	mux *http.ServeMux
}

// NewHandler creates a new HTTP handler with routes.
func NewHandler() *Handler {
	h := &Handler{mux: http.NewServeMux()}
	h.registerRoutes()
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) registerRoutes() {
	h.mux.HandleFunc("GET /hello", h.handleHello)
	h.mux.HandleFunc("GET /", h.handleRoot)
}

func (h *Handler) handleRoot(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"service": "http-server-example",
		"status":  "running",
	})
}

func (h *Handler) handleHello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "World"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Hello, %s!", name),
	})
}

func main() {
	// Create app with health checks enabled
	app := gaz.New(
		gaz.WithShutdownTimeout(30*time.Second),
		health.WithHealthChecks(health.DefaultConfig()),
	)

	// Register configuration using For[T]()
	config := DefaultServerConfig()
	if err := gaz.For[ServerConfig](app.Container()).Instance(config); err != nil {
		log.Fatalf("Failed to register config: %v", err)
	}

	// Register HTTP handler using For[T]()
	if err := gaz.For[*Handler](app.Container()).ProviderFunc(func(_ *gaz.Container) *Handler {
		return NewHandler()
	}); err != nil {
		log.Fatalf("Failed to register handler: %v", err)
	}

	// Register HTTP server (implements di.Starter and di.Stopper)
	if err := gaz.For[*Server](app.Container()).
		Eager(). // Start immediately
		Provider(func(c *gaz.Container) (*Server, error) {
			cfg, err := gaz.Resolve[ServerConfig](c)
			if err != nil {
				return nil, err
			}
			handler, err := gaz.Resolve[*Handler](c)
			if err != nil {
				return nil, err
			}
			return NewServer(cfg, handler), nil
		}); err != nil {
		log.Fatalf("Failed to register server: %v", err)
	}

	// Run the application
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
