# Domain Pitfalls: Go Application Framework / DI Container

**Domain:** Go DI Framework Design
**Researched:** 2026-01-26
**Focus:** What Go DI/framework projects commonly get wrong
**Confidence:** HIGH (verified with Context7, official documentation)

---

## Critical Pitfalls

Mistakes that cause rewrites, fundamental design changes, or make the framework unusable.

---

### Pitfall 1: Over-Abstraction / "Magic" Container Behavior

**What goes wrong:** Framework hides too much, making debugging impossible and behavior unpredictable. Users can't understand why their service wasn't instantiated or why shutdown happened in a particular order.

**Why it happens:** 
- Trying to make the API "convenient" by hiding complexity
- Copying patterns from Java/Spring without adapting to Go idioms
- Runtime reflection-heavy implementations that obscure control flow

**Consequences:**
- Users can't debug their own applications
- Error messages are cryptic ("failed to build container" with no context)
- Behavior differs between development and production
- Users abandon framework for explicit wiring

**Warning signs:**
- Users frequently ask "why isn't my service being created?"
- Stack traces don't show user code
- Need to read framework source to understand behavior

**Prevention:**
```go
// BAD: Magic that's hard to trace
container.AutoRegister("./services/...") // What got registered? In what order?

// GOOD: Explicit but ergonomic
app := gaz.New(
    gaz.Provide(NewUserService),  // Clear what's provided
    gaz.Provide(NewDB),
)
// Error shows exactly: "UserService requires DB, but DB.Connect failed: ..."
```

**Detection:** If you can't explain what happens at startup by reading main.go, you've gone too far.

**Phase to address:** Core Container Design (Phase 1)

---

### Pitfall 2: Reflection Performance at Hot Paths

**What goes wrong:** Using reflection for every service resolution, causing measurable latency in request handling paths.

**Why it happens:**
- Runtime DI is easiest to implement with reflection
- Assuming "startup-only" reflection, but resolution happens per-request
- Not benchmarking request-time vs startup-time operations

**Consequences:**
- 10-100x slower service resolution than direct calls
- Memory pressure from reflect.Value allocations
- Breaks when users have thousands of scoped requests/second

**Warning signs:**
- `reflect.*` appears in CPU profiles during request handling
- Memory allocations correlate with request volume
- Scoped services are significantly slower than singleton services

**Prevention:**
```go
// Approach 1: Code generation (like Wire)
// Pro: Zero runtime overhead
// Con: Build step, harder DX

// Approach 2: Cached reflection (like Fx/Dig)
// Resolve types once at startup, cache func pointers
type cachedProvider struct {
    fn     reflect.Value
    inTypes  []reflect.Type  // Pre-computed
    outTypes []reflect.Type  // Pre-computed
}

// Approach 3: Generics (like samber/do v2)
// Type-safe at compile time, no reflection for typed paths
func Invoke[T any](i *Injector) (T, error) {
    // Direct type assertion, no reflect
}
```

**What gaz should do:** Use generics for type-safe paths (Go 1.18+), reserve reflection only for dynamic registration (rare). Benchmark resolution in hot path.

**Phase to address:** Core Container Design (Phase 1), Performance Validation (Phase 3)

---

### Pitfall 3: Scope Complexity Explosion

**What goes wrong:** Framework offers too many scope types (Singleton, Scoped, Transient, PerRequest, PerGraph, Custom...), users don't know which to use, bugs arise from scope mismatches.

**Why it happens:**
- Copying scope models from other frameworks (ASP.NET, Spring)
- Adding scopes reactively when users request them
- Not realizing most Go apps need only 2-3 scope types

**Consequences:**
- "Captive dependency" bugs (singleton holding scoped reference)
- Mental overhead choosing scope for every service
- Scope-related memory leaks
- Framework complexity

**Warning signs:**
- Users frequently ask "which scope should I use?"
- Bugs where service holds stale reference
- Documentation needs extensive scope explanation

**Prevention:**
```go
// BAD: Too many choices
gaz.Singleton()    // Once per container
gaz.Scoped()       // Once per scope
gaz.Transient()    // Every time
gaz.PerRequest()   // What's different from Scoped?
gaz.PerGraph()     // What does this even mean?

// GOOD: Minimal scopes with clear semantics
// Default: Singleton (created once, cached)
gaz.Provide(NewDB)  // Singleton by default, what users want 90% of time

// Explicit: Transient (new instance every invocation)
gaz.Transient(NewRequestContext)  // Clear when you need fresh instance

// Scopes: Create explicit sub-containers for request/job context
reqScope := app.Scope()  // Child scope inherits singletons
defer reqScope.Close()   // Cleanup
```

**Phase to address:** Core Container Design (Phase 1), API Design (Phase 2)

---

### Pitfall 4: Lifecycle Hook Ordering Confusion

