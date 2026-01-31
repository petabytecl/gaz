# Phase 27: Error Standardization - Research

**Researched:** 2026-01-31
**Domain:** Go error handling, sentinel errors, typed errors, error wrapping
**Confidence:** HIGH

## Summary

This research covers consolidating all sentinel errors into a single `gaz/errors.go` file and standardizing error handling patterns across the gaz framework. The phase implements a clean-break approach (v3 philosophy) where all errors move to a central location with consistent naming and wrapping formats.

Go 1.13+ established the modern error handling paradigm with `errors.Is`, `errors.As`, and `%w` wrapping. The standard library's approach is authoritative: sentinel errors use `errors.New()`, typed errors implement `Error()`, `Unwrap()`, and optionally `Is()`/`As()` methods. The codebase currently has errors scattered across `di/errors.go`, `config/errors.go`, `worker/errors.go`, with inconsistent naming and wrapping patterns.

**Primary recommendation:** Create centralized `gaz/errors.go` with all sentinel errors using `ErrSubsystem*` naming, implement typed errors for recovery scenarios, and enforce `"pkg: context: %w"` wrapping format at package boundaries.

## Standard Stack

Go error handling requires no external libraries - the standard library provides everything needed.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `errors` | stdlib | `errors.New()`, `errors.Is()`, `errors.As()`, `errors.Unwrap()` | Go 1.13+ standard |
| `fmt` | stdlib | `fmt.Errorf()` with `%w` verb for wrapping | Go 1.13+ standard |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `errors.Join()` | Go 1.20+ | Combine multiple errors | Lifecycle shutdown collecting multiple failures |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Standard library | github.com/pkg/errors | Archived, adds stack traces but now outdated |
| `%w` verb | `%v` verb | Loses unwrap chain, breaks `errors.Is()` |

**No installation needed - stdlib only.**

## Architecture Patterns

### Recommended Error File Structure

```
gaz/
├── errors.go          # ALL sentinel errors + typed errors for the framework
├── di/
│   └── (no errors.go) # Removed - errors moved to gaz/errors.go
├── config/
│   └── (no errors.go) # Removed - errors moved to gaz/errors.go
├── worker/
│   └── (no errors.go) # Removed - errors moved to gaz/errors.go
├── health/
│   └── (no errors.go) # No errors currently defined
├── eventbus/
│   └── (no errors.go) # No errors currently defined
└── cron/
    └── (no errors.go) # Inline errors moved to gaz/errors.go
```

### Pattern 1: Sentinel Error Declaration
**What:** Predefined error values for common, expected conditions
**When to use:** When caller needs to branch logic based on specific error condition
**Example:**
```go
// Source: Go standard library pattern + user decisions
package gaz

import "errors"

// DI subsystem errors
var (
    // ErrDINotFound is returned when a requested service is not registered.
    ErrDINotFound = errors.New("di: not found")

    // ErrDICycle is returned when circular dependency detected.
    ErrDICycle = errors.New("di: circular dependency")

    // ErrDIDuplicate is returned when service already registered.
    ErrDIDuplicate = errors.New("di: duplicate registration")

    // ErrDITypeMismatch is returned when resolved service type doesn't match.
    ErrDITypeMismatch = errors.New("di: type mismatch")

    // ErrDIAlreadyBuilt is returned when registering after Build().
    ErrDIAlreadyBuilt = errors.New("di: already built")

    // ErrDIInvalidProvider is returned when provider signature is invalid.
    ErrDIInvalidProvider = errors.New("di: invalid provider")

    // ErrDINotSettable is returned when struct field cannot be set.
    ErrDINotSettable = errors.New("di: field not settable")
)

// Config subsystem errors
var (
    // ErrConfigValidation is returned when config validation fails.
    ErrConfigValidation = errors.New("config: validation failed")

    // ErrConfigNotFound is returned when config key doesn't exist.
    ErrConfigNotFound = errors.New("config: key not found")
)

// Worker subsystem errors
var (
    // ErrWorkerCircuitTripped is returned when worker exhausted restart attempts.
    ErrWorkerCircuitTripped = errors.New("worker: circuit breaker tripped")

    // ErrWorkerStopped indicates worker stopped normally (not an error condition).
    ErrWorkerStopped = errors.New("worker: stopped normally")

    // ErrWorkerCriticalFailed indicates critical worker failed, initiating shutdown.
    ErrWorkerCriticalFailed = errors.New("worker: critical worker failed")

    // ErrWorkerManagerRunning is returned when registering after manager started.
    ErrWorkerManagerRunning = errors.New("worker: manager already running")
)

// Cron subsystem errors
var (
    // ErrCronNotRunning is returned when scheduler health check fails.
    ErrCronNotRunning = errors.New("cron: scheduler not running")
)

// Module errors (gaz-specific)
var (
    // ErrModuleDuplicate is returned when module name already registered.
    ErrModuleDuplicate = errors.New("gaz: duplicate module")

    // ErrConfigKeyCollision is returned when config keys collide.
    ErrConfigKeyCollision = errors.New("gaz: config key collision")
)
```

