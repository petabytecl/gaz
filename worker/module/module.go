// Package module provides the gaz.Module for worker integration.
package module

import (
	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/worker"
)

// New creates a worker module that provides worker.Manager.
// This module registers the worker infrastructure for managing background workers.
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
//
//nolint:ireturn // Module is the expected return type for gaz modules
func New() gaz.Module {
	return gaz.NewModule("worker").
		Provide(worker.Module).
		Build()
}
