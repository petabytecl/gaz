# Phase 26: Module & Service Consolidation - Research

**Researched:** 2026-01-31
**Domain:** Go module system patterns, functional options, DI framework conventions
**Confidence:** HIGH

## Summary

This phase consolidates the gaz module system by removing the `gaz/service` package and standardizing `NewModule()` factory functions across all subsystem packages. The research confirms that the decisions made in CONTEXT.md align with established patterns from uber-go/fx and google/wire, representing current best practices in the Go DI ecosystem.

The core migration involves:
1. Moving `service.Builder` functionality directly into `gaz.App` methods (already partially exists)
2. Adding `NewModule()` factories to worker, cron, health, eventbus, and config packages
3. Documenting the di↔gaz relationship for new users
4. Removing the `gaz/service` package entirely (clean break for v3)

**Primary recommendation:** Follow the functional options pattern consistently across all NewModule() factories, returning `gaz.Module` directly with zero-config defaults.

## Standard Stack

This phase is internal refactoring with no new external dependencies.

### Core (Existing)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gaz | v3.0 | Core framework | Project itself |
| gaz/di | v3.0 | DI container | Standalone container |
| spf13/cobra | 1.x | CLI framework | Already integrated |
| spf13/pflag | 1.x | CLI flags | Already integrated |

### Pattern Reference
| Framework | Version | Pattern | Relevance |
|-----------|---------|---------|-----------|
| uber-go/fx | 1.x | `fx.Module("name", fx.Provide(...))` | Module factory pattern |
| google/wire | 0.x | `wire.NewSet(providers...)` | Provider set composition |

**No new dependencies required.** All patterns are implemented internally.

## Architecture Patterns

### Pattern 1: Functional Options for NewModule()

**What:** Each subsystem package exports a `NewModule()` function that accepts functional options and returns `gaz.Module`.

**When to use:** Every subsystem package (worker, cron, health, eventbus, config).

**Example:**
```go
// Source: Derived from uber-go/fx patterns and project CONTEXT.md decisions

// health/module_options.go
package health

// ModuleOption configures the health module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
    port          int
    livenessPath  string
    readinessPath string
    startupPath   string
}

func defaultModuleConfig() *moduleConfig {
    return &moduleConfig{
        port:          9090,
        livenessPath:  "/live",
        readinessPath: "/ready",
        startupPath:   "/startup",
    }
}

// WithPort sets the health server port.
func WithPort(port int) ModuleOption {
    return func(c *moduleConfig) {
        c.port = port
    }
}

// WithLivenessPath sets the liveness endpoint path.
func WithLivenessPath(path string) ModuleOption {
    return func(c *moduleConfig) {
        c.livenessPath = path
    }
}

// NewModule creates a health module with the given options.
// Returns a gaz.Module that registers health check components.
//
// Prerequisites: None (health module is self-contained)
//
// Example:
//
//     app := gaz.New()
//     app.Use(health.NewModule(health.WithPort(8081)))
func NewModule(opts ...ModuleOption) gaz.Module {
    cfg := defaultModuleConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    return gaz.NewModule("health").
        Provide(func(c *gaz.Container) error {
            // Register health.Config from module options
            healthCfg := Config{
                Port:          cfg.port,
                LivenessPath:  cfg.livenessPath,
                ReadinessPath: cfg.readinessPath,
                StartupPath:   cfg.startupPath,
            }
            if err := gaz.For[Config](c).Instance(healthCfg); err != nil {
                return fmt.Errorf("register health config: %w", err)
            }
            
            // Register ShutdownCheck, Manager, ManagementServer
            return registerComponents(c)
        }).
        Build()
}
```

### Pattern 2: Prerequisites Model

**What:** NewModule() functions document their prerequisites (registrations they expect to exist) via doc comments AND return runtime errors when prerequisites are missing.

**When to use:** Modules that depend on other registrations (e.g., worker module may expect logger).

**Example:**
```go
// Source: Project CONTEXT.md decisions

// worker/module.go
package worker

// NewModule creates a worker module with the given options.
// Returns a gaz.Module that configures worker management.
//
// Prerequisites:
//   - *slog.Logger must be registered (automatically registered by gaz.New())
//
// Example:
//
//     app := gaz.New()
//     app.Use(worker.NewModule(worker.WithMaxRestarts(3)))
func NewModule(opts ...ModuleOption) gaz.Module {
    cfg := defaultModuleConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    return gaz.NewModule("worker").
        Provide(func(c *gaz.Container) error {
            // Validate prerequisites
            if !gaz.Has[*slog.Logger](c) {
                return fmt.Errorf("worker module requires *slog.Logger to be registered")
            }
            // ... register components
            return nil
        }).
        Build()
}
```

