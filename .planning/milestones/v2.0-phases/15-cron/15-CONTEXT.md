# Phase 15: Cron - Context

**Gathered:** 2026-01-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Add scheduled task support wrapping `robfig/cron`. Jobs are DI-aware, discovered automatically, and integrated with the application lifecycle (graceful shutdown).
</domain>

<decisions>
## Implementation Decisions

### Job Interface & Definition
- **Interface-based approach:** Jobs must implement a `CronJob` interface.
- **Structure:**
  ```go
  type CronJob interface {
      Name() string              // For logging
      Schedule() string          // Cron expression ("@hourly", "*/5 * * * *")
      Timeout() time.Duration    // Execution timeout (return 0 for none)
      Run(ctx context.Context) error
  }
  ```
- **Self-contained:** The schedule is defined by the job itself, not at registration.

### Registration Pattern
- **Auto-discovery:** Jobs are registered as providers (`For[CronJob](c).Provide(NewMyJob)`) and discovered during `app.Build()`.
- **Transient Lifecycle:** The scheduler resolves a *new instance* of the job struct from the container for each execution (Transient scope recommended for jobs).
- **Disabling:** If `Schedule()` returns an empty string, the job is not scheduled (soft disable).

### Concurrency Policy
- **Strictly Skip:** If a job is still running when its next schedule triggers, the new run is skipped.
- **Single Node:** No distributed locking or clustering support in this phase.
- **Failure Handling:** If `Run()` returns an error, log it and wait for the next scheduled run. No immediate retries.

### Context & Lifecycle
- **Context:** The `ctx` passed to `Run()` is derived from the application context (cancelled on shutdown) combined with the job's optional `Timeout()`.
- **Logging:** Verbose default — log "Started", "Finished (duration)", and "Error" for every run.
- **Panic Recovery:** Recover panics, log the stack trace, and ensure the app doesn't crash.

### Claude's Discretion
- Exact naming of the interface and package structure (`gaz/cron` vs `gaz/scheduler`).
- Internal implementation details of the wrapper around `robfig/cron`.

</decisions>

<specifics>
## Specific Ideas

- "Simple, type-safe dependency injection with sane defaults" — avoiding complex config maps for schedules in v1.
- "Transient (New per Run)" decision implies the `robfig/cron.Job` wrapper needs to hold the `di.Container` to perform resolution.

</specifics>

<deferred>
## Deferred Ideas

- **Distributed Locking:** Support for Redis/Etcd locks for multi-node deployments.
- **Functional Registration:** Helper for `cron.Func("name", "@daily", func...)` (sticking to structs for now).
- **Configurable Concurrency:** Policies like "Allow Overlap" or "Replace" (sticking to Skip for simplicity).

</deferred>

---

*Phase: 15-cron*
*Context gathered: 2026-01-29*