**What goes wrong:** OnStart/OnStop hooks run in unexpected order, resources are accessed after they're closed, startup hangs because of circular hook dependencies.

**Why it happens:**
- Hooks registered implicitly during provider calls
- Order determined by internal graph, not registration order
- Stop hooks don't mirror start hook order (LIFO expected)

**Consequences:**
- Database connection used after Close()
- HTTP server can't stop because background worker holds connection
- Deadlocks on shutdown
- Flaky tests

**Warning signs:**
- "use of closed connection" errors during shutdown
- Shutdown hangs requiring SIGKILL
- Different behavior in tests vs production

**Prevention:**
```go
// Document and enforce: Stop hooks run in REVERSE of Start
// Fx does this correctly:
lc.Append(fx.Hook{
    OnStart: func(ctx context.Context) error {
        // Start in dependency order: DB first, then Server
        return server.Start()
    },
    OnStop: func(ctx context.Context) error {
        // Stop in REVERSE: Server first, then DB
        return server.Stop()
    },
})

// Make order visible
app.LifecycleOrder() // Returns ordered list for debugging

// Timeout ALL lifecycle operations
fx.StartTimeout(30*time.Second),
fx.StopTimeout(30*time.Second),
```

**Critical rule:** If service A depends on B, then:
- Start order: B.OnStart() → A.OnStart()
- Stop order: A.OnStop() → B.OnStop() (reverse!)

**Phase to address:** Lifecycle Management (Phase 2)

---

### Pitfall 5: Ignoring Cleanup/Shutdown

**What goes wrong:** Framework doesn't provide consistent way to clean up resources, leading to leaked connections, goroutines, file handles.

**Why it happens:**
- Focus on "getting services" not "releasing services"
- Go's GC gives false confidence (connections aren't GC'd properly)
- Shutdown often an afterthought

**Consequences:**
- Database connection pool exhaustion
- Goroutine leaks in tests
- Files left open
- "Too many open files" in production

**Warning signs:**
- Tests fail with resource exhaustion when run in parallel
- Production metrics show growing connection counts
- Memory grows over time even with stable traffic

**Prevention:**
```go
// Wire approach: Cleanup functions
func ProvideFile(path string) (*os.File, func(), error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, nil, err
    }
    cleanup := func() { f.Close() }
    return f, cleanup, nil
}

// samber/do approach: Shutdown interface
type Shutdownable interface {
    Shutdown(ctx context.Context) error
}

// Injector calls Shutdown on all services implementing it

// CRITICAL: Always support context for timeout
type ShutdownWithContext interface {
    Shutdown(ctx context.Context) error
}
```

**Phase to address:** Lifecycle Management (Phase 2)

---

## Technical Debt Patterns

Mistakes that cause delays, maintenance burden, and gradual degradation.

---

### Debt 1: Global Container Singleton

**What goes wrong:** Framework encourages or requires global container variable, making testing painful and preventing multiple isolated instances.

**Why it happens:**
- Convenient for initial examples
- Viper/other Go libs set this precedent
- Avoids passing injector everywhere

**Consequences:**
- Tests can't run in parallel (shared state)
- Can't have multiple containers with different config
- Hidden dependencies on global state
- Refactoring requires touching every file

**Prevention:**
```go
// BAD: Global singleton
var container = gaz.New()  // Package-level

func GetUserService() *UserService {
    return container.Invoke[*UserService]()  // Hidden dependency
}

// GOOD: Explicit container passing
func main() {
    app := gaz.New(...)
    
    // Container is parameter, not global
    server := NewServer(app)
    server.Run()
}
```

**Phase to address:** Core Container Design (Phase 1)

---

### Debt 2: Stringly-Typed Dependencies

**What goes wrong:** Services identified by string names, losing compile-time safety.

**Why it happens:**
- Easy to implement
- Allows dynamic registration
- Coming from Python/JS patterns

**Consequences:**
- Typos cause runtime errors, not compile errors
- Refactoring breaks hidden string references
- IDE can't help with discovery
- Can't use Go tooling effectively

**Warning signs:**
- Services registered with `container.Provide("userService", NewUserService)`
- Resolution like `container.Invoke("userService")`
- Runtime panics about "service not found"

**Prevention:**
```go
// BAD: String names
container.Provide("primary-db", NewDB)
container.Invoke("primary-db")  // Typo? Runtime error.

// GOOD: Type-based (generics)
gaz.Provide(injector, NewDB)
db := gaz.MustInvoke[*sql.DB](injector)  // Type-safe

// For multiple instances of same type, use wrapper types
type PrimaryDB struct{ *sql.DB }
type ReplicaDB struct{ *sql.DB }

gaz.Provide(injector, func() PrimaryDB { return PrimaryDB{primaryConn} })
gaz.Provide(injector, func() ReplicaDB { return ReplicaDB{replicaConn} })
```

