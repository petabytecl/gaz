# Features Research: v2.0 Concurrency (Workers, Cron, EventBus)

**Domain:** Go Application Framework - Background Processing & Event-Driven Architecture
**Researched:** 2026-01-27
**Confidence:** HIGH (Context7 + Official Documentation)

## Executive Summary

This research covers expected features for background workers, worker pools, cron scheduling, and in-app event bus in Go frameworks. The ecosystem has mature, well-established patterns from libraries like `robfig/cron`, `gammazero/workerpool`, `golang.org/x/sync/errgroup`, and `jilio/ebu`. Gaz already has strong lifecycle primitives (`Starter`/`Stopper` interfaces, graceful shutdown) that provide an excellent foundation.

**Key insight:** The differentiator for gaz is **deep lifecycle integration**. Most Go libraries for workers/cron/eventbus are standalone - they don't integrate with DI containers or application lifecycle. Gaz can provide seamless integration where workers/cron jobs/event handlers are automatically managed by the existing lifecycle system.

---

## Background Workers

### Table Stakes

Features users expect. Missing = framework feels incomplete.

| Feature | Why Expected | Complexity | Dependencies on Existing |
|---------|--------------|------------|--------------------------|
| **Context cancellation** | Graceful shutdown requires workers to respect `ctx.Done()` | Low | Existing `Stopper` interface |
| **Error handling** | Workers must report errors without crashing the app | Low | Existing error patterns |
| **Panic recovery** | Worker panics shouldn't crash the entire application | Low | Standard Go patterns |
| **Concurrency limiting** | Prevent resource exhaustion from too many goroutines | Medium | None |
| **Start/Stop lifecycle** | Workers must integrate with app.Run() and graceful shutdown | Low | `Starter`/`Stopper` already exist |
| **Wait for completion** | `StopWait()` pattern - wait for in-flight work before exit | Medium | Existing shutdown timeout |
| **Logging integration** | Workers should use app's structured logger | Low | `*slog.Logger` already in DI |

### Differentiators

Features that would make gaz stand out vs raw goroutines or standalone libraries.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **DI-aware workers** | Workers can inject dependencies from container | Low | Leverage existing `Resolve` |
| **Automatic registration** | Workers implementing `Worker` interface auto-start in app lifecycle | Medium | Like `Starter` but for long-running |
| **Health check integration** | Worker health feeds into existing health check system | Medium | Existing `/healthz` infrastructure |
| **Metrics/observability** | Worker count, queue depth, error rates visible | High | Prometheus integration |
| **Named workers** | Identify workers in logs/metrics by name | Low | Useful for debugging |
| **Restart policies** | Auto-restart workers that exit unexpectedly (with backoff) | Medium | Supervisor pattern |
| **Graceful draining** | Signal workers to stop accepting new work before shutdown | Medium | Two-phase shutdown |

### Anti-Features

Things to deliberately NOT build for workers.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Distributed workers** | Massive scope creep, requires external infrastructure (Redis, etc.) | Document integration with `asynq` for distributed needs |
| **Persistence/durability** | In-app workers are ephemeral by design | Use external queue for durable work |
| **Complex retry with dead-letter** | Overkill for in-app workers | Simple retry with backoff, log failures |
| **Priority queues** | Adds significant complexity | Single FIFO queue per pool |
| **Worker hot-reload** | Complex runtime management | Restart app for worker changes |

---

