# Phase 2: Lifecycle Management - Context

**Gathered:** 2026-01-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Deterministic startup and shutdown with lifecycle hooks. Services can register start/stop logic that executes in dependency order. App shuts down gracefully on signals. Integrates with Phase 1's DI container.

</domain>

<decisions>
## Implementation Decisions

### Hook Registration API
- Two mechanisms: interface-based for services, fluent methods for ad-hoc hooks
- Interface: `Starter` (Start(ctx) error) and `Stopper` (Stop(ctx) error) as separate interfaces
- All resolved services scanned for Starter/Stopper — no explicit opt-in needed
- Fluent API: `app.OnStart(name, func(ctx) error)` — named for debugging/logging

### Startup/Shutdown Behavior
- Leveled concurrent execution — concurrent within same dependency level, sequential across levels
- On startup failure: rollback (stop already-started hooks, return error)
- Per-hook timeout — hooks receive context that cancels after timeout
- Default timeout duration: Claude's discretion

### Signal Handling
- First signal (SIGTERM or SIGINT) triggers graceful shutdown
- Second signal forces immediate exit
- Automatic force exit after grace period: Claude's discretion
- Print message to stderr when signal received: "Received signal, shutting down..."

### Hook Ordering Control
- Dependency-based only — DI dependency graph determines order automatically
- Shutdown runs in reverse dependency order
- Circular dependencies are an error (not a warning)
- Dependency order computed at build time (app.Run()), fail fast if invalid

### Claude's Discretion
- Default timeout duration per hook
- Grace period before automatic force exit (if any)
- Exact stderr message format
- Internal data structures for tracking hook execution

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-lifecycle-management*
*Context gathered: 2026-01-26*
