# Feature Landscape: API Harmonization

**Domain:** Go DI Framework - API Consistency and Developer Experience
**Researched:** 2026-01-29
**Research Confidence:** HIGH (Context7 + Official Docs + Codebase Analysis + Industry Patterns)

---

## Executive Summary

This research documents the features that define a well-designed Go framework API, specifically for gaz v3.0's API Harmonization milestone. The goal is to create consistent, intuitive patterns across all gaz packages while maintaining the framework's "convention over configuration" philosophy.

**Key findings:**

1. **Table stakes** are primarily about consistency and predictability - users expect the same patterns everywhere
2. **Differentiators** come from reducing cognitive load - fewer concepts to learn, more "just works" behavior
3. **Anti-features** are magic behaviors that hide what's happening - Go developers prefer explicit over implicit

The research draws from:
- **uber-go/fx**: Lifecycle hooks, module patterns, error handling (Context7 - HIGH confidence)
- **google/wire**: Interface bindings, provider patterns (Context7 - HIGH confidence)
- **Industry consensus (2025)**: Generics for type safety, constructor injection, explicit dependencies
- **gaz codebase**: Existing patterns in `di/`, `config/`, `worker/`, `health/`, `gaztest/`

---

## Current gaz State (What's Already Built)

Understanding existing patterns is critical for harmonization:

| Pattern | Current Implementation | Location | Consistency Issue |
|---------|------------------------|----------|-------------------|
| Registration API | `For[T](c).Provider(fn)` fluent | `di/registration.go` | Solid foundation |
| Lifecycle interfaces | `Starter`, `Stopper` | `di/lifecycle.go`, `gaz/lifecycle.go` | Duplicated in two packages |
| Module builder | `NewModule(name).Provide().Build()` | `module_builder.go` | Different from di registration |
| Config unmarshaling | `LoadInto(target)` with Defaulter/Validator | `config/manager.go` | Good pattern |
| Error types | Per-package sentinel errors | `errors.go`, `di/`, `config/`, `worker/` | Inconsistent naming |
| Test utilities | `gaztest.New(t).Replace().Build()` | `gaztest/builder.go` | Good pattern |
| Service builder | `service.New().WithCmd().Build()` | `service/builder.go` | Similar to gaztest |

---

## Table Stakes: Must-Have Features for Professional Go Framework

Features users expect in any serious Go DI framework. Missing these feels incomplete or broken.

### 1. Type-Safe Registration API

**What:** Generic-based registration that catches type errors at compile time.

**Why Expected:** Go 1.18+ established generics as the standard. Reflection-based registration feels outdated.

| Criterion | Requirement | gaz Status |
|-----------|-------------|------------|
| Compile-time type checking | `For[*MyService](c).Provider(fn)` | Yes |
| Error on type mismatch | Provider return type must match T | Yes |
| IDE autocompletion | Generic type flows through chain | Yes |

**Acceptance Criteria:**
- [ ] All registration functions use generics (no `interface{}` in public API)
- [ ] Type mismatches are compile errors, not runtime panics
- [ ] Provider signature `func(*Container) (T, error)` is enforced by compiler

**gaz Alignment:** Already implemented via `For[T]()`. Table stakes met.

### 2. Context-Aware Lifecycle Hooks

**What:** All lifecycle operations accept `context.Context` for timeout/cancellation.

**Why Expected:** Standard Go pattern since Go 1.7. Essential for graceful shutdown.

| Interface | Expected Signature |
|-----------|-------------------|
| Starter | `OnStart(ctx context.Context) error` |
| Stopper | `OnStop(ctx context.Context) error` |
| Hook function | `func(ctx context.Context) error` |

**Acceptance Criteria:**
- [ ] All lifecycle interfaces use `context.Context` as first parameter
- [ ] Timeout enforcement via context deadline
- [ ] Per-hook timeouts configurable

**gaz Alignment:** Already implemented. Both `Starter` and `Stopper` accept context.

### 3. Consistent Error Types

**What:** Predictable error handling with sentinel errors and proper wrapping.

**Why Expected:** Go's `errors.Is()` and `errors.As()` are the standard. Magic string matching is fragile.

| Pattern | Expected | gaz Current Status |
|---------|----------|-------------------|
| Sentinel errors | `ErrNotFound`, `ErrCycle`, etc. | Yes, in di package |
| Wrapped errors | `fmt.Errorf("context: %w", err)` | Partial |
| Package prefix | `"di: ..."`, `"config: ..."` | Inconsistent |
| Error unwrapping | Implements `Unwrap() error` | Yes (ValidationError) |

