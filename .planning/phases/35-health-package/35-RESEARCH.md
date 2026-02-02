# Phase 35: Health Package + Integration - Research

**Researched:** 2026-02-01
**Domain:** Health check library replacement (alexliesenfeld/health -> internal health/internal/)
**Confidence:** HIGH

## Summary

This phase replaces the external `alexliesenfeld/health` library with an internal `health/internal/` package. The current codebase uses alexliesenfeld/health v0.8.1 for HTTP health endpoints with a custom IETFResultWriter already implemented. The replacement must provide identical functionality while adding features specified in CONTEXT.md (parallel execution, per-check timeouts, panic recovery, configurable visibility).

The migration follows the established pattern from prior phases (backoff, logger/tint, cron/internal): create internal package with equivalent API, update consumers, remove external dependency. The current usage is well-encapsulated in `health/manager.go` and `health/handlers.go`, making migration straightforward.

**Primary recommendation:** Create `health/internal/` package mirroring alexliesenfeld/health's core types and functional options pattern, then update `health/manager.go` to use internal package instead.

## Standard Stack

### Core (Internal Implementation)

| Component | Purpose | Why Required |
|-----------|---------|--------------|
| `health/internal/check.go` | Check struct, CheckFunc type | Per HLT-01 |
| `health/internal/checker.go` | NewChecker, Checker interface, CheckerResult | Per HLT-02, HLT-07 |
| `health/internal/status.go` | AvailabilityStatus enum (StatusUnknown, StatusUp, StatusDown) | Per HLT-08 |
| `health/internal/handler.go` | NewHandler, HTTP handler creation | Per HLT-04 |
| `health/internal/options.go` | WithCheck, WithResultWriter, WithStatusCode options | Per HLT-03, HLT-05, HLT-06 |
| `health/internal/writer.go` | ResultWriter interface, IETFResultWriter | Per HLT-09, HLT-11 |

### From Existing Codebase

| File | Purpose | Changes |
|------|---------|---------|
| `health/manager.go` | Manager struct, check registration | HLT-12: Replace imports |
| `health/handlers.go` | HTTP handler creation | Update to use health/internal/ |
| `health/writer.go` | IETFResultWriter (move to health/internal/) | Relocate or adapt |
| `health/types.go` | CheckFunc, Registrar interface | Keep as consumer interface |

## Architecture Patterns

### Recommended Package Structure

```
health/internal/
├── check.go        # Check struct, CheckerOption type
├── checker.go      # NewChecker, Checker interface, check execution
├── handler.go      # NewHandler, HTTP handler, HandlerOption type
├── options.go      # WithCheck, WithTimeout, WithResultWriter, etc.
├── status.go       # AvailabilityStatus enum constants
├── writer.go       # ResultWriter interface, IETFResultWriter
├── writer_ietf.go  # IETF health+json implementation
└── doc.go          # Package documentation
```

### Pattern 1: Functional Options (Checker)

**What:** Use functional options for configuring Checker and Handler
**When to use:** All public constructors
**Example:**
```go
// Source: alexliesenfeld/health pattern (verified via Context7)
type CheckerOption func(*checkerConfig)

func NewChecker(opts ...CheckerOption) Checker {
    cfg := defaultCheckerConfig()
    for _, opt := range opts {
        opt(&cfg)
    }
    return newChecker(cfg)
}

func WithCheck(check Check) CheckerOption {
    return func(cfg *checkerConfig) {
        cfg.checks[check.Name] = &check
    }
}

func WithTimeout(timeout time.Duration) CheckerOption {
    return func(cfg *checkerConfig) {
        cfg.timeout = timeout
    }
}
```

### Pattern 2: Check Struct with Context Callback

**What:** Check struct with Name and context-aware check function
**When to use:** All health checks
**Example:**
```go
// Source: alexliesenfeld/health Check struct (verified via Context7)
type Check struct {
    Name     string                              // Required: unique check name
    Check    func(ctx context.Context) error    // Required: check function
    Timeout  time.Duration                      // Optional: per-check timeout (default 5s)
    Critical bool                               // Optional: affects overall status (default true)
}
```

