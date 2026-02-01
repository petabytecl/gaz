# Feature Landscape: Dependency Replacement

**Project:** gaz - Dependency Internalization
**Researched:** 2026-02-01

## Overview

This document analyzes the exact features gaz currently uses from four external dependencies, categorizing requirements for internal implementations into table stakes (must have), differentiators (improvements), and anti-features (things to NOT implement).

---

## 1. jpillora/backoff

### Current Usage in gaz

**Files:** `worker/backoff.go`, `worker/supervisor.go`

**Exact API surface used:**
```go
// worker/backoff.go:67-74
func (c *BackoffConfig) NewBackoff() *backoff.Backoff {
    return &backoff.Backoff{
        Min:    c.Min,    // minimum delay (time.Duration)
        Max:    c.Max,    // maximum delay cap (time.Duration)
        Factor: c.Factor, // multiplier (float64)
        Jitter: c.Jitter, // randomization (bool)
    }
}

// worker/supervisor.go:137
s.backoff.Reset()

// worker/supervisor.go:141
delay := s.backoff.Duration()
```

**Features actually used:**
| Feature | Used | How |
|---------|------|-----|
| `Backoff` struct | YES | Created via struct literal |
| `Min` field | YES | Minimum delay (1s default) |
| `Max` field | YES | Maximum cap (5m default) |
| `Factor` field | YES | Multiplier (2 default) |
| `Jitter` field | YES | Enable randomization |
| `Duration()` method | YES | Get next backoff delay |
| `Reset()` method | YES | Reset after stable run |
| `Attempt` field | NO | Internal counter, not accessed |

### Table Stakes (Must Have)

| Feature | Reason | Reference Implementation |
|---------|--------|-------------------------|
| `NewExponentialBackoff()` constructor | Standard creation pattern | `_tmp_trust/srex/backoff/exponential.go:80` |
| `Min/Max` duration bounds | Core exponential behavior | srex uses `InitialInterval`/`MaxInterval` |
| `Factor/Multiplier` | Exponential growth rate | srex uses `Multiplier` field |
| `Jitter/RandomizationFactor` | Prevent thundering herd | srex has `RandomizationFactor` (0.5 default) |
| `NextBackOff()` method | Return next delay | srex interface: `NextBackOff() time.Duration` |
| `Reset()` method | Reset to initial state | srex interface: `Reset()` |
| Thread-safe usage | supervisor uses in goroutine | Note: jpillora/backoff is NOT thread-safe either |

### Differentiators (Improvements over jpillora/backoff)

| Feature | Value | From Reference |
|---------|-------|----------------|
| `BackOff` interface | Enables strategy pattern (constant, exponential, stop) | srex `BackOff` interface |
| `Stop` sentinel value | Signal "no more retries" | `const Stop time.Duration = -1` |
| `MaxElapsedTime` | Auto-stop after total elapsed time | srex `ExponentialBackOff.MaxElapsedTime` |
| Context support via `WithContext()` | Cancellation-aware backoff | srex `WithContext()` wrapper |
| `WithMaxRetries()` wrapper | Limit total retry attempts | srex `WithMaxRetries()` decorator |
| `Clock` interface | Testable time (mock time.Now) | srex `Clock` interface |

### Anti-Features (Do NOT Implement)

| Feature | Why Avoid | What gaz Needs Instead |
|---------|-----------|------------------------|
| `Retry()` helper function | gaz supervisor handles retry loop itself | Keep retry logic in supervisor |
| `Ticker` abstraction | Adds complexity; supervisor uses simple `time.After` | Use standard time.After |
| `Timer` abstraction | Over-engineering for gaz use case | Not needed |
| Notification callbacks | gaz uses slog logging directly | Use slog in supervisor |
| Generic retry with data | Over-engineered for backoff-only use | Not needed |

### Migration Path

**Current jpillora API:**
```go
b := &backoff.Backoff{Min: 1*time.Second, Max: 5*time.Minute, Factor: 2, Jitter: true}
delay := b.Duration()
b.Reset()
```

**Internal implementation API:**
```go
b := backoff.NewExponentialBackOff(
    backoff.WithInitialInterval(1*time.Second),
    backoff.WithMaxInterval(5*time.Minute),
    backoff.WithMultiplier(2),
    backoff.WithRandomizationFactor(0.5),
)
delay := b.NextBackOff()
b.Reset()
```