**Acceptance Criteria:**
- [ ] All packages use `Err<PackageName><ErrorType>` naming pattern
- [ ] All errors from public API are wrapped with context
- [ ] `errors.Is()` works for all sentinel errors
- [ ] Error messages include package prefix

**gaz Alignment:** Partial. Pattern exists but inconsistent across packages.

### 4. Builder Pattern with Validation

**What:** Fluent builders that validate at Build() time, not use time.

**Why Expected:** Catches configuration errors early. Standard Go pattern for complex configuration.

| Builder | Expected Methods | gaz Status |
|---------|------------------|------------|
| Registration | `For[T]().Named().Transient().Eager().Provider()` | Yes |
| Module | `NewModule(name).Provide().Flags().Build()` | Yes |
| Test App | `gaztest.New(t).Replace().Build()` | Yes |
| Service | `service.New().WithCmd().WithConfig().Build()` | Yes |

**Acceptance Criteria:**
- [ ] All builders validate input before returning
- [ ] Build() returns error, not panic
- [ ] Errors from builder methods are accumulated and returned at Build()
- [ ] Invalid state is not silently ignored

**gaz Alignment:** Good pattern exists. Needs verification of consistent validation.

### 5. Config Unmarshaling with Defaults and Validation

**What:** Struct tag-based configuration with automatic defaults and validation.

**Why Expected:** Industry standard (viper + validator). Reduces boilerplate.

| Feature | Expected Behavior |
|---------|------------------|
| Struct tags | `mapstructure:"key"`, `validate:"required"` |
| Defaults | `Defaulter` interface or struct tags |
| Validation | `go-playground/validator` integration |
| Env vars | `MYAPP_KEY` maps to `key` with prefix |

**Acceptance Criteria:**
- [ ] `LoadInto(target)` applies defaults, then validates
- [ ] Validation errors are detailed (field name, tag, value)
- [ ] Defaulter interface called after unmarshal, before validate
- [ ] Validator interface called after struct tag validation

**gaz Alignment:** Already implemented in `config/manager.go`. Excellent pattern.

### 6. Testing Utilities

**What:** Dedicated package for testing DI applications.

**Why Expected:** Testing DI without utilities is painful. fxtest set the standard.

| Feature | Expected |
|---------|----------|
| Test app wrapper | `gaztest.New(t)` with automatic cleanup |
| Mock injection | `Replace(mockInstance)` for interface swapping |
| RequireStart/Stop | Fatal on error (t.Helper pattern) |
| Shorter timeouts | 5s default instead of 30s |

**Acceptance Criteria:**
- [ ] `gaztest.New(t)` registers t.Cleanup automatically
- [ ] `Replace(instance)` works with type inference
- [ ] `RequireStart()`, `RequireStop()` use `t.Helper()`
- [ ] Errors are descriptive for debugging

**gaz Alignment:** Already implemented in `gaztest/`. Good foundation.

---

## Differentiators: Features That Set gaz Apart

Features that go beyond table stakes to provide competitive advantage.

### 1. Interface Auto-Detection for Lifecycle

**What:** Automatic registration of OnStart/OnStop when service implements Starter/Stopper.

**Why Valuable:** Reduces boilerplate. Services "just work" without explicit hook registration.

**Current Pattern (Explicit):**
```go
gaz.For[*HTTPServer](c).
    Provider(NewHTTPServer).
    OnStart(func(ctx context.Context, s *HTTPServer) error { return s.Start(ctx) }).
    OnStop(func(ctx context.Context, s *HTTPServer) error { return s.Stop(ctx) })
```

**Proposed Pattern (Auto-Detection):**
```go
// HTTPServer implements gaz.Starter and gaz.Stopper
gaz.For[*HTTPServer](c).Provider(NewHTTPServer) // Auto-detects OnStart/OnStop
```

**Acceptance Criteria:**
- [ ] Detection happens at registration time (not runtime)
- [ ] Explicit hooks override auto-detected hooks
- [ ] Works with both Starter and Stopper interfaces
- [ ] No additional imports or configuration required

**Complexity:** Medium - requires changes to `di.ServiceWrapper`

**Differentiator Level:** HIGH - Most frameworks require explicit hook registration.

### 2. Unified Module Pattern

**What:** Consistent bundling of providers, flags, and configuration into reusable modules.

**Why Valuable:** Reduces duplication. Makes packages self-contained and composable.

