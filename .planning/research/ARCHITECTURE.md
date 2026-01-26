# Architecture Patterns: Go Application Frameworks

**Domain:** Go Application Framework (DI + Lifecycle + CLI)
**Researched:** 2026-01-26
**Confidence:** HIGH (Context7 verified + local reference implementations)

## Executive Summary

Go application frameworks like uber-go/fx, google/wire, and go-kit follow distinct architectural patterns. Based on research of these frameworks and the existing dibx/gazx reference implementations, this document outlines the recommended architecture for **gaz** - a core package with optional subpackages for health, config, and logging.

**Key architectural decisions:**
- Reflection-based DI (not code generation) - aligns with type-safe generics goal
- Flat scope model (single container, no hierarchical scopes)
- Lifecycle hooks pattern similar to fx.Lifecycle
- Module/Provider pattern for extensibility

---

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              USER APPLICATION                            │
│  main.go: gaz.New(cmd).With(providers...).Run()                         │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           gaz (CORE PACKAGE)                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │
│  │   App        │  │  Container   │  │  Lifecycle   │  │  Providers  │ │
│  │   Builder    │──│  (DI)        │──│  Manager     │──│  (Modules)  │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └─────────────┘ │
│         │                 │                 │                 │         │
│         ▼                 ▼                 ▼                 ▼         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    Signal Handling + Graceful Shutdown           │   │
│  └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
          ┌─────────────────────────┼─────────────────────────┐
          ▼                         ▼                         ▼
┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│  gaz/health      │    │  gaz/config      │    │  gaz/log         │
│  (OPTIONAL)      │    │  (OPTIONAL)      │    │  (OPTIONAL)      │
│  HealthManager   │    │  ConfigLoader    │    │  Logger Setup    │
│  Liveness/Ready  │    │  Viper/Cobra     │    │  slog handlers   │
└──────────────────┘    └──────────────────┘    └──────────────────┘
```

---

## Component Responsibilities

| Component | Responsibility | Communicates With | Build Order |
|-----------|---------------|-------------------|-------------|
| **Container** | Type-safe DI, service registry, resolution | All components | 1st (foundation) |
| **App Builder** | Fluent API for app construction, Cobra integration | Container, Lifecycle, Providers | 2nd (uses Container) |
| **Lifecycle Manager** | OnStart/OnStop hooks, ordered execution | Container, App Builder | 3rd (uses Container) |
| **Provider System** | Module definition, dependency declaration | Container, Lifecycle | 4th (uses all core) |
| **Health Subpackage** | Health checks, liveness/readiness probes | Container (optional) | 5th (independent) |
| **Config Subpackage** | Configuration loading, Cobra flag binding | Container (optional) | 6th (independent) |
| **Log Subpackage** | slog setup, structured logging | Container (optional) | 7th (independent) |

---

## Recommended Architecture

### Layer 1: Container (DI Core)

The foundation layer providing type-safe dependency injection with generics.

**Pattern:** Reflection-based DI with generic type inference
**Source:** dibx reference implementation, verified against fx patterns (Context7)

```go
// Core container interface
type Container interface {
    Register(service any, opts ...Option) Container
    Resolve(target any) error
    MustResolve(target any)
}

// Generic provider type for type-safe registration
type Provider[T any] func(Container) (T, error)

// Service lifecycle types
type ServiceType string
const (
    ServiceTypeLazy      ServiceType = "lazy"      // Created on first use
    ServiceTypeEager     ServiceType = "eager"     // Created immediately
    ServiceTypeTransient ServiceType = "transient" // Created each request
)
```

**Key Design Decisions:**

| Decision | Choice | Rationale |
|----------|--------|-----------|
| DI Approach | Reflection-based | Generics require runtime type info; Wire code-gen doesn't support generics well |
| Type Safety | Generic Provider[T] | Eliminates interface{} casting at call sites |
| Scope Model | Flat (single scope) | Simpler than fx's hierarchical scopes; covers 90% of use cases |
| Registration API | Fluent chaining | `c.Register(x).Register(y)` pattern from dibx |

### Layer 2: Lifecycle Management

Manages application startup/shutdown with ordered hooks.

**Pattern:** Hook-based lifecycle (inspired by fx.Lifecycle)
**Source:** fx documentation (Context7), gazx reference implementation

```go
// Lifecycle interface for components needing start/stop
type Lifecycle interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
}

// LifecycleManager orchestrates component lifecycles
type LifecycleManager struct {
    components []Lifecycle
    started    bool
}

