# Phase 1: Core DI Container - Research

**Researched:** 2026-01-26
**Domain:** Go dependency injection with generics
**Confidence:** HIGH

## Summary

This research investigates patterns for building a type-safe dependency injection container in Go 1.21+ with generics. The investigation examines existing Go DI libraries (samber/do, uber-go/fx, uber-go/dig, google/wire), the internal dibx codebase being extracted from, and Go stdlib patterns for reflection and struct tags.

The standard approach for Go DI containers combines:
1. **Generic functions** for type-safe registration (`Register[T]`) and resolution (`Resolve[T]`)
2. **Provider functions** with signature `func(*Container) (T, error)` for lazy instantiation
3. **Fluent builder pattern** for registration options (Named, Transient, Eager)
4. **Map-based storage** with `sync.RWMutex` for thread-safe service registry
5. **Runtime circular dependency detection** via virtual scope / invoker chain tracking
6. **Reflection-based struct tag injection** using `reflect.StructTag.Lookup()`

The CONTEXT.md decisions lock in a fluent API: `gaz.For[T](c).Provider(fn)` for registration and `gaz.Resolve[T](c)` for resolution, with explicit error returns (no panics).

**Primary recommendation:** Build the container following the samber/do pattern but with the fluent API from CONTEXT.md. Use runtime cycle detection (invoker chain tracking during resolution), not static graph analysis. Keep internal storage as `map[string]any` keyed by type name or explicit name.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `reflect` | 1.21+ | Type introspection, struct tags | Required for generic type names and field injection |
| Go stdlib `sync` | 1.21+ | `sync.RWMutex` for thread-safety | Standard Go concurrency pattern |
| Go stdlib `errors` | 1.21+ | Sentinel errors, wrapping | `errors.Is()`, `errors.As()`, `%w` for chains |
| Go stdlib `fmt` | 1.21+ | Error formatting | Dependency chain error messages |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/google/uuid` | v1.6+ | Unique container/scope IDs | Optional, for introspection/debugging |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Runtime cycle detection | Static graph at registration | Static is faster but loses lazy instantiation semantics |
| `map[string]any` | Type-keyed map | No clean way to key by type in Go; string is standard |
| `sync.RWMutex` | `sync.Map` | RWMutex is more predictable for DI access patterns |

**No external dependencies required** - the container is pure Go stdlib.

## Architecture Patterns

### Recommended Project Structure
```
gaz/
├── container.go          # Container struct, New(), Build()
├── registration.go       # For[T]() fluent builder, registration types
├── resolution.go         # Resolve[T](), named resolution
├── service.go            # Service interface and implementations (lazy, eager, transient)
├── errors.go             # Sentinel errors (ErrNotFound, ErrCycle, etc.)
├── inject.go             # Struct field injection via tags
├── types.go              # Type name utilities (reflect-based)
└── container_test.go     # Tests
```

### Pattern 1: Provider Function Type
**What:** Generic provider function signature that receives container for dependency resolution
**When to use:** All lazy and transient service registrations
**Example:**
```go
// Provider creates a service instance, receiving the container for resolving dependencies
type Provider[T any] func(*Container) (T, error)

// Also support simple providers without error
type SimpleProvider[T any] func(*Container) T
```
**Source:** samber/do uses `func(Injector) (T, error)` - verified via Context7

### Pattern 2: Fluent Builder for Registration
**What:** Generic entry point returns builder struct with chainable methods
**When to use:** All service registrations
**Example:**
```go
// For returns a registration builder for type T
func For[T any](c *Container) *RegistrationBuilder[T] {
    return &RegistrationBuilder[T]{
        container: c,
        name:      TypeName[T](),  // Default: inferred from type
        scope:     ScopeSingleton, // Default: singleton
        lazy:      true,           // Default: lazy instantiation
    }
}

// Builder methods
func (b *RegistrationBuilder[T]) Named(name string) *RegistrationBuilder[T]
func (b *RegistrationBuilder[T]) Transient() *RegistrationBuilder[T]
func (b *RegistrationBuilder[T]) Eager() *RegistrationBuilder[T]
func (b *RegistrationBuilder[T]) Replace() *RegistrationBuilder[T]
func (b *RegistrationBuilder[T]) Provider(fn Provider[T]) error
func (b *RegistrationBuilder[T]) Instance(val T) error
```
**Source:** CONTEXT.md decision (locked)

### Pattern 3: Service Wrapper Interface
**What:** Internal interface wrapping provider/instance with lifecycle metadata
**When to use:** Internal storage - not exposed to users
**Example:**
```go
// serviceWrapper is internal - stores the provider and cached instance
type serviceWrapper interface {
    name() string
    typeName() string
    isLazy() bool
    isTransient() bool
    getInstance(c *Container, chain []string) (any, error)
}

