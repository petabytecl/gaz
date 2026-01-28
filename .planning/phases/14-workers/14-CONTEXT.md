# Phase 14: Workers - Context

**Gathered:** 2026-01-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Add background worker support with lifecycle integration, graceful shutdown, and panic recovery. Workers are services that run continuously in goroutines, auto-start with `app.Run()`, and auto-stop on shutdown. Creating workers is in scope; cron/scheduled tasks are Phase 15.

</domain>

<decisions>
## Implementation Decisions

### Worker registration API
- Use existing `For[T]().Provider()` pattern — workers implement a Worker interface and are auto-discovered
- Worker interface has `Start()`, `Stop()`, and `Name() string` methods (not `Run(ctx)`)
- `Start()` returns immediately — worker spawns its own goroutine internally
- `Stop()` signals shutdown without timeout context — worker decides when to return

### Error and restart behavior
- Panics are recovered and logged, then worker is restarted with exponential backoff
- Backoff pattern: 1s, 2s, 4s, 8s... with max delay, resets after stable run period
- Circuit breaker pattern: max N restarts in time window (e.g., 5 restarts in 10 min), then worker stays dead
- Critical workers can crash the app when they exhaust retries (configurable per-worker)

### Logging and naming
- Worker name is **required** via `Name() string` method on Worker interface
- Full lifecycle logging: start, stop, error, panic, restart events
- Structured logging via gaz logger with fields (`worker=MyWorker event=started`)

### Concurrency model
- Support worker pools via `WithPoolSize(n int)` registration option
- Pool workers named with index suffix: "queue-processor-1", "queue-processor-2"
- All workers start concurrently (parallel start, no ordering)

### Claude's Discretion
- Exact backoff algorithm parameters (initial delay, max delay, reset threshold)
- Circuit breaker window/threshold defaults
- How critical workers are marked (option or interface)
- Internal goroutine management patterns

</decisions>

<specifics>
## Specific Ideas

- Worker interface: `Start()`, `Stop()`, `Name() string` — non-blocking lifecycle
- Similar to existing Starter/Stopper pattern but for background work
- Pool workers should be easy to use for queue processing scenarios

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 14-workers*
*Context gathered: 2026-01-28*
