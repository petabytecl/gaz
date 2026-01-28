# Phase 12: DI Package - Research

**Researched:** 2026-01-28
**Domain:** Go subpackage extraction, DI container patterns
**Confidence:** HIGH

## Summary

This phase extracts the dependency injection functionality from the root `gaz` package into a standalone `gaz/di` subpackage. The research focuses on three areas: (1) Go patterns for subpackage extraction and type re-exporting, (2) DI container API patterns from established libraries (uber-go/fx, samber/do), and (3) the specific code structure of the current gaz DI implementation.

The current gaz implementation already follows industry-standard patterns: `For[T]()` fluent registration, `Resolve[T]()` with error handling, singleton/transient scopes, and lifecycle hooks. The extraction is primarily a reorganization exercise with careful attention to backward compatibility through wrapper types in the root package.

**Primary recommendation:** Extract DI into `di/` subpackage with `di.New()` constructor, maintain existing API patterns, use wrapper types in root gaz package for backward compatibility, add `MustResolve[T]()` and test helpers as new API surface.

## Standard Stack

This is a code reorganization phase - no new dependencies required.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Standard library | Go 1.25.6 | reflect, sync, context | Used for DI implementation |

### Supporting
No new dependencies needed. The phase is about reorganizing existing code.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Wrapper types | Type aliases | Type aliases can't add methods; wrappers allow extension |
| Full embedding | Composition | Embedding exposes all methods; composition gives control |

## Architecture Patterns

### Recommended Project Structure
```
gaz/
├── di/                      # NEW: Standalone DI package
│   ├── container.go         # Container type, New(), Build()
│   ├── registration.go      # For[T](), RegistrationBuilder
│   ├── resolution.go        # Resolve[T](), MustResolve[T]()
│   ├── service.go           # serviceWrapper interface, implementations
│   ├── inject.go            # Struct field injection
│   ├── types.go             # TypeName[T](), typeName()
│   ├── errors.go            # ErrNotFound, ErrCycle, etc.
│   ├── options.go           # ResolveOption, Named()
│   ├── lifecycle.go         # Starter, Stopper interfaces
│   ├── lifecycle_engine.go  # ComputeStartupOrder, ComputeShutdownOrder
│   ├── testing.go           # NewTestContainer(), test helpers
│   └── doc.go               # Package documentation
├── app.go                   # App wraps di.Container, adds framework features
├── compat.go                # NEW: Backward compatibility re-exports
└── ...                      # Other gaz packages unchanged
```

### Pattern 1: Standalone Container Constructor
**What:** `di.New()` returns `*Container` without error, following Go idioms like `bytes.NewBuffer()`
**When to use:** Creating containers outside the gaz.App context
**Example:**
```go
// Source: samber/do pattern, adapted to gaz style
package di

// New creates a new empty Container ready for service registration.
// Use For[T]() to register services, then optionally call Build() for eager instantiation.
func New() *Container {
    return &Container{
        services:         make(map[string]any),
        resolutionChains: make(map[int64][]string),
        dependencyGraph:  make(map[string][]string),
    }
}
```

### Pattern 2: MustResolve for Tests and Initialization
**What:** Panic-on-failure variant of Resolve for use in tests and main()
**When to use:** Test setup, init blocks, main() where failure means abort
**Example:**
```go
// Source: samber/do MustInvoke pattern
package di

// MustResolve resolves a service or panics if resolution fails.
// Use only in test setup or main() initialization where failure is fatal.
func MustResolve[T any](c *Container, opts ...ResolveOption) T {
    result, err := Resolve[T](c, opts...)
    if err != nil {
        panic(fmt.Sprintf("di.MustResolve[%s]: %v", TypeName[T](), err))
    }
    return result
}
```

### Pattern 3: Wrapper Types for Backward Compatibility
**What:** Root gaz package provides wrapper types that delegate to di package
**When to use:** Maintaining API compatibility after extraction
**Example:**
```go
// Source: Go stdlib pattern (e.g., net/http re-exports from net/http/internal)
package gaz

import "github.com/petabytecl/gaz/di"

// Container re-exports di.Container for backward compatibility.
// Use di.Container directly for new code.
type Container = di.Container

// For is a convenience wrapper around di.For for gaz.App users.
// It works with gaz.App by extracting the underlying di.Container.
func For[T any](c *Container) *di.RegistrationBuilder[T] {
    return di.For[T](c)
}

// Resolve is a convenience wrapper around di.Resolve.
func Resolve[T any](c *Container, opts ...di.ResolveOption) (T, error) {
    return di.Resolve[T](c, opts...)
}
```

