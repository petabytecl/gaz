# Requirements: gaz v4.0 Dependency Reduction

**Defined:** 2026-02-01
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v4.0 Requirements

Requirements for v4.0 milestone. Each maps to roadmap phases.

### Backoff Package

- [x] **BKF-01**: `backoff/` package exists with `BackOff` interface defining `NextBackOff()` and `Reset()` methods
- [x] **BKF-02**: `ExponentialBackOff` implementation with configurable InitialInterval, MaxInterval, Multiplier, RandomizationFactor
- [x] **BKF-03**: Overflow protection clamps result to MaxInterval when calculation exceeds int64
- [x] **BKF-04**: Jitter is thread-safe (uses math/rand/v2 or per-instance mutex)
- [x] **BKF-05**: `Stop` sentinel constant signals "no more retries"
- [x] **BKF-06**: `WithContext()` wrapper for cancellation-aware backoff
- [x] **BKF-07**: `worker/backoff.go` uses internal `backoff/` package instead of jpillora/backoff
- [x] **BKF-08**: `jpillora/backoff` removed from go.mod

### Cron Package

- [x] **CRN-01**: `cron/internal/` package exists with `Cron` scheduler type
- [x] **CRN-02**: Standard 5-field cron expression parser (minute, hour, dom, month, dow)
- [x] **CRN-03**: Descriptor support (@daily, @hourly, @weekly, @monthly, @yearly, @annually, @every)
- [x] **CRN-04**: `Start()` method begins scheduling, `Stop()` returns context for graceful wait
- [x] **CRN-05**: `AddJob(spec, Job)` registers jobs with cron schedules
- [x] **CRN-06**: `SkipIfStillRunning` job wrapper prevents overlapping executions
- [x] **CRN-07**: `Logger` interface compatible with slog patterns (Info, Error methods)
- [x] **CRN-08**: `WithLogger()` and `WithChain()` functional options
- [x] **CRN-09**: CRON_TZ prefix support for timezone-specific schedules
- [x] **CRN-10**: DST transitions handled correctly (spring forward skips, fall back runs once)
- [x] **CRN-11**: `cron/scheduler.go` uses internal `cron/internal/` package instead of robfig/cron/v3
- [x] **CRN-12**: `robfig/cron/v3` removed from go.mod

### Tint Package

- [x] **TNT-01**: `logger/tint/` package exists with `Handler` implementing `slog.Handler`
- [x] **TNT-02**: ANSI color output for log levels (DEBUG=blue, INFO=green, WARN=yellow, ERROR=red)
- [x] **TNT-03**: `Options.Level` filters logs by level (uses `slog.Leveler`)
- [x] **TNT-04**: `Options.AddSource` includes file:line in output
- [x] **TNT-05**: `Options.TimeFormat` customizes timestamp format
- [x] **TNT-06**: `WithAttrs()` returns new handler instance (not self) preserving attributes
- [x] **TNT-07**: `WithGroup()` returns new handler instance with group prefix
- [x] **TNT-08**: TTY detection auto-disables colors for non-terminal output
- [x] **TNT-09**: `Options.NoColor` explicitly disables color output
- [x] **TNT-10**: `logger/provider.go` uses internal `logger/tint/` package instead of lmittmann/tint
- [x] **TNT-11**: `lmittmann/tint` removed from go.mod

### Health Package

- [x] **HLT-01**: `health/internal/` package exists with `Check` struct (Name, Check func)
- [x] **HLT-02**: `NewChecker(opts...)` creates checker instance
- [x] **HLT-03**: `WithCheck(check)` option adds synchronous health check
- [x] **HLT-04**: `NewHandler(checker, opts...)` creates HTTP handler
- [x] **HLT-05**: `WithResultWriter(writer)` option for custom response format
- [x] **HLT-06**: `WithStatusCodeUp(code)` and `WithStatusCodeDown(code)` options
- [x] **HLT-07**: `CheckerResult` struct with Status and Details map
- [x] **HLT-08**: `AvailabilityStatus` enum (StatusUnknown, StatusUp, StatusDown)
- [x] **HLT-09**: `ResultWriter` interface for custom response formatting
- [x] **HLT-10**: Liveness handler returns 200 even on check failure (matches current behavior)
- [x] **HLT-11**: IETF health+json response format (built-in writer)
- [x] **HLT-12**: `health/manager.go` uses internal `health/internal/` package instead of alexliesenfeld/health
- [x] **HLT-13**: `alexliesenfeld/health` removed from go.mod

