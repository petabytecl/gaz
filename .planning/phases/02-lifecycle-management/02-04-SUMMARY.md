---
phase: 02-lifecycle-management
plan: 02-04
subsystem: lifecycle
tags: [app, lifecycle, signal, shutdown]

requires:
  - phase: 02-lifecycle-management
    provides: Lifecycle Engine
provides:
  - "gaz.App struct with Run/Stop"
  - "Graceful shutdown on SIGTERM"
  - "Parallel service startup"
affects:
  - 03-configuration
  - 04-observability

tech-stack:
  added: []
  patterns: [lifecycle-orchestration, graceful-shutdown]

key-files:
  created: [app.go, app_test.go]
  modified: [container.go, lifecycle_engine.go]

key-decisions:
  - "App.Run blocks until Stop() called or Signal received"
  - "Stop() can be called externally to initiate shutdown"
  - "Fixed bug in ComputeStartupOrder to correctly handle leaf dependencies"
  - "Fixed bug in Container.Build to ensure dependencies are recorded"

patterns-established:
  - "App wrapper orchestrates the Container and Lifecycle"

duration: 15min
completed: 2026-01-26
---

# Phase 02 Plan 04: Application Runtime Summary

**Implemented App.Run() with parallel startup, graceful shutdown, and signal handling.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-01-26T18:35:00Z
- **Completed:** 2026-01-26T18:50:00Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Implemented `gaz.App` struct as the main entry point
- Implemented `Run` method that orchestrates startup and blocking
- Implemented `Stop` method that orchestrates reverse-order shutdown
- Added signal handling (SIGTERM, SIGINT) for graceful shutdown
- Fixed critical bugs in dependency graph calculation and recording

## Task Commits

1. **Task 1: Create App Struct** - `500b539` (feat)
2. **Task 2: Implement Run and Stop** - `c04be20` (feat)
3. **Task 3: Signal Handling** - `962a8ec` (feat)

Additional fixes committed:
- `3f34267` fix(02-04): fix startup order calculation for leaf nodes
- `a61b3ee` fix(02-04): ensure dependencies are recorded during Build

## Files Created/Modified
- `app.go` - Main App struct and lifecycle logic
- `app_test.go` - Integration tests for App
- `container.go` - Fixed dependency recording in Build
- `lifecycle_engine.go` - Fixed startup order for dependencies

## Decisions Made
- `App.Run` is the blocking main loop. It waits for context cancellation, signal, or external `Stop()` call.
- `Stop()` initiates shutdown logic and signals `Run` to return.
- Parallel startup is implemented using `sync.WaitGroup` for each layer.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed startup order calculation for leaf nodes**
- **Found during:** Task 2 (Integration test failure)
- **Issue:** `ComputeStartupOrder` ignored services with 0 dependencies because they weren't keys in the graph map.
- **Fix:** Initialized `pendingCounts` for ALL services from the services map.
- **Files modified:** `lifecycle_engine.go`
- **Committed in:** `3f34267`

**2. [Rule 1 - Bug] Fixed dependency recording during Build**
- **Found during:** Task 2 (Integration test failure)
- **Issue:** Eager services instantiated via `Build` called `getInstance` directly, bypassing `resolveByName`'s chain management, so top-level dependencies weren't recorded.
- **Fix:** Updated `Build` to use `resolveByName` to ensure resolution chain is active.
- **Files modified:** `container.go`
- **Committed in:** `a61b3ee`

---

**Total deviations:** 2 auto-fixed (Bugs preventing correct lifecycle order)
**Impact on plan:** Essential fixes for the core requirement of ordered startup.

## Issues Encountered
- Test failures revealed the graph calculation bugs immediately. Fixed via TDD loop.

## Next Phase Readiness
- Lifecycle management is complete.
- Ready to move to Configuration (Phase 3).
