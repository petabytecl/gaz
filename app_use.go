package gaz

import (
	"fmt"

	"github.com/petabytecl/gaz/di"
)

// Use applies a module to the app's container.
// Modules bundle providers, configs, and other modules for reuse.
//
// Use accepts gaz.Module values built via gaz.NewModule().Build() or returned
// by feature module packages.
//
// Child modules bundled via ModuleBuilder.Use() are applied BEFORE the
// parent module's providers. This is for composition convenience, not
// dependency ordering (which is handled by the DI container).
//
// If the module provides CLI flags (via ModuleBuilder.Flags()) and the app
// has a Cobra command attached (via WithCobra), the flags are registered
// on the command's PersistentFlags.
//
// Returns error on duplicate module name (collected during Build()).
// Panics if called after Build().
//
// Example:
//
//	module := gaz.NewModule("database").
//	    Provide(func(c *gaz.Container) error {
//	        return gaz.For[*DB](c).Provider(NewDB)
//	    }).
//	    Build()
//
//	app := gaz.New().
//	    Use(module).
//	    Use(cacheModule).
//	    Build()
func (a *App) Use(m Module) *App {
	if a.built {
		panic("gaz: cannot add modules after Build()")
	}

	name := m.Name()

	if err := a.registerModuleIdentity(name); err != nil {
		a.buildErrors = append(a.buildErrors, err)
		return a
	}

	// Apply the module (which applies child modules first, then providers)
	if err := m.Apply(a); err != nil {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("module %s: %w", name, err))
	}

	return a
}

// UseDI applies a di.Module to the app's container.
//
// The di.Module interface has Register(c *Container) instead of Apply(app *App),
// which allows subsystem packages to export modules without importing gaz.
//
// Example:
//
//	app := gaz.New().
//	    UseDI(di.NewModuleFunc("custom", registerCustom)).
//	    Build()
func (a *App) UseDI(m di.Module) *App {
	if a.built {
		panic("gaz: cannot add modules after Build()")
	}

	name := m.Name()

	if err := a.registerModuleIdentity(name); err != nil {
		a.buildErrors = append(a.buildErrors, err)
		return a
	}

	// Apply the module by calling Register on the container
	if err := m.Register(a.container); err != nil {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("module %s: %w", name, err))
	}

	return a
}
