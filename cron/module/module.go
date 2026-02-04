// Package module provides the gaz.Module for cron integration.
package module

import (
	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/cron"
)

// New creates a cron module that provides cron.Scheduler.
// This module registers the cron infrastructure for scheduling jobs.
//
// Usage:
//
//	import cronmod "github.com/petabytecl/gaz/cron/module"
//
//	app := gaz.New(gaz.WithCobra(rootCmd))
//	app.Use(cronmod.New())
//
// The module provides:
//   - *cron.Scheduler for scheduling cron jobs
//
//nolint:ireturn // Module is the expected return type for gaz modules
func New() gaz.Module {
	return gaz.NewModule("cron").
		Provide(cron.Module).
		Build()
}