// Hook for ad-hoc lifecycle callbacks
type Hook struct {
    OnStart func(ctx context.Context) error
    OnStop  func(ctx context.Context) error
}
```

**Lifecycle Flow:**

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Build     │───▶│   Start     │───▶│    Run      │───▶│    Stop     │
│   Phase     │    │   Phase     │    │   Phase     │    │   Phase     │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
      │                  │                  │                  │
      ▼                  ▼                  ▼                  ▼
 Register         OnStart hooks      Wait for          OnStop hooks
 providers        (in order)         signal            (reverse order)
```

### Layer 3: App Builder

The user-facing API for constructing applications.

**Pattern:** Fluent Builder with Cobra integration
**Source:** gazx reference implementation

```go
type App struct {
    cmd       *cobra.Command
    container Container
    lifecycle *LifecycleManager
    providers []*Provider
}

func New(cmd *cobra.Command) *App {
    return &App{
        cmd:       cmd,
        container: NewContainer(),
        lifecycle: NewLifecycleManager(),
    }
}

func (a *App) With(p *Provider) *App {
    a.providers = append(a.providers, p)
    return a
}

func (a *App) Run() error {
    // Build → Start → Wait → Stop
}
```

### Layer 4: Provider/Module System

Encapsulates related services, flags, and lifecycle components.

**Pattern:** Module composition (similar to fx.Module)
**Source:** fx.Module pattern (Context7), gazx ModuleProvider

```go
type Provider struct {
    Name      string
    Flags     func(*pflag.FlagSet)          // CLI flag registration
    Services  []func(Container)              // Service providers
    Lifecycle []Lifecycle                    // Lifecycle components
    OnStart   []func(Container) error        // Start hooks
    OnStop    []func(Container) error        // Stop hooks
    Requires  []*Provider                    // Dependencies
}

func NewProvider(name string) *Provider {
    return &Provider{Name: name}
}

func (p *Provider) WithService(fn func(Container)) *Provider {
    p.Services = append(p.Services, fn)
    return p
}
```

---

## Data Flow

### Registration Flow (Build Phase)

```
User Code                Container                    Service Registry
    │                        │                              │
    │ Register(provider)     │                              │
    ├───────────────────────▶│                              │
    │                        │ Infer type from Provider[T]  │
    │                        │◀────────────────────────────▶│
    │                        │                              │
    │                        │ Store: name → ServiceEntry   │
    │                        ├─────────────────────────────▶│
    │                        │                              │
    │ Container (chainable)  │                              │
    │◀───────────────────────┤                              │
```

### Resolution Flow (Runtime)

```
User Code                Container                    Service Registry
    │                        │                              │
    │ Resolve(&target)       │                              │
    ├───────────────────────▶│                              │
    │                        │ Lookup by type name          │
    │                        ├─────────────────────────────▶│
    │                        │                              │
    │                        │      ServiceEntry            │
    │                        │◀─────────────────────────────┤
    │                        │                              │
    │                        │ [if lazy & not instantiated] │
    │                        │ Call Provider function       │
    │                        │                              │
    │                        │ [if has dependencies]        │
    │                        │ Recursively resolve deps     │
    │                        │                              │
    │       target set       │                              │
    │◀───────────────────────┤                              │
```

### Application Lifecycle Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                         APPLICATION LIFECYCLE                         │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  1. BUILD PHASE                                                       │
│     ├─ Create Container                                               │
│     ├─ Register core services (cmd, shutdowner)                       │
│     ├─ For each Provider:                                             │
│     │   ├─ Register required Providers (recursively)                  │
│     │   ├─ Register Services                                          │
│     │   └─ Register Lifecycle components                              │
│     └─ Validate dependencies (optional strict mode)                   │
│                                                                       │
│  2. START PHASE                                                       │
│     ├─ Start LifecycleManager (in registration order)                 │
│     ├─ Execute Provider.OnStart hooks                                 │
│     └─ Log "Application started"                                      │
│                                                                       │
│  3. RUN PHASE                                                         │
│     ├─ Wait for shutdown signal (SIGTERM, SIGINT, SIGQUIT)            │
│     └─ OR wait for programmatic Shutdown() call                       │
│                                                                       │
│  4. STOP PHASE                                                        │
│     ├─ Execute Provider.OnStop hooks (reverse order)                  │
│     ├─ Stop LifecycleManager (reverse order)                          │
│     ├─ Call Shutdown on all Shutdowner services                       │
│     └─ Log "Application stopped"                                      │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Project Structure

### Recommended Package Layout