## Worker Pools

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies on Existing |
|---------|--------------|------------|--------------------------|
| **Fixed-size pool** | Core pattern: `New(maxWorkers int)` | Low | None |
| **Submit(func())** | Non-blocking task submission | Low | None |
| **StopWait()** | Wait for all submitted tasks to complete | Low | Shutdown integration |
| **Stop()** | Stop immediately (don't wait for queue) | Low | Shutdown integration |
| **Context support** | `SubmitWithContext(ctx, func())` for cancellation | Low | Standard Go pattern |
| **Unbounded queue** | Tasks queue when pool is busy | Low | Standard pattern from `gammazero/workerpool` |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Lifecycle integration** | Pool auto-starts/stops with app lifecycle | Low | Implement `Starter`/`Stopper` |
| **Bounded waiting queue** | Backpressure when queue gets too large | Medium | Optional, prevents OOM |
| **TrySubmit()** | Non-blocking submit that returns false if pool is saturated | Low | Like `errgroup.TryGo()` |
| **Error aggregation** | Collect errors from all tasks (like `errgroup`) | Medium | Different from fire-and-forget |
| **Waiting count** | Expose queue depth for observability | Low | Metrics integration |
| **WaitingQueueSize()** | Query current queue depth | Low | Observability |
| **Pause/Resume** | Temporarily stop processing without losing queued work | Medium | Useful for maintenance |
| **Per-pool naming** | Named pools for logging/metrics | Low | Debugging aid |

### Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Dynamic pool resizing** | Complex, rarely needed | Create new pool if size needs change |
| **Task prioritization** | Significant complexity | Use separate pools for different priorities |
| **Task dependencies/DAG** | Not a pool concern, different abstraction | Document workflow patterns |
| **Rate limiting** | Separate concern | Compose with rate limiter |

---

## Cron/Scheduled Tasks

### Table Stakes

Based on `robfig/cron` (98.3 benchmark score, HIGH reputation):

| Feature | Why Expected | Complexity | Dependencies on Existing |
|---------|--------------|------------|--------------------------|
| **Standard 5-field cron expressions** | `* * * * *` (minute, hour, day, month, weekday) | Medium | None (use `robfig/cron` parser) |
| **Predefined schedules** | `@hourly`, `@daily`, `@weekly`, `@monthly` | Low | Convenience |
| **Fixed intervals** | `@every 5m`, `@every 1h30m` | Low | Common use case |
| **Start/Stop lifecycle** | Scheduler integrates with app lifecycle | Low | `Starter`/`Stopper` |
| **Graceful shutdown** | Wait for running jobs before exit | Medium | Existing shutdown patterns |
| **Panic recovery** | Jobs shouldn't crash scheduler | Low | `cron.Recover()` pattern |
| **Job entry IDs** | Return ID for later management | Low | Standard pattern |
| **Remove job by ID** | Unschedule a job at runtime | Low | Standard pattern |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **DI-aware jobs** | Cron jobs can inject dependencies | Medium | Unique to gaz |
| **Timezone support** | `CRON_TZ=America/New_York` or `WithLocation()` | Low | `robfig/cron` has this |
| **SkipIfStillRunning** | Don't start new if previous still running | Low | `cron.SkipIfStillRunning()` wrapper |
| **DelayIfStillRunning** | Queue next run if previous still running | Low | `cron.DelayIfStillRunning()` wrapper |
| **Seconds precision** | 6-field cron with seconds | Low | `cron.WithSeconds()` |
| **Job middleware chain** | Composable wrappers like HTTP middleware | Medium | `cron.WithChain()` |
| **Next run time query** | `Entry.Next` - when will job run next? | Low | Useful for UIs/debugging |
| **Health integration** | Cron scheduler health in `/healthz` | Medium | Show last run, next run, errors |
| **Named jobs** | Jobs have names for logging/metrics | Low | Critical for observability |

### Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Distributed cron** | Requires coordination (locks, leader election) | Use external scheduler (Kubernetes CronJob, etc.) |
| **Persistent job history** | Storage concern, not scheduler concern | Log to external system |
| **Job retries** | Jobs are scheduled, not queued tasks | Use worker pool for retry semantics |
| **Complex dependencies** | Cron is time-based, not event-based | Use workflow engine for complex deps |
| **Web UI for jobs** | Out of scope for framework | Document Asynqmon or similar for needs |

---

## EventBus

### Table Stakes

Based on `jilio/ebu` (93.2 benchmark score, HIGH reputation, uses generics):

| Feature | Why Expected | Complexity | Dependencies on Existing |
|---------|--------------|------------|--------------------------|
| **Type-safe publish/subscribe** | `Publish[T](bus, event)`, `Subscribe[T](bus, handler)` | Medium | Go 1.18+ generics |
| **Multiple handlers per event** | All subscribers receive the event | Low | Standard pattern |
| **Synchronous by default** | Handlers run in publisher's goroutine | Low | Predictable behavior |
| **Async handlers option** | `Subscribe(bus, handler, Async())` for concurrent | Low | Common pattern |
| **Once handlers** | Handler that auto-unsubscribes after first event | Low | Useful for setup events |
| **Handler errors** | Handlers can return errors (not panic) | Low | Error handling |
| **Unsubscribe** | Remove handler at runtime | Medium | Return subscription token |

### Differentiators

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Context propagation** | `SubscribeContext`, `PublishContext` for tracing/cancellation | Medium | `ebu` has this |
| **DI-aware handlers** | Event handlers can be resolved from container | Medium | Unique integration point |
| **Lifecycle integration** | EventBus auto-manages subscriber lifecycles | Medium | Start/Stop with app |
| **Wait for async completion** | `bus.Wait()` - wait for all async handlers | Low | `ebu` has this |
| **Panic recovery** | Handler panics don't crash app | Low | Standard pattern |
| **Event filtering** | Subscribe with filter function | Medium | Only receive matching events |
| **Middleware/interceptors** | Before/after publish hooks for logging, metrics | Medium | Observability |
| **Wildcard subscriptions** | Subscribe to multiple event types | High | Complex, maybe v3 |

### Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Distributed eventbus** | Requires external broker (NATS, Kafka, etc.) | Document integration patterns |
| **Persistent events** | Storage not eventbus concern | Use message queue |
| **Event sourcing** | Different architectural pattern entirely | Out of scope |
| **Guaranteed delivery** | In-process bus is best-effort | Use external queue for guarantees |
| **Event replay** | Requires persistence | Out of scope |
| **Schema registry** | Overkill for in-process events | Types are the schema |

---

## Complexity Assessment

### Simple (Low risk, 1-2 days each)

- **Worker interface** with `Starter`/`Stopper` - pattern already exists
- **Named logging** for workers/pools/jobs - leverage existing `*slog.Logger`
- **Context cancellation** throughout - standard Go pattern
- **Panic recovery wrappers** - simple deferred recover

### Medium (Some complexity, 3-5 days each)

- **Worker pool** with submit/stop/wait - well-understood pattern from `gammazero/workerpool`
- **Cron scheduler integration** - wrap `robfig/cron` with lifecycle integration
- **Type-safe eventbus** - follow `jilio/ebu` patterns with generics
- **Health check integration** - extend existing health system
- **DI integration for handlers** - connect to existing container

### Complex (Higher risk, 5+ days each)

- **Metrics/observability** - need to define metrics interface, possibly Prometheus
- **Restart policies with backoff** - supervisor patterns, edge cases
- **Bounded queue with backpressure** - blocking vs dropping semantics
- **Event middleware chain** - composition and ordering

---

## Integration with Existing Gaz Features

### Lifecycle Integration Pattern

All new components should follow the existing pattern:

```go
// Workers, pools, cron, eventbus all implement:
type Starter interface {
    OnStart(context.Context) error
}

type Stopper interface {
    OnStop(context.Context) error
}
```

This means:
- Auto-discovered by container during `Build()`
- Started in correct dependency order during `Run()`
- Stopped in reverse order during graceful shutdown
- Respect `ShutdownTimeout` and `PerHookTimeout`

### DI Integration Pattern

```go
// Register worker that needs database:
gaz.Provide[*MyWorker](app, func(c *gaz.Container) (*MyWorker, error) {
    db := gaz.MustResolve[*Database](c)
    return NewMyWorker(db), nil
}).AsWorker()

// The worker gets lifecycle and DI for free
```

### Health Integration

Workers/pools/cron should be able to report health:

```go
type HealthReporter interface {
    ReportHealth() health.ComponentStatus
}

// Worker pool might report:
// - active workers count
// - queue depth
// - error rate
```

---

## Recommended Phase Structure

Based on complexity and dependencies:

**Phase 1: Foundation**
- Worker interface + lifecycle integration
- Basic worker pool (Submit, StopWait)
- Context cancellation throughout

**Phase 2: Cron**
- Wrap `robfig/cron` with lifecycle
- DI-aware job registration
- Skip/Delay wrappers

**Phase 3: EventBus**
- Type-safe generics-based bus
- Sync/Async handlers
- Context propagation

**Phase 4: Polish**
- Health check integration
- Metrics (if scoped)
- Documentation

---

## Sources

### HIGH Confidence (Context7 + Official Documentation)

- `robfig/cron` - Context7 library ID: `/robfig/cron` - 98.3 benchmark score
  - Panic recovery with `cron.Recover()` wrapper
  - Job wrappers: `SkipIfStillRunning`, `DelayIfStillRunning`
  - Timezone support via `cron.WithLocation()` or `CRON_TZ=` prefix
  - Seconds field with `cron.WithSeconds()`
  - Graceful stop returns context for waiting: `ctx := c.Stop()`

- `jilio/ebu` - Context7 library ID: `/jilio/ebu` - 93.2 benchmark score
  - Type-safe generics: `Subscribe[T](bus, handler)`
  - Async handlers with `eventbus.Async()` option
  - Once handlers that auto-unsubscribe
  - Context propagation: `SubscribeContext`, `PublishContext`
  - Wait for async completion: `bus.Wait()`

- `golang.org/x/sync/errgroup` - Official Go sub-repository documentation
  - `SetLimit(n)` for bounded concurrency
  - `TryGo(f)` for non-blocking submission
  - Context integration with `WithContext(ctx)`
  - Error aggregation from parallel tasks

- `gammazero/workerpool` - GitHub (1.4k stars, MIT license)
  - `New(maxWorkers)` for fixed-size pool
  - `Submit(func())` for fire-and-forget
  - `StopWait()` for graceful shutdown
  - Unbounded queue by design

### MEDIUM Confidence (Authoritative Sources)

- `hibiken/asynq` - GitHub (12.8k stars) - Referenced for understanding distributed patterns
  - Shows what NOT to build in-process (Redis-backed, distributed)
  - Useful for documentation: "for distributed needs, use asynq"

### Existing Gaz Patterns Analyzed

- `lifecycle.go` - `Starter`/`Stopper` interfaces (lines 28-38)
- `service.go` - Service wrappers with lifecycle hooks (560+ lines)
- `app.go` - Startup/shutdown ordering, signal handling (810 lines)
  - Parallel layer-by-layer startup (lines 556-597)
  - Sequential shutdown with per-hook timeout (lines 732-789)
  - Force exit on global timeout (lines 676-690)

---

*Features research for: v2.0 Concurrency (Workers, Cron, EventBus)*
*Researched: Mon Jan 27 2026*
