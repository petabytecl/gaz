# Pitfalls Research: v2.0 Concurrency Primitives

**Researched:** 2026-01-27
**Overall Confidence:** HIGH (verified against Context7 docs, uber-go/fx patterns, robfig/cron docs, Watermill docs)

## Executive Summary

Adding workers, cron, and eventbus to gaz requires careful integration with the existing lifecycle system. The primary risks are:

1. **Goroutine leaks** - Workers that don't respect context cancellation
2. **Shutdown hangs** - OnStop hooks that block beyond per-hook timeout
3. **Race conditions** - State access during concurrent shutdown
4. **Panic propagation** - Unrecovered panics crashing the application
5. **Lifecycle ordering** - Workers depending on services that shut down first

gaz already has robust per-hook timeouts and blame logging, but workers/cron/eventbus introduce long-running goroutines that must be explicitly managed.

---

## Worker Pitfalls

### WRK-1: Goroutine Leaks from Orphaned Workers

**Severity:** CRITICAL

**What goes wrong:**
Workers started in OnStart continue running after OnStop completes because they don't check context cancellation. The application "shuts down" but goroutines remain, causing:
- Memory leaks in tests
- Process hangs in production
- Resource exhaustion over time

**Warning signs:**
- `go test -race` reports goroutine leaks
- Tests hang after completion
- `runtime.NumGoroutine()` increases over application restarts
- Process memory grows without explanation

**Root cause:**
Worker loops that use `for { ... }` without checking `<-ctx.Done()`.

**Prevention:**
```go
// BAD: No context check - will leak
func (w *Worker) Run() {
    for {
        w.doWork()
        time.Sleep(w.interval)
    }
}

// GOOD: Respects context cancellation
func (w *Worker) Run(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.doWork()
        }
    }
}
```

**Phase:** Workers phase - require context parameter in all worker interfaces

---

### WRK-2: Blocking OnStart Hooks

**Severity:** HIGH

**What goes wrong:**
OnStart hook runs the worker synchronously instead of spawning a goroutine. This blocks application startup, causing:
- Startup timeout
- Other services never start
- Application appears hung on boot

**Warning signs:**
- `fx.StartTimeout` exceeded (if using fx-style timeouts)
- gaz logs show startup never completes
- First service in startup order blocks forever

**Root cause:**
Misunderstanding lifecycle semantics - OnStart should *schedule* work, not *perform* work.

**Prevention:**
```go
// BAD: Blocks OnStart
func (w *Worker) OnStart(ctx context.Context) error {
    w.Run(ctx) // This blocks forever!
    return nil
}

// GOOD: Schedules work, returns immediately
func (w *Worker) OnStart(ctx context.Context) error {
    w.ctx, w.cancel = context.WithCancel(context.Background())
    go w.Run(w.ctx)
    return nil
}
```

**Phase:** Workers phase - document in API that OnStart must return quickly

**Source:** uber-go/fx docs: "hooks **must not** block to run long-running tasks synchronously, hooks **should** schedule long-running tasks in background goroutines"

---

### WRK-3: Missing Done Channel for Graceful Stop

**Severity:** HIGH

**What goes wrong:**
OnStop signals shutdown but doesn't wait for worker to actually stop. The hook returns before cleanup completes, causing:
- Data loss (worker processing item when app exits)
- Race conditions with dependent services
- gaz shutdown order violated

**Warning signs:**
- Logs show "service stopped" but worker still processing
- Data corruption or incomplete writes
- Race detector fires during shutdown

**Prevention:**
```go
type Worker struct {
    quit chan struct{}
    done chan struct{} // Signal that worker fully stopped
}

func (w *Worker) OnStop(ctx context.Context) error {
    close(w.quit) // Signal stop
    
    // Wait for worker to finish OR timeout
    select {
    case <-w.done:
        return nil // Clean shutdown
    case <-ctx.Done():
        return ctx.Err() // Timeout - gaz will blame-log this
    }
}
```

**Phase:** Workers phase - all worker types must expose Done() channel

---

### WRK-4: Panic in Worker Goroutine Crashes Application

