# Phase 15: Cron - Research

**Researched:** 2026-01-29
**Domain:** Scheduled task execution with robfig/cron v3, DI-aware jobs, lifecycle integration
**Confidence:** HIGH

## Summary

This phase adds scheduled task support to gaz by wrapping `robfig/cron/v3` with DI-aware job resolution and lifecycle integration. The research focused on robfig/cron v3 API (graceful shutdown, job wrappers, logger interface), patterns for wrapping jobs with dependency injection, and integrating with gaz's existing worker and lifecycle systems.

robfig/cron v3 provides all required primitives: `Stop()` returns a context that completes when running jobs finish (graceful shutdown), `WithChain()` enables composable job wrappers including `Recover()` for panic recovery and `SkipIfStillRunning()` for overlap prevention, and a `Logger` interface that can adapt to `log/slog`. The library supports both standard 5-field cron expressions and predefined schedules (`@hourly`, `@daily`, etc.).

The key design challenge is wrapping gaz's `CronJob` interface (with `Run(ctx) error`) into robfig/cron's simple `Job` interface (`Run()`) while providing transient DI resolution per execution, context with timeout support, and structured logging. The Scheduler should implement the existing `worker.Worker` interface for auto-discovery and lifecycle integration.

**Primary recommendation:** Create a `Scheduler` type that wraps robfig/cron with a custom `diJobWrapper` that resolves a fresh job instance from the container for each execution, manages context/timeout, and logs execution details via slog.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/robfig/cron/v3 | v3.0.1 | Cron scheduler | De-facto standard Go cron library; mature, well-tested, supports all cron features |
| Go stdlib `context` | 1.25+ | Timeout and cancellation | Standard pattern for goroutine lifecycle and cancellation |
| Go stdlib `log/slog` | 1.25+ | Structured logging | Already used by gaz; adapts cleanly to cron.Logger interface |
| Go stdlib `runtime/debug` | 1.25+ | Stack trace capture | For custom panic recovery with stack trace logging |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `sync` | 1.25+ | Mutex for job status tracking | Health check implementation |
| Go stdlib `time` | 1.25+ | Duration and timeout handling | Job timeout management |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| robfig/cron | go-co-op/gocron | gocron is newer but robfig/cron is industry standard, more widely used |
| robfig/cron | custom scheduler | robfig/cron handles edge cases (DST, leap seconds); not worth hand-rolling |
| cron.Recover() | custom panic recovery | Custom recovery allows slog integration and stack traces; use custom wrapper |

**Installation:**
```bash
go get github.com/robfig/cron/v3@v3.0.1
```

## Architecture Patterns

### Recommended Project Structure
```
gaz/
├── cron/
│   ├── doc.go            # Package documentation
│   ├── job.go            # CronJob interface definition
│   ├── scheduler.go      # Scheduler wrapping robfig/cron
│   ├── wrapper.go        # diJobWrapper for DI-aware job execution
│   ├── logger.go         # slog adapter for cron.Logger
│   └── scheduler_test.go # Scheduler tests
```

### Pattern 1: CronJob Interface (LOCKED)
**What:** Interface defining scheduled jobs with name, schedule, timeout, and context-aware execution
**When to use:** All scheduled tasks in gaz applications
**Example:**
```go
// Source: CONTEXT.md locked decision
type CronJob interface {
    // Name returns a human-readable identifier for logging
    Name() string

    // Schedule returns the cron expression or predefined schedule
    // Return empty string to disable this job
    Schedule() string

    // Timeout returns the execution timeout (0 for none)
    Timeout() time.Duration

    // Run executes the job with the given context
    // Context is cancelled on app shutdown or timeout
    Run(ctx context.Context) error
}
```

