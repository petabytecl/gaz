---
phase: 49-fix-critical-concurrency-bugs
plan: 02
subsystem: concurrency
tags: [goroutine, race-condition, context, error-handling, worker]

requires:
  - phase: none
    provides: existing app.go startup loop and worker/supervisor.go
provides:
  - Race-safe goroutine closure in parallel service startup
  - Complete error drain collecting all startup failures
  - Fresh context for worker OnStop (not cancelled supervisor context)
affects: [worker, lifecycle, startup]

tech-stack:
  added: []
  patterns: [value-parameter goroutine closure, errors.Join for multi-error, fresh-context-for-cleanup]

key-files:
  created: []
  modified: [app.go, app_test.go, worker/supervisor.go, worker/supervisor_test.go]

key-decisions:
  - "Used value parameters in goroutine closure instead of loop variable capture"
  - "Used errors.Join to combine multiple startup failures into single error"
  - "30-second hardcoded stop timeout (defaultStopTimeout constant) for OnStop fresh context"

patterns-established:
  - "Goroutine closure capture: always pass loop variables as function parameters"
  - "Error collection: drain buffered channels with range loop, join with errors.Join"
  - "Cleanup context: create fresh context.Background() with timeout for shutdown handlers"

requirements-completed: [CONC-01, CONC-02, CONC-05]

duration: 4min
completed: 2026-03-29
---

# Phase 49 Plan 02: Fix Critical Concurrency Bugs Summary

**Race-safe goroutine closures in startup, full multi-error drain with errors.Join, and fresh context for worker OnStop**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-29T20:32:54Z
- **Completed:** 2026-03-29T20:37:02Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Fixed goroutine closure variable capture race in parallel service startup (app.go)
- Changed startup error handling to drain all errors from channel and return joined error
- Fixed worker OnStop receiving cancelled context by creating fresh timeout context
- Added regression tests for both multi-error startup and fresh OnStop context

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix goroutine closure capture and startup error drain** - `9b7772b` (fix)
2. **Task 2: Fix worker OnStop cancelled context** - `b45b355` (fix)

## Files Created/Modified
- `app.go` - Goroutine closure uses value params; error drain uses range loop + errors.Join
- `app_test.go` - Added TestStartup_MultipleFailures_AllErrorsCollected regression test
- `worker/supervisor.go` - OnStop uses context.WithTimeout(context.Background(), defaultStopTimeout)
- `worker/supervisor_test.go` - Added TestSupervisor_OnStop_FreshContext regression test

## Decisions Made
- Used value parameters `go func(n string, s di.ServiceWrapper)` to capture loop variables safely
- Used `errors.Join` (stdlib) to combine multiple startup errors into a single error value
- Used 30-second hardcoded timeout constant (`defaultStopTimeout`) for OnStop fresh context, matching default StableRunPeriod

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Lint: magic number 30 in timeout argument**
- **Found during:** Task 2 (worker OnStop fix)
- **Issue:** golangci-lint mnd linter flagged `30*time.Second` as magic number
- **Fix:** Extracted to `defaultStopTimeout` constant
- **Files modified:** worker/supervisor.go
- **Verification:** `make lint` passes with 0 issues
- **Committed in:** b45b355 (amended into Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug/lint)
**Impact on plan:** Minor naming improvement, no scope change.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All concurrency bugs fixed with regression tests
- All tests pass with `-race` flag
- Linting passes

---
*Phase: 49-fix-critical-concurrency-bugs*
*Completed: 2026-03-29*
