package health

import (
	"fmt"

	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the health module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	port          int
	livenessPath  string
	readinessPath string
	startupPath   string
}

func defaultModuleConfig() *moduleConfig {
	cfg := DefaultConfig()
	return &moduleConfig{
		port:          cfg.Port,
		livenessPath:  cfg.LivenessPath,
		readinessPath: cfg.ReadinessPath,
		startupPath:   cfg.StartupPath,
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

// NewModule creates a health module with the given options.
// Returns a di.Module that registers health check components.
//
// Components registered:
//   - health.Config (from options or defaults)
//   - *health.ShutdownCheck
//   - *health.Manager
//   - *health.ManagementServer (eager, starts HTTP server)
//
// Example:
//
//	app := gaz.New()
//	app.UseDI(health.NewModule())                           // defaults
//	app.UseDI(health.NewModule(health.WithPort(8081)))      // custom port
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

		// Delegate to existing Module() for component registration
		return Module(c)
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

			return NewManagementServer(cfg, manager, shutdownCheck), nil
		}); err != nil {
		return fmt.Errorf("register management server: %w", err)
	}

	return nil
}
