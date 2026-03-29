---
phase: 24-lifecycle-interface
plan: 02
subsystem: di
tags: [lifecycle, di, breaking-change, v3.0]

requires:
  - phase: 24
    plan: 01
    provides: Worker interface with OnStart(ctx)/OnStop(ctx) pattern

provides:
  - DI package with interface-only lifecycle (no fluent hooks)
  - Simplified service wrappers without hook fields
  - Factory functions without hook parameters

affects: [24-03, 24-04, app.go, gaztest]

tech-stack:
  added: []
  patterns: [interface-based lifecycle, Starter/Stopper interfaces]

key-files:
  modified:
    - di/registration.go
    - di/service.go
    - di/registration_test.go
    - di/service_test.go
    - di/lifecycle_auto_test.go
    - app.go
    - gaztest/builder.go

key-decisions:
  - "Interface-only lifecycle - services must implement Starter/Stopper directly"
  - "No more fluent OnStart/OnStop on RegistrationBuilder"
  - "Factory functions simplified without hook parameters"

patterns-established:
  - "Use di.Starter interface for startup logic"
  - "Use di.Stopper interface for shutdown logic"

duration: 15min
completed: 2026-01-30
---

# Phase 24 Plan 02: Remove Fluent Hooks Summary

**Removed OnStart/OnStop fluent methods from RegistrationBuilder, making interface-based lifecycle (Starter/Stopper) the only mechanism for service lifecycle management**

## Performance

- **Duration:** 15 min (including session restart)
- **Started:** 2026-01-30T04:00:00Z
- **Completed:** 2026-01-30T04:15:00Z
- **Tasks:** 3
- **Files modified:** 7
- **Lines changed:** -529 net (593 removed, 64 added)

## Accomplishments

- RegistrationBuilder no longer has OnStart() or OnStop() fluent methods
- Service wrappers no longer have startHooks or stopHooks fields
- runStartHooks/runStopHooks methods deleted from baseService
- runStartLifecycle/runStopLifecycle simplified to interface-only checks
- Factory functions simplified (no hook parameters)
- All DI package tests pass
- Interface-based lifecycle (Starter/Stopper) continues to work correctly

## Task Commits

All changes committed atomically:

1. **Tasks 1-3: Complete fluent hooks removal** - `696725f` (refactor)

## Files Created/Modified

- `di/registration.go` - Removed OnStart/OnStop methods, startHooks/stopHooks fields, added doc comment directing to interfaces
- `di/service.go` - Removed hook fields, deleted runStartHooks/runStopHooks, simplified lifecycle methods, updated factory signatures
- `di/registration_test.go` - Deleted TestFor_OnStart_HookCalled, TestFor_OnStop_HookCalled, TestFor_BothHooks_CalledCorrectly
- `di/service_test.go` - Removed hook-related tests, updated factory call signatures
- `di/lifecycle_auto_test.go` - Deleted TestLifecycle_ExplicitOverridesImplicit test
- `app.go` - Updated NewInstanceServiceAny call (2 params)
- `gaztest/builder.go` - Updated NewInstanceServiceAny call (2 params)

## Decisions Made

1. **Interface-only lifecycle**: The fluent hook API was removed entirely. Services requiring lifecycle must implement `di.Starter` and/or `di.Stopper` interfaces directly on the type.

2. **No transition period**: This is a v3.0 breaking change - no deprecated API left behind.

3. **Doc comment guidance**: Added doc comment to RegistrationBuilder directing users to implement interfaces.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

### Build Breakage (Expected - Not Plan Scope)

The full project build (`go build ./...`) fails because `EventBus` and `cron.Scheduler` still use the old `worker.Worker` interface (`Start()`/`Stop()`) instead of the new `OnStart(ctx)`/`OnStop(ctx)` pattern.

**This is expected**: Plan 24-02 only modifies the DI package. The DI package builds and tests pass independently.

**Resolution path**:
- Plan 24-03 covers `cron.Scheduler` migration
- `EventBus` migration is NOT covered by any plan - needs to be added to 24-03 or handled separately

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- DI package interface-only lifecycle complete
- Ready for 24-03-PLAN.md (Migrate cron.Scheduler and example workers)
- **BLOCKER**: EventBus needs to be migrated to new worker.Worker interface (not in current plans)
  - Recommendation: Add EventBus migration to plan 24-03 or create hotfix before continuing

---
*Phase: 24-lifecycle-interface*
*Completed: 2026-01-30*