---

## 2. robfig/cron/v3

### Current Usage in gaz

**Files:** `cron/scheduler.go`, `cron/logger.go`, `cron/wrapper.go`

**Exact API surface used:**
```go
// cron/scheduler.go:51-54 - Cron creation
c := cron.New(
    cron.WithLogger(adapter),
    cron.WithChain(cron.SkipIfStillRunning(adapter)),
)

// cron/scheduler.go:86
s.cron.Start()

// cron/scheduler.go:108-109 - Graceful shutdown
cronCtx := s.cron.Stop()
<-cronCtx.Done()

// cron/scheduler.go:145
_, err := s.cron.AddJob(schedule, wrapper)

// cron/logger.go:21 - Logger interface implementation
func NewSlogAdapter(logger *slog.Logger) cron.Logger {
    return &slogAdapter{...}
}

// cron.Logger interface
Info(msg string, keysAndValues ...any)
Error(err error, msg string, keysAndValues ...any)
```

**Features actually used:**
| Feature | Used | How |
|---------|------|-----|
| `cron.New()` constructor | YES | Create scheduler |
| `WithLogger()` option | YES | Inject slog adapter |
| `WithChain()` option | YES | Apply job wrappers |
| `SkipIfStillRunning()` wrapper | YES | Prevent overlapping runs |
| `Start()` method | YES | Start scheduler |
| `Stop()` method | YES | Stop + return wait context |
| `AddJob(spec, Job)` method | YES | Register jobs |
| `cron.Job` interface (`Run()`) | YES | Job implementation |
| `cron.Logger` interface | YES | Logging adapter |
| Schedule expression parsing | YES | Standard cron expressions |
| Descriptor support (@daily, etc) | YES | Via standard parser |

**Features NOT used:**
| Feature | Status | Notes |
|---------|--------|-------|
| `AddFunc()` | NOT USED | gaz uses `AddJob()` |
| `Entry`, `EntryID` | NOT USED | gaz wraps jobs differently |
| `Remove()` | NOT USED | Jobs registered once at startup |
| `Entries()` | NOT USED | gaz tracks jobs separately |
| `WithSeconds()` option | NOT USED | Standard 5-field cron |
| `WithParser()` option | NOT USED | Default parser sufficient |
| `WithLocation()` option | NOT USED | Uses local time |
| `Recover()` wrapper | NOT USED | gaz has custom panic recovery |
| `DelayIfStillRunning()` wrapper | NOT USED | Uses SkipIfStillRunning |

### Table Stakes (Must Have)

| Feature | Reason | Reference Implementation |
|---------|--------|-------------------------|
| `New()` constructor with options | Standard pattern | `_tmp_trust/cronx/cron.go:100` |
| Standard cron parser (5-field) | Parse `"*/5 * * * *"` | `_tmp_trust/cronx/parser.go` |
| Descriptor support | Parse `@daily`, `@hourly`, etc | `_tmp_trust/cronx/parser.go:362-430` |
| `Start()` method | Begin scheduling | cronx has `Start()` |
| `Stop()` returning wait context | Graceful shutdown | cronx returns `context.Context` |
| `AddJob(spec, Job)` | Register jobs with schedule | cronx has `AddJob()` |
| `Job` interface (`Run()`) | Job abstraction | cronx `Job` interface |
| `Logger` interface (Info/Error) | Structured logging | cronx uses `logx.Logger` |
| `WithLogger()` option | Inject logger | cronx `WithLogger()` |
| `WithChain()` option | Apply job wrappers | cronx `WithChain()` |
| `SkipIfStillRunning()` wrapper | Prevent overlapping | cronx `SkipIfStillRunning()` |

### Differentiators (Improvements over robfig/cron)

| Feature | Value | Notes |
|---------|-------|-------|
| slog.Logger native support | Use `slog.Logger` directly, not adapter | Simpler integration |
| Timezone in spec (`CRON_TZ=`) | Already in cronx parser | More flexible scheduling |
| `@every` descriptor | Interval scheduling | Already in cronx |
| Clean interface boundaries | cronx has cleaner `ScheduleParser` interface | Better testability |