### Pattern 3: Parallel Check Execution with Timeout

**What:** Run checks concurrently with per-check timeouts and panic recovery
**When to use:** All synchronous check execution
**Example:**
```go
// Source: CONTEXT.md decisions
func (c *checker) runChecks(ctx context.Context) map[string]CheckResult {
    results := make(map[string]CheckResult)
    var mu sync.Mutex
    var wg sync.WaitGroup
    
    for _, check := range c.checks {
        wg.Add(1)
        go func(check *Check) {
            defer wg.Done()
            result := c.executeCheck(ctx, check)
            mu.Lock()
            results[check.Name] = result
            mu.Unlock()
        }(check)
    }
    
    wg.Wait()
    return results
}

func (c *checker) executeCheck(ctx context.Context, check *Check) CheckResult {
    // Per-check timeout (default 5s per CONTEXT.md)
    timeout := check.Timeout
    if timeout == 0 {
        timeout = c.defaultTimeout
    }
    
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    result := CheckResult{
        Status:    StatusUp,
        Timestamp: time.Now().UTC(),
    }
    
    // Panic recovery per CONTEXT.md
    err := func() (err error) {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("panic: %v", r)
            }
        }()
        return check.Check(ctx)
    }()
    
    if err != nil {
        result.Status = StatusDown
        result.Error = err
    }
    
    return result
}
```

### Pattern 4: ResultWriter Interface

**What:** Interface for custom response formatting
**When to use:** All HTTP response writing
**Example:**
```go
// Source: alexliesenfeld/health handler.go (verified via official repo)
type ResultWriter interface {
    Write(result *CheckerResult, statusCode int, w http.ResponseWriter, r *http.Request) error
}

type IETFResultWriter struct {
    showDetails      bool  // Per CONTEXT.md: hide by default
    showErrorDetails bool  // Per CONTEXT.md: configurable per environment
}
```

### Anti-Patterns to Avoid

- **Sequential check execution:** Don't run checks one-by-one; use goroutines for parallel execution per CONTEXT.md
- **Unbounded check execution:** Always apply per-check timeout to prevent hanging checks
- **Panic propagation:** Never let a check panic crash the health handler; recover and mark failed
- **Exposing internals by default:** Details should be hidden by default per CONTEXT.md (security)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| IETF health+json format | Custom JSON format | Existing IETFResultWriter pattern | Already implemented, follows RFC draft |
| Context timeout handling | Manual timers | `context.WithTimeout` | Standard Go pattern, handles cancellation |
| Panic recovery | Complex error handling | `defer/recover` pattern | Standard Go idiom |
| Concurrent map access | Regular map | `sync.Mutex` + map | Prevents data races |

**Key insight:** The alexliesenfeld/health library is well-designed with clear patterns. The internal implementation should mirror its API closely to minimize migration friction.

## Common Pitfalls

### Pitfall 1: Forgetting Check Name Uniqueness

**What goes wrong:** Duplicate check names silently overwrite each other
**Why it happens:** Map keyed by name without validation
**How to avoid:** Either: (a) validate uniqueness and error, or (b) document that last wins
**Warning signs:** Missing checks in health response

### Pitfall 2: Goroutine Leak on Timeout

**What goes wrong:** Check goroutine continues running after timeout, leaking resources
**Why it happens:** Context cancellation not checked inside check function
**How to avoid:** Checks should respect ctx.Done(); document this requirement
**Warning signs:** Increasing goroutine count over time

### Pitfall 3: Race Condition in Result Aggregation

**What goes wrong:** Data races when multiple goroutines update results map
**Why it happens:** Concurrent writes to shared map
**How to avoid:** Use mutex protection when collecting results
**Warning signs:** Race detector warnings, inconsistent results

