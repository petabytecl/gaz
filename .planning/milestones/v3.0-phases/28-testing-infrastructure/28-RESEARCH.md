# Phase 28: Testing Infrastructure - Research

**Researched:** 2026-01-31
**Domain:** Go Testing, Test Helpers, gaztest Enhancement, Documentation
**Confidence:** HIGH

## Summary

This phase enhances the gaztest package and creates comprehensive testing support for v3 patterns. The research focuses on three areas: (1) API extensions to gaztest for module registration and config injection, (2) per-subsystem test helpers following Go/testify conventions, and (3) testing documentation and examples.

The established patterns are clear: testify v1.11.1 is already in use, Go's `testing.TB` and `t.Helper()` patterns are well-documented, and the existing gaztest API provides a solid foundation. The key decisions from CONTEXT.md (RequireResolve, WithModules, WithConfigMap, Require* prefix helpers) align with Go testing best practices.

**Primary recommendation:** Extend gaztest with `WithModules(m ...di.Module)`, `WithConfigMap(map[string]any)`, and `RequireResolve[T](t, app)` generic helper. Create `testing.go` files in each subsystem package with mock factories, test configs, and `Require*` assertion helpers.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| testing | stdlib | Core Go testing | Built-in, universal |
| testify | v1.11.1 | Assertions/mocking | Already in go.mod, project standard |
| gaztest | internal | gaz app testing | Existing test package to enhance |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| testing.TB | stdlib | Universal test interface | Helper functions for T/B compatibility |
| t.Helper() | stdlib | Stack trace cleanup | Every test helper function |
| t.Cleanup() | stdlib | Automatic teardown | Resource management in helpers |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| testify/require | is (matryer) | Simpler API but testify already in use |
| testify/mock | gomock | Code generation vs runtime mocking |

**Installation:**
```bash
# Already in go.mod, no additional dependencies needed
# github.com/stretchr/testify v1.11.1
```

## Architecture Patterns

### Recommended Project Structure for Testing
```
gaztest/
├── builder.go          # Builder type with WithModules, WithConfigMap
├── app.go              # App type with RequireStart, RequireStop
├── helpers.go          # RequireResolve[T] generic helper
├── doc.go              # Package documentation
├── README.md           # Testing guide (NEW)
├── example_test.go     # Godoc examples (enhanced)
└── gaztest_test.go     # Package tests

health/
├── module.go
├── testing.go          # TestConfig(), NewTestConfig(opts...), mocks, RequireHealthy()
└── module_test.go

worker/
├── module.go
├── testing.go          # TestConfig(), MockWorker, RequireWorkerStarted()
└── module_test.go

cron/
├── module.go
├── testing.go          # TestConfig(), MockJob, RequireSchedulerRunning()
└── module_test.go

eventbus/
├── module.go
├── testing.go          # TestBus(), MockEvent, RequirePublished()
└── bus_test.go

config/
├── module.go
├── testing.go          # TestConfig(), MockConfigProvider
└── module_test.go
```

### Pattern 1: Testing Helper Function Pattern
**What:** Helper functions that fail tests on error, use `t.Helper()`, and use `testing.TB` for flexibility
**When to use:** All custom assertion helpers, setup functions, factory functions
**Example:**
```go
// Source: Go testing best practices 2025
func RequireResolve[T any](tb testing.TB, app *App) T {
    tb.Helper()
    result, err := gaz.Resolve[T](app.Container())
    if err != nil {
        tb.Fatalf("RequireResolve[%s]: %v", gaz.TypeName[T](), err)
    }
    return result
}
```

### Pattern 2: Test Config Pattern
**What:** Dual-pattern config generation: `TestConfig()` for defaults, `NewTestConfig(opts...)` for customization
**When to use:** Every subsystem that has configuration
**Example:**
```go
// Source: Codebase pattern analysis
// TestConfig returns sensible defaults for testing
func TestConfig() Config {
    return Config{
        Port:          0, // Random available port
        LivenessPath:  "/live",
        ReadinessPath: "/ready",
        StartupPath:   "/startup",
    }
}

// NewTestConfig creates a config with the given options
func NewTestConfig(opts ...ConfigOption) Config {
    cfg := TestConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    return cfg
}
```

### Pattern 3: Mock Factory Pattern
**What:** Factory functions that create pre-configured mocks for testing
**When to use:** When subsystem components need to be mocked in tests
**Example:**
```go
// Source: Testify patterns
// MockWorker implements worker.Worker for testing
type MockWorker struct {
    mock.Mock
}

func (m *MockWorker) Name() string { return "mock-worker" }
func (m *MockWorker) OnStart(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}
func (m *MockWorker) OnStop(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}

// NewMockWorker creates a mock worker with default expectations
func NewMockWorker() *MockWorker {
    m := &MockWorker{}
    m.On("OnStart", mock.Anything).Return(nil)
    m.On("OnStop", mock.Anything).Return(nil)
    return m
}
```

