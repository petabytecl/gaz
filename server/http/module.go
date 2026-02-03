package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the HTTP module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	port              int
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	readHeaderTimeout time.Duration
	handler           http.Handler
}

func defaultModuleConfig() *moduleConfig {
	cfg := DefaultConfig()
	return &moduleConfig{
		port:              cfg.Port,
		readTimeout:       cfg.ReadTimeout,
		writeTimeout:      cfg.WriteTimeout,
		idleTimeout:       cfg.IdleTimeout,
		readHeaderTimeout: cfg.ReadHeaderTimeout,
		handler:           nil, // Will use NotFoundHandler as default
	}
}

// WithPort sets the HTTP server port. Default is 8080.
func WithPort(port int) ModuleOption {
	return func(c *moduleConfig) {
		c.port = port
	}
}

// WithReadTimeout sets the read timeout. Default is 10 seconds.
func WithReadTimeout(d time.Duration) ModuleOption {
	return func(c *moduleConfig) {
		c.readTimeout = d
	}
}

// WithWriteTimeout sets the write timeout. Default is 30 seconds.
func WithWriteTimeout(d time.Duration) ModuleOption {
	return func(c *moduleConfig) {
		c.writeTimeout = d
	}
}

// WithIdleTimeout sets the idle timeout. Default is 120 seconds.
func WithIdleTimeout(d time.Duration) ModuleOption {
	return func(c *moduleConfig) {
		c.idleTimeout = d
	}
}

// WithReadHeaderTimeout sets the read header timeout. Default is 5 seconds.
func WithReadHeaderTimeout(d time.Duration) ModuleOption {
	return func(c *moduleConfig) {
		c.readHeaderTimeout = d
	}
}

// WithHandler sets the HTTP handler. Default is http.NotFoundHandler().
// For Gateway integration, the Gateway module will set the handler
// to proxy HTTP requests to gRPC services.
func WithHandler(h http.Handler) ModuleOption {
	return func(c *moduleConfig) {
		c.handler = h
	}
}

// NewModule creates an HTTP module with the given options.
// Returns a di.Module that registers HTTP server components.
//
// Components registered:
//   - http.Config (from options or defaults)
//   - *http.Server (eager, starts HTTP server)
//
// Example:
//
//	app := gaz.New()
//	app.UseDI(http.NewModule())                        // defaults
//	app.UseDI(http.NewModule(http.WithPort(3000)))     // custom port
//	app.UseDI(http.NewModule(http.WithHandler(mux)))   // custom handler
func NewModule(opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return di.NewModuleFunc("http", func(c *di.Container) error {
		// Register Config from module options
		httpCfg := Config{
			Port:              cfg.port,
			ReadTimeout:       cfg.readTimeout,
			WriteTimeout:      cfg.writeTimeout,
			IdleTimeout:       cfg.idleTimeout,
			ReadHeaderTimeout: cfg.readHeaderTimeout,
		}
		if err := di.For[Config](c).Instance(httpCfg); err != nil {
			return fmt.Errorf("register http config: %w", err)
		}

		// Store handler for provider closure
		handler := cfg.handler

		// Register Server (eager to participate in lifecycle)
		if err := di.For[*Server](c).
			Eager().
			Provider(func(c *di.Container) (*Server, error) {
				cfg, err := di.Resolve[Config](c)
				if err != nil {
					return nil, fmt.Errorf("resolve http config: %w", err)
				}

				// Try to resolve logger, use default if not available
				logger, err := di.Resolve[*slog.Logger](c)
				if err != nil {
					logger = slog.Default()
				}

				return NewServer(cfg, handler, logger), nil
			}); err != nil {
			return fmt.Errorf("register http server: %w", err)
		}

		return nil
	})
}

// Module registers the HTTP module components.
// It provides:
//   - *Server
//
// It assumes that http.Config has been registered in the container.
func Module(c *di.Container) error {
	// Register Server (eager to participate in lifecycle)
	if err := di.For[*Server](c).
		Eager().
		Provider(func(c *di.Container) (*Server, error) {
			cfg, err := di.Resolve[Config](c)
			if err != nil {
				return nil, fmt.Errorf("resolve http config: %w", err)
			}

			// Try to resolve logger, use default if not available
			logger, err := di.Resolve[*slog.Logger](c)
			if err != nil {
				logger = slog.Default()
			}

			// Use NotFoundHandler as default since no handler is provided
			return NewServer(cfg, nil, logger), nil
		}); err != nil {
		return fmt.Errorf("register http server: %w", err)
	}

	return nil
}
