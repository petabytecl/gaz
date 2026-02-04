// Package module provides the gaz.Module for health configuration with CLI flags.
package module

import (
	"fmt"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/health"
)

// New creates a health module that provides health.Config with CLI flags.
// This module registers CLI flags for health server configuration and
// provides the health components (ShutdownCheck, Manager, ManagementServer).
//
// Usage:
//
//	import healthmod "github.com/petabytecl/gaz/health/module"
//
//	app := gaz.New(gaz.WithCobra(rootCmd))
//	app.Use(healthmod.New())
//
// Flags registered:
//
//	--health-port           Health server port (default: 9090)
//	--health-liveness-path  Liveness endpoint path (default: /live)
//	--health-readiness-path Readiness endpoint path (default: /ready)
//	--health-startup-path   Startup endpoint path (default: /startup)
//
//nolint:ireturn // Module is the expected return type for gaz modules
func New() gaz.Module {
	defaultCfg := health.DefaultConfig()

	return gaz.NewModule("health-flags").
		Flags(defaultCfg.Flags).
		Provide(func(c *gaz.Container) error {
			// Register Config provider
			return gaz.For[health.Config](c).Provider(func(c *gaz.Container) (health.Config, error) {
				// Start with the default configuration which has flags bound to it
				cfg := defaultCfg

				// Try to load from config manager if available
				pv, pvErr := gaz.Resolve[*gaz.ProviderValues](c)
				if pvErr == nil {
					if unmarshalErr := pv.UnmarshalKey(cfg.Namespace(), &cfg); unmarshalErr != nil {
						// Ignore error, use defaults (key may not exist)
						_ = unmarshalErr
					}
				}

				cfg.SetDefaults()
				if validateErr := cfg.Validate(); validateErr != nil {
					return cfg, fmt.Errorf("validate health config: %w", validateErr)
				}

				return cfg, nil
			})
		}).
		Provide(health.Module).
		Build()
}
