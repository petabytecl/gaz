package worker

import (
	"log/slog"

	"github.com/petabytecl/gaz/di"
)

// Module registers worker infrastructure into the DI container.
// It provides a *Manager that can coordinate background workers.
//
// The logger is optional - if not registered, slog.Default() is used.
//
// For CLI/App integration with flags, use the worker/module subpackage:
//
//	import workermod "github.com/petabytecl/gaz/worker/module"
//	app.Use(workermod.New())
func Module(c *di.Container) error {
	return di.For[*Manager](c).Provider(func(c *di.Container) (*Manager, error) {
		// Logger is optional - use default if not registered
		logger := slog.Default()
		if l, err := di.Resolve[*slog.Logger](c); err == nil {
			logger = l
		}

		return NewManager(logger), nil
	})
}