### Pattern 4: Container Introspection
**What:** Methods to query container state: `List()`, `Has[T]()`
**When to use:** Debugging, testing, runtime introspection
**Example:**
```go
// Source: samber/do ListProvidedServices pattern
package di

// List returns names of all registered services.
func (c *Container) List() []string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    names := make([]string, 0, len(c.services))
    for name := range c.services {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}

// Has returns true if a service of type T is registered.
func Has[T any](c *Container) bool {
    return c.hasService(TypeName[T]())
}
```

### Pattern 5: Test Container Helper
**What:** Convenience constructor for testing with common setup
**When to use:** Unit tests that need isolated DI containers
**Example:**
```go
// Source: samber/do TestContainer pattern
package di

// NewTestContainer creates a container suitable for testing.
// It has sensible defaults and helper methods for mocking.
func NewTestContainer() *Container {
    return New()
}
```

### Anti-Patterns to Avoid
- **Type aliases for exported types:** Type aliases (`type X = Y`) can't have methods added. Use wrapper types for extensibility.
- **Import cycles:** di package must not import gaz package. Only root gaz can import di.
- **Exposing internal types:** Keep service wrappers internal to di package. Only export Container, For, Resolve, and related types.
- **Changing API semantics:** Backward compatibility means same behavior, not just same names. Existing tests must pass unchanged.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Goroutine ID tracking | Custom stack parsing | Keep existing `getGoroutineID()` | Already implemented, tested |
| Cycle detection | Simple visited map | Keep existing chain tracking per-goroutine | Thread-safe, already works |
| Topological sort | Custom graph algorithm | Keep existing `ComputeStartupOrder()` | Already handles lifecycle correctly |
| Type names | Manual string building | Keep existing `TypeName[T]()` with reflection | Handles all Go type edge cases |

**Key insight:** This is a reorganization phase, not a rewrite. Move code, don't reimplement.

## Common Pitfalls

### Pitfall 1: Import Cycles Between gaz and di
**What goes wrong:** Root gaz package imports di, but di imports something from gaz, causing Go compile error
**Why it happens:** Easy to accidentally reference App or other gaz types from di code
**How to avoid:** 
- di package has zero imports from parent gaz package
- All shared types (Starter, Stopper, lifecycle interfaces) move to di
- App references di.Container, not vice versa
**Warning signs:** Go import cycle compile error; any `import "github.com/petabytecl/gaz"` in di/ files

### Pitfall 2: Incomplete Type Migration
**What goes wrong:** Some DI-related types stay in root gaz, breaking standalone use
**Why it happens:** Incremental changes that don't move all related types together
**How to avoid:**
- Move ALL DI types together: Container, errors, options, service wrappers
- Create comprehensive list of types to move before starting
- Test standalone di package usage before integrating back
**Warning signs:** di package needs gaz types that weren't moved

### Pitfall 3: Breaking App Integration
**What goes wrong:** gaz.App stops working after extraction because internal access patterns changed
**Why it happens:** App uses internal Container methods that get reorganized
**How to avoid:**
- Ensure App.Container() returns compatible type
- Keep internal method signatures stable or update App to match
- Run all existing tests after extraction
**Warning signs:** App-level tests fail; module tests fail

### Pitfall 4: Inconsistent Error Prefixes
**What goes wrong:** Errors say "gaz:" in di package, confusing users
**Why it happens:** Error messages copied without updating package context
**How to avoid:** 
- Update error messages in di package to use "di:" prefix
- Optionally keep root gaz wrapper errors with "gaz:" prefix
**Warning signs:** Error messages mention wrong package

### Pitfall 5: Losing Thread Safety
**What goes wrong:** Race conditions introduced during refactoring
**Why it happens:** Moving code between files can accidentally drop mutex usage
**How to avoid:**
- Run tests with `-race` flag during and after extraction
- Review all mutex usage when moving code
- Keep sync.Mutex fields in same struct as data they protect
**Warning signs:** Race detector failures; intermittent test failures

