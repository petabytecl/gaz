package di

// Module represents a reusable bundle of providers that register
// services in a DI container.
//
// This interface is defined in the di package for adapters that should register
// directly into a DI container. Use gaz.App.UseDI() to apply one to an App.
//
// Example:
//
//	module := di.NewModuleFunc("custom", registerCustom)
//	app := gaz.New().UseDI(module)
type Module interface {
	// Name returns the module's identifier for debugging and error messages.
	Name() string

	// Register applies the module's providers to the container.
	Register(c *Container) error
}

// ModuleFunc is a convenience type for creating simple modules from functions.
// It implements the Module interface.
type ModuleFunc struct {
	name string
	fn   func(*Container) error
}

// NewModuleFunc creates a Module from a name and registration function.
// This is the simplest way to create a Module without defining a new type.
//
// Example:
//
//	module := di.NewModuleFunc("mymodule", func(c *di.Container) error {
//	    return di.For[*MyService](c).Provider(NewMyService)
//	})
func NewModuleFunc(name string, fn func(*Container) error) Module {
	return &ModuleFunc{name: name, fn: fn}
}

// Name returns the module name.
func (m *ModuleFunc) Name() string {
	return m.name
}

// Register calls the registration function.
func (m *ModuleFunc) Register(c *Container) error {
	return m.fn(c)
}
