---
phase: 14-workers
plan: 02
subsystem: worker
tags: [worker, supervisor, panic-recovery, circuit-breaker, goroutine, slog]

# Dependency graph
requires:
  - phase: 14-01
    provides: Worker interface, WorkerOptions, BackoffConfig with jpillora/backoff
provides:
  - Supervisor with panic recovery and automatic restarts
  - Circuit breaker preventing runaway restart loops
  - WorkerManager for coordinating multiple workers
  - Pool worker support with indexed names
  - Critical worker failure callback for app shutdown
affects: [14-03 App integration, 14-04 Tests]

# Tech tracking
tech-stack:
  added: []
  patterns: [Supervisor pattern, Circuit breaker, Panic recovery with stack traces]

key-files:
  created:
    - worker/supervisor.go
    - worker/manager.go
    - worker/errors.go
  modified: []

key-decisions:
  - "Supervisor is internal type, Manager is exported for App integration"
  - "Pool workers use pooledWorker wrapper with indexed names (worker-1, worker-2)"
  - "Circuit breaker uses simple counter+window, not external library"
  - "Critical worker failure triggers callback, not direct shutdown"

patterns-established:
  - "Supervisor pattern: each worker wrapped with panic recovery and restart logic"
  - "Circuit breaker: track failures in time window, trip after max restarts"
  - "Scoped logger: each supervisor has logger with worker name pre-attached"

# Metrics
duration: 2min
completed: 2026-01-28
---

# Phase 14 Plan 02: WorkerManager and Supervisor Summary

**Supervisor with defer/recover panic handling and stack traces, circuit breaker for restart limiting, and WorkerManager coordinating concurrent worker lifecycle with pool support**

## Performance

- **Duration:** 2 min
- **Started:** 2026-01-28T20:31:07Z
- **Completed:** 2026-01-28T20:32:53Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Supervisor wraps workers with panic recovery using defer/recover
- Panics logged with full stack traces via runtime/debug.Stack()
- Circuit breaker tracks failures in time window, trips after MaxRestarts
- WorkerManager coordinates multiple workers with concurrent start/stop
- Pool workers supported with indexed names ("worker-1", "worker-2")
- Critical worker failure callback for triggering app shutdown

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Supervisor with panic recovery and circuit breaker** - `95cc143` (feat)
2. **Task 2: Create WorkerManager for coordinating workers** - `d23a12b` (feat)

**Plan metadata:** (pending)

## Files Created/Modified

- `worker/errors.go` - Sentinel errors: ErrCircuitBreakerTripped, ErrWorkerStopped, ErrCriticalWorkerFailed
- `worker/supervisor.go` - Per-worker supervision with panic recovery, backoff restarts, circuit breaker
- `worker/manager.go` - WorkerManager for registration, concurrent start/stop, pool workers

## Decisions Made

- Supervisor is internal (unexported) - only Manager is exported for App integration
- Pool workers use a simple pooledWorker wrapper that delegates to the original worker
- Circuit breaker is hand-rolled (counter + window) per RESEARCH.md recommendation
- onCriticalFail callback pattern allows Manager to notify App without tight coupling
- Manager.Done() returns channel for external shutdown verification (WRK-06)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- Supervisor and Manager ready for App integration (14-03)
- Manager.SetCriticalFailHandler() ready for App shutdown integration
- All exports match what App will need: Manager, NewManager

---
*Phase: 14-workers*
*Completed: 2026-01-28*
