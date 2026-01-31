package config

import (
	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the config module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
	// Currently no configurable options exposed
	// Placeholder for future extensibility (e.g., WithWatcher)
}

func defaultModuleConfig() *moduleConfig {
	return &moduleConfig{}
}

// NewModule creates a config module with the given options.
// Returns a function compatible with gaz.Module registration that provides
// configuration infrastructure.
//
// Note: Configuration is typically set up via gaz.App.WithConfig().
// This module provides explicit opt-in for advanced use cases like
// additional config sources or watchers.
//
// Example:
//
//	app := gaz.New()
//	app.Module("config", config.NewModule())
func NewModule(opts ...ModuleOption) func(*di.Container) error {
	cfg := defaultModuleConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return func(c *di.Container) error {
		// Config infrastructure is set up in gaz.New() and
		// configured via WithConfig(). This module provides
		// a placeholder for future extensions like:
		// - Additional config sources
		// - Config watchers
		// - Remote config providers

		_ = c   // Future: use container for registration
		_ = cfg // Future: use cfg for configuration

		return nil
	}
}
