---
phase: 26-module-service-consolidation
plan: 04
subsystem: di
tags: [eventbus, config, module, di, functional-options]

requires:
  - phase: 26-01
    provides: Service consolidation complete, di package stable
provides:
  - eventbus.NewModule() factory function with functional options
  - config.NewModule() factory function with functional options
  - Consistent module API across all subsystem packages
affects: [26-05, documentation]

tech-stack:
  added: []
  patterns: [di.Container-based module functions]

key-files:
  created:
    - eventbus/module.go
    - eventbus/module_test.go
    - config/module.go
    - config/module_test.go
  modified: []

key-decisions:
  - "Used di package to avoid import cycle (eventbus/config imported by gaz package)"
  - "NewModule returns func(*di.Container) error, not gaz.Module interface"
  - "Followed health module pattern for consistency"

patterns-established:
  - "Module function pattern: func NewModule(opts ...ModuleOption) func(*di.Container) error"
  - "Import cycle avoidance: Use di package when package is imported by gaz"

duration: 3min
completed: 2026-01-31
---

# Phase 26 Plan 04: Eventbus and Config NewModule Summary

**Added NewModule() factory functions to eventbus and config packages using di.Container pattern to avoid import cycles**

## Performance

- **Duration:** 3 min
- **Started:** 2026-01-31T18:13:01Z
- **Completed:** 2026-01-31T18:17:00Z
- **Tasks:** 3
- **Files modified:** 4 (2 created, 2 test files created)

## Accomplishments

- Added `eventbus.NewModule()` with functional options pattern
- Added `config.NewModule()` with functional options pattern
- Both modules follow health module pattern using `di.Container`
- Complete test coverage for new functions

## Task Commits

Each task was committed atomically:

1. **Task 1: Create eventbus/module.go** - `93b396a` (feat)
2. **Task 2: Create config/module.go** - `e51bf0a` (feat)
3. **Task 3: Add tests** - `2c1d300` (test)

## Files Created/Modified

- `eventbus/module.go` - NewModule() factory with ModuleOption
- `eventbus/module_test.go` - Tests for NewModule()
- `config/module.go` - NewModule() factory with ModuleOption
- `config/module_test.go` - Tests for NewModule()

## Decisions Made

1. **Used di package instead of gaz package**
   - Both eventbus and config are imported by gaz/app.go
   - Importing gaz from these packages would create import cycle
   - Followed existing health module pattern

2. **NewModule returns function, not gaz.Module interface**
   - Returns `func(*di.Container) error` compatible with direct container use
   - Users can call directly: `module := eventbus.NewModule(); module(container)`
   - Consistent with health.Module pattern

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Import cycle resolution**
- **Found during:** Task 1 (eventbus/module.go creation)
- **Issue:** Plan suggested using gaz.NewModule() and gaz.Module, but eventbus is imported by gaz/app.go, creating import cycle
- **Fix:** Used di package directly, following health module pattern
- **Files modified:** eventbus/module.go, config/module.go
- **Verification:** `go build ./eventbus/... ./config/...` succeeds
- **Committed in:** 93b396a, e51bf0a

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required to avoid Go import cycle. Pattern is consistent with existing health module.

## Issues Encountered

None - once import cycle was resolved, implementation proceeded smoothly.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- MOD-03 requirement satisfied: eventbus and config packages export NewModule()
- Ready for 26-05 (final wave) or phase completion verification
- All five subsystem packages now have consistent NewModule() API:
  - health.Module (existing pattern)
  - worker.NewModule()
  - cron.NewModule()
  - eventbus.NewModule()
  - config.NewModule()

---
*Phase: 26-module-service-consolidation*
*Completed: 2026-01-31*
