# Feature Landscape: Go Application Frameworks

**Domain:** Go Application Framework (DI, Lifecycle, Config, Logging)
**Researched:** 2026-01-26
**Confidence:** HIGH (Context7 + internal code analysis)

## Table Stakes

Features users expect from any Go application framework. Missing these = product feels incomplete or amateurish.

| Feature | Why Expected | Complexity | gaz v1 Target | Notes |
|---------|--------------|------------|---------------|-------|
| **Dependency Injection** | Core value proposition; eliminates globals, enables testing | High | Yes | fx/wire/samber-do all provide this |
| **Constructor Registration** | How DI containers know what to build | Medium | Yes | `fx.Provide`, `wire.Build`, `dibx.Register` |
| **Lazy Instantiation** | Performance - don't build what isn't used | Medium | Yes | fx: lazy by default; Wire: compile-time |
| **Singleton Scoping** | Most services should be singletons | Low | Yes | Standard pattern in all frameworks |
| **Error Handling** | Constructors can fail; framework must propagate errors | Medium | Yes | Return `(T, error)` pattern universal |
| **Lifecycle Hooks (OnStart/OnStop)** | Apps need startup/shutdown orchestration | Medium | Yes | fx.Lifecycle, gazx.LifecycleManager |
| **Graceful Shutdown** | Signal handling (SIGTERM/SIGINT) | Medium | Yes | All production frameworks include this |
| **Context Propagation** | Cancellation and deadlines for cleanup | Medium | Yes | Go idiom; required for production |
| **Shutdown Timeouts** | Prevent hanging on shutdown | Low | Yes | `fx.StopTimeout`, `gazx.ShutdownTimeout` |
| **Logging Integration** | Observability is mandatory | Medium | Yes | fx: fxevent; gaz: slog integration |
| **Module System** | Code organization for large apps | Medium | Yes | `fx.Module`, `gazx.ModuleProvider` |
| **Testing Support** | DI must enable testing | Medium | Yes | fx: fxtest; gazx: NewTestBuilder |

## Differentiators

Features that set gaz apart from alternatives. Not expected, but create competitive advantage.

| Feature | Value Proposition | Complexity | gaz v1 Target | Framework Precedent |
|---------|-------------------|------------|---------------|---------------------|
| **Type-Safe Generics DI** | Compile-time safety, no `interface{}` casts, better IDE support | High | Yes | dibx does this; fx uses reflection |
| **Convention-over-Configuration** | Zero-config works; override when needed | Medium | Yes | Spring Boot influence; fx is explicit |
| **Cobra CLI Integration** | Unified app builder with CLI | Low | Yes | gazx has this; fx doesn't |
| **Health Check Subsystem** | Kubernetes-native readiness/liveness | Medium | Yes | gazx.HealthManager provides this |
| **Deterministic Startup Order** | Predictable initialization | High | Yes | fx has some; we can improve |
| **Deterministic Shutdown Order** | Reverse of startup; LIFO | High | Yes | gazx has this pattern |
| **Worker Group Management** | Background tasks with lifecycle | Medium | Yes | gazx.WorkerGroup provides this |
| **Typed Event Bus** | Inter-component communication | Medium | Yes | gazx.EventBus with generics |
| **Hierarchical Scopes** | Request-scoped or session-scoped DI | High | Defer | dibx supports; complex to use well |
| **Transient Registration** | New instance per request | Low | Yes | dibx.AsTransient() |
| **Eager Registration** | Instantiate immediately at registration | Low | Yes | dibx.AsEager() |
| **Named Services** | Multiple implementations of same type | Medium | Yes | fx: tags; dibx: WithName() |
| **Service Override** | Replace services (testing, customization) | Medium | Yes | dibx.Override() |
| **Struct Tag Injection** | Inject into struct fields by tag | Medium | Yes | dibx `dibx:"name"` tag |
| **Multi-Source Config** | Env, files, flags, remote sources | High | Yes | Viper-style but unified |
| **slog Context Propagation** | Structured logging with request context | Medium | Yes | Go 1.21+ slog native |
| **GOMAXPROCS Auto-Config** | Container-aware CPU limits | Low | Yes | gazx uses automaxprocs |
| **Memory Limit Auto-Config** | Container-aware memory limits | Low | Yes | gazx uses memlimit |

### Differentiator Deep-Dive

#### Type-Safe Generics DI (Primary Differentiator)

**What makes it special:**
- fx uses reflection and `interface{}` everywhere
- Wire generates code but still lacks generic elegance
- dibx/gaz uses Go 1.18+ generics for compile-time type safety