```
gaz/
├── go.mod
├── gaz.go              # Main entry point: New(), Run()
├── container.go        # DI container implementation
├── container_test.go
├── lifecycle.go        # Lifecycle manager
├── lifecycle_test.go
├── provider.go         # Provider/Module system
├── provider_test.go
├── app.go              # App builder
├── app_test.go
├── options.go          # Configuration options
├── errors.go           # Error types
├── types.go            # Core type definitions
│
├── health/             # Optional: Health checks
│   ├── health.go       # HealthManager, HealthCheck interface
│   ├── health_test.go
│   ├── provider.go     # Health provider for gaz integration
│   └── types.go        # HealthStatus, HealthResult, etc.
│
├── config/             # Optional: Configuration
│   ├── config.go       # Config loading, Viper integration
│   ├── config_test.go
│   ├── provider.go     # Config provider for gaz integration
│   └── bind.go         # Cobra flag binding helpers
│
├── log/                # Optional: Logging
│   ├── log.go          # slog setup, handlers
│   ├── log_test.go
│   ├── provider.go     # Log provider for gaz integration
│   └── handlers.go     # Custom slog handlers (JSON, text, etc.)
│
└── internal/           # Internal utilities
    └── reflect/        # Reflection helpers for DI
```

### Import Relationships

```
gaz/log ──────┐
              │
gaz/config ───┼──▶ gaz (core)
              │
gaz/health ───┘

Note: Subpackages depend on core, never the reverse.
Core can work without any subpackage (optional dependencies).
```

---

## Architectural Patterns to Follow

### Pattern 1: Generic Provider Registration

**What:** Use generics for type-safe service registration
**When:** All service registrations
**Why:** Eliminates interface{} casting, compile-time type checking

```go
// Good: Type-safe registration
func Provide[T any](c Container, provider func(Container) (T, error)) {
    name := typeNameOf[T]()
    c.register(name, &serviceEntry[T]{
        provider: provider,
        lifecycle: Lazy,
    })
}

// Usage - compiler knows types
Provide(c, func(c Container) (*UserService, error) {
    db := MustGet[*DB](c)
    return NewUserService(db), nil
})
```

### Pattern 2: Hook-Based Lifecycle

**What:** Components declare OnStart/OnStop callbacks
**When:** Managing startup/shutdown order
**Why:** Decouples component lifecycle from main application flow

```go
// Component registers its own lifecycle hooks
func NewHTTPServer(lifecycle *LifecycleManager, cfg *Config) *http.Server {
    srv := &http.Server{Addr: cfg.Addr}
    
    lifecycle.Append(Hook{
        OnStart: func(ctx context.Context) error {
            go srv.ListenAndServe()
            return nil
        },
        OnStop: func(ctx context.Context) error {
            return srv.Shutdown(ctx)
        },
    })
    
    return srv
}
```

### Pattern 3: Fluent Builder API

**What:** Chainable methods returning `*App`
**When:** Application construction
**Why:** Readable, IDE-friendly, self-documenting

```go
// Fluent construction
app := gaz.New(rootCmd).
    With(config.Provider()).
    With(log.Provider()).
    With(health.Provider()).
    With(myapp.Provider()).
    WithShutdownTimeout(30 * time.Second)

if err := app.Run(); err != nil {
    log.Fatal(err)
}
```

### Pattern 4: Optional Subpackage Integration

**What:** Core works standalone; subpackages enhance functionality
**When:** Designing extensibility
**Why:** Users only import what they need

```go
// Core only - no health, config, or log
app := gaz.New(cmd).With(myProvider).Run()

// With optional packages
app := gaz.New(cmd).
    With(health.Provider()).  // Adds /health, /ready endpoints
    With(config.Provider()).  // Adds viper config loading
    With(log.Provider()).     // Adds slog setup
    With(myProvider).
    Run()
```

### Pattern 5: Struct Tag-Based Resolution

**What:** Resolve dependencies via struct tags
**When:** Injecting multiple dependencies into a struct
**Why:** Reduces boilerplate, enables named dependencies

```go
type ServiceDeps struct {
    Logger *slog.Logger `gaz:""`
    DB     *sql.DB      `gaz:"name:primary"`
    Cache  *redis.Client `gaz:"optional"`
}

func NewService(c Container) (*Service, error) {
    var deps ServiceDeps
    if err := c.Resolve(&deps); err != nil {
        return nil, err
    }
    return &Service{deps: deps}, nil
}
```

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Hierarchical Scopes

**What:** Nested DI scopes (parent/child containers)
**Why bad:** Adds complexity, confuses service resolution, harder debugging
**Instead:** Use flat scope with named services for disambiguation

```go
// BAD: Hierarchical scopes
rootScope := gaz.New()
requestScope := rootScope.Scope("request")  // Creates child scope
requestScope.Register(requestLogger)         // Overrides parent

// GOOD: Flat scope with names
c := gaz.New()
c.Register(globalLogger)
c.Register(requestLogger, gaz.WithName("request"))
```

