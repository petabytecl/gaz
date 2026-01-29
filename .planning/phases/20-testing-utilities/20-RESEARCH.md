# Phase 20: Testing Utilities (gaztest) - Research

**Researched:** 2026-01-29
**Domain:** Go DI testing utilities, test lifecycle management
**Confidence:** HIGH

## Summary

This phase implements a `gaztest` package that provides test-friendly wrappers around gaz.App. The package enables:
1. Simple test app creation with `gaztest.New(t)`
2. Automatic cleanup via `t.Cleanup()` registration
3. Assertion methods (`RequireStart`, `RequireStop`) that fail tests on error
4. Mock injection via `Replace(instance)` with type inference
5. Shorter default timeouts (5s) suitable for test scenarios

The standard pattern established by Uber's `fxtest` package provides the definitive reference. Our implementation follows this pattern while adapting to gaz's builder API.

**Primary recommendation:** Implement a `Builder` struct with fluent API that creates a test-specific `App` wrapper, leveraging reflection for type inference in `Replace()` and automatic `t.Cleanup()` registration.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `testing` | stdlib | Test framework | Go's built-in testing, provides TB interface |
| `reflect` | stdlib | Type inference | Required for inferring type from mock instance |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify/require` | v1.9+ | Fatal assertions | Already in use across gaz tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom TB interface | testing.TB directly | testing.TB is more complete but we only need subset |
| Generics for Replace | Reflection | Generics cleaner but don't support type inference from argument |

## Architecture Patterns

### Recommended Project Structure
```
gaztest/
├── builder.go       # Builder type and fluent API
├── app.go           # TestApp wrapper type
├── doc.go           # Package documentation
└── gaztest_test.go  # Tests for the package
```

### Pattern 1: Builder Pattern with Fluent API

**What:** Method chaining pattern that accumulates configuration before creating the test app

**When to use:** Always - this is the primary API

**Example:**
```go
// Source: fxtest pattern adapted for gaz
func TestMyService(t *testing.T) {
    mockDB := &MockDatabase{}
    
    app := gaztest.New(t).
        WithTimeout(5*time.Second).
        Replace(mockDB).
        Build()
    
    // app is automatically cleaned up via t.Cleanup()
    // Start app for the test
    app.RequireStart()
    defer app.RequireStop()
    
    // ... test logic
}
```

### Pattern 2: Automatic Cleanup via t.Cleanup()

**What:** Register cleanup functions at Build() time that run after test completes

**When to use:** Always - this ensures resources are released even if test panics

**Example:**
```go
// Source: Go testing stdlib
func (b *Builder) Build() (*App, error) {
    app, err := b.buildApp()
    if err != nil {
        return nil, err
    }
    
    // Register cleanup - runs after test completes
    b.tb.Cleanup(func() {
        ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
        defer cancel()
        
        if err := app.Stop(ctx); err != nil {
            // Log but don't fail - test may have already called RequireStop()
            b.tb.Logf("cleanup: stop failed: %v", err)
        }
    })
    
    return app, nil
}
```

### Pattern 3: Type Inference via Reflection

**What:** Infer the type to replace from the mock instance itself

**When to use:** For Replace() method - allows `Replace(mockDB)` instead of `Replace[*Database](mockDB)`

**Example:**
```go
// Source: Common Go pattern with reflect package
func (b *Builder) Replace(instance any) *Builder {
    if instance == nil {
        b.errs = append(b.errs, errors.New("Replace: instance cannot be nil"))
        return b
    }
    
    instanceType := reflect.TypeOf(instance)
    typeName := di.TypeNameReflect(instanceType)
    
    b.replacements = append(b.replacements, replacement{
        typeName: typeName,
        instance: instance,
    })
    return b
}
```

### Pattern 4: TB Interface Subset

**What:** Define minimal interface matching testing.T and testing.B

**When to use:** For package API - allows both *testing.T and *testing.B

**Example:**
```go
// Source: fxtest/tb.go
// TB is a subset of testing.TB required by gaztest
type TB interface {
    Logf(string, ...any)
    Errorf(string, ...any)
    Fatalf(string, ...any)
    FailNow()
    Cleanup(func())
    Helper()
}
```

### Anti-Patterns to Avoid
- **Global test state:** Never use package-level variables for test state - breaks parallel tests
- **Forgetting cleanup:** Always use t.Cleanup() instead of relying on user to call Stop()
- **Assert vs Require:** In gaztest, only provide Require* methods - Assert* variants allow test to continue with invalid state
- **Ignoring context timeout:** Always respect context timeout in Start/Stop operations

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Type name from reflect.Type | Custom string formatting | `di.TypeNameReflect()` | Already handles pointers, packages, generics |
| Test fatal assertions | Custom panic wrapper | `t.Fatalf()` or `t.FailNow()` | Proper test failure semantics |
| Cleanup ordering | Manual defer chains | `t.Cleanup()` | LIFO ordering, survives panics |
| Service registration | New container API | `di.For[T]().Replace().Instance()` | Already supports replace semantics |

**Key insight:** The existing `di.For[T]().Replace().Instance()` pattern already supports replacing registrations. The gaztest package just needs to orchestrate this with test-friendly ergonomics.

## Common Pitfalls

### Pitfall 1: Forgetting that t.Cleanup runs after test function returns

**What goes wrong:** Code expects cleanup to run immediately on test failure
**Why it happens:** Confusion between `defer` and `t.Cleanup()` semantics
**How to avoid:** Use RequireStop() explicitly when you need synchronous stop
**Warning signs:** Tests timing out waiting for cleanup, resources not released

### Pitfall 2: Not handling already-stopped apps in cleanup

**What goes wrong:** Cleanup tries to stop an app that user already stopped, causing error or panic
**Why it happens:** User calls RequireStop() explicitly, then t.Cleanup also runs
**How to avoid:** Track app state, make Stop() idempotent, only log (don't fail) in cleanup
**Warning signs:** Test logs show double-stop errors

### Pitfall 3: Type inference failing for interface mocks

**What goes wrong:** Replace(mockService) where mockService is typed as interface gets wrong type
**Why it happens:** Go's reflection on interface values returns concrete type, not interface type
**How to avoid:** Document limitation, suggest using concrete pointer types for mocks
**Warning signs:** "type not found in container" errors when mock is provided

### Pitfall 4: Timeout too short for complex service graphs

**What goes wrong:** Test fails with timeout during startup
**Why it happens:** Default 5s timeout insufficient for services with slow init
**How to avoid:** Allow WithTimeout() override, document common timeout patterns
**Warning signs:** Intermittent timeout failures in CI but not locally

## Code Examples

Verified patterns from official sources:

### fxtest.New Pattern (from uber-go/fx)
```go
// Source: https://github.com/uber-go/fx/blob/master/fxtest/app.go
// New creates a new test application.
func New(tb TB, opts ...fx.Option) *App {
    allOpts := make([]fx.Option, 0, len(opts)+1)
    allOpts = append(allOpts, WithTestLogger(tb))
    allOpts = append(allOpts, opts...)

    app := fx.New(allOpts...)
    if err := app.Err(); err != nil {
        tb.Errorf("fx.New failed: %v", err)
        tb.FailNow()
    }

    return &App{
        App: app,
        tb:  tb,
    }
}