```go
// fx approach (runtime type checking)
fx.Provide(func() *MyService { ... })
fx.Invoke(func(s *MyService) { ... }) // type mismatch = runtime panic

// gaz approach (compile-time type checking)
gaz.Provide[*MyService](func(i gaz.Injector) (*MyService, error) { ... })
var svc *MyService
gaz.MustResolve(&svc) // type mismatch = compile error
```

**Confidence:** HIGH (verified in dibx source)

#### Convention-over-Configuration

**What makes it special:**
- fx requires explicit wiring for everything
- gaz can auto-wire common patterns
- Sensible defaults that work out of the box

**Examples:**
- Auto-register slog logger if none provided
- Auto-configure GOMAXPROCS for containers
- Default shutdown timeout (15s)
- Default health check endpoints

**Confidence:** MEDIUM (planned feature, not yet implemented)

## Anti-Features

Features to explicitly NOT build. Common mistakes in this domain.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Global Container** | Defeats purpose of DI; hides dependencies | Pass injector explicitly; no package-level container |
| **Service Locator Pattern** | Anti-pattern; hides dependencies | Constructor injection only |
| **init() Auto-Registration** | Non-deterministic, hard to test | Explicit registration in app builder |
| **Circular Dependencies** | Design smell; hard to reason about | Detect at registration; fail fast |
| **Over-Configuration** | Config fatigue; analysis paralysis | Sensible defaults; minimal required config |
| **Magic String Keys** | Typos cause runtime errors | Type-based registration as primary; named as escape hatch |
| **Implicit Startup Order** | Non-deterministic bugs | Explicit lifecycle phases |
| **RPC/Message Broker Built-in** | Scope creep; better as modules | Provide extension points; let users add go-micro |
| **Service Discovery Built-in** | Infrastructure concern | Integrate with existing (Consul, K8s) |
| **HTTP Server Built-in** | Many options; not framework concern | Example modules; users choose (std, chi, echo) |
| **Database Abstraction** | Too many options; not framework concern | Example patterns; users choose driver |
| **Compile-Time Code Generation** | Friction; requires extra tooling | Wire's approach; runtime DI is more ergonomic |

### Anti-Feature Rationale

#### Why NOT include RPC/Service Discovery

**go-micro** includes these, but:
- Creates vendor lock-in
- Forces architecture decisions
- Most teams have existing infrastructure
- Better as opt-in modules

**gaz approach:** Provide extension points (health checks, events) that integrate with any RPC framework.

#### Why NOT use Code Generation (Wire approach)

**Wire advantages:**
- Zero runtime overhead
- Compile-time validation

**Wire disadvantages:**
- Requires `go generate` step
- IDE support limited
- Can't do runtime reconfiguration
- More ceremony

**gaz approach:** Accept minimal runtime overhead for better DX. Use generics to catch type errors at compile time where possible.

## Feature Dependencies

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                    gaz Framework                        │
                    └─────────────────────────────────────────────────────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    ▼                         ▼                         ▼
             ┌─────────────┐          ┌─────────────┐          ┌─────────────┐
             │ DI Container │          │ App Builder │          │   Config    │
             │   (dibx)     │          │   (gazx)    │          │   System    │
             └─────────────┘          └─────────────┘          └─────────────┘
                    │                         │                         │
        ┌───────────┼───────────┐    ┌───────┼───────┐         ┌───────┼───────┐
        ▼           ▼           ▼    ▼       ▼       ▼         ▼       ▼       ▼
   ┌────────┐ ┌────────┐ ┌────────┐ ┌────┐ ┌────┐ ┌────┐  ┌─────┐ ┌─────┐ ┌─────┐
   │Register│ │Resolve │ │Scopes  │ │Life│ │Work│ │Hlth│  │ Env │ │File │ │Flags│
   │        │ │        │ │        │ │Cyc │ │Grp │ │Chk │  │     │ │     │ │     │
   └────────┘ └────────┘ └────────┘ └────┘ └────┘ └────┘  └─────┘ └─────┘ └─────┘