### Pattern 2: Typed Error for Recovery
**What:** Struct-based errors that carry metadata for caller recovery
**When to use:** When caller needs additional context to recover (which service? which field?)
**Example:**
```go
// Source: Go blog + CONTEXT.md decisions
package gaz

import "fmt"

// ResolutionError provides detailed context about DI resolution failures.
// Use errors.As() to extract resolution details for debugging/recovery.
type ResolutionError struct {
    ServiceName string   // The service that failed to resolve
    Chain       []string // Resolution chain leading to failure
    Cause       error    // Underlying error
}

func (e *ResolutionError) Error() string {
    if len(e.Chain) > 0 {
        return fmt.Sprintf("di: resolution failed: %s (chain: %v): %v",
            e.ServiceName, e.Chain, e.Cause)
    }
    return fmt.Sprintf("di: resolution failed: %s: %v", e.ServiceName, e.Cause)
}

func (e *ResolutionError) Unwrap() error {
    return e.Cause
}

// Is allows matching against sentinel errors in the cause chain.
func (e *ResolutionError) Is(target error) bool {
    return false // Let Unwrap handle chain traversal
}

// ParseError provides context about config parsing failures.
type ParseError struct {
    Key   string // Config key that failed
    Value any    // The problematic value
    Cause error  // Underlying error
}

func (e *ParseError) Error() string {
    return fmt.Sprintf("config: parse error for key %q: %v", e.Key, e.Cause)
}

func (e *ParseError) Unwrap() error {
    return e.Cause
}

// LifecycleError provides context about service lifecycle failures.
type LifecycleError struct {
    ServiceName string // Service that failed
    Phase       string // "start" or "stop"
    Cause       error  // Underlying error
}

func (e *LifecycleError) Error() string {
    return fmt.Sprintf("lifecycle: %s failed for %s: %v",
        e.Phase, e.ServiceName, e.Cause)
}

func (e *LifecycleError) Unwrap() error {
    return e.Cause
}
```

### Pattern 3: Consistent Wrapping Format
**What:** Standard format for adding context when wrapping errors
**When to use:** Always at package boundaries, when crossing layers
**Example:**
```go
// Source: CONTEXT.md decisions + Go best practices
// Format: "pkg: context: %w"

// Good - adds package context + descriptive action
if err != nil {
    return fmt.Errorf("di: resolve %s: %w", serviceName, err)
}

// Good - config layer wrapping
if err != nil {
    return fmt.Errorf("config: unmarshal key %q: %w", key, err)
}

// Good - worker layer wrapping
if err != nil {
    return fmt.Errorf("worker: start %s: %w", workerName, err)
}

// Bad - no package prefix
if err != nil {
    return fmt.Errorf("failed to start: %w", err) // Missing pkg prefix
}

// Bad - uses %v instead of %w
if err != nil {
    return fmt.Errorf("di: resolve: %v", err) // Breaks errors.Is chain
}
```

