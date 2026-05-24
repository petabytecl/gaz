package config

import (
	"github.com/petabytecl/gaz/di"
)

// ModuleOption configures the deprecated compatibility config module.
//
// Deprecated: [NewModule] is a no-op compatibility module. Use gaz.App
// WithConfig for application configuration, the config/module package for
// Cobra config flags, or [New] for standalone configuration loading.
type ModuleOption func(*moduleConfig)

type moduleConfig struct{}

// NewModule returns a no-op DI module kept for source compatibility.
//
// The config package owns standalone configuration loading through [New].
// gaz.App owns application configuration through WithConfig, and the
// config/module package owns Cobra config flags. This compatibility module
// deliberately registers no services.
//
// Example:
//
//	app := gaz.New()
//	app.UseDI(config.NewModule())
//
// Deprecated: NewModule does not register configuration infrastructure. Use
// gaz.App WithConfig for application configuration, the config/module package
// for Cobra config flags, or [New] for standalone configuration loading.
func NewModule(opts ...ModuleOption) di.Module {
	_ = opts

	return di.NewModuleFunc(ModuleName, func(c *di.Container) error {
		_ = c
		return nil
	})
}
