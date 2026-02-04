package health

import (
	"fmt"
	"log/slog"

	"github.com/petabytecl/gaz/di"
)

// Module registers the health module components.
// It provides:
// - *ShutdownCheck
// - *Manager
// - *ManagementServer
//
// It assumes that health.Config has been registered in the container
// (e.g. via health/module.New() or manual registration).
//
// For CLI flag support, use the health/module subpackage:
//
//	import healthmod "github.com/petabytecl/gaz/health/module"
//	app.Use(healthmod.New())
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
