# Architecture Patterns: Dependency Replacement

**Domain:** Internal dependency replacement in gaz framework
**Researched:** 2026-02-01
**Confidence:** HIGH (based on existing codebase analysis)

## Executive Summary

This document defines the architecture for replacing 4 external dependencies with internal implementations without breaking existing consumers. The key insight is that gaz's existing architecture already uses **adapter/wrapper patterns** that isolate external dependencies behind internal interfaces. This makes replacements straightforward - we modify internal wrappers, not public APIs.

## Current Dependency Map

| Package | External Dependency | Usage Location | Consumer Impact |
|---------|---------------------|----------------|-----------------|
| `worker` | `jpillora/backoff` | `worker/backoff.go`, `worker/supervisor.go` | `BackoffConfig` struct public, `NewBackoff()` returns external type |
| `cron` | `robfig/cron/v3` | `cron/scheduler.go`, `cron/logger.go` | `Scheduler` wraps external cron |
| `logger` | `lmittmann/tint` | `logger/provider.go` | Only used internally for text format |
| `health` | `alexliesenfeld/health` | `health/manager.go`, `health/handlers.go`, `health/writer.go` | Manager returns `health.Checker` (external type) |

## Recommended Architecture

### Pattern: Internal Interface + Adapter

Each replacement follows the same pattern:

```
External Dependency       Internal Implementation
       |                         |
       v                         v
   [Adapter]  <-  replaces  ->  [Direct Use]
       |                         |
       v                         v
   [Wrapper]  ----------------> [Wrapper] (unchanged API)
       |                         |
       v                         v
  [Consumer] -----------------> [Consumer] (unchanged)
```

**Key principle:** Replace at the adapter layer, keep wrapper APIs unchanged.

### Component Boundaries

```
gaz/
|-- backoff/           # NEW: standalone backoff package
|   |-- backoff.go     # BackOff interface
|   |-- exponential.go # ExponentialBackOff implementation
|   |-- options.go     # Functional options
|
|-- cron/internal/             # NEW: internal cron engine (forked from _tmp_trust/cronx)
|   |-- cron.go        # Cron scheduler core
|   |-- chain.go       # Job wrappers (Recover, SkipIfStillRunning)
|   |-- parser.go      # Schedule parsing
|   |-- logger.go      # Logger interface (uses slog, not logx)
|
|-- logger/tint/             # NEW: colored slog handler
|   |-- handler.go     # slog.Handler implementation with colors
|
|-- health/internal/           # NEW: health check core (replaces alexliesenfeld/health)
|   |-- checker.go     # Checker interface and implementation
|   |-- check.go       # Check struct and execution
|   |-- result.go      # CheckerResult, AvailabilityStatus
|   |-- handler.go     # HTTP handler creation
|
|-- worker/            # MODIFIED: use internal backoff
|   |-- backoff.go     # Config wraps backoff.ExponentialBackOff
|   |-- supervisor.go  # Uses internal backoff.BackOff
|
|-- cron/              # MODIFIED: use cron/internal
|   |-- scheduler.go   # Wraps internal.Cron instead of robfig/cron
|   |-- logger.go      # Adapter internal.Logger to slog
|
|-- logger/            # MODIFIED: use logger/tint
|   |-- provider.go    # Uses tint.Handler for text format
|
|-- health/            # MODIFIED: use health/internal
    |-- manager.go     # Returns internal.Checker
    |-- handlers.go    # Uses internal.NewHandler
    |-- writer.go      # Uses internal.CheckerResult
```

## Integration Points

### 1. worker/backoff - LOWEST RISK

**Current integration:**
```go
// worker/backoff.go
import "github.com/jpillora/backoff"

func (c *BackoffConfig) NewBackoff() *backoff.Backoff {
    return &backoff.Backoff{
        Min:    c.Min,
        Max:    c.Max,
        Factor: c.Factor,
        Jitter: c.Jitter,
    }
}

// worker/supervisor.go
type supervisor struct {
    backoff *backoff.Backoff
}
```

**Target integration:**
```go
// worker/backoff.go
import "github.com/petabytecl/gaz/backoff"

func (c *BackoffConfig) NewBackoff() backoff.BackOff {
    return &backoff.ExponentialBackOff{
        InitialInterval:     c.Min,
        MaxInterval:         c.Max,
        Multiplier:          c.Factor,
        RandomizationFactor: 0.5, // jitter equivalent
        MaxElapsedTime:      0,   // no limit
    }
}

// worker/supervisor.go
type supervisor struct {
    backoff backoff.BackOff  // interface, not concrete type
}
```