### Pattern 2: DI-Aware Job Wrapper
**What:** Wrapper that implements cron.Job, resolves fresh instance from container per execution
**When to use:** Bridge between gaz's CronJob and robfig/cron's Job interface
**Example:**
```go
// Source: Research synthesis - DI wrapper pattern
type diJobWrapper struct {
    container   *di.Container
    jobTypeName string          // Type name for resolving from container
    jobName     string          // Human-readable name for logging
    schedule    string          // Schedule expression for reference
    timeout     time.Duration   // Execution timeout
    appCtx      context.Context // Parent context (cancelled on shutdown)
    logger      *slog.Logger

    mu       sync.Mutex
    running  bool              // For health check status
    lastRun  time.Time
    lastErr  error
}

// Run implements cron.Job - called by robfig/cron scheduler
func (w *diJobWrapper) Run() {
    w.mu.Lock()
    w.running = true
    w.mu.Unlock()
    defer func() {
        w.mu.Lock()
        w.running = false
        w.mu.Unlock()
    }()

    // Resolve fresh instance from container (transient per execution)
    instance, err := w.container.ResolveByName(w.jobTypeName, nil)
    if err != nil {
        w.logger.Error("failed to resolve job", 
            "job", w.jobName, "error", err)
        return
    }

    job, ok := instance.(CronJob)
    if !ok {
        w.logger.Error("resolved instance is not CronJob",
            "job", w.jobName, "type", fmt.Sprintf("%T", instance))
        return
    }

    // Create context with timeout
    ctx := w.appCtx
    if w.timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(w.appCtx, w.timeout)
        defer cancel()
    }

    // Execute with logging
    start := time.Now()
    w.logger.Info("job started", "job", w.jobName)

    err = job.Run(ctx)
    elapsed := time.Since(start)

    w.mu.Lock()
    w.lastRun = time.Now()
    w.lastErr = err
    w.mu.Unlock()

    if err != nil {
        w.logger.Error("job failed",
            "job", w.jobName,
            "duration", elapsed,
            "error", err)
    } else {
        w.logger.Info("job finished",
            "job", w.jobName,
            "duration", elapsed)
    }
}
```

### Pattern 3: Scheduler as Worker
**What:** Scheduler implements worker.Worker for auto-discovery and lifecycle integration
**When to use:** Integrate cron scheduler with gaz lifecycle
**Example:**
```go
// Source: gaz worker pattern + robfig/cron
type Scheduler struct {
    cron      *cron.Cron
    logger    *slog.Logger
    container *di.Container
    appCtx    context.Context

    mu       sync.Mutex
    jobs     []*diJobWrapper
    running  bool
    stopCtx  context.Context // From cron.Stop()
}

// Implements worker.Worker
func (s *Scheduler) Name() string { return "cron.Scheduler" }

func (s *Scheduler) Start() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.running {
        return
    }
    s.running = true
    
    s.logger.Info("starting cron scheduler",
        "jobs", len(s.jobs))
    s.cron.Start()
}

func (s *Scheduler) Stop() {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if !s.running {
        return
    }
    s.running = false

    s.logger.Info("stopping cron scheduler, waiting for running jobs")
    
    // Stop() returns context that completes when running jobs finish
    ctx := s.cron.Stop()
    <-ctx.Done()
    
    s.logger.Info("cron scheduler stopped")
}
```

### Pattern 4: slog Logger Adapter
**What:** Adapter implementing cron.Logger interface with slog backend
**When to use:** Bridge robfig/cron logging to gaz's slog infrastructure
**Example:**
```go
// Source: robfig/cron Context7 docs + slog patterns
type slogAdapter struct {
    logger *slog.Logger
}

func (a *slogAdapter) Info(msg string, keysAndValues ...interface{}) {
    a.logger.Info(msg, keysAndValuesToSlog(keysAndValues)...)
}

func (a *slogAdapter) Error(err error, msg string, keysAndValues ...interface{}) {
    attrs := keysAndValuesToSlog(keysAndValues)
    attrs = append(attrs, slog.Any("error", err))
    a.logger.Error(msg, attrs...)
}

func keysAndValuesToSlog(kvs []interface{}) []any {
    var attrs []any
    for i := 0; i < len(kvs)-1; i += 2 {
        key, ok := kvs[i].(string)
        if !ok {
            continue
        }
        attrs = append(attrs, slog.Any(key, kvs[i+1]))
    }
    return attrs
}
```

### Pattern 5: Custom Panic Recovery with Stack Trace
**What:** Custom recovery wrapper that logs stack traces via slog instead of using cron.Recover()
**When to use:** Meet CONTEXT.md requirement for stack trace logging
**Example:**
```go
// Source: gaz worker/supervisor.go pattern adapted for cron
func (w *diJobWrapper) runWithRecovery() {
    defer func() {
        if r := recover(); r != nil {
            stack := debug.Stack()
            w.logger.Error("job panicked",
                "job", w.jobName,
                "panic", r,
                "stack", string(stack))
        }
    }()
    w.executeJob()
}
```

### Anti-Patterns to Avoid
- **Caching job instances:** Per CONTEXT.md, each execution must resolve a fresh instance (transient lifecycle)
- **Ignoring context cancellation:** Jobs must check `ctx.Done()` and exit promptly on shutdown
- **Using cron.Recover() directly:** It uses Printf logging; wrap with custom recovery for slog + stack traces
- **Blocking Stop():** Scheduler.Stop() should wait for jobs but not block indefinitely
- **Registering jobs after Start():** All jobs should be discovered and registered during Build()

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cron expression parsing | Custom parser | robfig/cron parser | Edge cases: DST, month lengths, leap years |
| Skip overlapping runs | Manual mutex per job | cron.SkipIfStillRunning | Built-in, tested, handles edge cases |
| Graceful shutdown waiting | Manual WaitGroup | cron.Stop() context | Returns context that completes when jobs finish |
| Schedule validation | Regex matching | cron.Parse() at registration | Returns errors for invalid expressions |
| Predefined schedules | String constants | @hourly, @daily, etc. | Built into robfig/cron parser |