**Severity:** HIGH

**What goes wrong:**
A panic in a worker goroutine propagates up, crashing the entire application instead of just that worker. Unlike HTTP handlers, goroutines don't automatically recover panics.

**Warning signs:**
- Application crashes with stack trace pointing to worker code
- No graceful shutdown occurs
- Process exits with non-zero code unexpectedly

**Prevention:**
```go
func (w *Worker) Run(ctx context.Context) {
    defer func() {
        if r := recover(); r != nil {
            w.logger.Error("worker panicked", "panic", r, "stack", debug.Stack())
            // Optionally: restart or notify
        }
    }()
    
    for {
        select {
        case <-ctx.Done():
            return
        case job := <-w.jobs:
            w.processWithRecovery(job)
        }
    }
}
```

**Phase:** Workers phase - provide PanicRecover wrapper in worker module

---

### WRK-5: Worker Pool Size Mismatch with Job Channel Buffer

**Severity:** MEDIUM

**What goes wrong:**
Unbuffered job channel with large worker pool causes producers to block when submitting jobs. Or: large buffer masks backpressure, causing memory exhaustion.

**Warning signs:**
- Job submission latency spikes
- Memory usage grows unboundedly
- "Thundering herd" when shutdown starts

**Prevention:**
- Bounded buffer = worker count (or small multiple)
- Document backpressure behavior
- Provide metrics: queue depth, processing time

**Phase:** Worker pools phase - configure reasonable defaults with escape hatch

---

## Cron Pitfalls

### CRN-1: Overlapping Job Execution

**Severity:** HIGH

**What goes wrong:**
A cron job scheduled every minute takes 90 seconds to complete. Without protection, a second instance starts while the first is still running. This causes:
- Database locks/deadlocks
- Rate limit exhaustion
- Double-processing of data

**Warning signs:**
- Duplicate log entries for same job run
- Database deadlocks correlating with cron schedule
- Resource usage doubles at cron interval

**Root cause:**
robfig/cron v3 runs each job in its own goroutine by default.

**Prevention:**
```go
// Use SkipIfStillRunning or DelayIfStillRunning wrapper
c := cron.New(
    cron.WithChain(
        cron.Recover(logger),
        cron.SkipIfStillRunning(logger), // Skip if previous still running
    ),
)
```

**Phase:** Cron phase - default to SkipIfStillRunning, document escape hatch

**Source:** Context7 robfig/cron docs - `SkipIfStillRunning` wrapper

---

### CRN-2: No Panic Recovery in Cron Jobs

**Severity:** HIGH

**What goes wrong:**
robfig/cron v3 no longer recovers panics by default (breaking change from v2). A panic in a cron job crashes the scheduler goroutine.

**Warning signs:**
- Cron jobs stop running after a crash
- Stack trace shows panic in cron job code
- No cron entries execute after a certain time

**Root cause:**
v3 intentional design: "Recovering panics can be surprising and is at odds with typical behavior of libraries"

**Prevention:**
```go
// Always configure Recover wrapper
c := cron.New(
    cron.WithChain(
        cron.Recover(logger),
    ),
)
```

**Phase:** Cron phase - MANDATORY panic recovery in gaz cron wrapper

**Source:** robfig/cron README: "By default, cron v3 will no longer recover panics in jobs"

---

### CRN-3: Improper Graceful Stop

**Severity:** HIGH

**What goes wrong:**
Calling `c.Stop()` without waiting for the returned context signals stop but doesn't wait for running jobs to complete. Jobs are interrupted mid-execution.

**Warning signs:**
- Jobs log "starting" but never log "completed"
- Data corruption from half-executed jobs
- OnStop hook completes but cron jobs still running

**Prevention:**
```go
// BAD: Doesn't wait for jobs
func (s *CronService) OnStop(ctx context.Context) error {
    s.cron.Stop() // Returns context, but ignored
    return nil
}

// GOOD: Waits for running jobs
func (s *CronService) OnStop(ctx context.Context) error {
    stopCtx := s.cron.Stop()
    
    select {
    case <-stopCtx.Done(): // Wait for jobs to finish
        return nil
    case <-ctx.Done(): // Or respect gaz per-hook timeout
        return ctx.Err()
    }
}
```

