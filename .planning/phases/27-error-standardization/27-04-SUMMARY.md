---
phase: 27-error-standardization
plan: 04
subsystem: errors
tags: [errors, worker, cron, re-export, errors.Is]

# Dependency graph
requires:
  - phase: 27-01
    provides: "Consolidated sentinel errors with ErrSubsystemAction naming"
  - phase: 27-02
    provides: "DI error re-export pattern"
  - phase: 27-03
    provides: "Config error re-export pattern"
provides:
  - "gaz.ErrWorker* re-export worker.Err* for errors.Is compatibility"
  - "gaz.ErrCronNotRunning re-exports cron.ErrNotRunning"
  - "cron/errors.go with canonical ErrNotRunning sentinel"
  - "Complete Phase 27 error standardization"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: ["subsystem error re-export pattern"]

key-files:
  created: [cron/errors.go]
  modified: [errors.go, cron/scheduler.go]

key-decisions:
  - "Re-export worker/cron errors instead of defining independent sentinels"
  - "Subsystem packages keep their errors.go as source of truth"
  - "gaz.ErrWorker*/ErrCron* are aliases ensuring errors.Is compatibility"

patterns-established:
  - "All four subsystems (di, config, worker, cron) use same re-export pattern"

# Metrics
duration: 2min
completed: 2026-02-01
---

# Phase 27 Plan 04: Worker/Cron Migration Summary

**Re-exported worker and cron errors from their subsystem packages, completing the consistent error re-export pattern across all subsystems**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-01T01:29:30Z
- **Completed:** 2026-02-01T01:30:32Z
- **Tasks:** 1 (modified from original plan)
- **Files modified:** 3

## Accomplishments

- Created cron/errors.go with ErrNotRunning as canonical sentinel error
- Updated cron/scheduler.go to use cron.ErrNotRunning instead of inline errors.New()
- Made gaz.ErrWorker* re-export worker.Err* instead of being independent sentinels
- Made gaz.ErrCronNotRunning re-export cron.ErrNotRunning
- All four subsystems (di, config, worker, cron) now use consistent re-export pattern
- errors.Is compatibility ensured across all packages

## Task Commits

1. **Modified Task: Re-export worker/cron errors** - `d85cd23` (feat)

Note: Original plan called for subsystems to import gaz, but Go import cycles prevented this. Instead, gaz re-exports subsystem errors.

## Files Created/Modified

- `cron/errors.go` - NEW: Canonical ErrNotRunning sentinel error
- `cron/scheduler.go` - Use cron.ErrNotRunning instead of inline errors.New()
- `errors.go` - Changed ErrWorker* and ErrCronNotRunning from independent errors.New() to re-exports

## Decisions Made

1. **Use re-export pattern consistently** - Same pattern as di and config: subsystem defines errors, gaz re-exports with ErrSubsystem* naming.

2. **Import cycle constraint** - gaz imports worker/cron, so they cannot import gaz. Re-export pattern is the solution.

## Deviations from Plan

### [Rule 3 - Blocking] Modified approach due to import cycle

- **Found during:** Plan analysis
- **Issue:** Original plan required worker/cron to import gaz (cycle: gaz -> worker/cron -> gaz)
- **Fix:** Keep subsystem errors.go as source, gaz re-exports for convenience
- **Impact:** Same user-facing API achieved, different internal structure

## Issues Encountered

None - modified approach worked correctly.

## User Setup Required

None.

## Next Phase Readiness

- Phase 27 complete with all requirements satisfied
- ERR-01: Sentinel errors consolidated in errors.go with re-exports
- ERR-02: ErrSubsystemAction naming convention established
- ERR-03: Typed errors with Unwrap for errors.Is/As
- Ready for Phase 28 (Testing Infrastructure)

---
*Phase: 27-error-standardization*
*Completed: 2026-02-01*
