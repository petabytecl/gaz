package gaz

import (
	"fmt"

	"github.com/petabytecl/gaz/di"
	"github.com/spf13/pflag"
)

// Use applies a module to the app's container.
// Modules bundle providers, configs, and other modules for reuse.
//
// Use accepts both gaz.Module (built via gaz.NewModule().Build()) and
// di.Module (returned by subsystem packages like health.NewModule()).
// This allows subsystem packages to export modules without importing gaz,
// avoiding import cycles.
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
//	    Use(health.NewModule()).    // di.Module from subsystem
//	    Use(cacheModule).
//	    Build()
func (a *App) Use(m Module) *App {
	if a.built {
		panic("gaz: cannot add modules after Build()")
	}

	name := m.Name()

	// Check for duplicate module name
	if a.modules[name] {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("%w: %s", ErrDuplicateModule, name))
		return a
	}
	a.modules[name] = true

	// Apply module flags if module provides them AND cobra command is available
	if flagsProvider, ok := m.(interface{ FlagsFn() func(*pflag.FlagSet) }); ok {
		if fn := flagsProvider.FlagsFn(); fn != nil {
			// If app has a cobra command, apply flags to it
			if a.cobraCmd != nil {
				fn(a.cobraCmd.PersistentFlags())
			}
		}
	}

	// Apply the module (which applies child modules first, then providers)
	if err := m.Apply(a); err != nil {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("module %s: %w", name, err))
	}

	return a
}

// UseDI applies a di.Module to the app's container.
// This is for subsystem packages (health, worker, cron, eventbus) that
// return di.Module to avoid import cycles with the gaz package.
//
// The di.Module interface has Register(c *Container) instead of Apply(app *App),
// which allows subsystem packages to export modules without importing gaz.
//
// Example:
//
//	app := gaz.New().
//	    UseDI(health.NewModule()).
//	    UseDI(worker.NewModule()).
//	    Build()
func (a *App) UseDI(m di.Module) *App {
	if a.built {
		panic("gaz: cannot add modules after Build()")
	}

	name := m.Name()

	// Check for duplicate module name
	if a.modules[name] {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("%w: %s", ErrDuplicateModule, name))
		return a
	}
	a.modules[name] = true

	// Apply the module by calling Register on the container
	if err := m.Register(a.container); err != nil {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("module %s: %w", name, err))
	}

	return a
}
