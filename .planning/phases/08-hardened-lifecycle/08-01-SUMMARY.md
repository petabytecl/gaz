---
phase: 08-hardened-lifecycle
plan: 01
subsystem: lifecycle
tags: [shutdown, timeout, context, graceful-exit]

# Dependency graph
requires:
  - phase: 07-validation-engine
    provides: Stable startup for testing shutdown
provides:
  - Per-hook timeout configuration (WithHookTimeout, PerHookTimeout)
  - Sequential shutdown orchestrator with blame logging
  - Global timeout force exit (exitFunc)
affects: [08-02, 08-03, testing]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-hook context timeout pattern"
    - "Blame logging with stderr fallback"
    - "Testable exitFunc variable"

key-files:
  created: []
  modified:
    - lifecycle.go
    - app.go

key-decisions:
  - "Sequential execution within layers for accurate blame tracking"
  - "10s default per-hook timeout, 30s global timeout"
  - "ERROR level for blame logging with stderr fallback"

patterns-established:
  - "Per-hook context.WithTimeout for timeout enforcement"
  - "Select on hook completion or context.Done for timeout detection"
  - "Global timeout goroutine with done channel for cancellation"

# Metrics
duration: 7min
completed: 2026-01-27
---

# Phase 8 Plan 1: Hardened Shutdown Orchestrator Summary

**Sequential shutdown with per-hook timeout enforcement, blame logging for hanging hooks, and global timeout force exit**

## Performance

- **Duration:** 7 min
- **Started:** 2026-01-27T13:54:35Z
- **Completed:** 2026-01-27T14:02:24Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added timeout configuration infrastructure (HookConfig.Timeout, WithHookTimeout, PerHookTimeout, WithPerHookTimeout)
- Rewrote stopServices() for sequential hook execution with per-hook timeout contexts
- Added blame logging for hooks that exceed timeout (Logger + stderr fallback)
- Added global timeout force exit goroutine in Stop() calling exitFunc(1)
- Made exitFunc testable for unit testing force exit behavior

## Task Commits

Each task was committed atomically:

1. **Task 1: Timeout Configuration Infrastructure** - `c879ec1` (feat)
2. **Task 2: Sequential Shutdown Orchestrator with Blame Logging** - `0cbc200` (feat)

## Files Created/Modified

- `lifecycle.go` - Added Timeout field to HookConfig, WithHookTimeout option
- `app.go` - Added PerHookTimeout, WithPerHookTimeout, sequential shutdown, blame logging, global timeout force exit

## Decisions Made

- **Sequential within layers:** Changed from parallel to sequential execution within shutdown layers for accurate blame tracking
- **Default timeouts:** 10s per-hook (defaultPerHookTimeout), 30s global (defaultShutdownTimeout)
- **Blame logging level:** ERROR for timeouts, INFO for successful completions
- **Stderr fallback:** Always write blame to stderr in addition to logger for guaranteed output

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] NewApp() missing PerHookTimeout default**
- **Found during:** Task 2 (test failures)
- **Issue:** Legacy NewApp() didn't initialize PerHookTimeout, causing immediate timeout (0s)
- **Fix:** Added `PerHookTimeout: defaultPerHookTimeout` to NewApp() options initialization
- **Files modified:** app.go
- **Verification:** All existing tests pass
- **Committed in:** 0cbc200 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Essential fix for backward compatibility with existing tests.

## Issues Encountered

None - plan executed successfully.

## Next Phase Readiness

- Shutdown orchestrator infrastructure complete
- Ready for 08-02 (Double-SIGINT force exit handling)
- exitFunc variable ready for testing in 08-03

---
*Phase: 08-hardened-lifecycle*
*Completed: 2026-01-27*
