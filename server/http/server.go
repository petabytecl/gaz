package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
)

// Server is a production-ready HTTP server with lifecycle management.
// It implements di.Starter and di.Stopper interfaces for integration
// with gaz's application lifecycle.
type Server struct {
	config   Config
	server   *http.Server
	logger   *slog.Logger
	handler  http.Handler
	listener net.Listener
	started  atomic.Bool
}

// NewServer creates a new HTTP server with the given configuration.
// If handler is nil, http.NotFoundHandler() is used as default.
// If logger is nil, slog.Default() is used.
func NewServer(cfg Config, handler http.Handler, logger *slog.Logger) *Server {
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		config:  cfg,
		handler: handler,
		logger:  logger,
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		},
	}
}

// SetHandler sets the HTTP handler for the server.
// This method panics if called after the server has started.
// Use this for late-binding scenarios such as Gateway integration.
func (s *Server) SetHandler(h http.Handler) {
	if s.started.Load() {
		panic("http: cannot set handler after server started")
	}
	s.handler = h
	s.server.Handler = h
}

// OnStart binds the port synchronously and then serves in a background goroutine.
// Returns an error immediately if the port cannot be bound (e.g., already in use).
// Implements di.Starter interface.
func (s *Server) OnStart(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("http server listen: %w", err)
	}
	s.listener = ln
	s.started.Store(true)
	s.logger.InfoContext(ctx, "HTTP server starting", "addr", ln.Addr().String())

	go func() {
		if serveErr := s.server.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			s.logger.Error("HTTP server error", "error", serveErr)
		}
	}()

	return nil
}

// OnStop gracefully shuts down the HTTP server.
// It waits for active connections to complete within the context deadline.
// Implements di.Stopper interface.
func (s *Server) OnStop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "HTTP server stopping")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	s.logger.InfoContext(ctx, "HTTP server stopped")
	return nil
}

// Addr returns the server's bound address.
// After OnStart, this returns the actual listener address (useful when port=0).
// Before OnStart, this returns the configured address in the form ":port".
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.server.Addr
}

// Port returns the configured port.
func (s *Server) Port() int {
	return s.config.Port
}
