# Phase 21: Service Builder + Unified Provider - Research

**Researched:** 2026-01-29
**Domain:** Go DI framework service builder and module patterns
**Confidence:** HIGH

## Summary

This phase introduces two convenience APIs to gaz: a Service Builder for creating production-ready applications with standard wiring, and a Unified Module system for bundling providers, flags, and lifecycle hooks into reusable units.

The existing codebase already has foundations for both:
- The current `app.Module(name, ...registrations)` method provides basic module grouping
- The `health.WithHealthChecks()` option pattern shows how standard providers can be auto-wired
- The `config.WithEnvPrefix()` pattern handles environment variable prefixing

The key insight from researching uber/fx, google/wire, and samber/do is that **modules should be self-contained but DI-resolved** - they bundle what they provide but let the container handle actual dependency resolution. The existing gaz approach of inferring dependencies from the DI graph (not explicit `DependsOn()`) aligns with industry best practices.

**Primary recommendation:** Implement a fluent `service.Builder()` that chains configuration options and returns `(App, error)`, and enhance the existing module system with a `ModuleBuilder` that supports `Flags()`, `Provide()`, and `OnStart()/OnStop()` methods while keeping module application via `app.Use(module)`.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gaz | current | DI container, App, lifecycle | Existing framework |
| cobra | 1.8.x | CLI integration | Already integrated |
| viper | 1.18.x | Config backend | Already integrated |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| pflag | 0.5.x | CLI flag parsing | Already used via cobra |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom builder | uber/fx module pattern | fx is more complex, gaz is simpler |
| Explicit DependsOn | Implicit DI resolution | Implicit is cleaner, matches gaz philosophy |

**Installation:** No additional dependencies required.

## Architecture Patterns

### Recommended Project Structure
```
service/
├── builder.go        # ServiceBuilder implementation
├── options.go        # ServiceBuilder option functions
└── module.go         # ModuleBuilder implementation

gaz/
├── app_module.go     # Enhanced app.Use(module) method
└── ...
```

### Pattern 1: Fluent Service Builder

**What:** A builder pattern that collects configuration and returns a fully-wired App.

**When to use:** Creating production services with standard components (health checks, logging, config).

**Example:**
```go
// Source: Derived from existing gaz patterns + fx.New() pattern
package service

// Builder configures a production-ready service.
type Builder struct {
    cmd       *cobra.Command
    config    any
    envPrefix string
    opts      []gaz.Option
    modules   []Module
    errs      []error
}

// New returns a new service builder.
func New() *Builder {
    return &Builder{}
}

// WithCmd sets the cobra command for CLI integration.
func (b *Builder) WithCmd(cmd *cobra.Command) *Builder {
    b.cmd = cmd
    return b
}

// WithConfig sets the config struct to load into.
func (b *Builder) WithConfig(cfg any) *Builder {
    b.config = cfg
    return b
}

// WithEnvPrefix sets the global environment variable prefix.
func (b *Builder) WithEnvPrefix(prefix string) *Builder {
    b.envPrefix = prefix
    return b
}

// WithOptions adds gaz.Options to the underlying app.
func (b *Builder) WithOptions(opts ...gaz.Option) *Builder {
    b.opts = append(b.opts, opts...)
    return b
}

// Use adds a module to the service.
func (b *Builder) Use(m Module) *Builder {
    b.modules = append(b.modules, m)
    return b
}

// Build creates the App with all configured components.
func (b *Builder) Build() (*gaz.App, error) {
    if len(b.errs) > 0 {
        return nil, errors.Join(b.errs...)
    }
    
    // Collect config options
    var configOpts []config.Option
    if b.envPrefix != "" {
        configOpts = append(configOpts, config.WithEnvPrefix(b.envPrefix))
    }
    
    // Create app with options
    app := gaz.New(b.opts...)
    
    // Configure if provided
    if b.config != nil {
        app.WithConfig(b.config, configOpts...)
    }
    
    // Apply modules
    for _, m := range b.modules {
        if err := app.Use(m); err != nil {
            return nil, err
        }
    }
    
    // Attach cobra if provided
    if b.cmd != nil {
        app.WithCobra(b.cmd)
    }
    
    return app, nil
}
```

### Pattern 2: Module Builder

**What:** A builder for creating self-contained, reusable modules that bundle providers, flags, and lifecycle.

**When to use:** Packaging reusable functionality (e.g., redis module, database module, HTTP module).