### Pattern 4: Sentinel Wrapping (Always Wrap)
**What:** Never return bare sentinels - always wrap with context
**When to use:** Per CONTEXT.md, wrap sentinels to force errors.Is usage
**Example:**
```go
// Source: Go blog recommendation + CONTEXT.md

// Good - wrapped sentinel forces errors.Is() usage
if !found {
    return fmt.Errorf("di: service %s: %w", name, ErrDINotFound)
}

// Also acceptable for very simple cases
if !found {
    return fmt.Errorf("%w: %s", ErrDINotFound, name)
}

// Caller checks with errors.Is (works through wrapping)
if errors.Is(err, gaz.ErrDINotFound) {
    // Handle not found case
}

// Bad - bare sentinel (allows direct comparison)
if !found {
    return ErrDINotFound // Allows callers to use err == ErrDINotFound
}
```

### Anti-Patterns to Avoid
- **String comparison for errors:** Never use `err.Error() == "..."` - use `errors.Is()`
- **Bare sentinel returns:** Always wrap with context for debuggability
- **Missing %w verb:** Using `%v` breaks error chain - always use `%w`
- **Scattered error definitions:** Errors in multiple files makes discovery hard
- **Inconsistent naming:** Mix of `ErrNotFound` vs `ErrServiceNotFound` vs `NotFoundError`

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Error wrapping | Custom wrap functions | `fmt.Errorf("...: %w", err)` | Standard, optimized, works with errors.Is/As |
| Error matching | Type assertions | `errors.Is(err, target)` | Handles wrapped chains correctly |
| Type extraction | Manual type switches | `errors.As(err, &target)` | Handles wrapped chains correctly |
| Multiple errors | Custom slice types | `errors.Join(err1, err2)` | Go 1.20+ standard, works with Is/As |
| Stack traces | github.com/pkg/errors | Structured logging with slog | More modern approach, errors as values |

**Key insight:** Go's error handling since 1.13 is complete. The stdlib `errors` and `fmt` packages handle all common cases. Third-party packages add complexity without benefits for most use cases.

## Common Pitfalls

### Pitfall 1: Breaking the Unwrap Chain
**What goes wrong:** Using `%v` instead of `%w` makes wrapped errors invisible to `errors.Is()`
**Why it happens:** Developer forgets or doesn't understand the difference
**How to avoid:** Always use `%w` for error arguments; lint for `%v` with error types
**Warning signs:** Tests using `errors.Is()` fail unexpectedly; errors "disappear" in chain

### Pitfall 2: Inconsistent Package Prefixes
**What goes wrong:** Error messages have different prefix styles: `"di:"`, `"DI:"`, `"container:"`
**Why it happens:** No enforced convention, each developer picks own style
**How to avoid:** Document standard (`"pkg: context"`) and review in code review
**Warning signs:** Error logs show inconsistent patterns, hard to grep

### Pitfall 3: Exposing Internal Errors
**What goes wrong:** Wrapping third-party errors exposes them as API
**Why it happens:** Using `%w` on all errors, including internal ones
**How to avoid:** Use `%v` for internal errors, `%w` only for errors that are part of API
**Warning signs:** User code depends on internal error types (like `sql.ErrNoRows`)

### Pitfall 4: Missing Is/Unwrap on Typed Errors
**What goes wrong:** Typed errors don't work with `errors.Is()` or `errors.As()`
**Why it happens:** Forgetting to implement `Unwrap()` method on error struct
**How to avoid:** Always implement `Error()` and `Unwrap()` on typed errors; `Is()` if needed
**Warning signs:** `errors.Is(err, ErrSentinel)` returns false even when sentinel is wrapped

### Pitfall 5: Over-Wrapping
**What goes wrong:** Error messages become unreadable: `"di: resolve: di: provider: di: instance:"`
**Why it happens:** Wrapping at every function call, not just package boundaries
**How to avoid:** Wrap at boundaries only; inner functions can return bare errors
**Warning signs:** Error messages repeat the same prefix multiple times

### Pitfall 6: Re-export Confusion
**What goes wrong:** `gaz.ErrNotFound` and `di.ErrNotFound` are same value but different symbols
**Why it happens:** Re-exporting errors for convenience creates two paths to same error
**How to avoid:** Choose ONE location (gaz/errors.go per CONTEXT.md), delete from subsystems
**Warning signs:** Confusion about which import to use; subtle test failures

## Code Examples

Verified patterns from official sources and codebase analysis:

