package server

import (
	"fmt"
	"net/http"

	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz"
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
// Returns a gaz.Module that registers CLI flags for port configuration.
// Use this when building CLI applications with gaz.WithCobra().
//
// Flags:
//   - --grpc-port        gRPC server port (default: 50051)
//   - --http-port        HTTP server port (default: 8080)
//   - --grpc-reflection  Enable gRPC reflection (default: true)
//   - --grpc-dev-mode    Enable gRPC development mode (default: false)
//
// Module options can set initial defaults, which flags can then override at runtime.
//
// Example:
//
//	app := gaz.New(gaz.WithCobra(cmd)).
//	    Use(server.NewModuleWithFlags()).
//	    Build()
//
// Example with option defaults:
//
//	app := gaz.New(gaz.WithCobra(cmd)).
//	    Use(server.NewModuleWithFlags(server.WithGRPCPort(9090))). // default 9090
//	    Build()
//	// --grpc-port flag defaults to 9090, user can override with --grpc-port=8888
func NewModuleWithFlags(opts ...ModuleOption) gaz.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return gaz.NewModule("server").
		Flags(func(fs *pflag.FlagSet) {
			// IntVar/BoolVar bind flag values directly to cfg fields.
			// Default values come FROM cfg (which may have been set by opts).
			// When flags are parsed, values are written TO cfg via these pointers.
			fs.IntVar(&cfg.grpcPort, "grpc-port", cfg.grpcPort, "gRPC server port")
			fs.IntVar(&cfg.httpPort, "http-port", cfg.httpPort, "HTTP server port")
			fs.BoolVar(&cfg.grpcReflection, "grpc-reflection", cfg.grpcReflection, "Enable gRPC reflection")
			fs.BoolVar(&cfg.grpcDevMode, "grpc-dev-mode", cfg.grpcDevMode, "Enable gRPC development mode")
		}).
		Provide(func(c *gaz.Container) error {
			// CRITICAL: This closure captures cfg by pointer reference.
			// When this provider EXECUTES (during app.Run(), after flag parsing),
			// cfg.grpcPort etc. contain the PARSED flag values, not the defaults.
			//
			// Flow:
			// 1. NewModuleWithFlags() called -> cfg created with defaults (or opts)
			// 2. app.Use() called -> Flags() binds &cfg.grpcPort to --grpc-port
			// 3. cobra parses args -> writes "9090" to cfg.grpcPort via pointer
			// 4. app.Run() -> container resolves eager services -> this Provide() runs
			// 5. cfg.grpcPort is now 9090, not 50051
			return registerServerComponents(cfg, c)
		}).
		Build()
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
