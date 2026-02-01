---
phase: 27-error-standardization
plan: 01
subsystem: errors
tags: [errors, sentinel, typed-errors, errors.Is, errors.As, Unwrap]

# Dependency graph
requires:
  - phase: 26-module-service-consolidation
    provides: "Stable module API with di.Module interface"
provides:
  - "Consolidated sentinel errors with ErrSubsystemAction naming"
  - "Typed errors (ResolutionError, LifecycleError, ValidationError)"
  - "Proper Unwrap implementations for errors.Is/As"
affects: [27-02, 27-03, error-handling, debugging]

# Tech tracking
tech-stack:
  added: []
  patterns: ["ErrSubsystemAction naming convention", "Typed errors with Unwrap"]

key-files:
  created: []
  modified: [errors.go]

key-decisions:
  - "Backward compat aliases point to di.Err* until migration complete"
  - "New ErrDI* sentinels defined independently for future migration"
  - "ValidationError/FieldError copied from config (not moved yet)"

patterns-established:
  - "ErrSubsystemAction: DI errors use ErrDI*, Config uses ErrConfig*, etc."
  - "Typed errors implement Unwrap() returning underlying cause"

# Metrics
duration: 5min
completed: 2026-02-01
---

# Phase 27 Plan 01: Error Consolidation Summary

**Consolidated 16 sentinel errors with ErrSubsystemAction naming and added typed errors (ResolutionError, LifecycleError, ValidationError) with proper Unwrap implementations**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-01T01:05:37Z
- **Completed:** 2026-02-01T01:10:12Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- Consolidated all 16 sentinel errors in `errors.go` with consistent `ErrSubsystemAction` naming
- Added typed errors for recovery scenarios: `ResolutionError`, `LifecycleError`, `ValidationError`
- All typed errors implement `Unwrap()` for `errors.Is` and `errors.As` compatibility
- Backward compatibility aliases preserve existing test compatibility

## Task Commits

Each task was committed atomically:

1. **Task 1: Create consolidated sentinel errors** - `0be5663` (feat)
2. **Task 2: Add typed errors for recovery scenarios** - `199cf48` (feat)

## Files Created/Modified

- `errors.go` - Consolidated sentinel errors (16) with subsystem grouping, backward compatibility aliases (8), and typed errors (ResolutionError, LifecycleError, ValidationError, FieldError)

## Decisions Made

1. **Backward compat aliases point to di.Err* for test compatibility** - Tests check `errors.Is(err, gaz.ErrDuplicate)` but errors originate from `di` package. Aliases must point to same error values until migration.

2. **New ErrDI* sentinels defined independently** - Future plans will migrate code to use new sentinels, then remove aliases and di import.

3. **ValidationError/FieldError copied, not moved** - Plan specified "DO NOT delete original yet" - config/errors.go still has its own copies for now.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added backward compat aliases pointing to di.Err***
- **Found during:** Task 1 verification
- **Issue:** Tests failed because `gaz.ErrDuplicate` (new sentinel) != `di.ErrDuplicate` (what di package returns)
- **Fix:** Backward compat aliases now point to `di.Err*` instead of new `ErrDI*` sentinels. This preserves test compatibility until migration.
- **Files modified:** errors.go
- **Verification:** All tests pass
- **Committed in:** 199cf48 (Task 2 included this fix)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary for test compatibility. The new ErrDI* sentinels exist and will be used after migration in subsequent plans.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All sentinel errors consolidated with proper naming convention
- Typed errors ready for use by subsystem packages
- Ready for 27-02 (update subsystem packages to use gaz errors)

---
*Phase: 27-error-standardization*
*Completed: 2026-02-01*