**Current Pattern:**
```go
// Module groups providers but flags are registered separately
module := gaz.NewModule("http").
    Provide(func(c *gaz.Container) error {
        return gaz.For[*HTTPServer](c).Provider(NewHTTPServer)
    }).
    Build()

// Flags registered separately
fs.Int("http-port", 8080, "HTTP server port")
```

**Proposed Pattern:**
```go
// Module bundles everything
var HTTPModule = gaz.NewModule("http").
    Flags(func(fs *pflag.FlagSet) {
        fs.Int("http-port", 8080, "HTTP server port")
    }).
    Provide(func(c *gaz.Container) error {
        return gaz.For[*HTTPServer](c).Provider(NewHTTPServer)
    }).
    Build()

// Single Use() call registers all
app.Use(HTTPModule)
```

**Acceptance Criteria:**
- [ ] Module.Flags() registers flags when module is applied
- [ ] Flags work with both Cobra and standalone
- [ ] Module composition via Use() works as expected
- [ ] Duplicate module detection remains functional

**Complexity:** Low - extend existing ModuleBuilder

**Differentiator Level:** MEDIUM - fx has modules but without integrated flags

### 3. Consistent Fluent API Across All Packages

**What:** Same API patterns in every gaz package.

**Why Valuable:** Learn once, use everywhere. Reduces cognitive load.

| Package | Builder Pattern | Build Method |
|---------|-----------------|--------------|
| di | `For[T](c).Provider(fn)` | Returns error |
| module | `NewModule(name).Provide().Build()` | Returns Module |
| gaztest | `New(t).Replace().Build()` | Returns (*App, error) |
| service | `New().WithCmd().Build()` | Returns (*App, error) |
| worker | `worker.New().WithPolicy().Build()` | Should follow pattern |

**Acceptance Criteria:**
- [ ] All builders follow same chain pattern
- [ ] Error accumulation in builders (not fail-fast)
- [ ] Build() is always the terminal method
- [ ] Documentation uses consistent terminology

**Complexity:** Medium - requires audit and updates across packages

**Differentiator Level:** HIGH - Most frameworks have inconsistent APIs across packages

### 4. Rich Error Context

**What:** Errors that tell you exactly what went wrong and where.

**Why Valuable:** Faster debugging. Reduces "what does this error mean" time.

**Current Error:**
```
gaz: duplicate module name: http
```

**Proposed Error:**
```
gaz: duplicate module name
    module: "http"
    first registered: service.go:42
    second attempt: main.go:18
    hint: use Replace() if intentional
```

**Acceptance Criteria:**
- [ ] Errors include relevant context (names, types, locations)
- [ ] Error types support structured access via errors.As()
- [ ] Hints for common mistakes
- [ ] Stack information when debugging enabled

**Complexity:** Medium - requires updating error creation throughout

**Differentiator Level:** MEDIUM - Good error messages are rare in DI frameworks

### 5. Opt-in Debug Mode

**What:** Verbose logging of DI operations for troubleshooting.

**Why Valuable:** Understanding DI ordering is hard. Visibility helps.

**Proposed Pattern:**
```go
app := gaz.New(gaz.WithDebug())
// Logs:
// gaz/di: registering *HTTPServer (singleton, lazy)
// gaz/di: registering *Database (singleton, eager)
// gaz/di: building container (2 services)
// gaz/di: resolving *Database (eager)
// gaz/di: resolved *Database in 1.2ms
// gaz/lifecycle: starting services (order: Database, HTTPServer)
```

**Acceptance Criteria:**
- [ ] Debug mode is opt-in (off by default)
- [ ] Uses standard slog for logging
- [ ] Shows resolution order
- [ ] Shows timing information
- [ ] Works with existing logger configuration

**Complexity:** Low-Medium - add logging calls at key points

**Differentiator Level:** MEDIUM - fx has debug logging, wire doesn't

---

## Anti-Features: Deliberate Non-Goals

Features to explicitly NOT build. These seem helpful but create problems.

### 1. Global Container / Singleton Pattern

**Why Requested:** "Just let me call gaz.Get[*Service]() anywhere"

