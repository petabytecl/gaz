# Stack Research: v2.0 Concurrency

**Researched:** 2026-01-27
**Focus:** Workers, Worker Pools, Cron/Scheduled Tasks, EventBus
**Confidence:** HIGH (verified via Context7 + official GitHub)

## Executive Summary

gaz v2.0 adds concurrency primitives that integrate with the existing `Starter`/`Stopper` lifecycle. The recommendations prioritize:
1. **Stdlib-first** for workers (goroutines + channels are sufficient)
2. **Minimal dependencies** for specialized needs (cron, eventbus)
3. **Lifecycle integration** as the unifying pattern

---

## Recommendations

### Workers / Worker Pools

**RECOMMENDATION: Build custom + optional pond for advanced pools**

| Component | Approach | Why |
|-----------|----------|-----|
| Simple workers | **Stdlib** (goroutine + context) | Sufficient for most cases, zero deps |
| Worker pools | **alitto/pond v2.6.0** | Modern API, context-aware, graceful shutdown |

**Why NOT ants:**
- ants v2.11.0 is excellent (14k stars, generics support), but:
- `Release()`/`ReleaseTimeout()` pattern doesn't match gaz's `Stopper.OnStop(context.Context)`
- No native `context.Context` support for pool lifecycle
- Requires adapting API patterns

**Why pond:**
- `pond.WithContext(ctx)` - pool stops when context cancels (maps to gaz lifecycle)
- `pool.StopAndWait()` - graceful shutdown compatible with `Stopper`
- v2 API uses Go 1.18+ generics
- Simpler API than ants

**gaz Integration Pattern:**
```go
// Worker implements Starter + Stopper
type Worker struct {
    pool *pond.Pool
    ctx  context.Context
    cancel context.CancelFunc
}

func (w *Worker) OnStart(ctx context.Context) error {
    w.ctx, w.cancel = context.WithCancel(ctx)
    w.pool = pond.NewPool(10, pond.WithContext(w.ctx))
    return nil
}

func (w *Worker) OnStop(ctx context.Context) error {
    w.cancel() // triggers pool shutdown via context
    return nil
}
```

**For simple workers (stdlib):**
```go
type SimpleWorker struct {
    cancel context.CancelFunc
    done   chan struct{}
}

func (w *SimpleWorker) OnStart(ctx context.Context) error {
    ctx, w.cancel = context.WithCancel(ctx)
    w.done = make(chan struct{})
    go w.run(ctx)
    return nil
}

func (w *SimpleWorker) OnStop(ctx context.Context) error {
    w.cancel()
    select {
    case <-w.done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

---

### Cron / Scheduled Tasks

**RECOMMENDATION: robfig/cron v3.0.1**

| Library | Stars | Decision |
|---------|-------|----------|
| **robfig/cron** | 14k | **USE** - Industry standard, graceful shutdown, Job interface |
| go-co-op/gocron | 5k+ | Alternative - higher-level API, more deps |

**Why robfig/cron:**
- `c.Stop()` returns context that completes when jobs finish - perfect for `Stopper`
- Job interface allows DI via constructors
- v3 has job wrappers for panic recovery, concurrency control
- Thread-safe add/remove of jobs at runtime
- Standard 5-field + optional seconds cron expressions
- Timezone support

**gaz Integration Pattern:**
```go
type Scheduler struct {
    cron *cron.Cron
}

func (s *Scheduler) OnStart(ctx context.Context) error {
    s.cron = cron.New(cron.WithSeconds()) // optional seconds precision
    // Jobs added via DI
    s.cron.Start()
    return nil
}

