package server

import (
	"fmt"
	"net/http"

	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/server/grpc"
	shttp "github.com/petabytecl/gaz/server/http"
)

// ModuleOption configures the unified server module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	// gRPC options
	grpcPort       int
	grpcReflection bool
	grpcDevMode    bool

	// HTTP options
	httpPort    int
	httpHandler http.Handler
}

func defaultModuleConfig() *moduleConfig {
	return &moduleConfig{
		grpcPort:       grpc.DefaultPort,
		grpcReflection: true,
		grpcDevMode:    false,
		httpPort:       shttp.DefaultPort,
		httpHandler:    nil,
	}
}

// WithGRPCPort sets the gRPC server port. Default is 50051.
func WithGRPCPort(port int) ModuleOption {
	return func(c *moduleConfig) {
		c.grpcPort = port
	}
}

// WithHTTPPort sets the HTTP server port. Default is 8080.
func WithHTTPPort(port int) ModuleOption {
	return func(c *moduleConfig) {
		c.httpPort = port
	}
}

// WithGRPCReflection enables or disables gRPC reflection. Default is true.
// When enabled, tools like grpcurl can introspect available services.
func WithGRPCReflection(enabled bool) ModuleOption {
	return func(c *moduleConfig) {
		c.grpcReflection = enabled
	}
}

// WithGRPCDevMode enables development mode for verbose gRPC error messages.
// Default is false.
func WithGRPCDevMode(enabled bool) ModuleOption {
	return func(c *moduleConfig) {
		c.grpcDevMode = enabled
	}
}

// WithHTTPHandler sets the HTTP handler. Default is http.NotFoundHandler().
// For Gateway integration, the Gateway module will typically set this
// to proxy HTTP requests to gRPC services.
func WithHTTPHandler(h http.Handler) ModuleOption {
	return func(c *moduleConfig) {
		c.httpHandler = h
	}
}

// NewModule creates a unified server module with the given options.
// Returns a di.Module that registers both gRPC and HTTP server components.
//
// The module ensures correct startup order:
//   - gRPC server starts first (port binding and service registration)
//   - HTTP server starts second (can depend on gRPC being available)
//
// Shutdown occurs in reverse order (HTTP first, then gRPC).
//
// Components registered:
//   - grpc.Config and *grpc.Server (from server/grpc package)
//   - http.Config and *http.Server (from server/http package)
//
// Example:
//
//	app := gaz.New()
//	app.Use(server.NewModule())                              // defaults
//	app.Use(server.NewModule(server.WithGRPCPort(9090)))     // custom gRPC port
//	app.Use(server.NewModule(server.WithHTTPPort(3000)))     // custom HTTP port
func NewModule(opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return di.NewModuleFunc("server", func(c *di.Container) error {
		// Register gRPC first (starts first, stops last).
		// This ensures gRPC is available before HTTP starts.
		grpcOpts := []grpc.ModuleOption{
			grpc.WithPort(cfg.grpcPort),
			grpc.WithReflection(cfg.grpcReflection),
			grpc.WithDevMode(cfg.grpcDevMode),
		}
		grpcModule := grpc.NewModule(grpcOpts...)
		if err := grpcModule.Register(c); err != nil {
			return fmt.Errorf("register grpc module: %w", err)
		}

		// Register HTTP second (starts second, stops first).
		// HTTP can depend on gRPC being available (e.g., Gateway).
		httpOpts := []shttp.ModuleOption{
			shttp.WithPort(cfg.httpPort),
		}
		if cfg.httpHandler != nil {
			httpOpts = append(httpOpts, shttp.WithHandler(cfg.httpHandler))
		}
		httpModule := shttp.NewModule(httpOpts...)
		if err := httpModule.Register(c); err != nil {
			return fmt.Errorf("register http module: %w", err)
		}

		return nil
	})
}
