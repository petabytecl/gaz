// Package module provides the gaz.Module for health configuration with CLI flags.
package module

import (
	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/health"
	"github.com/petabytecl/gaz/internal/configuredmodule"
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
func New() gaz.Module {
	defaultCfg := health.DefaultConfig()

	return gaz.NewModule("health-flags").
		Flags(defaultCfg.Flags).
		Provide(configuredmodule.ProvideDefaulted[health.Config, *health.Config](
			&defaultCfg,
			"health config",
			"validate health config",
		)).
		Provide(health.Module).
		Build()
}