func (s *Scheduler) OnStop(ctx context.Context) error {
    stopCtx := s.cron.Stop()
    select {
    case <-stopCtx.Done():
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Version note:** robfig/cron uses tags not releases. Latest stable: `v3.0.1` (Jan 2020). Despite age, it's stable and widely used.

---

### EventBus

**RECOMMENDATION: Build custom OR jilio/ebu for type-safety**

| Approach | When to Use |
|----------|-------------|
| **Custom (stdlib)** | Simple pub/sub, <5 event types, full control |
| **jilio/ebu v0.10.1** | Type-safe generics, async handlers, persistence |

**Why consider jilio/ebu:**
- Type-safe with Go generics - compile-time event type checking
- `eventbus.Subscribe[T](bus, handler)` - matches gaz's `For[T](c)` pattern
- Context support (`PublishContext`, `SubscribeContext`)
- `bus.Shutdown(ctx)` - graceful shutdown with timeout
- Async handlers with sequential option
- Zero core dependencies (optional otel, persistence packages)

**Why consider custom stdlib:**
- Simple pub/sub is ~50 lines of Go
- No external dependency
- Full control over behavior
- Easier to integrate with existing gaz patterns

**Recommended custom pattern:**
```go
type EventBus struct {
    mu       sync.RWMutex
    handlers map[reflect.Type][]any
    wg       sync.WaitGroup
}

func Publish[E any](bus *EventBus, event E) {
    // type-safe dispatch using generics
}

func Subscribe[E any](bus *EventBus, handler func(E)) {
    // type-safe registration
}
```

**ebu Integration Pattern (if using library):**
```go
type Bus struct {
    eb *eventbus.EventBus
}

func (b *Bus) OnStart(ctx context.Context) error {
    b.eb = eventbus.New()
    return nil
}

func (b *Bus) OnStop(ctx context.Context) error {
    return b.eb.Shutdown(ctx) // respects context timeout
}
```

---

## Integration Points with Existing gaz

| gaz Concept | New Component Integration |
|-------------|---------------------------|
| `Starter` interface | Workers/pools call `Start()` in `OnStart()` |
| `Stopper` interface | Workers/pools call graceful shutdown in `OnStop()` |
| `For[T](c).Provider()` | Register workers, scheduler, eventbus as singletons |
| `context.Context` | All new components accept/respect context |
| Per-hook timeout | Each component's shutdown respects `HookConfig.Timeout` |
| Dependency ordering | Workers may depend on DB, config - automatic ordering |

**Pattern: All concurrency primitives become lifecycle-aware services**

```go
// Example: Scheduler depends on DB connection
For[*Scheduler](c).Provider(func(c *Container) (*Scheduler, error) {
    db := Must[*DB](c) // ensures DB starts before scheduler
    return &Scheduler{db: db}, nil
})
```

---

## What NOT to Add

### Rejected: External Job Queues

| Library | Why NOT |
|---------|---------|
| machinery | Overkill - requires Redis/Mongo, for distributed systems |
| faktory | External process, language-agnostic design overhead |
| asynq | Redis-required, for distributed task queues |
| temporal | Enterprise workflow engine, massive complexity |

**Rationale:** gaz v2.0 targets in-process concurrency. Distributed job systems are a separate concern for users to add if needed.

### Rejected: Over-engineered Worker Pools

| Approach | Why NOT |
|----------|---------|
| Custom pool from scratch | stdlib goroutines often sufficient, pond covers edge cases |
| Multiple pool libraries | Pick one (pond) or none (stdlib) |
| Workstealing pools | Complexity not justified for typical use cases |

### Rejected: Complex EventBus Libraries

| Library | Why NOT |
|---------|---------|
| asaskevich/EventBus | No generics, string-based event names, less type-safe |
| mustafaturan/bus | More complex, less active development |
| olebedev/emitter | Node.js-style API, reflection heavy |

**Rationale:** Either build a simple type-safe bus (50 LOC) or use ebu for advanced needs. Middle ground adds dependency without clear benefit.

---

## Versions to Pin

```go
// go.mod additions for v2.0

// Required for cron
require github.com/robfig/cron/v3 v3.0.1

// Optional: if using worker pools beyond stdlib
require github.com/alitto/pond/v2 v2.6.0

// Optional: if using advanced eventbus
require github.com/jilio/ebu v0.10.1
```

**Go version:** Continue requiring Go 1.21+ (already required by gaz)

---

## Sources

| Source | URL | Confidence |
|--------|-----|------------|
| robfig/cron docs | Context7: /robfig/cron | HIGH |
| robfig/cron tags | https://github.com/robfig/cron/tags | HIGH |
| alitto/pond docs | Context7: /alitto/pond | HIGH |
| alitto/pond releases | https://github.com/alitto/pond/releases | HIGH |
| panjf2000/ants docs | Context7: /panjf2000/ants | HIGH |
| panjf2000/ants releases | https://github.com/panjf2000/ants/releases | HIGH |
| jilio/ebu docs | Context7: /jilio/ebu + GitHub README | HIGH |

---

## Decision Summary

| Capability | Recommendation | Dependency |
|------------|---------------|------------|
| Simple workers | **Stdlib** | None |
| Worker pools (optional) | **pond v2.6.0** | `github.com/alitto/pond/v2` |
| Cron/scheduling | **robfig/cron v3.0.1** | `github.com/robfig/cron/v3` |
| EventBus (simple) | **Custom stdlib** | None |
| EventBus (advanced) | **ebu v0.10.1** | `github.com/jilio/ebu` |

**Total new required dependencies: 1** (robfig/cron)
**Total new optional dependencies: 2** (pond, ebu)