### Pattern 4: Assertion Helper Pattern (Require* prefix)
**What:** Assertion helpers that fail tests immediately following testify's `require.*` convention
**When to use:** Domain-specific assertions for each subsystem
**Example:**
```go
// Source: Testify conventions + CONTEXT.md decision
// RequireHealthy asserts that the health manager reports healthy status
func RequireHealthy(tb testing.TB, m *Manager) {
    tb.Helper()
    checker := m.ReadinessChecker()
    result := checker.Check(context.Background())
    if result.Status != health.StatusUp {
        tb.Fatalf("RequireHealthy: expected status UP, got %s", result.Status)
    }
}
```

### Anti-Patterns to Avoid
- **Returning errors from helpers:** Helpers should call `t.Fatal()` or `t.Error()`, not return errors
- **Missing t.Helper():** Always call as first line to get correct stack traces
- **Global state without cleanup:** Use `t.Cleanup()` to restore any modified state
- **Over-abstraction:** Keep helpers atomic and focused

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Assertions | Custom if/error checks | testify/require | Battle-tested, consistent API |
| Mocking | Manual mock structs | testify/mock | Expectation setting, verification |
| Test cleanup | defer chains | t.Cleanup() | Works even on panic, registered order |
| Random ports | Fixed test ports | port 0 (OS-assigned) | Avoids port conflicts in parallel tests |
| Test timeouts | Manual context.WithTimeout | gaztest.WithTimeout() | Already handled by gaztest |

**Key insight:** gaztest already handles app lifecycle, auto-cleanup, and mock replacement. The extensions should follow the same patterns.

## Common Pitfalls

### Pitfall 1: Import Cycles with Testing Helpers
**What goes wrong:** Placing test helpers in gaztest that import subsystem packages, while subsystem packages might need to use gaztest
**Why it happens:** Testing helpers often need both the test framework and the subsystem types
**How to avoid:** Place subsystem-specific helpers in `{subsystem}/testing.go`, not in gaztest. The gaztest package only contains core app testing utilities.
**Warning signs:** "import cycle" compiler errors

### Pitfall 2: Forgetting t.Helper() in Nested Helpers
**What goes wrong:** Stack trace points to helper internals instead of test call site
**Why it happens:** Helper functions calling other helper functions
**How to avoid:** EVERY helper function must call `t.Helper()` as its first line
**Warning signs:** Test failure messages pointing to wrong line numbers

### Pitfall 3: Tight Coupling to Real Ports
**What goes wrong:** Tests fail intermittently due to port conflicts
**Why it happens:** Using fixed ports like `9090` in test configs
**How to avoid:** Use `Port: 0` in TestConfig() - OS assigns available port
**Warning signs:** "address already in use" errors in CI

### Pitfall 4: Missing Context Cancellation in Tests
**What goes wrong:** Tests hang or timeout waiting for goroutines
**Why it happens:** Goroutines waiting on contexts that are never cancelled
**How to avoid:** Always pass cancellable contexts, use t.Cleanup() to cancel
**Warning signs:** Tests timing out, goroutine leaks reported by race detector

### Pitfall 5: Expecting Synchronous Behavior from EventBus
**What goes wrong:** Assertions fail because events haven't been delivered yet
**Why it happens:** EventBus is asynchronous, Publish returns immediately
**How to avoid:** Use synchronization helpers or test-mode buffers
**Warning signs:** Flaky tests that pass when run slowly

## Code Examples

Verified patterns from the codebase and official sources:

### gaztest Builder Extension
```go
// Source: Codebase analysis + CONTEXT.md decisions
package gaztest

// WithModules adds modules to the test app during build
func (b *Builder) WithModules(modules ...di.Module) *Builder {
    b.modules = append(b.modules, modules...)
    return b
}

// WithConfigMap injects raw config values for testing
func (b *Builder) WithConfigMap(values map[string]any) *Builder {
    b.configValues = values
    return b
}
```

### Generic RequireResolve Helper
```go
// Source: Go generics + testify conventions
package gaztest

// RequireResolve resolves a service from the app or fails the test
func RequireResolve[T any](tb TB, app *App) T {
    tb.Helper()
    result, err := gaz.Resolve[T](app.Container())
    if err != nil {
        tb.Fatalf("RequireResolve[%s]: %v", reflect.TypeOf((*T)(nil)).Elem().String(), err)
    }
    return result
}
```