### Pattern 3: Service Builder Migration to App Methods

**What:** `service.Builder` functionality moves to direct `gaz.App` method chains.

**Before (service.Builder):**
```go
app, err := service.New().
    WithCmd(rootCmd).
    WithConfig(&cfg).
    WithEnvPrefix("MYAPP").
    Use(myModule).
    Build()
```

**After (gaz.App direct):**
```go
app := gaz.New().
    WithCobra(rootCmd).
    WithConfig(&cfg, config.WithEnvPrefix("MYAPP")).
    Use(myModule)

if err := app.Build(); err != nil {
    log.Fatal(err)
}
```

**Key differences:**
- `WithCmd` → `WithCobra` (already exists on gaz.App)
- `WithEnvPrefix` moves into `WithConfig` options (already supported)
- `Build()` returns error, not (*App, error) tuple
- Health auto-registration logic moves to gaz.App.Build()

### Recommended Project Structure After Consolidation

```
gaz/
├── app.go              # gaz.App with all methods (includes former service.Builder logic)
├── compat.go           # Re-exports from di package
├── module_builder.go   # gaz.Module interface and NewModule()
├── config/
│   ├── module.go       # config.NewModule() + ModuleOption
│   └── options.go      # config.Option (for Manager)
├── cron/
│   ├── module.go       # cron.NewModule() + ModuleOption
│   └── scheduler.go    # Scheduler implementation
├── eventbus/
│   ├── module.go       # eventbus.NewModule() + ModuleOption
│   └── bus.go          # EventBus implementation
├── health/
│   ├── module.go       # health.NewModule() + ModuleOption (replaces old Module func)
│   └── server.go       # ManagementServer
├── worker/
│   ├── module.go       # worker.NewModule() + ModuleOption
│   └── manager.go      # Manager implementation
├── di/                 # Standalone DI (kept public, documented as advanced)
└── service/            # REMOVED in v3
```

### Anti-Patterns to Avoid

- **Returning *ModuleBuilder:** NewModule() returns `gaz.Module` directly, not a builder. This prevents partially-built modules and aligns with fx.Module pattern.

- **Mixing option patterns:** Don't mix `With{Property}` with other naming conventions. All options use `With{Property}(value)` pattern.

- **Implicit dependencies:** Don't silently fail when prerequisites are missing. Return clear error messages.

- **Breaking existing behavior:** Health auto-registration via HealthConfigProvider must continue to work.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Functional options | Custom option pattern | Existing `With{Property}` convention | Consistency across packages |
| Module factory | Custom factory pattern | `gaz.NewModule().Provide().Build()` | Uses existing ModuleBuilder |
| Type re-export | Manual type alias | Type alias `type X = pkg.X` | Go standard pattern |

**Key insight:** The existing `gaz.NewModule()` builder provides all needed functionality internally. NewModule() factories are thin wrappers that configure options and delegate to the builder.

## Common Pitfalls

### Pitfall 1: Breaking Health Auto-Registration

**What goes wrong:** Removing service.Builder breaks apps that rely on HealthConfigProvider interface for automatic health module registration.

**Why it happens:** The auto-registration logic is currently in `service.Builder.Build()`.

**How to avoid:** Move the HealthConfigProvider check into `gaz.App.Build()`. When config implements HealthConfigProvider and health module is not already registered, auto-register it.

**Warning signs:** Tests using `testConfigWithHealth` struct fail.

### Pitfall 2: Option Name Collisions

**What goes wrong:** Two packages export the same option name (e.g., both `worker.WithTimeout` and `cron.WithTimeout`).

**Why it happens:** Not considering full API surface during design.

**How to avoid:** Each package's options are namespaced by import. Use descriptive names that are self-explanatory even without package prefix (`WithPort`, `WithMaxRetries`, not `WithP`, `WithMR`).

**Warning signs:** Ambiguous documentation, user confusion.

### Pitfall 3: Missing Prerequisites Check

**What goes wrong:** Module panics or returns confusing error when prerequisite is missing.

**Why it happens:** Assuming prerequisites are always registered.

**How to avoid:** 
1. Document prerequisites in doc comment
2. Check with `gaz.Has[T](c)` before using
3. Return descriptive error: `"worker module requires *slog.Logger to be registered"`

