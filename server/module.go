package server

import (
	"fmt"
	"net/http"

	"github.com/spf13/pflag"

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
// For CLI flag integration, use [NewModuleWithFlags] instead.
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
		return registerServerComponents(cfg, c)
	})
}

// NewModuleWithFlags creates a unified server module with CLI flag support.
// The flags are registered with the provided FlagSet and their values
// are read when the module registers components.
//
// Flags registered:
//   - --grpc-port        gRPC server port (default from options or 50051)
//   - --http-port        HTTP server port (default from options or 8080)
//   - --grpc-reflection  Enable gRPC reflection (default from options or true)
//   - --grpc-dev-mode    Enable gRPC development mode (default from options or false)
//
// Module options can set initial defaults, which flags can then override at runtime.
//
// Example:
//
//	rootCmd := &cobra.Command{}
//	app := gaz.New()
//	app.Use(server.NewModuleWithFlags(rootCmd.Flags()))
//	// Now --grpc-port, --http-port, --grpc-reflection, --grpc-dev-mode are available.
//
// Example with option defaults:
//
//	app.Use(server.NewModuleWithFlags(rootCmd.Flags(), server.WithGRPCPort(9090)))
//	// --grpc-port flag defaults to 9090, user can override with --grpc-port=8888
func NewModuleWithFlags(fs *pflag.FlagSet, opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Define flags with initial values from options.
	// Flag values are bound to pointers, allowing deferred evaluation.
	grpcPortFlag := fs.Int("grpc-port", cfg.grpcPort, "gRPC server port")
	httpPortFlag := fs.Int("http-port", cfg.httpPort, "HTTP server port")
	grpcReflectionFlag := fs.Bool("grpc-reflection", cfg.grpcReflection, "Enable gRPC reflection")
	grpcDevModeFlag := fs.Bool("grpc-dev-mode", cfg.grpcDevMode, "Enable gRPC development mode")

	return di.NewModuleFunc("server", func(c *di.Container) error {
		// CRITICAL: Read flag values HERE (deferred evaluation).
		// When this function EXECUTES (during app.Run(), after flag parsing),
		// the flag pointers contain the PARSED flag values, not the defaults.
		//
		// Flow:
		// 1. NewModuleWithFlags(fs) called -> flags registered on fs
		// 2. cobra parses args -> writes "9090" to *grpcPortFlag
		// 3. app.Run() -> container resolves eager services -> this function runs
		// 4. *grpcPortFlag is now 9090, not 50051
		cfg.grpcPort = *grpcPortFlag
		cfg.httpPort = *httpPortFlag
		cfg.grpcReflection = *grpcReflectionFlag
		cfg.grpcDevMode = *grpcDevModeFlag

		return registerServerComponents(cfg, c)
	})
}

// registerServerComponents is shared by both NewModule and NewModuleWithFlags.
// It registers gRPC and HTTP server components with the given config values.
func registerServerComponents(cfg *moduleConfig, c *di.Container) error {
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
}
