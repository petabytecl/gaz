---
phase: 24-lifecycle-interface
plan: 01
subsystem: worker
tags: [lifecycle, worker, context, error-handling]

requires:
  - phase: 23
    provides: STYLE.md with API naming conventions

provides:
  - Worker interface with OnStart(ctx)/OnStop(ctx) error signatures
  - Context propagation to worker lifecycle methods
  - Error handling for startup failures and shutdown issues

affects: [24-02, 24-03, cron.Scheduler, example workers]

tech-stack:
  added: []
  patterns: [context-aware lifecycle, error-returning lifecycle methods]

key-files:
  modified:
    - worker/worker.go
    - worker/supervisor.go
    - worker/manager.go
    - worker/manager_test.go
    - worker/supervisor_test.go
    - worker/doc.go

key-decisions:
  - "OnStart errors trigger restart logic (panic-equivalent)"
  - "OnStop errors are logged but shutdown continues"
  - "Log messages updated to 'worker OnStart'/'worker OnStop'"

patterns-established:
  - "Worker interface aligns with di.Starter/di.Stopper patterns"
  - "Context propagated to workers for cancellation signals"

duration: 5min
completed: 2026-01-30
---

# Phase 24 Plan 01: Worker Interface Migration Summary

**Migrated worker.Worker interface from Start()/Stop() to OnStart(ctx context.Context) error / OnStop(ctx context.Context) error, aligning with di.Starter/di.Stopper patterns**

## Performance

- **Duration:** 5 min
- **Started:** 2026-01-30T03:25:11Z
- **Completed:** 2026-01-30T03:30:07Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- Worker interface now has `OnStart(ctx context.Context) error` method
- Worker interface now has `OnStop(ctx context.Context) error` method
- Supervisor handles startup errors (triggers restart logic)
- Supervisor logs stop errors but continues shutdown (non-fatal)
- All worker package tests pass with new interface

## Task Commits

Each task was committed atomically:

1. **Task 1: Update Worker Interface Definition** - `0d2e2b1` (feat)
2. **Task 2: Update Worker Package Implementation** - `082a93b` (feat)
3. **Task 3: Update Worker Package Tests** - `ebdb3ba` (test)

## Files Created/Modified

- `worker/worker.go` - Updated Worker interface with OnStart/OnStop signatures
- `worker/supervisor.go` - Updated runWithRecovery to call OnStart/OnStop with context, handle errors
- `worker/manager.go` - Updated pooledWorker to delegate OnStart/OnStop with context
- `worker/manager_test.go` - Updated simpleWorker to implement new interface
- `worker/supervisor_test.go` - Updated mockWorker and panicWorker to implement new interface
- `worker/doc.go` - Updated package documentation with new interface examples

## Decisions Made

1. **Startup errors trigger restart logic**: When `OnStart(ctx)` returns an error, it's treated as equivalent to a panic - the failure count is incremented and restart logic is triggered.

2. **Stop errors are non-fatal**: When `OnStop(ctx)` returns an error, it's logged as a warning but shutdown continues. This prevents stop errors from blocking graceful shutdown.

3. **Log message updates**: Changed log messages from "worker starting"/"worker stopping" to "worker OnStart"/"worker OnStop" for consistency with the new method names.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated doc.go example**
- **Found during:** Task 3 (test updates)
- **Issue:** doc.go contained the old Start()/Stop() example in package documentation
- **Fix:** Updated example to show OnStart(ctx)/OnStop(ctx) with context handling
- **Files modified:** worker/doc.go
- **Verification:** `grep -rn "func.*Start()" worker/*.go | grep -v OnStart` returns no matches
- **Committed in:** ebdb3ba (part of Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Essential for complete interface migration. No scope creep.

## Issues Encountered

None - plan executed as specified.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Worker interface migration complete
- Ready for 24-02-PLAN.md (Remove fluent hooks from RegistrationBuilder)
- No blockers for next plan

---
*Phase: 24-lifecycle-interface*
*Completed: 2026-01-30*
