// Package module provides the gaz.Module for worker integration.
package module

import (
	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/worker"
)

// New creates a worker module that provides worker.Manager.
// This module registers the worker infrastructure for managing background workers.
// When used with gaz.App, the App-managed runtime reuses this Manager and
// applies its critical worker failure handler.
//
// Usage:
//
//	import workermod "github.com/petabytecl/gaz/worker/module"
//
//	app := gaz.New(gaz.WithCobra(rootCmd))
//	app.Use(workermod.New())
//
// The module provides:
//   - *worker.Manager for coordinating background workers
func New() gaz.Module {
	return gaz.NewModule(worker.ModuleName).
		Provide(worker.Module).
		Build()
}