**Why Problematic:**
- Breaks testability (can't parallel test with different mocks)
- Hidden dependencies (imports don't show dependencies)
- Race conditions in concurrent registration

**What to Do Instead:**
- Always inject Container explicitly
- Use constructor injection: `func NewService(dep *Dependency) *Service`

### 2. Spring-Style Field Injection Tags

**Why Requested:** "Just inject into struct fields automatically"

```go
// Anti-pattern
type MyService struct {
    Logger   *slog.Logger `inject:""`
    Database *sql.DB      `inject:""`
}
```

**Why Problematic:**
- Hides dependencies (not visible in constructor)
- Breaks refactoring (changing fields breaks magic)
- Requires reflection at resolution time
- Makes testing harder (can't use struct literal)

**What to Do Instead:**
- Constructor injection: `func NewService(logger *slog.Logger, db *sql.DB) *Service`
- Dependencies visible in function signature

### 3. Automatic Retry for Failed Providers

**Why Requested:** "If a service fails to start, retry automatically"

**Why Problematic:**
- Hides real errors (transient or permanent?)
- Delays startup without user awareness
- Hard to reason about state

**What to Do Instead:**
- Fail fast with clear error
- Let user implement retry in provider if needed
- Workers can have retry policies; providers should not

### 4. Dynamic Registration After Build

**Why Requested:** "Let me add services at runtime"

**Why Problematic:**
- Breaks static analysis
- Creates race conditions
- Dependency graph becomes unpredictable

**What to Do Instead:**
- Register everything before Build()
- Use factory patterns for dynamic instances
- Transient scope for per-request objects

### 5. Implicit Type Coercion

**Why Requested:** "Automatically convert string env vars to int/duration"

**Why Problematic:**
- Silent failures ("5" works, "five" crashes at runtime)
- Unexpected behavior (empty string becomes 0?)
- Type safety is Go's strength

**What to Do Instead:**
- Use mapstructure decode hooks (explicit)
- Fail fast on type mismatch
- Provide clear error messages

### 6. Automatic Config File Discovery

**Why Requested:** "Just find config.yaml anywhere on the system"

**Why Problematic:**
- Security issue (wrong config in production)
- Hard to debug (which file was loaded?)
- Different behavior per environment

**What to Do Instead:**
- Explicit config paths in priority order
- Clear precedence rules (flags > env > file)
- Log which config was loaded

---

## Feature Matrix by Category

| Feature | Category | Complexity | Priority | Depends On |
|---------|----------|------------|----------|------------|
| **Table Stakes** |
| Type-safe registration | Table Stakes | Done | P0 | - |
| Context-aware lifecycle | Table Stakes | Done | P0 | - |
| Builder pattern with validation | Table Stakes | Low | P1 | - |
| Consistent error types | Table Stakes | Medium | P1 | - |
| Config with defaults/validation | Table Stakes | Done | P0 | - |
| Testing utilities | Table Stakes | Done | P0 | - |
| **Differentiators** |
| Interface auto-detection | Differentiator | Medium | P1 | di changes |
| Unified module pattern | Differentiator | Low | P1 | - |
| Consistent fluent API | Differentiator | Medium | P1 | Audit |
| Rich error context | Differentiator | Medium | P2 | - |
| Opt-in debug mode | Differentiator | Low | P2 | slog integration |

---

## Alignment with gaz v3.0 Goals

| Goal | Relevant Features |
|------|------------------|
| **Developer Experience** | Interface auto-detection, consistent API, rich errors |
| **Consistency** | Error type naming, fluent API patterns, module pattern |
| **Documentation** | Builder validation (errors guide users), debug mode |
| **Extensibility** | Module pattern, unified provider interface |

---

## Config Unmarshaling: Best Practices

Based on research, gaz's config pattern should follow:

### Expected Flow

```
1. Load from sources (files, env, flags)
2. Unmarshal into struct (mapstructure)
3. Apply Defaulter interface
4. Apply struct tag validation (validator)
5. Apply Validator interface (custom logic)
6. Return config or detailed error
```

### Struct Tags

```go
type Config struct {
    // mapstructure for unmarshal key mapping
    Host string `mapstructure:"host" validate:"required"`
    
    // validate for struct-level validation
    Port int `mapstructure:"port" validate:"required,min=1,max=65535"`
    
    // Complex validation
    TLS struct {
        Enabled  bool   `mapstructure:"enabled"`
        CertFile string `mapstructure:"cert_file" validate:"required_if=Enabled true"`
    } `mapstructure:"tls"`
}
```

### Acceptance Criteria for Config

- [ ] All config structs use `mapstructure` tags
- [ ] Required fields use `validate:"required"`
- [ ] Cross-field validation uses `required_if`, `required_with`
- [ ] Custom validators registered for domain types
- [ ] Validation errors include field path

---

## Lifecycle Interface Design: Best Practices

Based on research, gaz's lifecycle interfaces should follow:

### Interface Definition

```go
// Starter is called during app.Start() in dependency order
type Starter interface {
    OnStart(ctx context.Context) error
}

// Stopper is called during app.Stop() in reverse dependency order
type Stopper interface {
    OnStop(ctx context.Context) error
}
```

### Execution Order

```
Start Order (dependency-first):
1. Database (no deps)
2. Cache (depends on nothing)
3. UserService (depends on Database)
4. HTTPServer (depends on UserService)

Stop Order (reverse):
1. HTTPServer
2. UserService
3. Cache
4. Database
```

### Timeout Enforcement

```go
// Per-service timeout with fallback to global
type HookConfig struct {
    Timeout time.Duration // Zero = use app default
}

// Hook execution with timeout
func runHook(ctx context.Context, fn HookFunc, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    return fn(ctx)
}
```

### Acceptance Criteria for Lifecycle

- [ ] Interfaces accept context as first parameter
- [ ] Start order follows dependency graph
- [ ] Stop order is reverse of start order
- [ ] Per-hook timeout is configurable
- [ ] Force exit if timeout exceeded

---

## Error Handling Strategy: Best Practices

Based on research, gaz's error handling should follow:

### Error Type Pattern

```go
// Package-level sentinel errors
var (
    ErrDINotFound     = errors.New("di: service not found")
    ErrDICycle        = errors.New("di: circular dependency")
    ErrDIDuplicate    = errors.New("di: duplicate registration")
    ErrDITypeMismatch = errors.New("di: type mismatch")
)

// Structured error with context
type ResolutionError struct {
    ServiceName string
    ServiceType string
    Cause       error
}

func (e *ResolutionError) Error() string {
    return fmt.Sprintf("di: failed to resolve %s (%s): %v", 
        e.ServiceName, e.ServiceType, e.Cause)
}

func (e *ResolutionError) Unwrap() error {
    return e.Cause
}
```

### Error Wrapping

```go
// Always wrap with context
func (c *Container) Resolve(name string) (any, error) {
    svc, err := c.get(name)
    if err != nil {
        return nil, &ResolutionError{
            ServiceName: name,
            ServiceType: c.getType(name),
            Cause:       err,
        }
    }
    return svc, nil
}
```

### Acceptance Criteria for Errors

- [ ] All sentinel errors follow `Err<Package><Type>` naming
- [ ] All public functions return wrapped errors
- [ ] `errors.Is()` works for sentinel matching
- [ ] `errors.As()` works for structured access
- [ ] Error messages are actionable

---

## Module Pattern: Best Practices

Based on research, gaz's module pattern should follow:

### Module Interface

```go
type Module interface {
    Name() string
    Apply(app *App) error
}

// Optional interface for flags
type FlagProvider interface {
    Flags() func(*pflag.FlagSet)
}

// Optional interface for env prefix
type EnvPrefixProvider interface {
    EnvPrefix() string
}
```

### Module Builder

```go
var HTTPModule = gaz.NewModule("http").
    // Bundle flags with module
    Flags(func(fs *pflag.FlagSet) {
        fs.Int("http-port", 8080, "HTTP server port")
        fs.Duration("http-timeout", 30*time.Second, "Request timeout")
    }).
    // Bundle config prefix
    WithEnvPrefix("http").
    // Bundle providers
    Provide(
        func(c *gaz.Container) error { return gaz.For[*Router](c).Provider(NewRouter) },
        func(c *gaz.Container) error { return gaz.For[*Server](c).Provider(NewServer) },
    ).
    // Bundle child modules
    Use(LoggingModule).
    // Build immutable module
    Build()
```

### Acceptance Criteria for Modules

- [ ] Modules are immutable after Build()
- [ ] Child modules applied before parent providers
- [ ] Flags registered when module applied (if cobra attached)
- [ ] Duplicate module detection by name
- [ ] Module name in error messages

---

## Sources

| Source | Type | Confidence |
|--------|------|------------|
| uber-go/fx documentation | Context7 | HIGH |
| google/wire documentation | Context7 | HIGH |
| Go 1.25 stdlib | Official | HIGH |
| WebSearch (Go DI patterns 2025) | Community | MEDIUM |
| WebSearch (Go config patterns) | Community | MEDIUM |
| gaz codebase analysis | Internal | HIGH |
| spf13/viper patterns | Context7/Official | HIGH |

---

*Feature research for: gaz v3.0 API Harmonization*
*Researched: 2026-01-29*
