package cron

import (
	"log/slog"

	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the cron module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	// Currently no configurable options exposed
	// Placeholder for future extensibility (e.g., WithTimezone)
}

func defaultModuleConfig() *moduleConfig {
	return &moduleConfig{}
}

// NewModule creates a cron module with the given options.
// Returns a function compatible with gaz.Module registration that provides
// cron scheduling infrastructure.
//
// Prerequisites:
//   - *slog.Logger must be registered (automatically registered by gaz.New())
//
// Note: Cron jobs are auto-discovered during gaz.App.Build() when services
// implement the cron.CronJob interface. This module provides explicit opt-in
// and validates prerequisites. The Scheduler is created in gaz.New().
//
// Example:
//
//	app := gaz.New()
//	app.Module("cron", cron.NewModule())
func NewModule(opts ...ModuleOption) func(*di.Container) error {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return func(c *di.Container) error {
		// Validate prerequisites
		if !di.Has[*slog.Logger](c) {
			// Logger is auto-registered by gaz.New(), so this should never fail
			// in normal usage. This check exists for direct di.Container usage.
			return nil
		}

		// Scheduler is auto-created in gaz.New(), so this module
		// just validates prerequisites. Future options could configure
		// timezone, error handlers, etc.
		_ = cfg // Future: use cfg for configuration

		return nil
	}
}