**API Change:** `NewBackoff()` return type changes from `*backoff.Backoff` to `backoff.BackOff` (interface). This is a **breaking change** for any consumer that stores the concrete type, but the interface has the same methods (`Duration()` -> `NextBackOff()`, `Reset()`).

**Migration strategy:** Change return type to interface. Document method name change (`Duration()` -> `NextBackOff()`).

### 2. cron/scheduler - MEDIUM RISK

**Current integration:**
```go
// cron/scheduler.go
import "github.com/robfig/cron/v3"

type Scheduler struct {
    cron *cron.Cron
}

func NewScheduler(...) *Scheduler {
    c := cron.New(
        cron.WithLogger(adapter),
        cron.WithChain(cron.SkipIfStillRunning(adapter)),
    )
    return &Scheduler{cron: c}
}

// cron/logger.go
func NewSlogAdapter(logger *slog.Logger) cron.Logger {
    return &slogAdapter{logger: logger}
}
```

**Target integration:**
```go
// cron/scheduler.go
import "github.com/petabytecl/gaz/cron/internal"

type Scheduler struct {
    cron *internal.Cron
}

func NewScheduler(...) *Scheduler {
    c := internal.New(
        internal.WithLogger(NewSlogAdapter(logger)),
        internal.WithChain(internal.SkipIfStillRunning(NewSlogAdapter(logger))),
    )
    return &Scheduler{cron: c}
}

// cron/logger.go - NEW INTERFACE
// cron/internal uses its own Logger interface that matches slog patterns
type CronLogger interface {
    Info(msg string, keysAndValues ...any)
    Error(err error, msg string, keysAndValues ...any)
}
```

**API Change:** Internal only. `Scheduler` public API unchanged.

**Key consideration:** The cron/internal reference uses `logx.Logger` (go-logr/logr). We must replace this with an slog-compatible interface:
```go
// In cron/internal package
type Logger interface {
    Info(msg string, keysAndValues ...any)
    Error(err error, msg string, keysAndValues ...any)
}
```

This matches the existing `cron.Logger` interface from robfig/cron, so `slogAdapter` continues to work.

### 3. logger/provider - LOWEST RISK

**Current integration:**
```go
// logger/provider.go
import "github.com/lmittmann/tint"

func NewLogger(cfg *Config) *slog.Logger {
    if cfg.Format == "text" {
        handler = tint.NewHandler(os.Stdout, &tint.Options{
            Level:      lvl,
            AddSource:  cfg.AddSource,
            TimeFormat: "15:04:05.000",
        })
    }
}
```

**Target integration:**
```go
// logger/provider.go
import "github.com/petabytecl/gaz/logger/tint"

func NewLogger(cfg *Config) *slog.Logger {
    if cfg.Format == "text" {
        handler = tint.NewHandler(os.Stdout, &tint.Options{
            Level:      lvl,
            AddSource:  cfg.AddSource,
            TimeFormat: "15:04:05.000",
        })
    }
}
```

**API Change:** None. `tint.Handler` implements `slog.Handler` just like `tint.Handler`.

**Implementation note:** logger/tint needs to implement:
- ANSI color codes for levels (ERROR=red, WARN=yellow, INFO=green, DEBUG=gray)
- Timestamp formatting
- Source file/line info
- Attribute formatting with colors
- Windows console support (optional, can skip initially)

### 4. health/manager - HIGHEST RISK

**Current integration:**
```go
// health/manager.go
import "github.com/alexliesenfeld/health"

type Manager struct {
    livenessChecks  []health.Check
    readinessChecks []health.Check
}

func (m *Manager) LivenessChecker(opts ...health.CheckerOption) health.Checker {
    return health.NewChecker(finalOpts...)
}

// health/handlers.go
func (m *Manager) NewLivenessHandler() http.Handler {
    checker := m.LivenessChecker()
    return health.NewHandler(checker,
        health.WithResultWriter(NewIETFResultWriter()),
    )
}

// health/writer.go
func (rw *IETFResultWriter) Write(
    result *health.CheckerResult,
    statusCode int,
    w http.ResponseWriter,
    _ *http.Request,
) error
```