### Integration

- [x] **INT-01**: All existing tests pass after migration
- [x] **INT-02**: Test coverage maintained at 90%+ after migration
- [x] **INT-03**: No import cycles introduced by new packages

## Future Requirements

Deferred to v4.1 or later.

### Backoff

- **BKF-09**: `WithMaxRetries(n)` wrapper limits total attempts
- **BKF-10**: `Retry()` and `RetryWithData()` helper functions

### Cron

- **CRN-13**: `Entry`/`EntryID` tracking for job management
- **CRN-14**: `Remove(id)` for dynamic job removal
- **CRN-15**: `Entries()` introspection

### Tint

- **TNT-12**: `ReplaceAttr` callback for attribute transformation
- **TNT-13**: Custom color themes
- **TNT-14**: Windows console API support

### Health

- **HLT-14**: `WithPeriodicCheck()` for background health polling
- **HLT-15**: `WithCacheDuration()` for result caching
- **HLT-16**: `WithTimeout()` global check timeout

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Ticker abstraction in backoff | Adds complexity, supervisor uses time.After |
| 6-field cron (seconds) | Standard 5-field sufficient for gaz |
| Cron dynamic job removal | Jobs are static at startup |
| Windows terminal colors | gaz targets Linux environments |
| Periodic health checks | Synchronous checks sufficient |
| Health result caching | Fresh checks needed per request |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

### Phase 32: Backoff Package

| Requirement | Phase | Status |
|-------------|-------|--------|
| BKF-01 | 32 | Complete |
| BKF-02 | 32 | Complete |
| BKF-03 | 32 | Complete |
| BKF-04 | 32 | Complete |
| BKF-05 | 32 | Complete |
| BKF-06 | 32 | Complete |
| BKF-07 | 32 | Complete |
| BKF-08 | 32 | Complete |

### Phase 33: Tint Package

| Requirement | Phase | Status |
|-------------|-------|--------|
| TNT-01 | 33 | Complete |
| TNT-02 | 33 | Complete |
| TNT-03 | 33 | Complete |
| TNT-04 | 33 | Complete |
| TNT-05 | 33 | Complete |
| TNT-06 | 33 | Complete |
| TNT-07 | 33 | Complete |
| TNT-08 | 33 | Complete |
| TNT-09 | 33 | Complete |
| TNT-10 | 33 | Complete |
| TNT-11 | 33 | Complete |

### Phase 34: Cron Package

| Requirement | Phase | Status |
|-------------|-------|--------|
| CRN-01 | 34 | Complete |
| CRN-02 | 34 | Complete |
| CRN-03 | 34 | Complete |
| CRN-04 | 34 | Complete |
| CRN-05 | 34 | Complete |
| CRN-06 | 34 | Complete |
| CRN-07 | 34 | Complete |
| CRN-08 | 34 | Complete |
| CRN-09 | 34 | Complete |
| CRN-10 | 34 | Complete |
| CRN-11 | 34 | Complete |
| CRN-12 | 34 | Complete |

### Phase 35: Health Package + Integration

| Requirement | Phase | Status |
|-------------|-------|--------|
| HLT-01 | 35 | Complete |
| HLT-02 | 35 | Complete |
| HLT-03 | 35 | Complete |
| HLT-04 | 35 | Complete |
| HLT-05 | 35 | Complete |
| HLT-06 | 35 | Complete |
| HLT-07 | 35 | Complete |
| HLT-08 | 35 | Complete |
| HLT-09 | 35 | Complete |
| HLT-10 | 35 | Complete |
| HLT-11 | 35 | Complete |
| HLT-12 | 35 | Complete |
| HLT-13 | 35 | Complete |
| INT-01 | 35 | Complete |
| INT-02 | 35 | Complete |
| INT-03 | 35 | Complete |

**Coverage:**
- v4.0 requirements: 47 total
- Mapped to phases: 47 ✓
- Complete: 47 ✓
- Unmapped: 0

---
*Requirements defined: 2026-02-01*
*Last updated: 2026-02-02 — v4.0 milestone complete*