### Subsystem Testing.go Pattern (health example)
```go
// Source: Codebase analysis + CONTEXT.md decisions
package health

import (
    "context"
    "testing"
)

// TestConfig returns a Config suitable for testing
func TestConfig() Config {
    return Config{
        Port:          0, // OS-assigned port
        LivenessPath:  "/live",
        ReadinessPath: "/ready",
        StartupPath:   "/startup",
    }
}

// NewTestConfig returns a Config with the given options applied
func NewTestConfig(opts ...func(*Config)) Config {
    cfg := TestConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    return cfg
}

// RequireHealthy asserts the manager reports healthy status
func RequireHealthy(tb testing.TB, m *Manager) {
    tb.Helper()
    checker := m.ReadinessChecker()
    result := checker.Check(context.Background())
    if result.Status != "up" {
        tb.Fatalf("RequireHealthy: expected status 'up', got '%s'", result.Status)
    }
}

// RequireUnhealthy asserts the manager reports unhealthy status
func RequireUnhealthy(tb testing.TB, m *Manager) {
    tb.Helper()
    checker := m.ReadinessChecker()
    result := checker.Check(context.Background())
    if result.Status == "up" {
        tb.Fatalf("RequireUnhealthy: expected status not 'up', got '%s'", result.Status)
    }
}
```

### Example Test Using Enhanced API
```go
// Source: Pattern synthesis
func TestWorkerIntegration(t *testing.T) {
    // Create test app with modules and config
    app, err := gaztest.New(t).
        WithModules(worker.NewModule()).
        WithConfigMap(map[string]any{
            "worker.pool_size": 2,
        }).
        Build()
    require.NoError(t, err)

    app.RequireStart()
    defer app.RequireStop()

    // Resolve and verify
    mgr := gaztest.RequireResolve[*worker.Manager](t, app)
    require.NotNil(t, mgr)
}
```

### Godoc Example Pattern
```go
// Source: Go documentation conventions
package gaztest_test

import (
    "github.com/petabytecl/gaz"
    "github.com/petabytecl/gaz/gaztest"
    "github.com/petabytecl/gaz/health"
)

// Example_withModules demonstrates using WithModules to register modules.
func Example_withModules() {
    t := &testing.T{} // In real tests, use the provided *testing.T

    app, err := gaztest.New(t).
        WithModules(health.NewModule(health.WithPort(0))).
        Build()
    if err != nil {
        fmt.Println("build failed:", err)
        return
    }

    app.RequireStart()
    defer app.RequireStop()

    // Access health manager via container
    mgr, _ := gaz.Resolve[*health.Manager](app.Container())
    _ = mgr // Use in your test
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual defer cleanup | t.Cleanup() | Go 1.14 | Auto-cleanup even on panic |
| *testing.T only | testing.TB | Go 1.x | Works with benchmarks and fuzz |
| Type-specific helpers | Generic helpers | Go 1.18 | Single RequireResolve[T] works for all types |
| Return errors from helpers | t.Fatal in helpers | Best practice | Cleaner test code, no if/err checks |

**Deprecated/outdated:**
- Returning cleanup functions from helpers (use t.Cleanup instead)
- Type-specific assertion helpers (use generics)
- Fixed port numbers in tests (use port 0)

## Open Questions

Things that couldn't be fully resolved:

1. **EventBus test synchronization**
   - What we know: EventBus is async, tests need to wait for delivery
   - What's unclear: Best pattern for synchronizing in tests without modifying production code
   - Recommendation: Create a `TestBus()` helper that returns a bus with smaller buffer and provides a `WaitForDelivery()` method

2. **Standalone example location**
   - What we know: Godoc examples go in `*_test.go` files, standalone scenarios need a home
   - What's unclear: Whether to use `examples/` directory or `testdata/`
   - Recommendation: Use `gaztest/examples_test.go` for comprehensive Godoc examples, avoid separate directory

## Sources

### Primary (HIGH confidence)
- Go testing package documentation (stdlib)
- Testify v1.11.1 README and godoc
- Existing gaztest package in codebase
- CONTEXT.md decisions for this phase

### Secondary (MEDIUM confidence)
- Google Search: Go testing helper patterns 2025
- Existing subsystem test patterns in codebase (health_test.go, worker_test.go)

### Tertiary (LOW confidence)
- None - all patterns verified against primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - testify already in use, patterns well-established
- Architecture: HIGH - following existing codebase conventions
- API extensions: HIGH - CONTEXT.md provides clear decisions
- Pitfalls: HIGH - based on codebase analysis and Go testing conventions

**Research date:** 2026-01-31
**Valid until:** 2026-03-01 (stable Go testing patterns, unlikely to change)