**Phase:** Cron phase - document `<-cron.Stop()` pattern

**Source:** Context7 robfig/cron: "Stop the scheduler and wait for running jobs to complete"

---

### CRN-4: Job Context Not Propagated

**Severity:** MEDIUM

**What goes wrong:**
Cron jobs don't receive a context, so they can't respect shutdown signals or propagate tracing context. Long-running jobs continue after shutdown starts.

**Warning signs:**
- No tracing spans for cron jobs
- Jobs don't respond to shutdown
- Logs lack request ID correlation

**Prevention:**
```go
// robfig/cron doesn't support context in AddFunc
// Use custom Job interface:
type ContextJob interface {
    Run(ctx context.Context)
}

// Wrap with context-aware scheduler
type contextJobWrapper struct {
    ctx context.Context
    job ContextJob
}

func (w *contextJobWrapper) Run() {
    w.job.Run(w.ctx)
}
```

**Phase:** Cron phase - provide context-aware job wrapper

---

## EventBus Pitfalls

### EVT-1: Unbounded Channel Buffer

**Severity:** CRITICAL

**What goes wrong:**
In-memory eventbus with unbounded buffer accepts messages faster than subscribers process them. Memory grows until OOM kills the process.

**Warning signs:**
- Memory usage grows linearly over time
- No backpressure on publishers
- OOM kills during high-traffic periods

**Prevention:**
```go
// Watermill GoChannel with bounded buffer
pubSub := gochannel.NewGoChannel(
    gochannel.Config{
        OutputChannelBuffer: 1000, // Bounded
    },
    logger,
)
```

**Phase:** EventBus phase - require explicit buffer size, no default unbounded

---

### EVT-2: CloseTimeout Not Configured

**Severity:** HIGH

**What goes wrong:**
Watermill router doesn't have CloseTimeout configured. On shutdown, it waits indefinitely for handlers to complete, blocking gaz shutdown and triggering force exit.

**Warning signs:**
- Shutdown hangs at eventbus stop
- gaz blame logs show eventbus exceeding per-hook timeout
- Global timeout force exit triggered

**Prevention:**
```go
router, _ := message.NewRouter(message.RouterConfig{
    CloseTimeout: 30 * time.Second, // MUST configure
}, logger)
```

**Phase:** EventBus phase - require CloseTimeout <= gaz PerHookTimeout

**Source:** Watermill docs: "It will wait for a timeout configured in RouterConfig.CloseTimeout"

---

### EVT-3: Subscriber Doesn't Ack Messages

**Severity:** HIGH

**What goes wrong:**
Message handler processes message but forgets to call `msg.Ack()`. Message stays in-flight forever (for persistent backends) or blocks in-memory channel.

**Warning signs:**
- Message count grows in broker
- Same messages redelivered after restart
- In-memory channel deadlocks

**Prevention:**
```go
// BAD: Forgets to Ack
func handler(msg *message.Message) error {
    process(msg)
    return nil // No Ack!
}

// GOOD: Always Ack
func handler(msg *message.Message) error {
    process(msg)
    msg.Ack()
    return nil
}
```

**Phase:** EventBus phase - wrap handlers to auto-Ack on nil error

---

### EVT-4: No Poison Queue for Failed Messages

**Severity:** MEDIUM

**What goes wrong:**
Messages that fail processing repeatedly get retried forever, blocking the queue and wasting resources on unprocessable messages.

**Warning signs:**
- Same error logged repeatedly for same message
- Queue processing stalls
- Retry count metrics go to infinity

**Prevention:**
```go
// Watermill poison queue middleware
poisonQueue, _ := middleware.PoisonQueue(pubSub, "dead_letter_topic")

router.AddMiddleware(
    middleware.Retry{MaxRetries: 3}.Middleware,
    poisonQueue, // Send to DLQ after retries exhausted
)
```

**Phase:** EventBus phase - provide poison queue option

**Source:** Watermill docs - PoisonQueue middleware