**Target integration:**
```go
// health/manager.go
import "github.com/petabytecl/gaz/health/internal"

type Manager struct {
    livenessChecks  []internal.Check
    readinessChecks []internal.Check
}

func (m *Manager) LivenessChecker(opts ...internal.CheckerOption) internal.Checker {
    return internal.NewChecker(finalOpts...)
}

// health/handlers.go
func (m *Manager) NewLivenessHandler() http.Handler {
    checker := m.LivenessChecker()
    return internal.NewHandler(checker,
        internal.WithResultWriter(NewIETFResultWriter()),
    )
}

// health/writer.go - implements internal.ResultWriter
func (rw *IETFResultWriter) Write(
    result *internal.CheckerResult,
    statusCode int,
    w http.ResponseWriter,
    _ *http.Request,
) error
```

**API Change:** Method signatures change from `health.*` types to `internal.*` types. This is a **breaking change** for any consumer that:
1. Uses `health.CheckerOption` type directly
2. Stores `health.Checker` interface
3. Accesses `health.CheckerResult` fields

**Migration strategy:**
1. health/internal types must exactly match alexliesenfeld/health API signatures
2. Type aliases can ease migration: `type Check = internal.Check`
3. Deprecation period with both available (optional)

## New Components Needed

### 1. backoff/ package (fork from _tmp_trust/srex/backoff)

**Files to create:**
| File | Purpose | Source |
|------|---------|--------|
| `backoff/backoff.go` | `BackOff` interface, `Stop` constant | `_tmp_trust/srex/backoff/backoff.go` |
| `backoff/exponential.go` | `ExponentialBackOff` implementation | `_tmp_trust/srex/backoff/exponential.go` |
| `backoff/retry.go` | `Retry()`, `RetryWithData()` helpers | `_tmp_trust/srex/backoff/retry.go` |
| `backoff/context.go` | Context-aware backoff wrapper | `_tmp_trust/srex/backoff/context.go` |

**Modifications from source:**
- Remove `github.com/pkg/errors` dependency (use standard `errors` package)
- Keep API surface minimal (no ticker, no tries initially)

### 2. cron/internal/ package (fork from _tmp_trust/cronx)

**Files to create:**
| File | Purpose | Source |
|------|---------|--------|
| `cron/internal/cron.go` | `Cron` scheduler, `Entry`, `Schedule` | `_tmp_trust/cron/internal/cron.go` |
| `cron/internal/chain.go` | `JobWrapper`, `Recover`, `SkipIfStillRunning` | `_tmp_trust/cron/internal/chain.go` |
| `cron/internal/parser.go` | Cron expression parsing | `_tmp_trust/cron/internal/parser.go` |
| `cron/internal/spec.go` | Schedule spec types | `_tmp_trust/cron/internal/spec.go` |
| `cron/internal/option.go` | Functional options | `_tmp_trust/cron/internal/option.go` |
| `cron/internal/constantdelay.go` | `ConstantDelaySchedule` | `_tmp_trust/cron/internal/constantdelay.go` |

**Modifications from source:**
- Replace `logx.Logger` with local slog-compatible interface:
  ```go
  type Logger interface {
      Info(msg string, keysAndValues ...any)
      Error(err error, msg string, keysAndValues ...any)
  }
  ```
- Remove Azure DevOps import paths
- Keep full robfig/cron API compatibility

### 3. logger/tint/ package (new implementation)

**Files to create:**
| File | Purpose |
|------|---------|
| `logger/tint/handler.go` | `Handler` implementing `slog.Handler` |
| `logger/tint/color.go` | ANSI color constants and helpers |
| `logger/tint/options.go` | `Options` struct with `Level`, `AddSource`, `TimeFormat` |

**Implementation scope:**
- Unix terminal support (ANSI escape codes)
- Level-based coloring (ERROR=red, WARN=yellow, INFO=green, DEBUG=gray)
- Timestamp formatting
- Key=value attribute formatting
- Source file:line formatting (when `AddSource=true`)

**Out of scope (can add later):**
- Windows console API support
- Custom color schemes
- ReplaceAttr hooks

### 4. health/internal/ package (new implementation)

