---
phase: 14-workers
plan: 03
subsystem: worker
tags: [worker, app-integration, lifecycle, auto-discovery, graceful-shutdown]

# Dependency graph
requires:
  - phase: 14-02
    provides: WorkerManager and Supervisor with panic recovery
provides:
  - App with WorkerManager lifecycle integration
  - Workers auto-discovered during Build()
  - Workers auto-start after services in Run()
  - Workers auto-stop before services in Stop()
  - Critical worker failure triggers app shutdown
  - gaz.Worker type alias for convenience
affects: [14-04 Tests]

# Tech tracking
tech-stack:
  added: []
  patterns: [Auto-discovery via interface check, Lifecycle integration]

key-files:
  created: []
  modified:
    - app.go
    - compat.go

key-decisions:
  - "Workers discovered during Build() by checking Worker interface on resolved services"
  - "Workers start after all Starter hooks complete (services ready)"
  - "Workers stop before Stopper hooks run (services still available)"
  - "Critical failure callback triggers graceful app shutdown"

patterns-established:
  - "Auto-discovery pattern: ForEachService + interface type assertion"
  - "Lifecycle ordering: services start → workers start → workers stop → services stop"

# Metrics
duration: 2min
completed: 2026-01-28
---

# Phase 14 Plan 03: App Integration Summary

**WorkerManager integrated with App lifecycle for auto-discovery during Build(), auto-start in Run() after services, and auto-stop in Stop() before services**

## Performance

- **Duration:** 2 min
- **Started:** 2026-01-28T20:35:37Z
- **Completed:** 2026-01-28T20:38:16Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- WorkerManager initialized in New() with critical failure handler
- Workers auto-discovered during Build() by checking Worker interface
- Workers auto-start after all Starter hooks complete in Run()
- Workers auto-stop before Stopper hooks run in Stop()
- gaz.Worker type alias added for convenience

## Task Commits

Each task was committed atomically:

1. **Task 1: Add WorkerManager to App struct and discover workers** - `db494e2` (feat)
2. **Task 2: Integrate workers with Run()/Stop() lifecycle** - `0c9e511` (feat)

**Plan metadata:** (pending)

## Files Created/Modified

- `app.go` - Added workerMgr field, discoverWorkers(), worker lifecycle in Run()/Stop()
- `compat.go` - Added Worker type alias re-exporting worker.Worker

## Decisions Made

- Workers discovered during Build() using interface type assertion on resolved services
- Transient services skipped during discovery (may resolve multiple times)
- Workers start after all services are started (workers may depend on services)
- Workers stop before services stop (workers may depend on services)
- Critical failure handler triggers app.Stop() in goroutine (non-blocking)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- App integration complete, ready for tests (14-04)
- WorkerManager fully integrated with app lifecycle
- gaz.Worker alias available for users

---
*Phase: 14-workers*
*Completed: 2026-01-28*