### Sentinel Error Definition
```go
// Source: Go standard library pattern
package gaz

import "errors"

// ErrDINotFound is returned when a requested service is not found.
// Check with errors.Is(err, ErrDINotFound).
var ErrDINotFound = errors.New("di: not found")
```

### Using Sentinel Errors in Code
```go
// Source: di/container.go pattern (current codebase)
// Good: wrap sentinel with context
if !ok {
    return nil, fmt.Errorf("%w: %s", ErrDINotFound, name)
}

// Caller usage
svc, err := gaz.Resolve[*MyService](container)
if errors.Is(err, gaz.ErrDINotFound) {
    // Service not registered - handle appropriately
    return nil, fmt.Errorf("mypackage: required service missing: %w", err)
}
```

### Typed Error with Unwrap
```go
// Source: config/errors.go (current codebase) + Go blog
type ValidationError struct {
    Errors []FieldError
}

func (ve ValidationError) Error() string {
    // Format error message
    msgs := make([]string, len(ve.Errors))
    for i, e := range ve.Errors {
        msgs[i] = e.String()
    }
    return fmt.Sprintf("config: validation failed:\n%s", strings.Join(msgs, "\n"))
}

// Unwrap returns sentinel so errors.Is works
func (ve ValidationError) Unwrap() error {
    return ErrConfigValidation
}

// Caller usage
err := config.Load(target)
if errors.Is(err, gaz.ErrConfigValidation) {
    var ve gaz.ValidationError
    if errors.As(err, &ve) {
        for _, fieldErr := range ve.Errors {
            log.Printf("Invalid field: %s", fieldErr.Namespace)
        }
    }
}
```

### Migration: Updating Error References
```go
// Before (di/errors.go and gaz/errors.go with re-exports)
import "github.com/petabytecl/gaz/di"
if errors.Is(err, di.ErrNotFound) { ... }

// After (gaz/errors.go only)
import "github.com/petabytecl/gaz"
if errors.Is(err, gaz.ErrDINotFound) { ... }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `err == ErrX` | `errors.Is(err, ErrX)` | Go 1.13 (2019) | Works with wrapped errors |
| `err.(*T)` type assertion | `errors.As(err, &target)` | Go 1.13 (2019) | Works with wrapped errors |
| `%v` for all errors | `%w` for wrappable errors | Go 1.13 (2019) | Enables error chain |
| `github.com/pkg/errors` | stdlib `errors` + `fmt` | Go 1.13 (2019) | No external dependency |
| Per-package errors.go | Central errors.go (v3 decision) | v3.0 | Single source of truth |

**Deprecated/outdated:**
- `github.com/pkg/errors`: Archived, use stdlib
- Direct error comparison (`==`): Use `errors.Is()` for wrapped error support
- Type assertions for errors: Use `errors.As()` for wrapped error support

## Claude's Discretion Recommendations

### 1. Re-export from Subsystem Packages: NO
**Recommendation:** Do NOT re-export errors from subsystem packages.
**Reasoning:**
- Creates two paths to same error (`gaz.ErrDINotFound` vs `di.ErrDINotFound`)
- Confuses users about which to import
- Clean break philosophy: one location only
- Current codebase already has confusion with `gaz/errors.go` re-exporting from `di/errors.go`

### 2. Organization within gaz/errors.go: GROUPED BY SUBSYSTEM
**Recommendation:** Group errors by subsystem with comment headers.
**Reasoning:**
- Easier to find all DI errors together
- Matches naming convention (ErrDI*, ErrConfig*, etc.)
- Aligns with how users think about the framework

```go
// gaz/errors.go

// DI container errors
var (
    ErrDINotFound = ...
    ErrDICycle = ...
    // ...
)

// Config subsystem errors
var (
    ErrConfigValidation = ...
    ErrConfigNotFound = ...
)

// Worker subsystem errors
var (
    ErrWorkerCircuitTripped = ...
    // ...
)

// Typed errors below
type ResolutionError struct { ... }
type ValidationError struct { ... }
```

### 3. Wrap Context Identifiers: INCLUDE SERVICE/TYPE NAMES
**Recommendation:** Include service names and type names in wrap context.
**Reasoning:**
- Makes debugging much easier: "di: resolve *MyService: not found"
- Already the pattern in current codebase
- Cost is minimal (string formatting)

```go
// Good - includes service name
fmt.Errorf("di: resolve %s: %w", serviceName, err)

