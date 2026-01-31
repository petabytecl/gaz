---
phase: 26-module-service-consolidation
plan: 06
subsystem: di
tags: [di, module, api-consistency, gap-closure]

# Dependency graph
requires:
  - phase: 26-02
    provides: di.Module interface and di.NewModuleFunc helper
provides:
  - All 5 subsystem modules return di.Module consistently
  - MOD-03 requirement fully satisfied
affects: [27-error-standardization, users-of-subsystem-modules]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - di.NewModuleFunc wrapping pattern for subsystem modules

key-files:
  created: []
  modified:
    - worker/module.go
    - worker/module_test.go
    - cron/module.go
    - cron/module_test.go
    - eventbus/module.go
    - eventbus/module_test.go
    - config/module.go
    - config/module_test.go

key-decisions:
  - "All subsystem NewModule() functions now return di.Module instead of func(*di.Container) error"
  - "Use di.NewModuleFunc() wrapper to match health.NewModule() pattern"

patterns-established:
  - "Subsystem modules: NewModule(opts...) di.Module using di.NewModuleFunc(name, fn)"

# Metrics
duration: 3min
completed: 2026-01-31
---

# Phase 26 Plan 06: Gap Closure - NewModule Return Type Fix Summary

**All 4 subsystem modules (worker, cron, eventbus, config) now return di.Module, matching health.NewModule() pattern**

## Performance

- **Duration:** 3 min
- **Started:** 2026-01-31T19:18:08Z
- **Completed:** 2026-01-31T19:21:32Z
- **Tasks:** 4
- **Files modified:** 8

## Accomplishments

- Fixed worker.NewModule() return type from `func(*di.Container) error` to `di.Module`
- Fixed cron.NewModule() return type from `func(*di.Container) error` to `di.Module`
- Fixed eventbus.NewModule() return type from `func(*di.Container) error` to `di.Module`
- Fixed config.NewModule() return type from `func(*di.Container) error` to `di.Module`
- All 5 subsystem packages now have consistent NewModule() → di.Module API
- MOD-03 requirement fully satisfied

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix worker.NewModule() return type** - `c576d34` (fix)
2. **Task 2: Fix cron.NewModule() return type** - `e1c87a2` (fix)
3. **Task 3: Fix eventbus.NewModule() return type** - `a6489ea` (fix)
4. **Task 4: Fix config.NewModule() return type** - `7b3f68a` (fix)

## Files Created/Modified

- `worker/module.go` - Return di.Module, use di.NewModuleFunc("worker", ...)
- `cron/module.go` - Return di.Module, use di.NewModuleFunc("cron", ...)
- `eventbus/module.go` - Return di.Module, use di.NewModuleFunc("eventbus", ...)
- `config/module.go` - Return di.Module, use di.NewModuleFunc("config", ...)
- `**/module_test.go` - Updated to use module.Register(c) instead of moduleFn(c)

## Decisions Made

None - followed plan exactly with pattern from health.NewModule()

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated module tests for new return type**
- **Found during:** Verification step
- **Issue:** Tests were calling NewModule() result as function: `moduleFn(c)` 
- **Fix:** Changed to use di.Module interface: `module.Register(c)` and added `module.Name()` assertions
- **Files modified:** worker/module_test.go, cron/module_test.go, eventbus/module_test.go, config/module_test.go
- **Verification:** `go test ./worker/... ./cron/... ./eventbus/... ./config/...` passes
- **Committed in:** 2abe002

---

**Total deviations:** 1 auto-fixed (blocking)
**Impact on plan:** Required fix for tests to compile with new return type. No scope creep.

## Issues Encountered

None

## Next Phase Readiness

- Phase 26 gap closed: all MOD requirements fully met
- Ready for Phase 27: Error Standardization (ERR-01, ERR-02, ERR-03)

---
*Phase: 26-module-service-consolidation*
*Completed: 2026-01-31*