### Anti-Pattern 2: Service Locator Pattern

**What:** Passing container everywhere, calling Resolve() ad-hoc
**Why bad:** Hides dependencies, makes testing harder, violates IoC
**Instead:** Inject dependencies explicitly in providers

```go
// BAD: Service locator
func HandleRequest(c Container, req Request) {
    logger := MustGet[*Logger](c)  // Hidden dependency
    db := MustGet[*DB](c)          // Hidden dependency
    // ...
}

// GOOD: Explicit injection
func NewHandler(logger *Logger, db *DB) *Handler {
    return &Handler{logger: logger, db: db}
}
```

### Anti-Pattern 3: Circular Dependencies

**What:** A depends on B depends on A
**Why bad:** Cannot instantiate, deadlock during resolution
**Instead:** Extract shared functionality to third service, use lazy resolution, or use interfaces

```go
// BAD: Circular
type ServiceA struct { B *ServiceB }
type ServiceB struct { A *ServiceA }

// GOOD: Break with interface
type ServiceA struct { B BInterface }
type ServiceB struct { /* no A reference */ }
type Orchestrator struct { A *ServiceA; B *ServiceB }
```

### Anti-Pattern 4: Eager Everything

**What:** Instantiating all services at startup
**Why bad:** Slow startup, wastes memory for unused services
**Instead:** Use lazy (default) for most services, eager only for startup validation

```go
// BAD: All eager
c.Register(serviceA, gaz.AsEager())
c.Register(serviceB, gaz.AsEager())
c.Register(serviceC, gaz.AsEager())

// GOOD: Lazy by default, eager for critical services only
c.Register(serviceA)  // Lazy (default)
c.Register(serviceB)  // Lazy
c.Register(dbPool, gaz.AsEager())  // Eager - validate connection at startup
```

### Anti-Pattern 5: God Provider

**What:** Single provider registering 50+ services
**Why bad:** Hard to understand, test, and maintain
**Instead:** Break into focused providers by domain

```go
// BAD: God provider
var AppProvider = gaz.NewProvider("app").
    WithService(newUserService).
    WithService(newOrderService).
    WithService(newPaymentService).
    // ... 47 more services

// GOOD: Domain-focused providers
var UserProvider = gaz.NewProvider("user").WithService(newUserService)
var OrderProvider = gaz.NewProvider("order").WithService(newOrderService)
var PaymentProvider = gaz.NewProvider("payment").WithService(newPaymentService)

app.With(UserProvider).With(OrderProvider).With(PaymentProvider)
```

---

## Framework Comparison Summary

| Aspect | uber-go/fx | google/wire | go-kit | gaz (Recommended) |
|--------|-----------|-------------|--------|-------------------|
| **DI Approach** | Reflection | Code generation | Manual | Reflection + Generics |
| **Type Safety** | Runtime | Compile-time | N/A | Compile-time (generics) |
| **Scope Model** | Hierarchical | N/A | N/A | Flat |
| **Lifecycle** | Hook-based | N/A | N/A | Hook-based |
| **Modules** | fx.Module | wire.NewSet | Endpoints | Provider pattern |
| **CLI Integration** | None | None | None | Cobra-native |
| **Learning Curve** | Medium | Low | High | Low |

---

## Build Order Implications

Based on component dependencies, the recommended implementation order:

```
Phase 1: Foundation
├── Container (DI core with generics)
├── Types (Provider[T], Container interface)
└── Options (registration options)

Phase 2: Lifecycle
├── Lifecycle interface and manager
├── Hook system
└── Shutdown handling

Phase 3: App Builder
├── App struct and fluent API
├── Cobra integration
└── Signal handling

Phase 4: Provider System
├── Provider struct
├── Module composition
└── Dependency resolution

Phase 5: Optional Subpackages (can be parallel)
├── gaz/health (HealthManager, checks)
├── gaz/config (Viper, Cobra binding)
└── gaz/log (slog setup)
```

---

## Sources

**HIGH Confidence (Context7 verified):**
- uber-go/fx: Module pattern, Lifecycle hooks, fx.Provide/fx.Invoke
- google/wire: Code generation patterns, Provider sets
- spf13/viper: Config binding, Cobra integration

**HIGH Confidence (Local reference implementations):**
- tmp/dibx/: Reflection-based DI with generics, scope patterns
- tmp/gazx/: App builder, ModuleProvider, lifecycle management

**MEDIUM Confidence (Official docs):**
- go-kit/kit: Endpoint/Transport/Service layering pattern
