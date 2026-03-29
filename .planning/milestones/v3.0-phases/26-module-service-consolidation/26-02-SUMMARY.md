---
phase: 26-module-service-consolidation
plan: 02
subsystem: api
tags: [health, module, di, functional-options, dependency-injection]

# Dependency graph
requires:
  - phase: 26-01
    provides: "health module using di package directly (no import cycle)"
provides:
  - "health.NewModule() factory with functional options"
  - "di.Module interface for subsystem modules"
  - "gaz.App.UseDI() method for di.Module acceptance"
  - "ModuleOption type with WithPort, WithLivenessPath, WithReadinessPath, WithStartupPath"
affects: [26-03, 26-04, 26-05, 27-error-standardization]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Functional options pattern for module configuration"
    - "di.Module interface for subsystem modules (breaks import cycle)"
    - "gaz.App.UseDI() for accepting di.Module from subsystems"

key-files:
  created:
    - "di/module.go"
  modified:
    - "health/module.go"
    - "health/module_test.go"
    - "app_use.go"

key-decisions:
  - "NewModule() returns di.Module instead of gaz.Module to break import cycle"
  - "Added di.Module interface in di package for subsystem compatibility"
  - "Added gaz.App.UseDI() to accept di.Module from subsystems"

patterns-established:
  - "Subsystem NewModule() returns di.Module with Register(c *Container) method"
  - "Functional options: WithPort(8081), WithLivenessPath('/health/live')"
  - "di.NewModuleFunc() helper for simple module creation"

# Metrics
duration: 8min
completed: 2026-01-31
---

# Phase 26 Plan 02: Health NewModule Summary

**health.NewModule() factory with functional options pattern returning di.Module to break import cycle**

## Performance

- **Duration:** 8 min
- **Started:** 2026-01-31T18:12:07Z
- **Completed:** 2026-01-31T18:20:37Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- Added health.NewModule() factory accepting variadic ModuleOption
- Created di.Module interface in di package to break import cycle
- Added gaz.App.UseDI() method to accept di.Module from subsystem packages
- Implemented WithPort, WithLivenessPath, WithReadinessPath, WithStartupPath options
- Full test coverage for NewModule with all option combinations

## Task Commits

Each task was committed atomically:

1. **Task 1: Add ModuleOption and With* functions** - `e5b02ed` (feat)
2. **Task 2: Add NewModule() factory function** - `e7bebcf` (feat)
3. **Task 3: Add tests for NewModule()** - `feac5be` (test)

## Files Created/Modified

**Created:**
- `di/module.go` - di.Module interface and di.NewModuleFunc helper

**Modified:**
- `health/module.go` - Added ModuleOption, With* functions, NewModule()
- `health/module_test.go` - Added TestNewModule with 4 test cases
- `app_use.go` - Added gaz.App.UseDI() for di.Module acceptance

## Decisions Made

1. **NewModule() returns di.Module instead of gaz.Module** - Due to import cycle (gaz imports health, health cannot import gaz), NewModule() returns di.Module which has Register(c *Container) instead of Apply(app *App). This follows the established pattern from 26-01.

2. **Added di.Module interface** - New interface in di package that subsystem packages can implement and return without importing gaz. Methods: Name() string, Register(c *Container) error.

3. **Added gaz.App.UseDI()** - Separate method from Use() that accepts di.Module. This keeps the API explicit about which module type is being used.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added di.Module interface to break import cycle**
- **Found during:** Task 2 (NewModule factory implementation)
- **Issue:** Plan specified returning gaz.Module, but health cannot import gaz (gaz imports health)
- **Fix:** Created di.Module interface in di/module.go with Register(c *Container) method, added gaz.App.UseDI() to accept it
- **Files modified:** di/module.go (created), app_use.go (modified)
- **Verification:** go build ./... passes, no import cycles
- **Committed in:** e7bebcf (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking issue)
**Impact on plan:** The di.Module approach is architecturally superior - it provides a clean separation between the DI layer (di package) and the application layer (gaz package). Other subsystem packages (worker, cron, eventbus, config) will follow this pattern.

## Issues Encountered

None - deviation was handled smoothly.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- MOD-03 health complete: health.NewModule() exports with functional options
- Ready for 26-03: worker/cron NewModule factories (will use same di.Module pattern)
- Pattern established for remaining subsystem packages

---
*Phase: 26-module-service-consolidation*
*Completed: 2026-01-31*