**Warning signs:** Cryptic "service not found" errors at runtime.

### Pitfall 4: Breaking Existing Module() Signatures

**What goes wrong:** Existing code using `health.Module(c)` breaks.

**Why it happens:** Renaming `Module` to `NewModule` without deprecation.

**How to avoid:** Since v3 is a clean break, this is acceptable. But ensure all internal usages are updated.

**Warning signs:** Build failures in tests.

### Pitfall 5: Circular Import Between gaz and Subsystem Packages

**What goes wrong:** `health.NewModule()` imports `gaz`, and `gaz` imports `health`.

**Why it happens:** gaz.App needs to check HealthConfigProvider.

**How to avoid:** 
1. Keep `HealthConfigProvider` interface in `health` package
2. gaz.App imports health only for the interface check
3. NewModule() in health imports gaz (allowed, not circular)

**Warning signs:** Import cycle compile errors.

## Code Examples

### Example 1: Complete NewModule() Implementation (Health)

```go
// Source: Derived from existing health.Module + uber-go/fx patterns

package health

import (
    "fmt"
    "github.com/petabytecl/gaz"
)

// ModuleOption configures the health module.
type ModuleOption func(*moduleConfig)

type moduleConfig struct {
    port          int
    livenessPath  string
    readinessPath string
    startupPath   string
}

func defaultModuleConfig() *moduleConfig {
    cfg := DefaultConfig()
    return &moduleConfig{
        port:          cfg.Port,
        livenessPath:  cfg.LivenessPath,
        readinessPath: cfg.ReadinessPath,
        startupPath:   cfg.StartupPath,
    }
}

// WithPort sets the health server port. Default is 9090.
func WithPort(port int) ModuleOption {
    return func(c *moduleConfig) {
        c.port = port
    }
}

// WithLivenessPath sets the liveness endpoint path. Default is "/live".
func WithLivenessPath(path string) ModuleOption {
    return func(c *moduleConfig) {
        c.livenessPath = path
    }
}

// WithReadinessPath sets the readiness endpoint path. Default is "/ready".
func WithReadinessPath(path string) ModuleOption {
    return func(c *moduleConfig) {
        c.readinessPath = path
    }
}

// WithStartupPath sets the startup endpoint path. Default is "/startup".
func WithStartupPath(path string) ModuleOption {
    return func(c *moduleConfig) {
        c.startupPath = path
    }
}

// NewModule creates a health module with the given options.
// Returns a gaz.Module that registers health check components.
//
// Components registered:
//   - health.Config (from options or defaults)
//   - *health.ShutdownCheck
//   - *health.Manager
//   - *health.ManagementServer (eager, starts HTTP server)
//
// Example:
//
//     app := gaz.New()
//     app.Use(health.NewModule())                           // defaults
//     app.Use(health.NewModule(health.WithPort(8081)))      // custom port
func NewModule(opts ...ModuleOption) gaz.Module {
    cfg := defaultModuleConfig()
    for _, opt := range opts {
        opt(cfg)
    }

    return gaz.NewModule("health").
        Provide(func(c *gaz.Container) error {
            // Register Config from module options
            healthCfg := Config{
                Port:          cfg.port,
                LivenessPath:  cfg.livenessPath,
                ReadinessPath: cfg.readinessPath,
                StartupPath:   cfg.startupPath,
            }
            if err := gaz.For[Config](c).Instance(healthCfg); err != nil {
                return fmt.Errorf("register health config: %w", err)
            }

            // Register components (same as old Module function)
            if err := gaz.For[*ShutdownCheck](c).
                ProviderFunc(func(_ *gaz.Container) *ShutdownCheck {
                    return NewShutdownCheck()
                }); err != nil {
                return fmt.Errorf("register shutdown check: %w", err)
            }

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

            if err := gaz.For[*ManagementServer](c).
                Eager().
                Provider(func(c *gaz.Container) (*ManagementServer, error) {
                    resolvedCfg, err := gaz.Resolve[Config](c)
                    if err != nil {
                        return nil, err
                    }
                    manager, err := gaz.Resolve[*Manager](c)
                    if err != nil {
                        return nil, err
                    }
                    shutdownCheck, err := gaz.Resolve[*ShutdownCheck](c)
                    if err != nil {
                        return nil, err
                    }
                    return NewManagementServer(resolvedCfg, manager, shutdownCheck), nil
                }); err != nil {
                return fmt.Errorf("register management server: %w", err)
            }

            return nil
        }).
        Build()
}
```

