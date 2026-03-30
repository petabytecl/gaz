---
phase: 51-design-and-api-improvements
plan: 03
subsystem: framework
tags: [refactor, lifecycle, cron, shutdown, timer]

requires:
  - phase: none
    provides: none
provides:
  - "Split app.go into focused files (app.go, app_build.go, app_run.go, app_shutdown.go)"
  - "Cron scheduler with cancellable context"
  - "Shutdown error joining (startup + stop errors)"
  - "Timer leak fixes in shutdown and supervisor"
affects: [framework-core, worker, cron]

tech-stack:
  added: []
  patterns:
    - "File split by concern: types/build/run/shutdown"
    - "context.WithCancel for subsystem lifecycle management"
    - "errors.Join for multi-error rollback reporting"
    - "time.NewTimer+Stop instead of time.After for leak prevention"

key-files:
  created:
    - app_build.go
    - app_run.go
    - app_shutdown.go
  modified:
    - app.go
    - app_test.go
    - worker/supervisor.go
    - .golangci.yml

key-decisions:
  - "Expanded golangci exclusion paths to match split file pattern (app_build|app_run|app_shutdown)"
  - "cronCtx/cronCancel fields on App struct for cron lifecycle"

patterns-established:
  - "App lifecycle split: types in app.go, build in app_build.go, run in app_run.go, shutdown in app_shutdown.go"

requirements-completed: [DSGN-01, DSGN-03, DSGN-04, DSGN-06, DSGN-10]

duration: 8min
completed: 2026-03-30
---

# Phase 51 Plan 03: App File Split and Lifecycle Fixes Summary

**Split 1172-line app.go into 4 focused files and fixed cron context propagation, shutdown error joining, duplicate comment, and timer leaks**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-30T00:22:24Z
- **Completed:** 2026-03-30T00:30:51Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Split app.go from 1172 lines into 4 focused files: app.go (304), app_build.go (532), app_run.go (181), app_shutdown.go (202)
- Cron scheduler now receives a cancellable context (cronCtx) that is cancelled during Stop(), replacing context.Background()
- Shutdown rollback now joins startup and stop errors via errors.Join instead of discarding stop errors
- Removed duplicate "Option configures App settings" comment
- Replaced time.After with time.NewTimer+Stop in doStop() and worker/supervisor.go to prevent timer leaks
- Updated golangci exclusion paths to match split file pattern

## Task Commits

Each task was committed atomically:

1. **Task 1: Split app.go into four focused files** - `d236dab` (refactor)
2. **Task 2: Fix cron context, shutdown error join, and timer leaks** - `582f371` (fix)
3. **Golangci config update for split files** - `e89707d` (chore)

## Files Created/Modified
- `app.go` - Types, constructors, options, config methods (304 lines, down from 1172)
- `app_build.go` - Build() and all build-phase helpers (initializeLogger, initializeSubsystems, etc.)
- `app_run.go` - Run(), waitForShutdownSignal(), handleSignalShutdown() with error joining
- `app_shutdown.go` - Stop(), doStop(), stopServices(), logBlame() with timer fix and cronCancel
- `app_test.go` - Added tests for cron context, error joining, and timer leak fix
- `worker/supervisor.go` - Replaced time.After with time.NewTimer in restart delay
- `.golangci.yml` - Updated exclusion paths from `app\.go` to `app(_build|_run|_shutdown)?\.go`

## Decisions Made
- Expanded golangci-lint exclusion paths to match the new split file pattern, since pre-existing gocognit/nestif suppressions only matched `app.go`
- Added cronCtx/cronCancel fields to App struct rather than passing context through method parameters, keeping the API unchanged

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated golangci-lint exclusion paths**
- **Found during:** Post-task verification (lint check)
- **Issue:** Pre-existing gocognit/nestif lint exclusions only matched `app.go`, not the new split files
- **Fix:** Changed path pattern from `app\.go` to `app(_build|_run|_shutdown)?\.go`
- **Files modified:** .golangci.yml
- **Verification:** `make lint` no longer reports gocognit/nestif on split files
- **Committed in:** e89707d

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to maintain lint-clean status after file split. No scope creep.

## Issues Encountered
- 6 pre-existing nolintlint issues in logger/module/module.go and worker/module/module.go (unused //nolint:ireturn directives) -- these are from another plan's golangci config changes and are out of scope

## Known Stubs
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All app lifecycle code is now in focused files
- Cron scheduler properly respects app lifecycle context
- Shutdown error reporting is comprehensive
- No timer leaks in shutdown or worker restart paths

---
*Phase: 51-design-and-api-improvements*
*Completed: 2026-03-30*