**Example:**
```go
// Source: Derived from fx.Module + samber/do.Package patterns
package gaz

// Module is the interface that all modules implement.
type Module interface {
    Name() string
    Apply(app *App) error
}

// ModuleBuilder provides a fluent API for creating modules.
type ModuleBuilder struct {
    name         string
    providers    []func(*Container) error
    flagsFn      func(*pflag.FlagSet)
    configPrefix string
    errs         []error
}

// NewModule starts building a module with the given name.
func NewModule(name string) *ModuleBuilder {
    return &ModuleBuilder{name: name}
}

// Flags registers CLI flags for this module.
// The flags function receives a FlagSet to register flags on.
func (b *ModuleBuilder) Flags(fn func(*pflag.FlagSet)) *ModuleBuilder {
    b.flagsFn = fn
    return b
}

// Provide adds provider functions to the module.
// Each function receives *Container and returns error.
func (b *ModuleBuilder) Provide(fns ...func(*Container) error) *ModuleBuilder {
    b.providers = append(b.providers, fns...)
    return b
}

// WithEnvPrefix sets the config key prefix for this module.
// Combined with service prefix: MYAPP_ + redis = MYAPP_REDIS_
func (b *ModuleBuilder) WithEnvPrefix(prefix string) *ModuleBuilder {
    b.configPrefix = prefix
    return b
}

// Build returns the completed Module.
func (b *ModuleBuilder) Build() Module {
    return &builtModule{
        name:         b.name,
        providers:    b.providers,
        flagsFn:      b.flagsFn,
        configPrefix: b.configPrefix,
    }
}

// builtModule is the concrete Module implementation.
type builtModule struct {
    name         string
    providers    []func(*Container) error
    flagsFn      func(*pflag.FlagSet)
    configPrefix string
}

func (m *builtModule) Name() string { return m.name }

func (m *builtModule) Apply(app *App) error {
    // Register all providers
    for _, p := range m.providers {
        if err := p(app.Container()); err != nil {
            return fmt.Errorf("module %s: %w", m.name, err)
        }
    }
    return nil
}
```

### Pattern 3: Auto-Registration of Health Check

**What:** Service builder automatically wires health check when config provides health settings.

**When to use:** Production services that need health endpoints.

**Example:**
```go
// Source: Existing health.WithHealthChecks pattern
func (b *Builder) Build() (*gaz.App, error) {
    // ... existing code ...
    
    // Auto-register health check if config implements health.ConfigProvider
    if hp, ok := b.config.(interface{ HealthConfig() health.Config }); ok {
        app.Use(health.Module(hp.HealthConfig()))
    }
    
    return app, nil
}
```

### Anti-Patterns to Avoid

- **Explicit DependsOn declarations:** Don't add `DependsOn()` to modules - let DI graph infer dependencies
- **Module namespacing:** Module names are for debugging, not type namespacing
- **Implicit registration:** Don't auto-register modules - require explicit `app.Use()`
- **Mutable modules after build:** Once `Build()` is called, module should be immutable

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Dependency ordering | Custom topological sort | Existing `ComputeStartupOrder` | Already handles cycles, layers |
| Flag binding | Manual viper.BindPFlag | Existing `RegisterCobraFlags` | Handles types, namespacing |
| Config validation | Custom validation | Existing `config.ValidateStruct` | Uses go-playground/validator |
| Duplicate detection | Manual tracking | Existing `app.modules` map | Already detects duplicate modules |
| Env var naming | Manual transformation | `config.WithEnvPrefix` | Handles `_` to `.` conversion |

**Key insight:** The existing gaz infrastructure already handles the hard problems. The service builder and module system are thin convenience layers on top.

## Common Pitfalls

### Pitfall 1: Module Registration Order Sensitivity

**What goes wrong:** Assuming modules must be registered in dependency order.
**Why it happens:** Confusion about when providers are instantiated.
**How to avoid:** Clarify that DI resolution happens at Build() time, not at registration time. Order of `app.Use()` calls doesn't matter.
**Warning signs:** Documentation or tests that rely on specific registration order.

### Pitfall 2: Env Prefix Stacking Confusion

**What goes wrong:** Users expect `MYAPP_REDIS_HOST` but get `MYAPP_HOST` or vice versa.
**Why it happens:** Unclear how service prefix and module prefix combine.
**How to avoid:** 
- Document clearly: service prefix `MYAPP_` + module prefix `redis` + key `host` = `MYAPP_REDIS_HOST`
- Use `strings.ToUpper(servicePrefix + "_" + modulePrefix + "_" + key)` with proper underscore handling
**Warning signs:** Tests that hardcode env var names without using the same transformation logic.

### Pitfall 3: Module Flag Collision

**What goes wrong:** Two modules register flags with the same name.
**Why it happens:** Each module independently defines its flags.
**How to avoid:**
- Module flags should be namespaced by module name: `--redis-host` not `--host`
- Use `configKeyToFlagName()` transformation consistently
**Warning signs:** Flag registration errors in tests with multiple modules.

### Pitfall 4: Health Check Not Auto-Registering

**What goes wrong:** Service builder doesn't register health check even though config has health settings.
**Why it happens:** Config struct doesn't implement the expected interface.
**How to avoid:**
- Define a clear interface: `interface{ HealthConfig() health.Config }`
- Or: check if `health.Config` is registered in container after config loading
- Document: "Health check auto-registers when X"
**Warning signs:** Services running without health endpoints when they should have them.

### Pitfall 5: Module Lifecycle Hook Ordering