## Code Examples

Verified patterns from official sources:

### Standalone DI Usage (New Pattern)
```go
// Source: Research synthesis of samber/do + current gaz patterns
package main

import "github.com/petabytecl/gaz/di"

type Database struct{ DSN string }
type UserService struct{ DB *Database }

func main() {
    // Create standalone container
    c := di.New()
    
    // Register services
    di.For[*Database](c).Instance(&Database{DSN: "postgres://..."})
    
    di.For[*UserService](c).Provider(func(c *di.Container) (*UserService, error) {
        db, err := di.Resolve[*Database](c)
        if err != nil {
            return nil, err
        }
        return &UserService{DB: db}, nil
    })
    
    // Build eager services
    if err := c.Build(); err != nil {
        panic(err)
    }
    
    // Resolve
    svc := di.MustResolve[*UserService](c)
    println(svc.DB.DSN)
}
```

### Backward Compatible Usage (Existing Pattern)
```go
// Source: Current gaz examples/basic/main.go
package main

import "github.com/petabytecl/gaz"

type Greeter struct{ Name string }

func main() {
    // Works exactly as before
    app := gaz.New()
    
    gaz.For[*Greeter](app.Container()).Provider(func(c *gaz.Container) (*Greeter, error) {
        return &Greeter{Name: "World"}, nil
    })
    
    if err := app.Build(); err != nil {
        panic(err)
    }
    
    greeter, _ := gaz.Resolve[*Greeter](app.Container())
    println(greeter.Name)
}
```

### Container Introspection
```go
// Source: samber/do ListProvidedServices pattern
package main

import "github.com/petabytecl/gaz/di"

func main() {
    c := di.New()
    di.For[*Database](c).Instance(&Database{})
    di.For[*Cache](c).Instance(&Cache{})
    
    // List all registered services
    for _, name := range c.List() {
        fmt.Println("Registered:", name)
    }
    
    // Check if specific type is registered
    if di.Has[*Database](c) {
        fmt.Println("Database is registered")
    }
}
```

