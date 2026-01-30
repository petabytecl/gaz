# gaz Style Guide

This document defines API conventions for contributors modifying gaz. All rules use MUST/SHOULD/MAY per [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

**Scope:** API conventions for internal contributors. Code style conventions (formatting, comments) are deferred to a later pass.

## Naming Conventions

### Package Names

Package names MUST be lowercase, short, and avoid underscores.

**Rationale:** Go convention; makes imports readable and avoids conflicts.

**Good:**
```go
// Source: gaz/config/
package config

// Source: gaz/health/
package health
```

**Bad:**
```go
// Don't use underscores or mixed case
package health_check
package healthCheck
```

### Type Names

Exported types MUST use PascalCase. Unexported types MUST use camelCase.

**Rationale:** Go visibility rules; PascalCase exports, camelCase keeps internal.

**Good:**
```go
// Source: gaz/module_builder.go
type ModuleBuilder struct { ... }  // Exported
type builtModule struct { ... }    // Unexported
```

**Bad:**
```go
type module_builder struct { ... }  // Underscores
type Builtmodule struct { ... }     // Inconsistent casing
```

### Interface Names

Single-method interfaces SHOULD use `-er` suffix. Multi-method interfaces SHOULD use descriptive nouns.

**Rationale:** Go idiom; `Reader`, `Writer`, `Closer` are standard patterns.

**Good:**
```go
// Single method: use -er suffix
type Starter interface {
    Start(context.Context) error
}

// Multiple methods: descriptive noun
type Backend interface {
    Get(key string) any
    Set(key string, value any)
    IsSet(key string) bool
}
```

**Bad:**
```go
// Avoid -er for multi-method interfaces
type Backender interface { ... }

// Avoid generic names
type IBackend interface { ... }
```

## Constructor Patterns

### NewX() Pattern

Constructors MUST use `NewX()` when the package exports multiple types.

**Rationale:** Disambiguates which type is being constructed when multiple exist.

**Good:**
```go
// Source: gaz/health/manager.go
// NewManager creates a new Health Manager.
func NewManager() *Manager {
    return &Manager{}
}

// Source: gaz/health/shutdown.go
// NewShutdownCheck creates a new shutdown check.
func NewShutdownCheck() *ShutdownCheck {
    return &ShutdownCheck{}
}
```

**Bad:**
```go
// Ambiguous when multiple types exist
func New() *Manager { ... }
```

### New() Pattern

Constructors MUST use `New()` when the package exports a single primary type.

**Rationale:** Avoids stutter; `config.New()` is clearer than `config.NewConfig()`.

**Good:**
```go
// Source: gaz/config/manager.go
// New creates a new Manager with the given options.
func New(opts ...Option) *Manager {
    m := &Manager{
        fileName:    "config",
        fileType:    "yaml",
        searchPaths: []string{"."},
        defaults:    make(map[string]any),
    }
    for _, opt := range opts {
        opt(m)
    }
    return m
}

// Source: gaz/service/builder.go
// New returns a new service Builder.
func New() *Builder {
    return &Builder{}
}
```

**Bad:**
```go
// Stutters
func NewConfig() *Config { ... }
func NewBuilder() *Builder { ... }
```

### NewXWithY() Pattern

Variant constructors MUST use `NewXWithY()` when providing alternative setup.

**Rationale:** Signals different initialization path without overloading `New()`.

**Good:**
```go
// Source: gaz/config/manager.go
// NewWithBackend creates a Manager with a custom backend.
func NewWithBackend(backend Backend, opts ...Option) *Manager {
    if backend == nil {
        panic("config: backend cannot be nil")
    }
    m := &Manager{
        backend:     backend,
        fileName:    "config",
        ...
    }
    return m
}
```

**Bad:**
```go
// Unclear what differs from New()
func New2() *Manager { ... }
func NewAlt() *Manager { ... }
```

### Builder Pattern

Use the Builder pattern when types have many optional configurations.

**Rationale:** Readable configuration without constructor parameter explosion.

**Good:**
```go
// Source: gaz/module_builder.go
// NewModule creates a new ModuleBuilder with the given name.
func NewModule(name string) *ModuleBuilder {
    return &ModuleBuilder{name: name}
}

func (b *ModuleBuilder) Provide(fns ...func(*Container) error) *ModuleBuilder {
    b.providers = append(b.providers, fns...)
    return b
}

func (b *ModuleBuilder) Build() Module {
    return &builtModule{...}
}

// Usage:
// module := gaz.NewModule("database").
//     Provide(DBProvider).
//     Flags(dbFlags).
//     Build()
```

```go
// Source: gaz/service/builder.go
// Builder constructs a gaz.App with fluent configuration.
func New() *Builder {
    return &Builder{}
}

func (b *Builder) WithCmd(cmd *cobra.Command) *Builder {
    b.cmd = cmd
    return b
}

func (b *Builder) Build() (*gaz.App, error) {
    ...
}

// Usage:
// app, err := service.New().
//     WithCmd(rootCmd).
//     WithConfig(cfg).
//     Build()
```

**Bad:**
```go
// Too many constructor parameters
func NewModule(name string, providers []Provider, flags FlagFn, prefix string) Module {
    ...
}
```

## Error Conventions

### Sentinel Error Naming

Sentinel error variables MUST use the `Err` prefix followed by a descriptive name.

**Rationale:** Go convention; enables `errors.Is()` checks and clear identification.

**Good:**
```go
// Source: gaz/di/errors.go
var (
    // ErrNotFound is returned when a requested service is not registered.
    ErrNotFound = errors.New("di: service not found")

    // ErrCycle is returned when a circular dependency is detected.
    ErrCycle = errors.New("di: circular dependency detected")

    // ErrDuplicate is returned when attempting to register an existing service.
    ErrDuplicate = errors.New("di: service already registered")
)
```

```go
// Source: gaz/worker/errors.go
var (
    // ErrCircuitBreakerTripped indicates max restarts exceeded.
    ErrCircuitBreakerTripped = errors.New("worker: circuit breaker tripped, max restarts exceeded")

    // ErrWorkerStopped indicates a worker stopped normally.
    ErrWorkerStopped = errors.New("worker: stopped normally")
)
```

**Bad:**
```go
// Missing Err prefix
var NotFound = errors.New("not found")

// Wrong casing
var errNotFound = errors.New("not found")  // Unexported sentinel is rarely useful
```

[AUTOMATABLE] Error variable naming (`Err*` prefix) can be enforced with custom golangci-lint rule.

### Error Message Format

Error messages MUST use the format `"pkg: description"` — lowercase, no trailing punctuation.

**Rationale:** Consistent formatting when errors are wrapped and printed in chains.

**Good:**
```go
// Source: gaz/di/errors.go
ErrNotFound = errors.New("di: service not found")
ErrCycle = errors.New("di: circular dependency detected")

// Source: gaz/worker/errors.go
ErrCircuitBreakerTripped = errors.New("worker: circuit breaker tripped, max restarts exceeded")
```

**Bad:**
```go
// Capitalized
ErrNotFound = errors.New("Service not found")

// Trailing punctuation
ErrNotFound = errors.New("di: service not found.")

// Missing package prefix
ErrNotFound = errors.New("service not found")
```

### Error Wrapping

Error wrapping MUST use `fmt.Errorf("context: %w", err)` with lowercase context.

**Rationale:** Preserves error chain for `errors.Is()` and `errors.As()` while adding context.

**Good:**
```go
// Source: gaz/health/module.go
if err := gaz.For[*ShutdownCheck](c).
    ProviderFunc(func(_ *gaz.Container) *ShutdownCheck {
        return NewShutdownCheck()
    }); err != nil {
    return fmt.Errorf("register shutdown check: %w", err)
}

if err := gaz.For[*Manager](c).Provider(...); err != nil {
    return fmt.Errorf("register manager: %w", err)
}
```

**Bad:**
```go
// Capitalized context
return fmt.Errorf("Register manager: %w", err)

// Using %v loses error chain
return fmt.Errorf("register manager: %v", err)

// No context
return err
```

### Error Types (for completeness)

Error types SHOULD use the `Error` suffix when a struct type is needed.

**Rationale:** Distinguishes error types from regular types; enables `errors.As()` matching.

**Note:** gaz currently uses sentinel errors (`var Err*`), not error types. This convention is documented for completeness if error types are needed in the future.

**Good:**
```go
// Not currently used in gaz, but if needed:
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}
```

**Bad:**
```go
// Missing Error suffix
type Validation struct { ... }

// Using Err prefix for types (Err is for variables)
type ErrValidation struct { ... }
```

## Module Patterns

### Module Factory Function

Module factory functions MUST use the signature `func Module(c *gaz.Container) error`.

**Rationale:** Consistent signature enables composition with `NewModule().Provide()`.

**Good:**
```go
// Source: gaz/health/module.go
// Module registers the health module components.
func Module(c *gaz.Container) error {
    // Register ShutdownCheck
    if err := gaz.For[*ShutdownCheck](c).
        ProviderFunc(func(_ *gaz.Container) *ShutdownCheck {
            return NewShutdownCheck()
        }); err != nil {
        return fmt.Errorf("register shutdown check: %w", err)
    }

    // Register Manager
    if err := gaz.For[*Manager](c).
        Provider(func(c *gaz.Container) (*Manager, error) {
            m := NewManager()
            // Wire up dependencies...
            return m, nil
        }); err != nil {
        return fmt.Errorf("register manager: %w", err)
    }

    return nil
}
```

**Bad:**
```go
// Wrong signature
func Module() error { ... }
func Module(c *gaz.Container) { ... }  // Missing error return

// Wrong name
func RegisterModule(c *gaz.Container) error { ... }
```

### ModuleBuilder Pattern

Use `NewModule(name).Provide(...).Build()` for bundling multiple providers.

**Rationale:** Clean composition with naming, flags, and child modules.

**Good:**
```go
// Source: gaz/module_builder.go
// Create module with providers
module := gaz.NewModule("database").
    Provide(func(c *gaz.Container) error {
        return gaz.For[*DB](c).Provider(NewDB)
    }).
    Build()

// Compose modules
observability := gaz.NewModule("observability").
    Use(loggingModule).
    Use(metricsModule).
    Build()

// Module with flags
redisModule := gaz.NewModule("redis").
    Flags(func(fs *pflag.FlagSet) {
        fs.String("redis-host", "localhost", "Redis server host")
    }).
    Provide(RedisProvider).
    Build()
```

**Bad:**
```go
// Direct struct construction
module := &builtModule{name: "database", ...}

// Missing Build() call
module := gaz.NewModule("database").Provide(...)  // Returns *ModuleBuilder, not Module
```

### Error Wrapping in Modules

Module registration errors MUST be wrapped with context using the pattern `"register X: %w"`.

**Rationale:** Clear error messages when module registration fails.

**Good:**
```go
// Source: gaz/health/module.go
if err := gaz.For[*Manager](c).Provider(...); err != nil {
    return fmt.Errorf("register manager: %w", err)
}

if err := gaz.For[*ManagementServer](c).Provider(...); err != nil {
    return fmt.Errorf("register management server: %w", err)
}
```

**Bad:**
```go
// No context
if err := gaz.For[*Manager](c).Provider(...); err != nil {
    return err
}

// Inconsistent format
if err := gaz.For[*Manager](c).Provider(...); err != nil {
    return fmt.Errorf("Manager registration failed: %w", err)
}
```

## Exception Process

When a convention cannot be followed:

1. **Document the reason** in a code comment referencing STYLE.md
2. **Get approval** in code review with explicit acknowledgment
3. **Justify why** the exception is necessary (not just convenient)

**Example:**
```go
// STYLE.md exception: Using NewHealthManager() instead of NewManager()
// because health package also exports health.Manager from external library.
// Approved in PR #123.
func NewHealthManager() *Manager {
    return &Manager{}
}
```

**When exceptions are appropriate:**
- External library constraints (naming conflicts)
- Backward compatibility requirements
- Performance-critical code paths

**When exceptions are NOT appropriate:**
- Personal preference
- "It's faster to write"
- "Other projects do it this way"

## Automatable Rules

The following conventions can be enforced with linters:

| Convention | Automatable | Linter |
|------------|-------------|--------|
| Error variable naming (`Err*` prefix) | Yes | Custom golangci-lint rule |
| Error message format (`pkg: msg`) | Partially | Custom rule |
| Package doc exists | Yes | revive |
| Doc comment starts with name | Yes | revive |
| Constructor naming (`New*`) | No | Semantic, context-dependent |

[AUTOMATABLE] markers appear inline where linter enforcement is feasible.

---

*Last updated: 2026-01-30*
*Phase: 23-foundation-style-guide*
