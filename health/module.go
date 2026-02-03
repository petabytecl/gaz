package health

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the health module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	port          int
	livenessPath  string
	readinessPath string
	startupPath   string
	enableGRPC    bool
	grpcInterval  time.Duration
}

func defaultModuleConfig() *moduleConfig {
	cfg := DefaultConfig()
	return &moduleConfig{
		port:          cfg.Port,
		livenessPath:  cfg.LivenessPath,
		readinessPath: cfg.ReadinessPath,
		startupPath:   cfg.StartupPath,
		enableGRPC:    false,
		grpcInterval:  DefaultGRPCCheckInterval,
	}
}

// WithPort sets the health server port. Default is 9090.
func WithPort(port int) ModuleOption {
	return func(c *moduleConfig) {
		c.port = port
	}
}

// WithLivenessPath sets the liveness endpoint path. Default is "/live".
func WithLivenessPath(path string) ModuleOption {
	return func(c *moduleConfig) {
		c.livenessPath = path
	}
}

// WithReadinessPath sets the readiness endpoint path. Default is "/ready".
func WithReadinessPath(path string) ModuleOption {
	return func(c *moduleConfig) {
		c.readinessPath = path
	}
}

// WithStartupPath sets the startup endpoint path. Default is "/startup".
func WithStartupPath(path string) ModuleOption {
	return func(c *moduleConfig) {
		c.startupPath = path
	}
}

// WithGRPC enables the gRPC health server.
// When enabled, GRPCServer is registered and can be resolved by the gRPC server module.
// The GRPCServer syncs status from the Manager's readiness checks to the standard
// grpc.health.v1.Health service.
func WithGRPC() ModuleOption {
	return func(c *moduleConfig) {
		c.enableGRPC = true
	}
}

// WithGRPCInterval sets the gRPC health check polling interval.
// Default is 5 seconds. Only effective when WithGRPC() is also used.
func WithGRPCInterval(d time.Duration) ModuleOption {
	return func(c *moduleConfig) {
		if d > 0 {
			c.grpcInterval = d
		}
	}
}

// NewModule creates a health module with the given options.
// Returns a di.Module that registers health check components.
//
// Components registered:
//   - health.Config (from options or defaults)
//   - *health.ShutdownCheck
//   - *health.Manager
//   - *health.ManagementServer (eager, starts HTTP server)
//   - *health.GRPCServer (eager, if WithGRPC() is used)
//
// Example:
//
//	app := gaz.New()
//	app.UseDI(health.NewModule())                           // defaults
//	app.UseDI(health.NewModule(health.WithPort(8081)))      // custom port
//	app.UseDI(health.NewModule(health.WithGRPC()))          // enable gRPC health
func NewModule(opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return di.NewModuleFunc("health", func(c *di.Container) error {
		// Register Config from module options
		healthCfg := Config{
			Port:          cfg.port,
			LivenessPath:  cfg.livenessPath,
			ReadinessPath: cfg.readinessPath,
			StartupPath:   cfg.startupPath,
		}
		if err := di.For[Config](c).Instance(healthCfg); err != nil {
			return fmt.Errorf("register health config: %w", err)
		}

		// Register core components
		if err := Module(c); err != nil {
			return err
		}

		// Register GRPCServer if enabled
		if cfg.enableGRPC {
			if err := registerGRPCServer(c, cfg.grpcInterval); err != nil {
				return err
			}
		}

		return nil
	})
}

// Module registers the health module components.
// It provides:
// - *ShutdownCheck
// - *Manager
// - *ManagementServer
//
// It assumes that health.Config has been registered in the container
// (e.g. via gaz.WithHealthChecks or manual registration).
func Module(c *di.Container) error {
	// Register ShutdownCheck
	if err := di.For[*ShutdownCheck](c).
		ProviderFunc(func(_ *di.Container) *ShutdownCheck {
			return NewShutdownCheck()
		}); err != nil {
		return fmt.Errorf("register shutdown check: %w", err)
	}

	// Register Manager
	if err := di.For[*Manager](c).
		Provider(func(c *di.Container) (*Manager, error) {
			m := NewManager()

			// Wire up shutdown check
			shutdownCheck, err := di.Resolve[*ShutdownCheck](c)
			if err != nil {
				return nil, err
			}

			// Register as readiness check
			m.AddReadinessCheck("shutdown", shutdownCheck.Check)

			return m, nil
		}); err != nil {
		return fmt.Errorf("register manager: %w", err)
	}

	// Register ManagementServer (implements di.Starter and di.Stopper)
	if err := di.For[*ManagementServer](c).
		Eager().
		Provider(func(c *di.Container) (*ManagementServer, error) {
			cfg, err := di.Resolve[Config](c)
			if err != nil {
				return nil, err
			}

			manager, err := di.Resolve[*Manager](c)
			if err != nil {
				return nil, err
			}

			shutdownCheck, err := di.Resolve[*ShutdownCheck](c)
			if err != nil {
				return nil, err
			}

			// Logger is optional - use default if not registered
			logger := slog.Default()
			if l, resolveErr := di.Resolve[*slog.Logger](c); resolveErr == nil {
				logger = l
			}

			return NewManagementServer(cfg, manager, shutdownCheck, logger), nil
		}); err != nil {
		return fmt.Errorf("register management server: %w", err)
	}

	return nil
}

// registerGRPCServer registers the GRPCServer component.
func registerGRPCServer(c *di.Container, interval time.Duration) error {
	// Register GRPCServer (implements di.Starter and di.Stopper)
	if err := di.For[*GRPCServer](c).
		Eager().
		Provider(func(c *di.Container) (*GRPCServer, error) {
			manager, err := di.Resolve[*Manager](c)
			if err != nil {
				return nil, fmt.Errorf("resolve health manager: %w", err)
			}

			// Logger is optional - use default if not registered
			logger := slog.Default()
			if l, resolveErr := di.Resolve[*slog.Logger](c); resolveErr == nil {
				logger = l
			}

			return NewGRPCServer(manager, logger, WithCheckInterval(interval)), nil
		}); err != nil {
		return fmt.Errorf("register grpc health server: %w", err)
	}

	return nil
}