**Files to create:**
| File | Purpose |
|------|---------|
| `health/internal/check.go` | `Check` struct, `CheckFunc` type |
| `health/internal/checker.go` | `Checker` interface, `NewChecker()` |
| `health/internal/result.go` | `CheckerResult`, `CheckResult`, `AvailabilityStatus` |
| `health/internal/options.go` | `CheckerOption`, `WithCheck`, `WithTimeout`, etc. |
| `health/internal/handler.go` | `NewHandler()`, `ResultWriter` interface |

**API surface (must match alexliesenfeld/health):**
```go
// Types
type Check struct {
    Name    string
    Timeout time.Duration
    Check   func(context.Context) error
}

type AvailabilityStatus int
const (
    StatusUnknown AvailabilityStatus = iota
    StatusUp
    StatusDown
)

type CheckResult struct {
    Status    AvailabilityStatus
    Timestamp time.Time
    Error     error
}

type CheckerResult struct {
    Status  AvailabilityStatus
    Details map[string]CheckResult
}

// Interfaces
type Checker interface {
    Check(ctx context.Context) CheckerResult
}

type ResultWriter interface {
    Write(result *CheckerResult, statusCode int, w http.ResponseWriter, r *http.Request) error
}

// Functions
func NewChecker(opts ...CheckerOption) Checker
func NewHandler(checker Checker, opts ...HandlerOption) http.Handler

// Options
func WithCheck(check Check) CheckerOption
func WithTimeout(timeout time.Duration) CheckerOption
func WithCacheDuration(duration time.Duration) CheckerOption
func WithResultWriter(writer ResultWriter) HandlerOption
func WithStatusCodeUp(code int) HandlerOption
func WithStatusCodeDown(code int) HandlerOption
```

## Data Flow Changes

### Before (External Dependencies)

```
app.go
  |-> worker.NewManager()
        |-> supervisor.newSupervisor()
              |-> BackoffConfig.NewBackoff()
                    |-> jpillora/backoff.Backoff <- EXTERNAL

  |-> cron.NewScheduler()
        |-> robfig/cron.New() <- EXTERNAL
              |-> cron.WithLogger()
              |-> cron.WithChain()

  |-> logger.NewLogger()
        |-> lmittmann/tint.NewHandler() <- EXTERNAL

health.Module()
  |-> Manager.LivenessChecker()
        |-> alexliesenfeld/health.NewChecker() <- EXTERNAL
  |-> Manager.NewLivenessHandler()
        |-> alexliesenfeld/health.NewHandler() <- EXTERNAL
```

### After (Internal Implementations)

```
app.go
  |-> worker.NewManager()
        |-> supervisor.newSupervisor()
              |-> BackoffConfig.NewBackoff()
                    |-> backoff.ExponentialBackOff <- INTERNAL

  |-> cron.NewScheduler()
        |-> internal.New() <- INTERNAL
              |-> internal.WithLogger()
              |-> internal.WithChain()

  |-> logger.NewLogger()
        |-> tint.NewHandler() <- INTERNAL

health.Module()
  |-> Manager.LivenessChecker()
        |-> internal.NewChecker() <- INTERNAL
  |-> Manager.NewLivenessHandler()
        |-> internal.NewHandler() <- INTERNAL
```

## Suggested Build Order

Based on dependency analysis and risk assessment:

### Phase 1: backoff (Lowest Risk, No Dependencies)

**Rationale:**
- Completely standalone package
- No imports from other gaz packages
- No external dependencies in target
- Single consumer (worker/supervisor)
- Well-defined interface from reference implementation

**Steps:**
1. Create `backoff/` package (fork from `_tmp_trust/srex/backoff`)
2. Remove `github.com/pkg/errors` dependency
3. Update `worker/backoff.go` to use internal backoff
4. Update `worker/supervisor.go` to use `backoff.BackOff` interface
5. Remove `jpillora/backoff` from `go.mod`

### Phase 2: logger/tint (Low Risk, No Dependencies)

**Rationale:**
- Completely standalone package
- Only implements `slog.Handler` interface
- Single integration point (logger/provider.go)
- Limited feature scope needed

**Steps:**
1. Create `logger/tint/` package
2. Implement `slog.Handler` with color support
3. Update `logger/provider.go` to use `tint.NewHandler`
4. Remove `lmittmann/tint` from `go.mod`

### Phase 3: cron/internal (Medium Risk, Logger Interface)

