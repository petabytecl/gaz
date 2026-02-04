package eventbus

import (
	"log/slog"

	"github.com/petabytecl/gaz/di"
)

// Module registers eventbus infrastructure into the DI container.
// It provides a *EventBus for in-process pub/sub messaging.
//
// If *EventBus is already registered (e.g., by gaz.App), this is a no-op.
// The logger is optional - if not registered, slog.Default() is used.
//
// For CLI/App integration with flags, use the eventbus/module subpackage:
//
//	import eventbusmod "github.com/petabytecl/gaz/eventbus/module"
//	app.Use(eventbusmod.New())
func Module(c *di.Container) error {
	// Skip if already registered (e.g., by gaz.App)
	if di.Has[*EventBus](c) {
		return nil
	}

	return di.For[*EventBus](c).Provider(func(c *di.Container) (*EventBus, error) {
		// Logger is optional - use default if not registered
		logger := slog.Default()
		if l, err := di.Resolve[*slog.Logger](c); err == nil {
			logger = l
		}

		return New(logger), nil
	})
}