**Phase to address:** Core Container Design (Phase 1)

---

### Debt 3: No Health Check Integration

**What goes wrong:** Framework provides services but no way to check if they're healthy, leading to silent failures.

**Why it happens:**
- Health checks seen as "application concern"
- Added as afterthought, inconsistent API
- Different services have different health semantics

**Consequences:**
- K8s kills pods without knowing why
- Database down but app reports "healthy"
- No unified health endpoint

**Prevention:**
```go
// Provide optional health check interface
type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}

// Framework aggregates all health checks
status := injector.HealthCheckWithContext(ctx)
// Returns map[string]error for each service

// Configure timeouts and parallelism
gaz.NewWithOpts(&gaz.Opts{
    HealthCheckTimeout:       100 * time.Millisecond,
    HealthCheckParallelism:   10,
})
```

**Phase to address:** Lifecycle Management (Phase 2), Extensions (Phase 4)

---

### Debt 4: Constructor Signature Explosion

**What goes wrong:** Constructors require 10+ parameters because framework injects everything as separate arguments.

**Why it happens:**
- Direct translation of dependencies to constructor args
- No grouping mechanism
- Adding dependencies means changing signatures

**Consequences:**
- Hard to read constructors
- Breaking changes when adding dependencies
- Mocking in tests requires many stubs

**Prevention:**
```go
// BAD: Signature explosion
func NewServer(
    db *sql.DB,
    cache *redis.Client,
    logger *zap.Logger,
    config *Config,
    metrics *Metrics,
    tracer *Tracer,
    authService *AuthService,
    userService *UserService,
    // ... growing forever
) *Server

// GOOD: Parameter struct (Fx pattern)
type ServerParams struct {
    fx.In
    DB      *sql.DB
    Cache   *redis.Client `optional:"true"`
    Logger  *zap.Logger
    Config  *Config
}

func NewServer(p ServerParams) *Server {
    // Access via p.DB, p.Cache, etc.
}

// GOOD: Functional options for optional deps
func NewServer(db *sql.DB, logger *zap.Logger, opts ...ServerOption) *Server
```

**Phase to address:** API Design (Phase 2)

---

## Performance Traps

Patterns that cause performance degradation.

---

### Trap 1: Allocating on Every Resolution

**What goes wrong:** Each `Invoke[T]()` call allocates even for cached singletons.

**Why it happens:**
- Type assertion boxes values
- Map lookups allocate on miss path
- Creating temporary closures

**Benchmark to include:**
```go
func BenchmarkInvoke(b *testing.B) {
    injector := setupInjector()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = gaz.MustInvoke[*UserService](injector)
    }
}
// Target: < 100 ns/op, 0 allocs/op for cached singleton
```

**Prevention:**
```go
// Cache typed accessors
type typedCache[T any] struct {
    once    sync.Once
    value   T
}

// Pre-warm common types at startup
injector.WarmCache[*UserService]()
```

**Phase to address:** Core Container Design (Phase 1), Performance (Phase 3)

---

### Trap 2: Lock Contention on Container

**What goes wrong:** Single RWMutex protects entire container, creating contention under load.

**Why it happens:**
- Simple implementation uses single lock
- Works fine in tests with low concurrency
- Not tested under production load

**Warning signs:**
- Lock contention visible in profiles
- Latency increases non-linearly with concurrency
- pprof shows mutex wait time

**Prevention:**
```go
// Shard by type, not single lock
type Injector struct {
    shards [256]struct {
        sync.RWMutex
        services map[reflect.Type]any
    }
}

func (i *Injector) shard(t reflect.Type) *shard {
    hash := fnv.Sum64(t.String())
    return &i.shards[hash % 256]
}

// Or: Lock-free for read path (singletons are immutable after startup)
```

**Phase to address:** Core Container Design (Phase 1), Performance (Phase 3)

---

### Trap 3: Eager Loading Everything

**What goes wrong:** All services instantiated at startup even if never used.

**Why it happens:**
- "Fail fast" philosophy taken too far
- Not distinguishing startup-critical from optional services
- Warming caches that may never be accessed

**Consequences:**
- Slow startup time
- Resources allocated but unused
- Memory waste for optional features

**Prevention:**
```go
// Default: Lazy (instantiate on first use)
gaz.Provide(NewExpensiveService)

// Explicit: Eager (instantiate at startup)
gaz.Eager(NewCriticalService)  // Fail fast for this one

// Or: Eager validation, lazy instantiation
gaz.ValidateOnly(NewExpensiveService)  // Check deps exist, don't create
```

**Phase to address:** Core Container Design (Phase 1)

---

## API Design Mistakes

Patterns that make the API confusing or hard to use.

---