---

### EVT-5: Deadlock from Synchronous Publish in Handler

**Severity:** HIGH

**What goes wrong:**
Handler publishes to a topic that the same handler subscribes to (or creates a publish cycle). With blocking publish and bounded buffers, deadlock occurs.

**Warning signs:**
- Application hangs with no error
- SIGQUIT shows all goroutines blocked on channel operations
- Only happens under specific message sequences

**Prevention:**
- Async publish (goroutine or queue)
- Separate handler for published messages
- Detect publish cycles at registration time

**Source:** Watermill troubleshooting: "SIGQUIT signal to get goroutine dump for diagnosing deadlocks"

**Phase:** EventBus phase - document deadlock patterns, provide async publish option

---

## Integration Pitfalls

### INT-1: Lifecycle Ordering - EventBus Before Subscribers

**Severity:** CRITICAL

**What goes wrong:**
EventBus shuts down before worker that subscribes to it. Worker blocks trying to publish final status, causing shutdown hang.

**Warning signs:**
- Shutdown hangs at subscriber's OnStop
- Blame log shows subscriber timing out
- Logs show "publish failed: subscriber closed"

**Root cause:**
gaz stops in reverse startup order. If eventbus starts before worker, it stops after worker. But if worker publishes during its OnStop, eventbus may already be closed.

**Prevention:**
1. Design: Workers shouldn't publish during OnStop
2. Ordering: Register eventbus as dependency of workers
3. Timeout: Configure publish timeout < per-hook timeout

```go
// Declare explicit dependency
For[*Worker](app.Container()).
    DependsOn[*EventBus](). // EventBus stops AFTER worker
    ...
```

**Phase:** Worker + EventBus phases - document dependency patterns

---

### INT-2: Context Not Propagated to Background Work

**Severity:** HIGH

**What goes wrong:**
Workers receive `context.Background()` instead of the shutdown-aware context. They can't detect when to stop, requiring separate quit channels and complicating the API.

**Warning signs:**
- Workers don't respond to context cancellation
- Need both context AND quit channel in worker struct
- Inconsistent shutdown behavior

**Prevention:**
```go
// OnStart receives ctx but it's startup context, not shutdown context
// Need to create dedicated context for worker lifetime

func (w *Worker) OnStart(ctx context.Context) error {
    // ctx here is startup context - wrong for long-running work
    
    // Create worker-scoped context
    w.ctx, w.cancel = context.WithCancel(context.Background())
    go w.Run(w.ctx)
    return nil
}

func (w *Worker) OnStop(ctx context.Context) error {
    w.cancel() // Cancel worker context
    // Wait with shutdown context...
}
```

**Phase:** Workers phase - provide helper for worker context management

---

### INT-3: Per-Hook Timeout Too Short for Worker Drain

**Severity:** HIGH

**What goes wrong:**
gaz default per-hook timeout (10s) is shorter than time needed for workers to drain their queues. Workers get interrupted mid-drain, losing work.

**Warning signs:**
- Blame logs during normal shutdown
- Data loss proportional to queue depth at shutdown
- Tests require artificially small queues

**Prevention:**
1. Configure per-hook timeout based on expected drain time
2. Provide worker-specific timeout via `WithHookTimeout`
3. Bound queue size to ensure drain time < timeout

```go
For[*Worker](app.Container()).
    OnStop(func(ctx context.Context, w *Worker) error {
        // ...
    }, gaz.WithHookTimeout(30*time.Second)). // Override default
```

**Phase:** Workers phase - document timeout sizing, provide guidance

---

### INT-4: Race Condition During Shutdown Ordering

**Severity:** MEDIUM

**What goes wrong:**
Multiple workers in same dependency layer stop concurrently. If they share state (mutex, map), concurrent access during shutdown causes race or deadlock.

**Warning signs:**
- Race detector fires during shutdown
- Intermittent deadlocks on shutdown
- Tests fail with `-race` but pass without

**Prevention:**
- Minimize shared state between workers
- If sharing required, use concurrent-safe structures
- Document shutdown concurrency in API