### Pitfall 4: Exposing Sensitive Error Details

**What goes wrong:** Database connection strings, internal paths exposed in health response
**Why it happens:** Error messages passed through to response without filtering
**How to avoid:** Configure error visibility per environment (hide in prod per CONTEXT.md)
**Warning signs:** Security audit findings, information disclosure

### Pitfall 5: Breaking Liveness Handler Behavior

**What goes wrong:** Liveness handler returns 503 on failure, causing K8s to restart pod
**Why it happens:** Forgetting HLT-10 requirement (200 OK even on check failure)
**How to avoid:** Test liveness handler specifically with failing checks
**Warning signs:** Pods restarting unexpectedly

### Pitfall 6: Import Cycle with health/ Package

**What goes wrong:** `health/internal/` imports `health/` types, creating cycle
**Why it happens:** Trying to use existing types from consumer package
**How to avoid:** health/internal/ must be completely independent; health/ imports health/internal/
**Warning signs:** Go build errors about import cycles (INT-03)

## Code Examples

### Check Struct Definition (HLT-01)

```go
// Source: alexliesenfeld/health config.go (verified via official repo)
// Check configures a health check.
type Check struct {
    // Name must be unique among all checks. Required.
    Name string
    
    // Check is the function that performs the health check.
    // Must return nil if healthy, error if unhealthy.
    Check func(ctx context.Context) error
    
    // Timeout overrides the default timeout for this check.
    // Zero means use default (5s per CONTEXT.md).
    Timeout time.Duration
    
    // Critical determines if this check affects overall status.
    // Default is true. Non-critical checks report independently.
    Critical bool
}
```

### Checker Interface (HLT-02)

```go
// Source: alexliesenfeld/health check.go (verified via official repo)
// Checker executes health checks and returns aggregated results.
type Checker interface {
    // Check runs all configured health checks and returns the result.
    // The context may contain a deadline that will be respected.
    Check(ctx context.Context) CheckerResult
}
```

### CheckerResult Struct (HLT-07)

```go
// Source: alexliesenfeld/health check.go (verified via official repo)
// CheckerResult holds the aggregated health status and details.
type CheckerResult struct {
    // Status is the aggregated availability status.
    Status AvailabilityStatus
    // Details contains per-check results.
    Details map[string]CheckResult
}

// CheckResult holds a single check's result.
type CheckResult struct {
    // Status is the check's availability status.
    Status AvailabilityStatus
    // Timestamp is when the check was executed.
    Timestamp time.Time
    // Error is the check error (nil if healthy).
    Error error
}
```

### AvailabilityStatus Enum (HLT-08)

```go
// Source: alexliesenfeld/health check.go (verified via official repo)
// AvailabilityStatus represents system/component availability.
type AvailabilityStatus string

const (
    // StatusUnknown means the status is not yet known.
    StatusUnknown AvailabilityStatus = "unknown"
    // StatusUp means the system/component is available.
    StatusUp AvailabilityStatus = "up"
    // StatusDown means the system/component is unavailable.
    StatusDown AvailabilityStatus = "down"
)
```

### IETF health+json Response (HLT-11)

```go
// Source: draft-inadarei-api-health-check-06 (verified via IETF)
// IETF format uses "pass", "fail", "warn" for status values.
type ietfResponse struct {
    Status string                 `json:"status"`           // Required: "pass", "fail", "warn"
    Checks map[string][]ietfCheck `json:"checks,omitempty"` // Optional: per-component checks
}

type ietfCheck struct {
    Status string `json:"status"`          // "pass", "fail", "warn"
    Time   string `json:"time,omitempty"`  // ISO8601 timestamp
    Output string `json:"output,omitempty"` // Error message (omit for pass)
}

// Map internal status to IETF status
func mapToIETFStatus(s AvailabilityStatus) string {
    switch s {
    case StatusUp:
        return "pass"
    case StatusDown:
        return "fail"
    case StatusUnknown:
        return "warn"
    default:
        return "warn"
    }
}
```