### Anti-Features (Do NOT Implement)

| Feature | Why Avoid | What gaz Needs Instead |
|---------|-----------|------------------------|
| `Entry`/`EntryID` tracking | gaz tracks jobs in `diJobWrapper` array | Use existing wrapper tracking |
| `Entries()` introspection | Adds complexity, not used | Not needed |
| `Remove()` dynamic removal | Jobs are static at startup | Not needed |
| `Schedule()` direct method | `AddJob()` is sufficient | Not needed |
| `Run()` blocking mode | gaz uses `Start()` non-blocking | Not needed |
| `FuncJob` adapter | gaz uses `Job` interface directly | Not needed |
| `SecondOptional` parser option | Over-engineering | Standard 5-field sufficient |
| `Recover()` wrapper | gaz has custom recovery in `diJobWrapper` | Keep custom recovery |
| `DelayIfStillRunning()` wrapper | Skip is preferred over delay | Not needed |

### Migration Path

**Current robfig/cron API:**
```go
c := cron.New(
    cron.WithLogger(adapter),
    cron.WithChain(cron.SkipIfStillRunning(adapter)),
)
c.AddJob("*/5 * * * *", job)
c.Start()
ctx := c.Stop()
<-ctx.Done()
```

**Internal implementation API:**
```go
c := cron.New(
    cron.WithLogger(logger),  // Direct slog.Logger
    cron.WithChain(cron.SkipIfStillRunning(logger)),
)
c.AddJob("*/5 * * * *", job)
c.Start()
ctx := c.Stop()
<-ctx.Done()
```

---

## 3. lmittmann/tint

### Current Usage in gaz

**Files:** `logger/provider.go`

**Exact API surface used:**
```go
// logger/provider.go:22-26
handler = tint.NewHandler(os.Stdout, &tint.Options{
    Level:      lvl,           // *slog.LevelVar
    AddSource:  cfg.AddSource, // bool
    TimeFormat: "15:04:05.000", // string
})
```

**Features actually used:**
| Feature | Used | How |
|---------|------|-----|
| `tint.NewHandler()` | YES | Create colored handler |
| `tint.Options.Level` | YES | Dynamic log level |
| `tint.Options.AddSource` | YES | Include source location |
| `tint.Options.TimeFormat` | YES | Custom time format |

**Features NOT used:**
| Feature | Status | Notes |
|---------|--------|-------|
| `Options.ReplaceAttr` | NOT USED | No attribute transformation |
| `Options.NoColor` | NOT USED | Always colored when text format |
| `tint.Attr()` for colored attrs | NOT USED | Standard attributes only |
| `tint.Err()` helper | NOT USED | Uses `slog.Any("error", err)` |

### Table Stakes (Must Have)

| Feature | Reason | Implementation Notes |
|---------|--------|---------------------|
| `NewHandler(w, opts)` constructor | Standard slog.Handler pattern | Return `slog.Handler` |
| ANSI color output | Visual distinction of levels | Level-based colors |
| `Options.Level` (Leveler) | Filter by log level | Use `slog.Leveler` |
| `Options.AddSource` (bool) | Include file:line | Standard slog option |
| `Options.TimeFormat` (string) | Custom timestamp format | Use time.Format layout |
| Implement `slog.Handler` interface | Drop-in replacement | `Enabled`, `Handle`, `WithAttrs`, `WithGroup` |
| Structured attribute output | Key=value pairs | Match tint format |

### Differentiators (Improvements over tint)

| Feature | Value | Notes |
|---------|-------|-------|
| Level-aware coloring | Different colors per level | DEBUG=blue, INFO=green, WARN=yellow, ERROR=red |
| Attribute highlighting | Highlight error attributes in red | Better visibility |
| `Options.NoColor` | Disable colors (for CI/logs) | Useful for non-TTY output |
| Terminal detection | Auto-detect TTY | Use `isatty` or `term` package |

### Anti-Features (Do NOT Implement)

| Feature | Why Avoid | What gaz Needs Instead |
|---------|-----------|------------------------|
| `ReplaceAttr` callback | Adds complexity, not used | Not needed for gaz |
| `tint.Attr()` color customization | Over-engineering | Standard coloring sufficient |
| Windows colorable support | gaz targets Linux | Use standard ANSI codes |
| Custom level names (`TRC`) | Not used in gaz | Standard level names |