// lazySingleton implements serviceWrapper
type lazySingleton[T any] struct {
    serviceName string
    provider    Provider[T]
    instance    T
    built       bool
    mu          sync.Mutex
}
```
**Source:** dibx internal `Service[T]` interface pattern

### Pattern 4: Invoker Chain for Cycle Detection
**What:** Track resolution chain during provider execution to detect cycles at runtime
**When to use:** During `Resolve[T]()` and within provider execution
**Example:**
```go
// Resolution context tracks the current resolution chain
type resolutionContext struct {
    chain []string  // Service names being resolved
}

func (r *resolutionContext) detectCycle(name string) error {
    for _, svc := range r.chain {
        if svc == name {
            return fmt.Errorf("%w: %s", ErrCycle, 
                strings.Join(append(r.chain, name), " -> "))
        }
    }
    return nil
}
```
**Source:** dibx `virtualScope.detectCircularDependency()` - verified in codebase

### Pattern 5: Type Name from Generics
**What:** Generate consistent string keys from generic type parameters using reflection
**When to use:** Default service naming, type matching
**Example:**
```go
// TypeName returns the fully-qualified type name for T
func TypeName[T any]() string {
    var zero T
    return typeName(reflect.TypeOf(&zero).Elem())
}

func typeName(t reflect.Type) string {
    if name := t.Name(); name != "" {
        if pkg := t.PkgPath(); pkg != "" {
            return pkg + "." + name
        }
        return name
    }
    // Handle pointers, slices, etc.
    switch t.Kind() {
    case reflect.Pointer:
        return "*" + typeName(t.Elem())
    case reflect.Slice:
        return "[]" + typeName(t.Elem())
    // ... other kinds
    }
    return t.String()
}
```
**Source:** dibx `converter.GetType[T]()` - verified in codebase

### Anti-Patterns to Avoid
- **Storing injector in services:** Services should resolve dependencies in their provider, not store the container for later use (causes runtime errors instead of startup errors)
- **Interface{} return without type info:** Always track the concrete type name separately for error messages
- **Panic on missing service:** Return errors, not panics - let caller decide how to handle
- **Global default container:** Avoid global state; require explicit container parameter

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Type name generation | Custom reflect code | Pattern from dibx converter | Edge cases: pointers, slices, generics, unnamed types |
| Struct tag parsing | Manual string parsing | `reflect.StructTag.Lookup()` | Standard Go API handles quoting, escaping correctly |
| Error wrapping | String concatenation | `fmt.Errorf("context: %w", err)` | Enables `errors.Is()`, `errors.As()` |
| Thread-safe lazy init | Double-checked locking | `sync.Once` or mutex pattern | Avoid subtle race conditions |
| Panic recovery in providers | Ignore panics | `defer/recover` wrapper | Convert panics to errors for consistent handling |

**Key insight:** The dibx codebase already has working implementations of these patterns. Extract and simplify rather than rewrite.

## Common Pitfalls

### Pitfall 1: Interface Type Names
**What goes wrong:** Registering `*MyStruct` but resolving as `MyInterface` - names don't match
**Why it happens:** Go reflection gives different strings for interface types vs concrete types
**How to avoid:** 
- Use `reflect.Type.AssignableTo()` for interface matching
- Consider explicit interface binding: `Bind[*MyStruct, MyInterface]()`
**Warning signs:** "Service not found" when you know you registered it

### Pitfall 2: Pointer vs Value Types
**What goes wrong:** Register `*Config` but resolve `Config` (no pointer)
**Why it happens:** `reflect.TypeOf(Config{})` vs `reflect.TypeOf(&Config{})` are different
**How to avoid:** Document that types must match exactly; consider auto-handling `*T` when `T` is registered
**Warning signs:** Type mismatch errors despite seemingly correct registration

### Pitfall 3: Concurrent Resolution Races
**What goes wrong:** Multiple goroutines resolve the same lazy singleton, provider runs twice
**Why it happens:** Check-then-build race condition without proper locking
**How to avoid:** Hold mutex during build, use double-checked locking pattern:
```go
func (s *lazySingleton[T]) getInstance(c *Container, chain []string) (any, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.built {
        return s.instance, nil
    }
    
    instance, err := s.provider(c)
    if err != nil {
        return nil, err
    }
    
    s.instance = instance
    s.built = true
    return instance, nil
}
```
**Warning signs:** Intermittent errors in concurrent tests, multiple initialization logs

### Pitfall 4: Lost Error Context in Chains
**What goes wrong:** Deep dependency error says "failed to create X" without showing the full chain
**Why it happens:** Errors wrapped without the resolution path context
**How to avoid:** Include the dependency chain in error messages:
```go
return nil, fmt.Errorf("resolving %s -> %s: %w", 
    strings.Join(chain, " -> "), name, err)