### Handler Creation (HLT-04, HLT-05, HLT-06)

```go
// Source: alexliesenfeld/health handler.go (verified via official repo)
type HandlerOption func(*handlerConfig)

func NewHandler(checker Checker, opts ...HandlerOption) http.Handler {
    cfg := &handlerConfig{
        statusCodeUp:   http.StatusOK,
        statusCodeDown: http.StatusServiceUnavailable,
        resultWriter:   NewIETFResultWriter(),
    }
    for _, opt := range opts {
        opt(cfg)
    }
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        result := checker.Check(r.Context())
        statusCode := cfg.statusCodeUp
        if result.Status == StatusDown || result.Status == StatusUnknown {
            statusCode = cfg.statusCodeDown
        }
        cfg.resultWriter.Write(&result, statusCode, w, r)
    })
}

func WithResultWriter(w ResultWriter) HandlerOption {
    return func(cfg *handlerConfig) { cfg.resultWriter = w }
}

func WithStatusCodeUp(code int) HandlerOption {
    return func(cfg *handlerConfig) { cfg.statusCodeUp = code }
}

func WithStatusCodeDown(code int) HandlerOption {
    return func(cfg *handlerConfig) { cfg.statusCodeDown = code }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Sequential checks | Parallel execution | CONTEXT.md decision | Faster response times |
| Expose all details | Hide by default | CONTEXT.md decision | Security improvement |
| All checks critical | Critical vs Warning | CONTEXT.md decision | Graceful degradation |
| External library | Internal package | This phase | Reduced dependencies |

**Current alexliesenfeld/health features used:**
- `NewChecker` with `WithCheck` - used for all probe types
- `NewHandler` with `WithResultWriter`, `WithStatusCodeUp`, `WithStatusCodeDown`
- `health.Check` struct with Name and Check function
- `health.CheckerResult` with Status and Details

**alexliesenfeld/health features NOT used:**
- `WithPeriodicCheck` - not used, all checks are synchronous
- `WithCacheDuration` - not used
- `WithStatusListener` - not used
- `WithInterceptors` - not used
- `WithInfo` - not used

## Open Questions

1. **Warning Check Visibility**
   - What we know: CONTEXT.md specifies critical vs warning checks
   - What's unclear: Should warning checks appear in response when overall status is "pass"?
   - Recommendation: Include warning checks in details; they provide operational visibility without affecting overall status

2. **Error Message Filtering**
   - What we know: CONTEXT.md says "configurable per environment"
   - What's unclear: How to configure (environment variable, option, both)?
   - Recommendation: Use option `WithShowErrorDetails(bool)` on ResultWriter; let consumer decide based on environment

3. **Default Check Timeout Value**
   - What we know: CONTEXT.md says "default 5s"
   - What's unclear: Should global timeout also be 5s or longer (alexliesenfeld uses 10s global)?
   - Recommendation: 5s per-check default, 10s global timeout (matches alexliesenfeld pattern)

## Sources

### Primary (HIGH confidence)

- **Context7 /alexliesenfeld/health** - API patterns, Check struct, Checker interface, Handler options
- **alexliesenfeld/health GitHub repo** - check.go, config.go, handler.go source code
- **IETF draft-inadarei-api-health-check-06** - IETF health+json format specification
- **Existing codebase** - health/manager.go, health/handlers.go, health/writer.go

### Secondary (MEDIUM confidence)

- **CONTEXT.md** - User decisions on parallel execution, timeouts, visibility, criticality

### Tertiary (LOW confidence)

None - all findings verified with primary sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Based on existing codebase and alexliesenfeld/health source
- Architecture: HIGH - Established pattern from prior phases and library source
- Pitfalls: HIGH - Common Go concurrency issues, verified against library implementation
- IETF format: HIGH - Verified against RFC draft

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (30 days - stable domain)
