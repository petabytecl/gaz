package gaz

import "fmt"

// Use applies a module to the app's container.
// Modules bundle providers, configs, and other modules for reuse.
//
// Child modules bundled via ModuleBuilder.Use() are applied BEFORE the
// parent module's providers. This is for composition convenience, not
// dependency ordering (which is handled by the DI container).
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

	// Check for duplicate module name
	if a.modules[name] {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("%w: %s", ErrDuplicateModule, name))
		return a
	}
	a.modules[name] = true

	// Apply the module (which applies child modules first, then providers)
	if err := m.Apply(a); err != nil {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("module %s: %w", name, err))
	}

	return a
}