**Key insight:** robfig/cron handles all scheduling complexity. Our code focuses only on DI integration, context/timeout wrapping, and slog logging.

## Common Pitfalls

### Pitfall 1: Job Resolution Failure During Execution
**What goes wrong:** Container returns error when resolving job type during scheduled execution
**Why it happens:** Job type not registered or dependency graph broken
**How to avoid:** Validate job resolution during discoverCronJobs() at Build() time; log warning if Schedule() is empty
**Warning signs:** Error logs with "failed to resolve job" during runtime

### Pitfall 2: Context Cancellation Not Respected
**What goes wrong:** Job continues running after shutdown signal, delaying graceful shutdown
**Why it happens:** Job doesn't check ctx.Done() in long-running operations
**How to avoid:** Document contract that jobs MUST respect context; pass ctx to all blocking calls
**Warning signs:** Shutdown takes longer than expected; jobs log "completed" after Stop()

### Pitfall 3: Panic in Job Crashes App
**What goes wrong:** Unrecovered panic in job propagates and crashes application
**Why it happens:** Using bare cron.Job without recovery wrapper
**How to avoid:** Always wrap job execution with defer/recover; use diJobWrapper's recovery
**Warning signs:** App crashes with panic stack trace from job code

### Pitfall 4: Memory Leak from Job Instance Retention
**What goes wrong:** Job instances accumulate in memory over time
**Why it happens:** Holding references to resolved job instances after execution
**How to avoid:** Resolve job in Run(), use it, let it be GC'd; no caching
**Warning signs:** Memory growth over time correlating with job execution frequency

### Pitfall 5: Duplicate Job Registration
**What goes wrong:** Same job runs twice per schedule
**Why it happens:** Job registered both via CronJob interface and direct AddJob call
**How to avoid:** Single discovery path via interface check; no manual registration API
**Warning signs:** Job logs show double execution at each schedule trigger

### Pitfall 6: Health Check Race Condition
**What goes wrong:** Health check returns stale job status
**Why it happens:** Reading job status without mutex protection
**How to avoid:** Use mutex around running/lastRun/lastErr fields in diJobWrapper
**Warning signs:** Flaky health check results; test race detector failures

## Code Examples

Verified patterns from official sources:

### Creating Scheduler with Wrappers
```go
// Source: robfig/cron Context7 docs - combined wrappers
import "github.com/robfig/cron/v3"

func NewScheduler(logger *slog.Logger, container *di.Container) *Scheduler {
    cronLogger := &slogAdapter{logger: logger.With("component", "cron")}
    
    c := cron.New(
        cron.WithLogger(cronLogger),
        cron.WithChain(
            // Note: We use custom panic recovery instead of cron.Recover()
            // to get stack traces via slog
            cron.SkipIfStillRunning(cronLogger),
        ),
    )
    
    return &Scheduler{
        cron:      c,
        logger:    logger.With("component", "cron.Scheduler"),
        container: container,
    }
}
```

### Registering Jobs with Schedule Validation
```go
// Source: robfig/cron Context7 docs - AddJob with validation
func (s *Scheduler) RegisterJob(wrapper *diJobWrapper) error {
    if wrapper.schedule == "" {
        s.logger.Info("job schedule disabled",
            "job", wrapper.jobName)
        return nil // Not an error, job opted out
    }
    
    // Wrap with custom panic recovery
    recoveryJob := &recoveryWrapper{
        delegate: wrapper,
        logger:   s.logger,
    }
    
    // AddJob returns entry ID and error
    _, err := s.cron.AddJob(wrapper.schedule, recoveryJob)
    if err != nil {
        return fmt.Errorf("invalid schedule for job %s: %w", 
            wrapper.jobName, err)
    }
    
    s.jobs = append(s.jobs, wrapper)
    s.logger.Info("job registered",
        "job", wrapper.jobName,
        "schedule", wrapper.schedule)
    return nil
}
```

### Graceful Shutdown with Wait
```go
// Source: robfig/cron Context7 docs - graceful shutdown
func (s *Scheduler) Stop() {
    s.logger.Info("stopping cron scheduler")
    
    // Stop() returns a context that is cancelled when all running jobs complete
    ctx := s.cron.Stop()
    
    // Wait for running jobs to finish
    <-ctx.Done()
    
    s.logger.Info("cron scheduler stopped, all jobs finished")
}
```