```
**Warning signs:** Error messages that don't help identify which dependency failed

### Pitfall 5: Unexported Struct Fields
**What goes wrong:** Struct tag injection silently skips unexported fields
**Why it happens:** Go reflection can't set unexported fields (`CanSet() == false`)
**How to avoid:** Check `CanSet()` and return an error if a tagged field can't be set
**Warning signs:** Fields remain nil despite correct registration

### Pitfall 6: Eager Services Resolving Lazy Dependencies
**What goes wrong:** Eager service registered before its dependencies, panics on Build()
**Why it happens:** Eager services instantiate during registration/build, but dependencies may not be registered yet
**How to avoid:** 
- Process eager services in a separate Build() phase after all registrations
- Validate dependency graph at Build() time
**Warning signs:** Panics during container setup, order-dependent registration

## Code Examples

Verified patterns from official sources and the internal codebase:

### Fluent Registration (from CONTEXT.md)
```go
// Register a lazy singleton with provider
err := gaz.For[*DatabasePool](c).Provider(func(c *gaz.Container) (*DatabasePool, error) {
    config, err := gaz.Resolve[*Config](c)
    if err != nil {
        return nil, err
    }
    return NewDatabasePool(config)
})

// Register with a name
err := gaz.For[*sql.DB](c).Named("primary").Provider(NewPrimaryDB)

// Register a pre-built instance
err := gaz.For[*Config](c).Instance(loadedConfig)

// Register transient (new instance per resolution)
err := gaz.For[*RequestContext](c).Transient().Provider(NewRequestContext)

// Register eager (instantiate at Build time)
err := gaz.For[*ConnectionPool](c).Eager().Provider(NewConnectionPool)
```

### Resolution (from CONTEXT.md)
```go
// Basic resolution
db, err := gaz.Resolve[*DatabasePool](c)
if errors.Is(err, gaz.ErrNotFound) {
    // Handle missing dependency
}

// Named resolution
primaryDB, err := gaz.Resolve[*sql.DB](c, gaz.Named("primary"))
```

### Struct Tag Injection (from CONTEXT.md)
```go
type MyHandler struct {
    DB     *DatabasePool    `gaz:"inject"`
    Logger *slog.Logger     `gaz:"inject,name=app-logger"`
    Cache  *CacheService    `gaz:"inject,optional"`  // nil if not registered
    
    // Non-injected fields
    requestCount int
}