### Migration Path

**Current tint API:**
```go
handler = tint.NewHandler(os.Stdout, &tint.Options{
    Level:      lvl,
    AddSource:  cfg.AddSource,
    TimeFormat: "15:04:05.000",
})
```

**Internal implementation API:**
```go
handler = color.NewHandler(os.Stdout, &color.Options{
    Level:      lvl,
    AddSource:  cfg.AddSource,
    TimeFormat: "15:04:05.000",
    NoColor:    !isTerminal,  // Optional enhancement
})
```

---

## 4. alexliesenfeld/health

### Current Usage in gaz

**Files:** `health/manager.go`, `health/handlers.go`, `health/writer.go`, `health/types.go`

**Exact API surface used:**
```go
// health/manager.go - Check struct
health.Check{
    Name:  name,
    Check: check,  // func(context.Context) error
}

// health/manager.go:61, 76, 91 - Build checkers
health.WithCheck(c)
health.NewChecker(finalOpts...)

// health/handlers.go:16-20 - Create HTTP handlers
health.NewHandler(checker,
    health.WithResultWriter(NewIETFResultWriter()),
    health.WithStatusCodeUp(http.StatusOK),
    health.WithStatusCodeDown(http.StatusOK),  // or 503
)

// health/writer.go - ResultWriter interface
type IETFResultWriter struct{}
func (rw *IETFResultWriter) Write(
    result *health.CheckerResult,
    statusCode int,
    w http.ResponseWriter,
    _ *http.Request,
) error

// health/writer.go - CheckerResult access
result.Status                           // health.AvailabilityStatus
result.Details                          // map[string]*CheckResult
checkResult.Status                      // health.AvailabilityStatus
checkResult.Timestamp                   // time.Time
checkResult.Error                       // error

// health/writer.go - AvailabilityStatus constants
health.StatusUp
health.StatusDown
health.StatusUnknown
```

**Features actually used:**
| Feature | Used | How |
|---------|------|-----|
| `health.Check` struct | YES | Check configuration |
| `health.Check.Name` | YES | Check identifier |
| `health.Check.Check` | YES | Check function |
| `health.NewChecker()` | YES | Create checker |
| `health.WithCheck()` | YES | Add synchronous check |
| `health.NewHandler()` | YES | Create HTTP handler |
| `health.WithResultWriter()` | YES | Custom JSON format (IETF) |
| `health.WithStatusCodeUp()` | YES | 200 on success |
| `health.WithStatusCodeDown()` | YES | 200 or 503 on failure |
| `health.CheckerResult` | YES | Result structure |
| `health.AvailabilityStatus` | YES | Status enum |
| `health.StatusUp/Down/Unknown` | YES | Status constants |

**Features NOT used:**
| Feature | Status | Notes |
|---------|--------|-------|
| `WithPeriodicCheck()` | NOT USED | All checks are synchronous |
| `WithTimeout()` global | NOT USED | Per-check timeout via context |
| `WithCacheDuration()` | NOT USED | Fresh checks each request |
| `WithStatusListener()` | NOT USED | No status change callbacks |
| `WithDisabledAutostart()` | NOT USED | Auto-start is fine |
| `Check.Timeout` field | NOT USED | Uses context deadline |
| `checker.Start()/Stop()` | NOT USED | Automatic lifecycle |
| `checker.IsStarted()` | NOT USED | No lifecycle introspection |

### Table Stakes (Must Have)

| Feature | Reason | Implementation Notes |
|---------|--------|---------------------|
| `Check` struct with Name/Check | Check configuration | Keep same structure |
| `NewChecker(opts...)` | Create checker instance | Functional options |
| `WithCheck()` option | Add synchronous check | Executed per request |
| `Checker` interface | Abstract checker | For testability |
| `NewHandler(checker, opts...)` | Create HTTP handler | Return `http.Handler` |
| `WithResultWriter()` option | Custom response format | For IETF format |
| `WithStatusCodeUp()` option | HTTP status on success | Default 200 |
| `WithStatusCodeDown()` option | HTTP status on failure | Default 503 |
| `ResultWriter` interface | Custom response format | `Write(result, status, w, r)` |
| `CheckerResult` struct | Aggregated results | Status + Details map |
| `CheckResult` per check | Individual result | Status + Timestamp + Error |
| `AvailabilityStatus` enum | Up/Down/Unknown | For status representation |