### Health Check Implementation
```go
// Source: gaz health patterns + cron status tracking
func (s *Scheduler) HealthCheck(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if !s.running {
        return errors.New("scheduler not running")
    }
    
    // Check for stuck jobs (running longer than threshold)
    for _, job := range s.jobs {
        job.mu.Lock()
        isRunning := job.running
        lastErr := job.lastErr
        job.mu.Unlock()
        
        if isRunning {
            // Job is running - not necessarily unhealthy
            continue
        }
        if lastErr != nil {
            // Last run failed - could report degraded
            // For v1, just return nil (last error is logged)
        }
    }
    
    return nil
}
```

### Discovery During Build
```go
// Source: gaz app.go discoverWorkers pattern adapted for cron
func (a *App) discoverCronJobs() error {
    a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
        if svc.IsTransient() {
            return // Expected - cron jobs should be transient
        }
        
        // Try to resolve and check for CronJob interface
        instance, err := a.container.ResolveByName(name, nil)
        if err != nil {
            return
        }
        
        if job, ok := instance.(cron.CronJob); ok {
            wrapper := cron.NewJobWrapper(
                a.container,
                name,                // type name for resolution
                job.Name(),          // human name for logging
                job.Schedule(),
                job.Timeout(),
                a.ctx,               // app context
                a.Logger,
            )
            
            if err := a.scheduler.RegisterJob(wrapper); err != nil {
                a.Logger.Error("failed to register cron job",
                    "job", job.Name(),
                    "error", err)
            }
        }
    })
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| cron v1/v2 with seconds field | cron v3 standard 5-field | v3.0.0 (June 2019) | Standard cron expression format by default |
| cron.ErrorLog field | cron.WithLogger() option | v3.0.0 (June 2019) | Functional options pattern |
| Automatic panic recovery | Opt-in via WithChain(Recover()) | v3.0.0 (June 2019) | Must explicitly enable recovery |
| Printf logging | go-logr compatible interface | v3.0.0 (June 2019) | Structured logging support |
| Manual shutdown tracking | Stop() returns context | v3.0.0 (June 2019) | Clean graceful shutdown pattern |

**Deprecated/outdated:**
- `cron.WithPanicLogger`: Removed in v3; use `cron.WithChain(cron.Recover(logger))`
- `cron.WithVerboseLogger`: Removed in v3; use `cron.WithLogger(VerbosePrintfLogger(...))`
- `gopkg.in/robfig/cron.v2`: Use `github.com/robfig/cron/v3` import path
- Seconds field by default: v3 uses 5-field standard format; opt in with `cron.WithSeconds()` if needed

## Open Questions

Things that couldn't be fully resolved:

1. **Scheduler as Worker vs. Separate Integration**
   - What we know: Scheduler could implement worker.Worker for auto-discovery OR be managed separately in app.go
   - What's unclear: Whether scheduler should be discovered like other workers or handled specially
   - Recommendation: Implement worker.Worker interface; fits existing pattern, auto-discovery works

2. **Health Check Granularity**
   - What we know: CRN-09 requires health check exposing job status
   - What's unclear: What exactly to report (running count? last errors? individual job status?)
   - Recommendation: Start simple - report scheduler running + count of registered jobs; expand if needed

3. **Job Type Registration**
   - What we know: Jobs should be registered via `For[CronJob](c).Transient().Provider(NewMyJob)`
   - What's unclear: Should gaz provide a helper function or just document the pattern?
   - Recommendation: Document pattern in examples; no special helper needed for v1

## Sources

### Primary (HIGH confidence)
- `/robfig/cron` Context7 documentation - Job wrappers, graceful shutdown, logger interface, predefined schedules
- https://github.com/robfig/cron README - v3 upgrade guide, breaking changes, new features
- gaz codebase: worker/manager.go, worker/supervisor.go, worker/worker.go - Worker interface patterns
- gaz codebase: di/service.go, di/container.go - Transient service pattern, resolution
- gaz codebase: app.go - discoverWorkers pattern, lifecycle integration
- CONTEXT.md locked decisions - CronJob interface, transient lifecycle, SkipIfStillRunning

### Secondary (MEDIUM confidence)
- gaz health/manager.go - Health check registration pattern

### Tertiary (LOW confidence)
- None - all findings verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - robfig/cron v3 verified via Context7 and official README
- Architecture: HIGH - Patterns derived from existing gaz code and CONTEXT.md decisions
- Pitfalls: HIGH - Based on documented robfig/cron behavior and gaz patterns

**Research date:** 2026-01-29
**Valid until:** 2026-02-28 (30 days - stable library, locked design decisions)
