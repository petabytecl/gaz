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