### Differentiators (Improvements over alexliesenfeld/health)

| Feature | Value | Notes |
|---------|-------|-------|
| Simpler API | Remove unused features | No periodic checks, no caching |
| Built-in IETF writer | Include IETF format by default | Currently custom in gaz |
| slog integration | Log check failures | Structured logging |
| Check metadata | Additional context fields | For richer responses |

### Anti-Features (Do NOT Implement)

| Feature | Why Avoid | What gaz Needs Instead |
|---------|-----------|------------------------|
| `WithPeriodicCheck()` | Not used, adds goroutine complexity | Synchronous checks only |
| `WithCacheDuration()` | Not used, fresh checks needed | Real-time checks |
| `WithTimeout()` global | Per-check timeout via context | Context deadlines |
| `WithStatusListener()` | Not used | Not needed |
| `WithDisabledAutostart()` | Over-engineering | Not needed |
| `WithInterceptors()` | Not used | Not needed |
| `checker.Start()/Stop()` lifecycle | Synchronous checks don't need this | Automatic |
| `checker.Check()` manual trigger | Not used | Via HTTP handler |
| `GetRunningPeriodicCheckCount()` | Not applicable | Not needed |

### Migration Path

**Current alexliesenfeld/health API:**
```go
// Manager
m.livenessChecks = append(m.livenessChecks, health.Check{
    Name:  name,
    Check: check,
})

checker := health.NewChecker(
    health.WithCheck(check1),
    health.WithCheck(check2),
)

handler := health.NewHandler(checker,
    health.WithResultWriter(NewIETFResultWriter()),
    health.WithStatusCodeUp(http.StatusOK),
    health.WithStatusCodeDown(http.StatusServiceUnavailable),
)
```

**Internal implementation API:**
```go
// Manager (same structure)
m.livenessChecks = append(m.livenessChecks, health.Check{
    Name:  name,
    Check: check,
})

checker := health.NewChecker(
    health.WithCheck(check1),
    health.WithCheck(check2),
)

handler := health.NewHandler(checker,
    health.WithResultWriter(health.IETFWriter()),  // Built-in
    health.WithStatusCodeUp(http.StatusOK),
    health.WithStatusCodeDown(http.StatusServiceUnavailable),
)
```

---

## Feature Dependencies

### Dependency Graph

```
backoff (internal)
    └── used by: worker/supervisor.go

cron (internal)
    └── used by: cron/scheduler.go

color (tint replacement)
    └── used by: logger/provider.go

health (internal)
    └── used by: health/manager.go, health/handlers.go
```

### Cross-Package Dependencies

- **backoff**: No dependencies on other gaz packages
- **cron**: Uses slog for logging (already in gaz)
- **color**: Implements slog.Handler (standard library)
- **health**: No dependencies on other gaz packages

---

## Implementation Priority

Based on complexity and value:

| Priority | Package | Complexity | Reason |
|----------|---------|------------|--------|
| 1 | color (tint) | LOW | Simple handler, minimal API |
| 2 | backoff | LOW-MED | Reference impl available, small API |
| 3 | health | MEDIUM | Larger API but well-defined |
| 4 | cron | MEDIUM-HIGH | Parser complexity, scheduler logic |

---

## Summary

### Total Features to Implement

| Package | Table Stakes | Differentiators | Anti-Features |
|---------|--------------|-----------------|---------------|
| backoff | 7 | 6 | 5 |
| cron | 11 | 4 | 9 |
| color (tint) | 7 | 4 | 4 |
| health | 12 | 4 | 9 |
| **Total** | **37** | **18** | **27** |

### Key Takeaways

1. **backoff**: Reference implementation in `_tmp_trust/srex/backoff/` covers table stakes; strip retry/ticker complexity
2. **cron**: Reference implementation in `_tmp_trust/cronx/` is comprehensive; adapt logger interface to slog
3. **color (tint)**: Simple implementation, focus on ANSI color output with slog.Handler interface
4. **health**: Most API surface used; simplify by removing periodic checks and caching