```

**Core Dependencies:**
1. **DI Container** → Required for everything; foundation layer
2. **Lifecycle** → Requires DI (hooks register via DI)
3. **App Builder** → Requires DI + Lifecycle
4. **Health Checks** → Requires Lifecycle (starts/stops with app)
5. **Worker Groups** → Requires Lifecycle
6. **Event Bus** → Requires Lifecycle
7. **Config** → Standalone but registers values into DI
8. **Logging (slog)** → Standalone; integrates with DI

**Build Order:**
1. DI Container (core/dibx)
2. Lifecycle Management 
3. Config System
4. App Builder (core/gazx)
5. Health Checks
6. Worker Groups
7. Event Bus
8. slog Integration
9. Cobra Integration

## MVP Definition

### Phase 1: Foundation (Must Have)

Core functionality that makes the framework usable at all:

1. **Type-Safe DI Container**
   - Generic registration: `Register[T](provider)`
   - Generic resolution: `Resolve[T]() -> T`
   - Singleton scoping (lazy by default)
   - Transient scoping option
   - Named services
   - Error propagation

2. **Basic Lifecycle**
   - OnStart/OnStop hooks
   - Graceful shutdown (signal handling)
   - Shutdown timeout

3. **Minimal App Builder**
   - `gaz.New()` entry point
   - Provider registration
   - Run method

### Phase 2: Production-Ready

Features needed for real production use:

1. **Advanced DI**
   - Struct tag injection
   - Service override
   - Circular dependency detection

2. **Advanced Lifecycle**
   - Deterministic startup order
   - Deterministic shutdown order (LIFO)
   - Start/stop timeouts per hook

3. **Health Checks**
   - Readiness checks
   - Liveness checks
   - Health manager

4. **Config System**
   - Environment variables
   - Flag integration
   - Type-safe config structs

5. **Cobra Integration**
   - CLI app builder
   - Subcommand support
   - Flag binding

6. **slog Integration**
   - Default logger setup
   - Context propagation
   - Framework event logging

### Defer to Post-MVP

- Hierarchical scopes (complex, niche use case)
- Event bus (nice-to-have, not essential)
- Worker groups (can use goroutines directly)
- Decorator pattern (advanced use case)
- Value groups (fx feature, advanced)
- Validation framework (separate concern)

## Competitor Analysis

### Uber fx

**What it does well:**
- Battle-tested at Uber scale
- Comprehensive feature set
- Excellent documentation
- Good module system

**Where gaz can improve:**
- Type safety (fx uses reflection)
- Convention-over-configuration (fx is explicit)
- Simpler API (fx has many concepts)
- CLI integration (fx doesn't include)
- Health checks (fx doesn't include)

**Source:** Context7 /uber-go/fx (HIGH confidence)

### Google Wire

**What it does well:**
- Zero runtime overhead
- Compile-time validation
- No runtime reflection

**Where gaz can improve:**
- No code generation required
- Better DX (no go generate step)
- Runtime flexibility
- Simpler tooling

**Source:** Context7 /google/wire (HIGH confidence)

### go-micro

**What it does well:**
- Full microservices framework
- RPC, messaging, service discovery
- Pluggable architecture

**Where gaz differs:**
- gaz is focused on DI + lifecycle (not full microservices)
- gaz doesn't include transport layer
- gaz is smaller scope, more composable
- Teams can add go-micro on top of gaz if needed

**Source:** Context7 /micro/go-micro (HIGH confidence)

### samber/do

**What it does well:**
- Go 1.18+ generics
- Simple API
- Lightweight

**Where gaz can improve:**
- Lifecycle management (do doesn't include)
- App framework features (do is just DI)
- Config integration
- CLI integration
- Health checks

**Source:** Training data (MEDIUM confidence - verify with official docs)

## Internal Patterns (from tmp/dibx and tmp/gazx)

### Patterns to Adopt

| Pattern | Source | Description |
|---------|--------|-------------|
| Generic Provider | dibx | `Provider[T] func(Injector) (T, error)` |
| Service Types | dibx | Lazy, Eager, Transient service wrappers |
| Scope Hierarchy | dibx | Parent-child scope relationships |
| Lifecycle Interface | gazx | `Start(ctx) error`, `Stop(ctx) error` |
| Module Provider | gazx | Fluent builder for modules |
| Health Check Types | gazx | ReadinessHealthType, LivenessHealthType |
| Event Bus | gazx | Typed events with generics |
| Worker Group | gazx | Managed background tasks |

### Patterns to Improve

| Current Pattern | Issue | Improvement |
|-----------------|-------|-------------|
| Package-level `Provide[T]` | Deprecated in dibx | Method-only: `injector.Register()` |
| Separate hooks | Multiple hook types | Unified hook system |
| Reflection for type names | Runtime overhead | Cache type names at registration |

## Sources

### High Confidence (Context7/Official)

- uber-go/fx documentation via Context7
- google/wire documentation via Context7
- micro/go-micro documentation via Context7
- Internal dibx source code (tmp/dibx/)
- Internal gazx source code (tmp/gazx/)

### Medium Confidence (Verified)

- GitHub uber-go/fx README (WebFetch verified)

### Lower Confidence (Training Data)

- samber/do patterns (verify with official docs before implementation)
- go-kit architecture (verify if comparing)