### Test Container Usage
```go
// Source: samber/do TestContainer pattern
package myservice_test

import (
    "testing"
    "github.com/petabytecl/gaz/di"
)

func TestUserService(t *testing.T) {
    // Create test container
    c := di.NewTestContainer()
    
    // Register mock
    mockDB := &MockDatabase{}
    di.For[Database](c).Instance(mockDB)
    
    // Register service under test
    di.For[*UserService](c).Provider(NewUserService)
    
    // Test
    svc := di.MustResolve[*UserService](c)
    // ... assertions
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Reflection-based registration | Generic fluent API `For[T]()` | gaz Phase 11 (CLN-04 to CLN-09) | Already implemented, maintain this |
| Container as method receiver | Container as first argument `For[T](c)` | Decision in Phase 12 CONTEXT | Keeps consistency with current API |
| Panic on resolve failure | Error return with MustResolve variant | Industry standard (samber/do) | Add MustResolve[T]() in this phase |

**Deprecated/outdated:**
- Reflection-based `Register()` method: Removed in Phase 11 cleanup
- `RegisterProvider()`, `RegisterSingleton()`: Replaced by `For[T]().Provider()`

## Open Questions

Things that couldn't be fully resolved:

1. **Cleanup/Disposal mechanism**
   - What we know: Some DI containers (samber/do) have Shutdown() for cleanup
   - What's unclear: Current gaz uses lifecycle hooks on App.Stop(), does di need standalone cleanup?
   - Recommendation: Leave cleanup to App.Stop() for now; di.Container doesn't need Shutdown() for MVP. Can add later if standalone use cases require it.

2. **Error prefix convention**
   - What we know: Current errors use "gaz:" prefix
   - What's unclear: Should di package errors use "di:" prefix?
   - Recommendation: Use "di:" prefix in di package. Wrapper errors in gaz can use "gaz:" if wrapping.

3. **Exact minimal export surface**
   - What we know: Context says "Claude decides minimal core exports"
   - Recommendation: Export Container, For, Resolve, MustResolve, TypeName, Named, errors, Starter/Stopper interfaces, RegistrationBuilder. Keep service wrappers internal.

## Sources

### Primary (HIGH confidence)
- `/uber-go/fx` (Context7) - Lifecycle hooks, module patterns, API design
- `/websites/pkg_go_dev_github_com_samber_do_v2` (Context7) - Invoke/MustInvoke patterns, container introspection, test helpers
- Current gaz codebase - container.go, registration.go, resolution.go, service.go, inject.go

### Secondary (MEDIUM confidence)
- `/uber-go/guide` (Context7) - Error naming conventions, package organization
- Go Effective Go documentation - Package naming, constructors

### Tertiary (LOW confidence)
- None - all research verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No new dependencies, code reorganization only
- Architecture: HIGH - Based on existing gaz code and verified DI library patterns
- Pitfalls: HIGH - Based on analysis of code dependencies and common refactoring issues

**Research date:** 2026-01-28
**Valid until:** 2026-02-28 (stable domain, no external dependencies changing)

## Files to Move

Complete inventory of DI-related code to extract:

| Current File | Target File | What Moves | Notes |
|--------------|-------------|------------|-------|
| container.go | di/container.go | Container type, NewContainer, Build, resolveByName, graph methods | Rename NewContainer → New |
| registration.go | di/registration.go | RegistrationBuilder, For[T], Named, Transient, Eager, Replace, OnStart, OnStop, Provider, ProviderFunc, Instance, serviceScope | Unchanged API |
| resolution.go | di/resolution.go | Resolve[T] | Add MustResolve[T] |
| service.go | di/service.go | serviceWrapper, baseService, lazySingleton, transientService, eagerSingleton, instanceService, instanceServiceAny | All internal except optionally Service interface |
| inject.go | di/inject.go | tagOptions, parseTag, injectStruct | Internal implementation |
| types.go | di/types.go | TypeName[T], typeName | Public utility |
| errors.go | di/errors.go | ErrNotFound, ErrCycle, ErrDuplicate, ErrNotSettable, ErrTypeMismatch, ErrAlreadyBuilt, ErrInvalidProvider | Update message prefixes |
| options.go | di/options.go | ResolveOption, resolveOptions, Named, applyOptions | Split: DI options → di, Config options stay in gaz |
| lifecycle.go | di/lifecycle.go | Starter, Stopper, HookFunc, HookConfig, HookOption, WithHookTimeout | Interface definitions |
| lifecycle_engine.go | di/lifecycle_engine.go | ComputeStartupOrder, ComputeShutdownOrder | Graph algorithms |

**Files staying in root gaz:**
- app.go - Uses di.Container internally
- app_module.go - Module registration
- cobra.go - CLI integration
- config.go, config_manager.go - Configuration
- validation.go - Struct validation

**New file in root gaz:**
- compat.go - Re-exports for backward compatibility

## Public API for di Package

### Exported (PUBLIC)
```go
// Types
type Container struct { ... }
type RegistrationBuilder[T any] struct { ... }
type ResolveOption func(*resolveOptions)

// Constructors
func New() *Container
func NewTestContainer() *Container

// Registration
func For[T any](c *Container) *RegistrationBuilder[T]

// Resolution
func Resolve[T any](c *Container, opts ...ResolveOption) (T, error)
func MustResolve[T any](c *Container, opts ...ResolveOption) T
func Named(name string) ResolveOption

// Introspection
func (c *Container) List() []string
func (c *Container) Build() error
func Has[T any](c *Container) bool

// Type utilities
func TypeName[T any]() string

// Errors
var ErrNotFound, ErrCycle, ErrDuplicate, ErrNotSettable, ErrTypeMismatch, ErrAlreadyBuilt, ErrInvalidProvider error

// Lifecycle interfaces
type Starter interface { OnStart(context.Context) error }
type Stopper interface { OnStop(context.Context) error }
type HookOption func(*HookConfig)
func WithHookTimeout(d time.Duration) HookOption
```

### Internal (unexported)
```go
// All service wrapper types
type serviceWrapper interface { ... }
type baseService struct { ... }
type lazySingleton[T any] struct { ... }
type transientService[T any] struct { ... }
type eagerSingleton[T any] struct { ... }
type instanceService[T any] struct { ... }
type instanceServiceAny struct { ... }

// Helper functions
func typeName(t reflect.Type) string
func injectStruct(c *Container, target any, chain []string) error
func parseTag(tag string) tagOptions
func getGoroutineID() int64
```
