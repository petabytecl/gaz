package gaz

import (
	"fmt"
)

// Module registers a named group of providers.
// The name is used for debugging and error messages.
// Duplicate module names result in ErrModuleDuplicate error during Build().
//
// Each registration should be a function that accepts *Container and returns error.
// This allows using the For[T]() fluent API within modules.
//
// Example:
//
//	app.Module("database",
//	    func(c *gaz.Container) error { return gaz.For[*DB](c).Provider(NewDB) },
//	    func(c *gaz.Container) error { return gaz.For[*UserRepo](c).Provider(NewUserRepo) },
//	).Module("http",
//	    func(c *gaz.Container) error { return gaz.For[*Server](c).Provider(NewServer) },
//	)
func (a *App) Module(name string, registrations ...func(*Container) error) *App {
	if a.built {
		panic("gaz: cannot add modules after Build()")
	}

	// Check for duplicate module name
	if a.modules[name] {
		a.buildErrors = append(a.buildErrors,
			fmt.Errorf("%w: %s", ErrModuleDuplicate, name))
		return a
	}
	a.modules[name] = true

	// Register each provider with module context
	for _, reg := range registrations {
		if err := reg(a.container); err != nil {
			a.buildErrors = append(a.buildErrors,
				fmt.Errorf("module %s: %w", name, err))
		}
	}

	return a
}
