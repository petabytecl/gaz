package grpc

import (
	"fmt"
	"log/slog"

	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the gRPC module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	port       int
	reflection bool
	devMode    bool
}

func defaultModuleConfig() *moduleConfig {
	cfg := DefaultConfig()
	return &moduleConfig{
		port:       cfg.Port,
		reflection: cfg.Reflection,
		devMode:    false,
	}
}

// WithPort sets the gRPC server port. Default is 50051.
func WithPort(port int) ModuleOption {
	return func(c *moduleConfig) {
		c.port = port
	}
}

// WithReflection enables or disables gRPC reflection. Default is true.
func WithReflection(enabled bool) ModuleOption {
	return func(c *moduleConfig) {
		c.reflection = enabled
	}
}

// WithDevMode enables development mode for verbose error messages. Default is false.
func WithDevMode(enabled bool) ModuleOption {
	return func(c *moduleConfig) {
		c.devMode = enabled
	}
}

// NewModule creates a gRPC module with the given options.
// Returns a di.Module that registers gRPC server components.
//
// Components registered:
//   - grpc.Config (from options or defaults)
//   - *grpc.Server (eager, starts on app start)
//
// The Server is registered as Eager so it starts when the application starts.
//
// Example:
//
//	app := gaz.New()
//	app.Use(grpc.NewModule())                          // defaults
//	app.Use(grpc.NewModule(grpc.WithPort(9090)))       // custom port
//	app.Use(grpc.NewModule(grpc.WithReflection(false))) // disable reflection
func NewModule(opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return di.NewModuleFunc("grpc", func(c *di.Container) error {
		// Register Config from module options.
		grpcCfg := Config{
			Port:           cfg.port,
			Reflection:     cfg.reflection,
			MaxRecvMsgSize: DefaultMaxMsgSize,
			MaxSendMsgSize: DefaultMaxMsgSize,
		}
		if err := di.For[Config](c).Instance(grpcCfg); err != nil {
			return fmt.Errorf("register grpc config: %w", err)
		}

		// Delegate to existing Module() for component registration.
		return Module(c, cfg.devMode)
	})
}

// Module registers the gRPC module components.
// It provides:
//   - *Server (eager, starts on app start)
//
// It assumes that grpc.Config has been registered in the container
// (e.g., via NewModule or manual registration).
//
// The devMode parameter controls whether panic details are exposed in error responses.
func Module(c *di.Container, devMode bool) error {
	// Register Server (implements di.Starter and di.Stopper).
	if err := di.For[*Server](c).
		Eager().
		Provider(func(c *di.Container) (*Server, error) {
			cfg, err := di.Resolve[Config](c)
			if err != nil {
				return nil, fmt.Errorf("resolve grpc config: %w", err)
			}

			logger, err := di.Resolve[*slog.Logger](c)
			if err != nil {
				return nil, fmt.Errorf("resolve logger: %w", err)
			}

			return NewServer(cfg, logger, c, devMode), nil
		}); err != nil {
		return fmt.Errorf("register grpc server: %w", err)
	}

	return nil
}