### Example 2: Migration Pattern for Service Builder

```go
// Source: Codebase analysis - before and after

// BEFORE: Using service.Builder (to be removed)
package main

import (
    "github.com/petabytecl/gaz/service"
    "github.com/petabytecl/gaz/health"
)

func main() {
    app, err := service.New().
        WithCmd(rootCmd).
        WithConfig(&cfg).
        WithEnvPrefix("MYAPP").
        Use(myModule).
        Build()
    if err != nil {
        log.Fatal(err)
    }
    app.Run(context.Background())
}

// AFTER: Using gaz.App directly (v3 pattern)
package main

import (
    "github.com/petabytecl/gaz"
    "github.com/petabytecl/gaz/config"
    "github.com/petabytecl/gaz/health"
)

func main() {
    app := gaz.New().
        WithCobra(rootCmd).
        WithConfig(&cfg, config.WithEnvPrefix("MYAPP")).
        Use(myModule)
    
    // Health auto-registers if cfg implements HealthConfigProvider
    // Or explicitly:
    // app.Use(health.NewModule(health.WithPort(8081)))
    
    if err := app.Build(); err != nil {
        log.Fatal(err)
    }
    if err := app.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

### Example 3: Package Relationship Documentation (MOD-04)

```go
// Source: Pattern for di package doc.go update

// Package di provides a lightweight, type-safe dependency injection container.
//
// # When to Use di vs gaz
//
// Most applications should import "github.com/petabytecl/gaz" directly:
//
//     import "github.com/petabytecl/gaz"
//
//     app := gaz.New()
//     gaz.For[*MyService](app.Container()).Provider(NewMyService)
//
// The gaz package re-exports all di types (Container, For, Resolve, etc.)
// and adds application lifecycle, configuration, workers, cron, and health.
//
// Import di directly only when:
//   - You need standalone DI without gaz.App lifecycle
//   - You're building a library that depends only on the container
//   - You want to minimize import surface
//
// # Re-exported Types
//
// The following are re-exported by the gaz package:
//   - Container (as gaz.Container)
//   - For[T] (as gaz.For[T])
//   - Resolve[T] (as gaz.Resolve[T])
//   - Has[T] (as gaz.Has[T])
//   - Named (as gaz.Named)
//   - RegistrationBuilder (as gaz.RegistrationBuilder)
//
// For full application development, prefer the gaz package.
package di
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `service.New().Build()` | `gaz.New().Build()` | v3.0 | One fewer package to import |
| `health.Module(c)` | `health.NewModule(opts...)` | v3.0 | Consistent with fx/wire patterns |
| Separate service package | Unified gaz package | v3.0 | Simplified API surface |

**Deprecated/outdated:**
- `gaz/service` package: Removed in v3, functionality in gaz.App
- `health.Module(c *Container) error`: Replaced by `health.NewModule() gaz.Module`

## Open Questions

1. **Worker/Cron/EventBus module scope**
   - What we know: These are currently auto-initialized in gaz.New()
   - What's unclear: Should NewModule() allow reconfiguration, or are they fixed?
   - Recommendation: NewModule() for these packages allows *additional* configuration but doesn't replace auto-initialization. Document this behavior clearly.

2. **Config module utility**
   - What we know: Config is auto-loaded via gaz.App.WithConfig()
   - What's unclear: What would config.NewModule() provide beyond current options?
   - Recommendation: config.NewModule() could register additional config sources or watchers. Implement minimally, expand if needed.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `gaz/service/builder.go`, `gaz/app.go`, `gaz/module_builder.go`
- Codebase analysis: `gaz/health/module.go`, `gaz/compat.go`
- Context7 `/uber-go/fx` - Module pattern, fx.Provide, fx.Module
- Context7 `/google/wire` - Provider sets, composition patterns

### Secondary (MEDIUM confidence)
- Project decisions: `26-CONTEXT.md` (user-confirmed design choices)
- Project requirements: `REQUIREMENTS.md` (MOD-01, MOD-02, MOD-03, MOD-04)

### Tertiary (LOW confidence)
- None - all patterns verified against codebase and authoritative sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Internal refactoring, no new dependencies
- Architecture: HIGH - Patterns verified against uber-go/fx and codebase
- Pitfalls: HIGH - Based on codebase analysis and common Go patterns

**Research date:** 2026-01-31
**Valid until:** 60 days (stable internal patterns)