**Phase:** Workers phase - document concurrency during shutdown

---

### INT-5: Worker Starts Before Dependency Ready

**Severity:** HIGH

**What goes wrong:**
Worker OnStart runs but dependency (DB, cache, etc.) isn't ready yet. Worker fails immediately or blocks waiting for dependency.

**Warning signs:**
- Startup errors like "connection refused"
- Retry loops in OnStart blocking startup
- Intermittent startup failures

**Root cause:**
gaz starts services in parallel within layers. If dependency is in same layer as worker, order is undefined.

**Prevention:**
1. Declare explicit dependencies: `DependsOn[*Database]()`
2. Use health checks before starting workers
3. Build retry/backoff into worker, not OnStart

**Phase:** Workers phase - document dependency requirements

---

## Phase-Specific Warnings

| Phase | Likely Pitfall | Priority | Mitigation |
|-------|---------------|----------|------------|
| Workers | WRK-2 (Blocking OnStart) | P0 | Document OnStart semantics clearly |
| Workers | WRK-1 (Goroutine Leaks) | P0 | Require context in worker interface |
| Workers | WRK-3 (No Done Channel) | P0 | Provide Done() in base worker |
| Worker Pools | WRK-5 (Buffer sizing) | P1 | Document sizing guidelines |
| Cron | CRN-2 (No Panic Recovery) | P0 | Default to Recover wrapper |
| Cron | CRN-1 (Overlapping Jobs) | P0 | Default to SkipIfStillRunning |
| Cron | CRN-3 (Improper Stop) | P1 | Document `<-cron.Stop()` pattern |
| EventBus | EVT-1 (Unbounded Buffer) | P0 | Require explicit buffer size |
| EventBus | EVT-2 (CloseTimeout) | P0 | Validate against PerHookTimeout |
| Integration | INT-1 (Lifecycle Order) | P0 | Document dependency patterns |

---

## Testing Recommendations

### Goroutine Leak Detection
```go
func TestWorkerDoesNotLeak(t *testing.T) {
    before := runtime.NumGoroutine()
    
    app := gaz.New()
    app.ProvideWorker(NewWorker)
    app.Build()
    app.Run(ctx)
    app.Stop(ctx)
    
    // Give goroutines time to exit
    time.Sleep(100 * time.Millisecond)
    
    after := runtime.NumGoroutine()
    assert.Equal(t, before, after, "goroutine leak detected")
}
```

### Shutdown Timeout Verification
```go
func TestShutdownRespectsTimeout(t *testing.T) {
    start := time.Now()
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := app.Stop(ctx)
    elapsed := time.Since(start)
    
    assert.Less(t, elapsed, 6*time.Second, "shutdown exceeded timeout")
}
```

### Race Condition Detection
```bash
go test -race ./...
```

---

## Sources

| Topic | Source | Confidence |
|-------|--------|------------|
| Cron patterns | Context7: /robfig/cron | HIGH |
| Cron panic recovery | robfig/cron README (Breaking Changes) | HIGH |
| Watermill router | Context7: /threedotslabs/watermill | HIGH |
| Watermill deadlocks | Watermill troubleshooting docs | HIGH |
| Fx lifecycle | Context7: /uber-go/fx | HIGH |
| gaz lifecycle | Local codebase analysis | HIGH |
| Worker pool patterns | Go concurrency patterns (training data) | MEDIUM |

---

## Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Cron pitfalls | HIGH | Context7 + official docs |
| EventBus pitfalls | HIGH | Watermill docs verified |
| Worker patterns | HIGH | uber-go/fx patterns + Go standard patterns |
| gaz integration | HIGH | Analyzed actual codebase (lifecycle.go, app.go, service.go) |
| Lifecycle ordering | HIGH | Verified gaz shutdown order in lifecycle_engine.go |

---

## Open Questions

1. **Worker restart policy** - Should crashed workers auto-restart? Needs design decision.
2. **Metrics integration** - How to expose worker queue depth, processing time? Phase-specific research.
3. **Distributed cron** - Leader election for cron in multi-instance deployment? Out of scope for v2.0.
