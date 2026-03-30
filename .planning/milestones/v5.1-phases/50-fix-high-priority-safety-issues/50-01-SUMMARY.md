---
phase: 50-fix-high-priority-safety-issues
plan: 01
subsystem: di, eventbus
tags: [race-condition, memory-leak, concurrency, goroutine-safety]

# Dependency graph
requires: []
provides:
  - "Race-safe EventBus Close/Publish — channels closed under write lock"
  - "Panic-safe DI resolution chain — deferred clearChain prevents goroutine-keyed leak"
affects: [eventbus, di]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hold RWMutex RLock during channel send to prevent send-on-closed-channel"
    - "Deferred clearChain at top-level resolution entry points for panic safety"

key-files:
  created: []
  modified:
    - eventbus/bus.go
    - eventbus/bus_test.go
    - di/container.go
    - di/container_test.go

key-decisions:
  - "Publish holds RLock during channel send (not just during handler collection) to prevent race with Close"
  - "clearChain at top-level entry points instead of wrapping every push/pop pair — simpler and catches all panic paths"
  - "resolveEager extracted as helper for Build() to enable deferred cleanup"

patterns-established:
  - "Top-level resolution methods defer clearChain when chain is empty (outermost call)"

requirements-completed: [SAFE-01, SAFE-02]

# Metrics
duration: 6min
completed: 2026-03-29
---

# Phase 50 Plan 01: Fix EventBus Close/Publish Race and DI Resolution Chain Leak Summary

**Race-safe EventBus Close with channels closed under write lock; panic-safe DI resolution chain with deferred clearChain at all entry points**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-29T20:50:02Z
- **Completed:** 2026-03-29T20:56:24Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Fixed P1 send-on-closed-channel panic in EventBus by holding write lock during channel close AND holding RLock during Publish channel send
- Fixed P1 resolution chain memory leak in DI container by adding deferred clearChain at all top-level resolution entry points (Build, ResolveByName, ResolveAllByName, ResolveGroup, ResolveAllByType)
- Added clearChain method and resolveEager helper for clean panic-safe patterns
- All tests pass with -race flag, linter clean

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix EventBus Close/Publish race** - `76225dc` (test) + `801b57f` (fix)
2. **Task 2: Fix DI resolution chain leak on panic** - `c6b85d6` (test) + `7c880da` (fix)

_TDD workflow: RED (failing test) then GREEN (fix) for each task_

## Files Created/Modified
- `eventbus/bus.go` - Close() now closes channels under write lock; Publish() holds RLock during send
- `eventbus/bus_test.go` - Added TestEventBus_ConcurrentClosePublish (50 goroutines + Close), CloseIdempotent, PublishAfterCloseIsNoop
- `di/container.go` - Added clearChain method, resolveEager helper, deferred clearChain at all top-level resolution entry points
- `di/container_test.go` - Added TestResolutionChain_CleanedAfterNormalResolve, CleanedAfterProviderPanic, CleanedAfterBuildPanic, CleanedAfterResolveAllPanic, TestClearChain_RemovesEntireEntry

## Decisions Made
- Publish holds RLock during channel send (not just during handler lookup) -- this is the key fix since the race window was between RUnlock and the send
- clearChain deletes entire map entry (not just popping) at top-level boundaries -- prevents stale entries from goroutine ID reuse
- Extracted resolveEager helper for Build() to cleanly defer clearChain

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Known Stubs
None

## Next Phase Readiness
- EventBus and DI container race/leak issues resolved
- Ready for remaining safety fixes (X-Request-ID validation, Vanguard health paths, logger delegation)

## Self-Check: PASSED

All files exist, all commits found, key patterns verified.

---
*Phase: 50-fix-high-priority-safety-issues*
*Completed: 2026-03-29*
