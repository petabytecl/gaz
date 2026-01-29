---
phase: 19
plan: 01
subsystem: di
tags: [lifecycle, reflection, interfaces]

requires:
  - phase: 18
    provides: [concurrency-safety]
provides:
  - "Auto-detection of Starter/Stopper interfaces for services"
  - "Updated HasLifecycle logic for LazySingleton and EagerSingleton"
affects:
  - phase: 21

tech-stack:
  added: []
  patterns: [interface-checking, reflection]

key-files:
  created: [di/lifecycle_auto_test.go]
  modified: [di/service.go]

key-decisions:
  - "Used reflection on zero value and pointer to check implementation"
  - "Extracted common check logic to hasLifecycleImpl helper"

metrics:
  duration: 10m
  completed: 2026-01-29
---

# Phase 19 Plan 01: Lifecycle Auto-Detection Summary

**Auto-detection of Starter/Stopper interfaces for services using reflection on type T**

## Performance

- **Duration:** 10m
- **Started:** 2026-01-29T00:00:00Z (Approx)
- **Completed:** 2026-01-29T00:10:00Z (Approx)
- **Tasks:** 1 (TDD cycle)
- **Files modified:** 2

## Accomplishments
- Implemented automatic detection of `Starter` and `Stopper` interfaces
- Removed need for explicit `.OnStart()` registration for interface implementors
- Handled both value and pointer receivers using `any(zero)` and `any(new(T))`
- Added comprehensive tests for auto-detection

## Task Commits

1. **test(19-01): add failing test for lifecycle auto-detection** - `6808ffc`
2. **feat(19-01): implement lifecycle auto-detection** - `a763366`

## Files Created/Modified
- `di/service.go` - Added `hasLifecycleImpl` helper, updated `HasLifecycle` methods
- `di/lifecycle_auto_test.go` - Added tests for auto-detection

## Decisions Made
- **Reflection Strategy:** Checks both `T` (via zero value) and `*T` (via `new(T)`) to ensure all implementation patterns are caught (value receivers vs pointer receivers).
- **Helper Extraction:** Created `hasLifecycleImpl[T]` to share logic between `lazySingleton` and `eagerSingleton`, reducing duplication.

## Deviations from Plan

None - plan executed exactly as written.

## Next Phase Readiness
- Lifecycle auto-detection is ready
- Next plan (19-02) can proceed with CLI args injection
