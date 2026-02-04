package gaz

import (
	"fmt"

	"github.com/spf13/pflag"
)

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
	flagsFn      func(*pflag.FlagSet) // CLI flags registration function
	envPrefix    string               // config key prefix for the module
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

// Flags registers CLI flags for this module.
// The flags function receives a FlagSet to register module-specific flags.
// Flags should be namespaced by module name (e.g., "redis-host" not "host")
// to avoid collisions with other modules.
//
// The flags function is called when the module is applied to an App that has
// a Cobra command attached (via WithCobra). If no Cobra command is attached,
// the flags function is not called.
//
// Example:
//
//	module := gaz.NewModule("redis").
//	    Flags(func(fs *pflag.FlagSet) {
//	        fs.String("redis-host", "localhost", "Redis server host")
//	        fs.Int("redis-port", 6379, "Redis server port")
//	    }).
//	    Build()
func (b *ModuleBuilder) Flags(fn func(*pflag.FlagSet)) *ModuleBuilder {
	b.flagsFn = fn
	return b
}

// WithEnvPrefix sets the config key prefix for this module.
// When combined with service-level env prefix, the module prefix becomes
// a sub-prefix. For example:
//
//	Service prefix: "MYAPP_"
//	Module prefix:  "redis"
//	Config key:     "host"
//	Result:         "MYAPP_REDIS_HOST"
//
// Example:
//
//	module := gaz.NewModule("redis").
//	    WithEnvPrefix("redis").
//	    Build()
func (b *ModuleBuilder) WithEnvPrefix(prefix string) *ModuleBuilder {
	b.envPrefix = prefix
	return b
}

// Build creates the final Module.
// After Build() is called, the ModuleBuilder should not be reused.
func (b *ModuleBuilder) Build() Module {
	return &builtModule{
		name:         b.name,
		providers:    b.providers,
		childModules: b.childModules,
		flagsFn:      b.flagsFn,
		envPrefix:    b.envPrefix,
	}
}

// builtModule is the concrete implementation of Module.
type builtModule struct {
	name         string
	providers    []func(*Container) error
	childModules []Module
	flagsFn      func(*pflag.FlagSet)
	envPrefix    string
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
			return fmt.Errorf("%w: %s", ErrModuleDuplicate, childName)
		}
		app.modules[childName] = true

		if err := child.Apply(app); err != nil {
			return fmt.Errorf("child module %s: %w", childName, err)
		}
	}

	// Register module flags if present
	if m.flagsFn != nil {
		app.AddFlagsFn(m.flagsFn)
	}

	// Then apply this module's providers
	for _, p := range m.providers {
		if err := p(app.container); err != nil {
			return err
		}
	}

	return nil
}

// FlagsFn returns the flags registration function, or nil if none was set.
// This is used by App.Use() to apply module flags to the cobra command.
func (m *builtModule) FlagsFn() func(*pflag.FlagSet) {
	return m.flagsFn
}

// EnvPrefix returns the module's environment prefix, or empty string if none was set.
func (m *builtModule) EnvPrefix() string {
	return m.envPrefix
}