// Good - includes config key
fmt.Errorf("config: unmarshal key %q: %w", key, err)

// Less useful - no specific context
fmt.Errorf("di: resolve: %w", err)
```

## Open Questions

Things that couldn't be fully resolved:

1. **EventBus error sentinels**
   - What we know: EventBus currently has no explicit sentinel errors
   - What's unclear: Should `ErrEventBusClosed` be added, or is silent no-op sufficient?
   - Recommendation: Keep current behavior (silent no-op on closed bus per CONTEXT.md), add sentinel only if users need to detect closed state

2. **Health check errors**
   - What we know: Health checks return inline `errors.New("scheduler not running")`
   - What's unclear: Should these become sentinels like `ErrHealthCheckFailed`?
   - Recommendation: Add `ErrCronNotRunning` sentinel since users may want to programmatically detect this

## Sources

### Primary (HIGH confidence)
- Go official blog: "Working with Errors in Go 1.13" - https://go.dev/blog/go1.13-errors
- Go `errors` package documentation - https://pkg.go.dev/errors
- Codebase analysis: `di/errors.go`, `config/errors.go`, `worker/errors.go`, `gaz/errors.go`
- CONTEXT.md decisions (user-locked choices)

### Secondary (MEDIUM confidence)
- Google Search synthesis: Go error handling best practices 2025/2026
- Uber Go Style Guide patterns (referenced in search results)

### Tertiary (LOW confidence)
- None - all findings verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - stdlib only, verified with official docs
- Architecture: HIGH - patterns from official blog + existing codebase
- Naming conventions: HIGH - locked by CONTEXT.md decisions
- Typed errors: HIGH - patterns from official blog + existing `ValidationError`
- Claude's discretion items: MEDIUM - recommendations based on best practices

**Research date:** 2026-01-31
**Valid until:** 90 days (stable patterns, no expected changes to Go error handling)

## Summary Checklist for Implementation

### Files to Create/Modify
- [ ] `gaz/errors.go` - Expand with all sentinel errors
- [ ] `gaz/errors.go` - Add typed errors (ResolutionError, ParseError, LifecycleError)

### Files to Delete (errors only)
- [ ] `di/errors.go` - All sentinels move to gaz/errors.go
- [ ] `config/errors.go` - All errors move to gaz/errors.go  
- [ ] `worker/errors.go` - All sentinels move to gaz/errors.go

### Update References
- [ ] All `di.ErrNotFound` -> `gaz.ErrDINotFound`
- [ ] All `di.ErrCycle` -> `gaz.ErrDICycle`
- [ ] All `config.ErrKeyNotFound` -> `gaz.ErrConfigNotFound`
- [ ] All `worker.ErrCircuitBreakerTripped` -> `gaz.ErrWorkerCircuitTripped`
- [ ] Update wrapping format to `"pkg: context: %w"`

### Naming Mapping (Old -> New)
| Old | New |
|-----|-----|
| `di.ErrNotFound` | `ErrDINotFound` |
| `di.ErrCycle` | `ErrDICycle` |
| `di.ErrDuplicate` | `ErrDIDuplicate` |
| `di.ErrNotSettable` | `ErrDINotSettable` |
| `di.ErrTypeMismatch` | `ErrDITypeMismatch` |
| `di.ErrAlreadyBuilt` | `ErrDIAlreadyBuilt` |
| `di.ErrInvalidProvider` | `ErrDIInvalidProvider` |
| `config.ErrConfigValidation` | `ErrConfigValidation` |
| `config.ErrKeyNotFound` | `ErrConfigNotFound` |
| `worker.ErrCircuitBreakerTripped` | `ErrWorkerCircuitTripped` |
| `worker.ErrWorkerStopped` | `ErrWorkerStopped` |
| `worker.ErrCriticalWorkerFailed` | `ErrWorkerCriticalFailed` |
| `worker.ErrManagerAlreadyRunning` | `ErrWorkerManagerRunning` |
| `gaz.ErrDuplicateModule` | `ErrModuleDuplicate` |
| `gaz.ErrConfigKeyCollision` | `ErrConfigKeyCollision` |
