package health

import (
	"context"

	"github.com/petabytecl/gaz"
)

// Module registers the health module components.
// It provides:
// - *ShutdownCheck
// - *Manager
// - *ManagementServer
//
// It assumes that health.Config has been registered in the container
// (e.g. via gaz.WithHealthChecks or manual registration).
func Module(c *gaz.Container) error {
	// Register ShutdownCheck
	if err := gaz.For[*ShutdownCheck](c).
		ProviderFunc(func(_ *gaz.Container) *ShutdownCheck {
			return NewShutdownCheck()
		}); err != nil {
		return err
	}

	// Register Manager
	if err := gaz.For[*Manager](c).
		Provider(func(c *gaz.Container) (*Manager, error) {
			m := NewManager()

			// Wire up shutdown check
			shutdownCheck, err := gaz.Resolve[*ShutdownCheck](c)
			if err != nil {
				return nil, err
			}

			// Register as readiness check
			m.AddReadinessCheck("shutdown", shutdownCheck.Check)

			return m, nil
		}); err != nil {
		return err
	}

	// Register ManagementServer
	if err := gaz.For[*ManagementServer](c).
		OnStart(func(ctx context.Context, s *ManagementServer) error {
			return s.OnStart(ctx)
		}).
		OnStop(func(ctx context.Context, s *ManagementServer) error {
			return s.OnStop(ctx)
		}).
		Eager().
		Provider(func(c *gaz.Container) (*ManagementServer, error) {
			cfg, err := gaz.Resolve[Config](c)
			if err != nil {
				return nil, err
			}

			manager, err := gaz.Resolve[*Manager](c)
			if err != nil {
				return nil, err
			}

			shutdownCheck, err := gaz.Resolve[*ShutdownCheck](c)
			if err != nil {
				return nil, err
			}

			return NewManagementServer(cfg, manager, shutdownCheck), nil
		}); err != nil {
		return err
	}

	return nil
}