**Rationale:**
- Fork is mostly complete in `_tmp_trust/cronx`
- Main work is replacing `logx.Logger` with slog-compatible interface
- Scheduler wrapper hides implementation details

**Steps:**
1. Create `cron/internal/` package (fork from `_tmp_trust/cronx`)
2. Define local `Logger` interface (matching cron.Logger)
3. Update all `logx.Logger` references
4. Remove Azure DevOps imports
5. Update `cron/scheduler.go` to use cron/internal
6. Update `cron/logger.go` adapter if needed
7. Remove `robfig/cron/v3` from `go.mod`

### Phase 4: health/internal (Highest Risk, Full API Surface)

**Rationale:**
- Must replicate full alexliesenfeld/health API
- Multiple integration points (manager, handlers, writer)
- External type exposure in public APIs
- Requires careful testing

**Steps:**
1. Create `health/internal/` package
2. Implement `Check`, `Checker`, `CheckerResult` types
3. Implement `NewChecker()` with options
4. Implement `NewHandler()` with options
5. Implement `ResultWriter` interface
6. Update `health/manager.go`
7. Update `health/handlers.go`
8. Update `health/writer.go`
9. Remove `alexliesenfeld/health` from `go.mod`

## Import Cycle Considerations

### Current Dangerous Relationships

gaz has existing import cycle constraints documented in `.planning/phases/26-*`:

```
gaz (app.go)
  |-> imports: di, config, cron, worker, health, logger, eventbus

subsystem packages (di, config, cron, worker, health)
  |-> CANNOT import: gaz (would create cycle)
  |-> CAN import: di (for module registration)
```

### New Package Placement

All new packages are **leaf packages** with no gaz dependencies:

```
backoff/    <- imports only stdlib
logger/tint/      <- imports only stdlib (log/slog)
cron/internal/      <- imports only stdlib
health/internal/    <- imports only stdlib (net/http)
```

This placement is **safe** because:
1. New packages don't import any gaz packages
2. Existing packages import new packages (not vice versa)
3. No new import cycles possible

### Package Import Graph (Post-Migration)

```
                    gaz (app.go)
                         |
         +---------------+---------------+
         |               |               |
         v               v               v
      worker          cron           health
         |               |               |
         v               v               v
      backoff         cron/internal          health/internal
         |               |               |
         v               v               v
       stdlib         stdlib         stdlib

       logger
         |
         v
       logger/tint
         |
         v
       stdlib
```

## Testing Strategy

### Unit Tests (Per Package)

| Package | Test Focus |
|---------|------------|
| `backoff/` | Exponential calculation, jitter, reset |
| `logger/tint/` | Color output, level formatting, time formatting |
| `cron/internal/` | Schedule parsing, job execution, chain wrappers |
| `health/internal/` | Check execution, timeout handling, result aggregation |

### Integration Tests (Consumer Packages)

| Package | Test Focus |
|---------|------------|
| `worker/` | Supervisor restart with backoff delays |
| `cron/` | Scheduler lifecycle, job registration |
| `logger/` | Text format output with colors |
| `health/` | HTTP handlers, IETF response format |

### Migration Validation

For each phase, verify:
- [ ] All existing tests pass
- [ ] External dependency removed from go.mod
- [ ] `go build ./...` succeeds
- [ ] No import cycles: `go list -f '{{.ImportPath}} {{.Imports}}' ./...`

## Risk Assessment

| Component | Risk | Mitigation |
|-----------|------|------------|
| backoff | LOW | Well-isolated, single consumer |
| logger/tint | LOW | Simple slog.Handler, feature parity easy |
| cron/internal | MEDIUM | Larger codebase, logger interface change |
| health/internal | HIGH | Full API replication, multiple consumers, public types |

## Sources

- **HIGH confidence:** Direct codebase analysis of:
  - `worker/backoff.go`, `worker/supervisor.go`
  - `cron/scheduler.go`, `cron/logger.go`
  - `logger/provider.go`
  - `health/manager.go`, `health/handlers.go`, `health/writer.go`
  - `_tmp_trust/srex/backoff/*`
  - `_tmp_trust/cron/internal/*`

- **HIGH confidence:** Context7 documentation for:
  - robfig/cron Logger interface
  - alexliesenfeld/health Checker API

- **HIGH confidence:** Existing gaz architecture patterns from `.planning/phases/26-*` (import cycle solutions)