### Mistake 1: Too Many Ways to Do the Same Thing

**What goes wrong:** Multiple APIs for registration, multiple APIs for resolution, unclear which to use.

**This is a known pain point from dibx/gazx:**
> "Too many options/knobs"
> "Unclear when to use which service type"

**Example of confusion (from existing Fx):**
```go
// All of these register a service - which should I use?
fx.Provide(New)
fx.Supply(value)
fx.Annotate(New, fx.As(new(Interface)))
fx.Decorate(wrapper)
fx.Replace(value)
fx.Invoke(New)  // Also registers if it returns something?
```

**Prevention:**
```go
// ONE way to provide
gaz.Provide(NewService)

// ONE way to provide a value directly
gaz.ProvideValue(existingInstance)

// ONE way to get
svc := gaz.MustInvoke[*Service](i)

// Annotations as options, not separate functions
gaz.Provide(NewService, 
    gaz.As[ServiceInterface](),  // Bind to interface
    gaz.Transient(),             // Scope option
)
```

**Phase to address:** API Design (Phase 2)

---

### Mistake 2: Interface Binding Verbosity

**What goes wrong:** Binding concrete type to interface requires too much ceremony.

**Wire's approach (explicit, verbose):**
```go
wire.Bind(new(Fooer), new(*MyFooer))  // What do these "new" calls mean?
```

**Prevention:**
```go
// Clear generic syntax
gaz.Provide(NewMyService, gaz.As[ServiceInterface]())

// Or infer from return type
func NewMyService() ServiceInterface {
    return &myService{}
}
gaz.Provide(NewMyService)  // Automatically binds to ServiceInterface
```

**Phase to address:** API Design (Phase 2)

---

### Mistake 3: Silent Failures

**What goes wrong:** Optional dependencies silently nil, services not registered without error.

**Prevention:**
```go
// Make optional explicit
type Params struct {
    Cache *redis.Client `optional:"true"`
}

// Provide clear error for missing required
// "gaz: cannot build *UserService: requires *sql.DB which is not provided"

// Warn on unused providers (they might be mistakes)
app.WarnUnusedProviders()
```

**Phase to address:** Error Handling (Phase 2)

---

## "Looks Done But Isn't" Checklist

Things that work in demos but fail in production.

| Feature | Demo Version | Production Requirement |
|---------|--------------|------------------------|
| Lifecycle | OnStart works | OnStop runs in reverse order, with timeout |
| Scopes | Child scope created | Child scope cleanup, parent scope isolation |
| Errors | Panic on missing | Structured errors with dependency chain |
| Config | Hardcoded values | Env override, secrets, validation |
| Testing | Works in isolation | Parallel tests don't interfere |
| Performance | Works with 10 services | Works with 500 services, 10K requests/sec |
| Debugging | Prints log | Dependency graph visualization, cycle detection |
| Shutdown | Exits cleanly | Graceful with timeout, drain connections |

---

## Pitfall-to-Phase Mapping

| Phase | Pitfalls to Actively Prevent |
|-------|------------------------------|
| **Phase 1: Core Container** | Magic/over-abstraction, Reflection performance, Global singleton, Stringly-typed, Allocation per resolve, Lock contention |
| **Phase 2: Lifecycle & API** | Scope explosion, Hook ordering, Cleanup missing, Signature explosion, Too many options, Silent failures |
| **Phase 3: Performance** | Validate reflection impact, Benchmark resolution, Load test concurrency |
| **Phase 4: Extensions** | Health checks, Config integration |
| **Phase 5: Polish** | Error messages, Debugging tools, Documentation |

---

## Sources

| Source | Confidence | What it provided |
|--------|------------|------------------|
| uber-go/fx documentation (Context7) | HIGH | Lifecycle hooks, modules, groups, annotations |
| google/wire FAQ | HIGH | Compile-time DI tradeoffs, multiple bindings |
| samber/do documentation (Context7) | HIGH | Generics-based DI, health checks, scopes |
| spf13/viper (Context7) | MEDIUM | Config anti-patterns, global state issues |
| knadh/koanf (Context7) | MEDIUM | Lighter config alternative |
| Internal dibx/gazx experience | HIGH | "Too many options/knobs" pain point |

---

## Summary

**Top 3 mistakes to avoid in gaz:**

1. **Too many options** - Convention over configuration. One obvious way to do things.
2. **Scope complexity** - Start with singleton (default) + transient + explicit child scopes. Nothing more.
3. **Magic lifecycle** - Make hook order visible and predictable. Stop in reverse of start.

**Design principles:**

- **Explicit over magic:** Users should understand what happens by reading their code
- **Compile-time over runtime:** Use generics, catch errors at compile time
- **Minimal surface:** Don't add API until there's proven need
- **Performance by default:** Benchmark resolution paths, not just startup