**What goes wrong:** Module's OnStart runs before a dependency's OnStart.
**Why it happens:** Module hooks don't participate in dependency graph.
**How to avoid:**
- Module hooks should be registered via provider lifecycle hooks, not separate module-level hooks
- Use existing `OnStart()/OnStop()` on `RegistrationBuilder`
**Warning signs:** Race conditions or nil pointers during startup.

## Code Examples

Verified patterns from existing codebase:

### Existing Module Pattern
```go
// Source: health/module.go
func Module(c *gaz.Container) error {
    // Register ShutdownCheck
    if err := gaz.For[*ShutdownCheck](c).
        ProviderFunc(func(_ *gaz.Container) *ShutdownCheck {
            return NewShutdownCheck()
        }); err != nil {
        return fmt.Errorf("register shutdown check: %w", err)
    }
    
    // Register Manager with dependency on ShutdownCheck
    if err := gaz.For[*Manager](c).
        Provider(func(c *gaz.Container) (*Manager, error) {
            m := NewManager()
            shutdownCheck, err := gaz.Resolve[*ShutdownCheck](c)
            if err != nil {
                return nil, err
            }
            m.AddReadinessCheck("shutdown", shutdownCheck.Check)
            return m, nil
        }); err != nil {
        return fmt.Errorf("register manager: %w", err)
    }
    
    return nil
}
```

### Existing App.Module Pattern
```go
// Source: app_module.go
func (a *App) Module(name string, registrations ...func(*Container) error) *App {
    if a.built {
        panic("gaz: cannot add modules after Build()")
    }
    
    // Check for duplicate module name
    if a.modules[name] {
        a.buildErrors = append(a.buildErrors,
            fmt.Errorf("%w: %s", ErrDuplicateModule, name))
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
```

### Config Env Prefix Transformation
```go
// Source: config/manager.go
// When envPrefix is "APP", key "database.host" becomes "APP_DATABASE__HOST"
eb.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))

// For provider flags (simpler):
// key "redis.host" becomes env var "REDIS_HOST"
envKey := strings.ToUpper(strings.ReplaceAll(fullKey, ".", "_"))
```

### Fluent Builder Usage Example
```go
// Target API for Phase 21
func main() {
    cfg := &AppConfig{}
    
    app, err := service.New().
        WithCmd(rootCmd).
        WithConfig(cfg).
        WithEnvPrefix("MYAPP").
        Use(health.NewModule()).
        Use(redis.NewModule()).
        Build()
    if err != nil {
        log.Fatal(err)
    }
    
    app.Run(context.Background())
}
```

### Module Builder Usage Example
```go
// Target API for Phase 21
var RedisModule = gaz.NewModule("redis").
    Flags(func(fs *pflag.FlagSet) {
        fs.String("redis-host", "localhost", "Redis server host")
        fs.Int("redis-port", 6379, "Redis server port")
    }).
    Provide(
        func(c *gaz.Container) error {
            return gaz.For[*redis.Client](c).
                OnStop(func(ctx context.Context, r *redis.Client) error {
                    return r.Close()
                }).
                Provider(NewRedisClient)
        },
    ).
    Build()
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual app setup | Service builder | This phase | Less boilerplate |
| Module functions | ModuleBuilder | This phase | Bundled flags/config |
| Separate flag registration | Module.Flags() | This phase | Co-located with providers |

**Deprecated/outdated:**
- The current `app.Module(name, ...funcs)` API will remain for backward compatibility but `app.Use(module)` becomes the preferred pattern for complex modules

## Open Questions

Things that couldn't be fully resolved:

1. **Health check auto-registration trigger**
   - What we know: Context says "auto-registers when config provides health settings"
   - What's unclear: Exact mechanism - config interface check vs. container check
   - Recommendation: Use interface check `interface{ HealthConfig() health.Config }` - more explicit

2. **Module config prefix interaction with global prefix**
   - What we know: Global `MYAPP_` + module `redis` = `MYAPP_REDIS_HOST`
   - What's unclear: How to handle modules that want NO prefix
   - Recommendation: Empty module prefix means use only global prefix

3. **Module.Flags() timing**
   - What we know: Flags must be registered before `cmd.Execute()`
   - What's unclear: Who calls the flags function and when
   - Recommendation: `app.Use(module)` should immediately call `module.Flags(cmd.PersistentFlags())` if cmd is available

## Sources

### Primary (HIGH confidence)
- Context7 /uber-go/fx - Module pattern, fx.Module, fx.Provide, lifecycle hooks
- Context7 /google/wire - ProviderSet composition pattern
- Context7 /websites/pkg_go_dev_github_com_samber_do_v2 - Package function, scope pattern
- Existing gaz codebase - app.go, app_module.go, health/module.go, config/manager.go

### Secondary (MEDIUM confidence)
- Google Search: "Go service builder pattern fluent API dependency injection 2025 2026" - confirmed fluent builder is preferred for complex initialization

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - using existing gaz patterns, no new dependencies
- Architecture: HIGH - patterns derived from uber/fx, verified against existing code
- Pitfalls: HIGH - identified from existing gaz code and common DI issues

**Research date:** 2026-01-29
**Valid until:** 2026-02-28 (stable domain, 30 days)