// RequireStart calls Start, failing the test if an error is encountered.
func (app *App) RequireStart() *App {
    startCtx, cancel := context.WithTimeout(context.Background(), app.StartTimeout())
    defer cancel()

    if err := app.Start(startCtx); err != nil {
        app.tb.Errorf("application didn't start cleanly: %v", err)
        app.tb.FailNow()
    }
    return app
}
```

### t.Cleanup Usage Pattern
```go
// Source: Go testing stdlib documentation
func setupTestEnv(t *testing.T) *TestEnv {
    db := connectDatabase()
    
    t.Cleanup(func() {
        db.Close()
    })
    
    return &TestEnv{DB: db}
}
```

### Type Inference via Reflection
```go
// Source: Adapted from Go reflection patterns
func typeNameFromInstance(instance any) string {
    t := reflect.TypeOf(instance)
    if t == nil {
        return "nil"
    }
    
    // Handle pointer types
    if t.Kind() == reflect.Pointer {
        return "*" + t.Elem().PkgPath() + "." + t.Elem().Name()
    }
    
    if t.PkgPath() != "" {
        return t.PkgPath() + "." + t.Name()
    }
    return t.Name()
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `defer cleanup()` | `t.Cleanup(cleanup)` | Go 1.14 (2020) | Better test semantics, survives panics |
| Explicit mock types | Type inference via reflection | Common practice | Simpler API |
| `assert` variants | `require` only in test utilities | Community consensus | Prevents invalid state |

**Deprecated/outdated:**
- Manual cleanup tracking: Use t.Cleanup() instead
- Assert variants: Require-only is best practice for test utilities

## Implementation Approach

Based on research, here's the recommended implementation strategy:

### 1. Types Structure

```go
// gaztest/builder.go

// TB is the testing interface required by gaztest
type TB interface {
    Logf(string, ...any)
    Errorf(string, ...any)
    Fatalf(string, ...any)
    FailNow()
    Cleanup(func())
    Helper()
}

// Builder configures a test application
type Builder struct {
    tb           TB
    timeout      time.Duration
    replacements []replacement
    options      []gaz.Option
    errs         []error
}

type replacement struct {
    typeName string
    instance any
}

// App wraps gaz.App with test-friendly methods
type App struct {
    *gaz.App
    tb      TB
    timeout time.Duration
    stopped bool
}
```

### 2. Builder Flow

```go
// New creates a Builder for configuring test apps
func New(tb TB) *Builder {
    return &Builder{
        tb:      tb,
        timeout: 5 * time.Second, // Default test timeout
    }
}

// WithTimeout sets custom timeout for start/stop operations
func (b *Builder) WithTimeout(d time.Duration) *Builder {
    b.timeout = d
    return b
}

// Replace registers a mock instance to replace a type in the container
func (b *Builder) Replace(instance any) *Builder {
    if instance == nil {
        b.errs = append(b.errs, errors.New("Replace: instance cannot be nil"))
        return b
    }
    
    typeName := di.TypeNameReflect(reflect.TypeOf(instance))
    b.replacements = append(b.replacements, replacement{
        typeName: typeName,
        instance: instance,
    })
    return b
}

// Build creates the test app with all configured replacements
func (b *Builder) Build() (*App, error) {
    if len(b.errs) > 0 {
        return nil, errors.Join(b.errs...)
    }
    
    // Create base gaz app with test timeouts
    gazApp := gaz.New(
        gaz.WithShutdownTimeout(b.timeout),
        gaz.WithPerHookTimeout(b.timeout),
    )
    
    // Apply replacements to container
    for _, r := range b.replacements {
        if !gazApp.Container().HasService(r.typeName) {
            return nil, fmt.Errorf("Replace: type %s not registered in container", r.typeName)
        }
        // Use di.For reflection-based instance registration with Replace
        svc := di.NewInstanceServiceAny(r.typeName, r.typeName, r.instance, nil, nil)
        gazApp.Container().Register(r.typeName, svc)
    }
    
    // Build and validate
    if err := gazApp.Build(); err != nil {
        return nil, err
    }
    
    app := &App{
        App:     gazApp,
        tb:      b.tb,
        timeout: b.timeout,
    }
    
    // Register automatic cleanup
    b.tb.Cleanup(func() {
        if !app.stopped {
            ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
            defer cancel()
            if err := app.Stop(ctx); err != nil {
                b.tb.Logf("gaztest cleanup: stop failed: %v", err)
            }
        }
    })
    
    return app, nil
}
```

### 3. TestApp Methods

```go
// RequireStart starts the app or fails the test
func (a *App) RequireStart() *App {
    a.tb.Helper()
    
    ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
    defer cancel()
    
    if err := a.App.Start(ctx); err != nil {
        a.tb.Fatalf("gaztest: app didn't start: %v", err)
    }
    return a
}

// RequireStop stops the app or fails the test
func (a *App) RequireStop() {
    a.tb.Helper()
    
    if a.stopped {
        return // Idempotent
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
    defer cancel()
    
    if err := a.App.Stop(ctx); err != nil {
        a.tb.Fatalf("gaztest: app didn't stop: %v", err)
    }
    a.stopped = true
}
```

### 4. Key Design Decisions

**Build() returns (App, error):** Per CONTEXT.md decision - caller handles errors, allows test assertions on build failures.

**Replace before Build:** Replacements must be called before Build() because the container is validated during Build().

**Type not in container returns error:** Per CONTEXT.md - replacing a type not in container returns error from Build().

**Single timeout for all operations:** Per CONTEXT.md - start and stop share the same timeout value.

## Open Questions

Things that couldn't be fully resolved:

1. **Whether App should embed *gaz.App or wrap it**
   - What we know: fxtest embeds *fx.App, exposing full API
   - What's unclear: Do we want to expose all gaz.App methods or just subset?
   - Recommendation: Embed *gaz.App for full access, document that Container() is available for advanced use

2. **Handling Replace for types not yet registered**
   - What we know: CONTEXT.md says error from Build() if type not in container
   - What's unclear: Should we defer validation to Build() or validate eagerly in Replace()?
   - Recommendation: Defer to Build() - allows builder pattern without ordering constraints

3. **Logger configuration for test apps**
   - What we know: fxtest redirects logs to testing.T
   - What's unclear: Should gaztest suppress logs or redirect to t.Log()?
   - Recommendation: Create test logger that redirects to t.Log() for debugging

## Sources

### Primary (HIGH confidence)
- Context7 `/uber-go/fx` - fxtest patterns, Replace/Supply mechanics
- pkg.go.dev/go.uber.org/fx/fxtest - Official fxtest API documentation
- GitHub uber-go/fx/fxtest/app.go - Source implementation

### Secondary (MEDIUM confidence)
- Go testing package documentation for t.Cleanup() semantics
- Go reflect package for type inference patterns

### Tertiary (LOW confidence)
- None - all critical patterns verified with official sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Uses only stdlib + existing gaz patterns
- Architecture: HIGH - Follows established fxtest pattern
- Pitfalls: MEDIUM - Based on common DI testing issues, not gaz-specific experience

**Research date:** 2026-01-29
**Valid until:** 60 days (stable pattern, unlikely to change)
