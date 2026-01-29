package gaz

import "fmt"

// Module represents a reusable bundle of providers.
// Modules can be composed using the Use() method to bundle child modules.
// When a module is applied, its child modules are applied first.
type Module interface {
	// Name returns the module's identifier for debugging and error messages.
	Name() string

	// Apply applies the module's providers to the app.
	// Child modules are applied before the parent module's providers.
	Apply(app *App) error
}

// ModuleBuilder constructs Module instances via fluent API.
// Use NewModule(name) to start building a module.
//
// Example:
//
//	module := gaz.NewModule("database").
//	    Provide(func(c *gaz.Container) error {
//	        return gaz.For[*DB](c).Provider(NewDB)
//	    }).
//	    Build()
//
//	app.Use(module)
type ModuleBuilder struct {
	name         string
	providers    []func(*Container) error
	childModules []Module
	errs         []error
}

// NewModule creates a new ModuleBuilder with the given name.
// The name is used for debugging, error messages, and duplicate detection.
//
// Example:
//
//	module := gaz.NewModule("redis").
//	    Provide(RedisProvider).
//	    Build()
func NewModule(name string) *ModuleBuilder {
	return &ModuleBuilder{name: name}
}

// Provide adds provider functions to the module.
// Each function receives *Container and returns error.
// This method is chainable.
//
// Example:
//
//	module := gaz.NewModule("http").
//	    Provide(
//	        func(c *gaz.Container) error { return gaz.For[*Router](c).Provider(NewRouter) },
//	        func(c *gaz.Container) error { return gaz.For[*Server](c).Provider(NewServer) },
//	    ).
//	    Build()
func (b *ModuleBuilder) Provide(fns ...func(*Container) error) *ModuleBuilder {
	b.providers = append(b.providers, fns...)
	return b
}

// Use bundles another module to be applied when this module is applied.
// Child modules are applied BEFORE this module's providers.
// This is for composition/bundling convenience, not dependency ordering
// (dependency ordering is handled by the DI container).
//
// Example:
//
//	logging := gaz.NewModule("logging").Provide(LoggerProvider).Build()
//	metrics := gaz.NewModule("metrics").Provide(MetricsProvider).Build()
//
//	observability := gaz.NewModule("observability").
//	    Use(logging).
//	    Use(metrics).
//	    Build()
func (b *ModuleBuilder) Use(m Module) *ModuleBuilder {
	b.childModules = append(b.childModules, m)
	return b
}

// Build creates the final Module.
// After Build() is called, the ModuleBuilder should not be reused.
func (b *ModuleBuilder) Build() Module {
	return &builtModule{
		name:         b.name,
		providers:    b.providers,
		childModules: b.childModules,
	}
}

// builtModule is the concrete implementation of Module.
type builtModule struct {
	name         string
	providers    []func(*Container) error
	childModules []Module
}

// Name returns the module name.
func (m *builtModule) Name() string {
	return m.name
}

// Apply applies the module's providers to the app.
// Child modules are applied FIRST (in order), then the parent's providers.
// Each module's name is registered in app.modules for duplicate detection.
func (m *builtModule) Apply(app *App) error {
	// Apply child modules FIRST (composition)
	for _, child := range m.childModules {
		childName := child.Name()

		// Check for duplicate child module name
		if app.modules[childName] {
			return fmt.Errorf("%w: %s", ErrDuplicateModule, childName)
		}
		app.modules[childName] = true

		if err := child.Apply(app); err != nil {
			return fmt.Errorf("child module %s: %w", childName, err)
		}
	}

	// Then apply this module's providers
	for _, p := range m.providers {
		if err := p(app.container); err != nil {
			return err
		}
	}

	return nil
}
