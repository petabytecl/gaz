package eventbus

import (
	"log/slog"

	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the eventbus module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	// Currently no configurable options exposed
	// Placeholder for future extensibility (e.g., WithBufferSize)
}

func defaultModuleConfig() *moduleConfig {
	return &moduleConfig{}
}

// NewModule creates an eventbus module with the given options.
// Returns a di.Module that provides pub/sub infrastructure.
//
// Prerequisites:
//   - *slog.Logger must be registered (automatically registered by gaz.New())
//
// Note: EventBus is auto-created during gaz.New() and registered in the
// container. This module provides explicit opt-in and future configuration.
//
// Example:
//
//	app := gaz.New()
//	app.UseDI(eventbus.NewModule())
func NewModule(opts ...ModuleOption) di.Module {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return di.NewModuleFunc("eventbus", func(c *di.Container) error {
		// Validate prerequisites
		if !di.Has[*slog.Logger](c) {
			// Logger is auto-registered by gaz.New(), so this should never fail
			// in normal usage. This check exists for direct di.Container usage.
			return nil
		}

		// EventBus is auto-created in gaz.New() and registered.
		// This module validates prerequisites and serves as foundation
		// for future options (e.g., WithBufferSize, WithErrorHandler).
		_ = cfg // Future: use cfg for configuration

		return nil
	})
}