// Auto-injection on resolution
handler, err := gaz.Resolve[*MyHandler](c)
// handler.DB, handler.Logger are populated
// handler.Cache is nil if CacheService not registered and optional
```

### Sentinel Errors (from CONTEXT.md)
```go
var (
    ErrNotFound       = errors.New("gaz: service not found")
    ErrCycle          = errors.New("gaz: circular dependency detected")
    ErrDuplicate      = errors.New("gaz: service already registered")
    ErrNotSettable    = errors.New("gaz: field is not settable")
    ErrTypeMismatch   = errors.New("gaz: type mismatch")
)
```

### Panic Recovery in Providers (from dibx)
```go
// Source: dibx/provider.go
func handleProviderPanic[T any](
    provider Provider[T],
    c *Container,
) (svc T, err error) {
    defer func() {
        if r := recover(); r != nil {
            if e, ok := r.(error); ok {
                err = e
            } else {
                err = fmt.Errorf("gaz: provider panic: %v", r)
            }
        }
    }()
    
    return provider(c)
}
```

### Struct Field Iteration for Injection (from dibx)
```go
// Source: dibx/invoke.go pattern
func injectStruct(c *Container, target reflect.Value, chain []string) error {
    if target.Kind() != reflect.Ptr || target.Elem().Kind() != reflect.Struct {
        return errors.New("gaz: target must be pointer to struct")
    }
    
    structVal := target.Elem()
    structType := structVal.Type()
    
    for i := 0; i < structVal.NumField(); i++ {
        field := structType.Field(i)
        fieldVal := structVal.Field(i)
        
        tagValue, hasTag := field.Tag.Lookup("gaz")
        if !hasTag {
            continue
        }
        
        if !fieldVal.CanSet() {
            return fmt.Errorf("%w: field %s.%s", ErrNotSettable, 
                structType.Name(), field.Name)
        }
        
        // Parse tag: "inject" or "inject,name=foo" or "inject,optional"
        opts := parseTag(tagValue)
        
        serviceName := opts.name
        if serviceName == "" {
            serviceName = typeName(field.Type)
        }
        
        instance, err := resolveByName(c, serviceName, chain)
        if err != nil {
            if opts.optional && errors.Is(err, ErrNotFound) {
                continue  // Leave as zero value
            }
            return fmt.Errorf("injecting %s.%s: %w", 
                structType.Name(), field.Name, err)
        }
        
        instanceVal := reflect.ValueOf(instance)
        if !instanceVal.Type().AssignableTo(fieldVal.Type()) {
            return fmt.Errorf("%w: cannot assign %s to field %s.%s of type %s",
                ErrTypeMismatch, instanceVal.Type(), structType.Name(), 
                field.Name, fieldVal.Type())
        }
        
        fieldVal.Set(instanceVal)
    }
    
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `interface{}` containers | Generic containers with `[T any]` | Go 1.18 (2022) | Type-safe at compile time |
| Global container variable | Explicit container parameter | Modern best practice | Testable, no hidden state |
| Panic on errors | Return errors | Always preferred | Recoverable, testable |
| XML/config-based DI | Code-based registration | Go idiom | IDE support, type checking |
| Hierarchical scopes | Flat scopes (Singleton/Transient) | Simplification | Reduced complexity |

**Deprecated/outdated:**
- **samber/do v1 API:** v2 uses different registration functions; use v2 patterns
- **uber/dig reflect-based resolution:** fx provides better developer experience
- **google/wire code generation:** Still valid but adds build complexity

## Open Questions

Things that couldn't be fully resolved:

1. **Build() vs Lazy Eager Resolution**
   - What we know: Eager services should instantiate at startup, not on first resolve
   - What's unclear: Should Build() be required, or should first resolve of any service trigger eager instantiation?
   - Recommendation: Require explicit `Build()` call - makes eager instantiation timing explicit

2. **Interface Binding Syntax**
   - What we know: Often want to register `*ConcreteType` as `InterfaceType`
   - What's unclear: Best fluent API syntax for this
   - Recommendation: Consider `gaz.For[MyInterface](c).Provider(NewConcrete)` where Provider returns interface type, or explicit `gaz.Bind[*Concrete, MyInterface](c)`

3. **Reflection Type Cache**
   - What we know: Type name generation via reflection can be expensive
   - What's unclear: Impact on performance with many registrations
   - Recommendation: Profile before optimizing; likely negligible for <1000 services

## Sources

### Primary (HIGH confidence)
- `/samber/do` Context7 - Service registration, invocation, circular dependency detection, struct injection
- `/uber-go/fx` Context7 - Lifecycle management, named services, error handling patterns
- Internal dibx codebase - Existing implementation patterns (converter, service types, invoker chain)
- Internal CONTEXT.md - API decisions locked by user

### Secondary (MEDIUM confidence)
- `/google/wire` Context7 - Binding error messages, multiple bindings handling
- Go `reflect` package documentation - StructTag, Type, Value APIs

### Tertiary (LOW confidence)
- None - all findings verified with Context7 or codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Go stdlib only, no external deps needed
- Architecture: HIGH - Patterns extracted from working dibx codebase and verified against samber/do
- Pitfalls: HIGH - Observed directly in dibx code and documented in samber/do

**Research date:** 2026-01-26
**Valid until:** 2026-02-26 (stable domain, Go generics patterns well-established)
