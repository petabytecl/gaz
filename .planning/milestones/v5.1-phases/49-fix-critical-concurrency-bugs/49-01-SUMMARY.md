---
phase: 49-fix-critical-concurrency-bugs
plan: 01
subsystem: di
tags: [concurrency, race-condition, sync.Once, mutex, di]

# Dependency graph
requires:
  - phase: 48-server-module-gateway-removal
    provides: "v5.0 complete codebase as baseline"
provides:
  - "Race-safe lazySingleton Start/Stop with mutex protection"
  - "Race-safe Container.Build() using sync.Once for single execution"
  - "Concurrent regression tests for both fixes"
affects: [di, app-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns: ["sync.Once for idempotent Build()", "mutex-guarded lifecycle methods matching eagerSingleton pattern"]

key-files:
  created: []
  modified:
    - di/service.go
    - di/container.go
    - di/service_test.go
    - di/container_test.go

key-decisions:
  - "Used sync.Once for Container.Build() instead of mutex check-and-set to eliminate TOCTOU race"
  - "lazySingleton Start/Stop now mirrors eagerSingleton pattern with s.mu.Lock()/defer s.mu.Unlock()"

patterns-established:
  - "All singleton service types (lazy, eager) must hold mutex in Start/Stop before reading built/instance fields"

requirements-completed: [CONC-03, CONC-04]

# Metrics
duration: 3min
completed: 2026-03-29
---

# Phase 49 Plan 01: Fix Critical Concurrency Bugs Summary

**Race-safe lazySingleton lifecycle methods and Container.Build() using sync.Once, with concurrent regression tests**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-29T20:33:12Z
- **Completed:** 2026-03-29T20:35:42Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Fixed data race in lazySingleton.Start() and Stop() that read s.built and s.instance without holding s.mu
- Replaced Container.Build() TOCTOU pattern (lock-check-unlock-work-lock-set-unlock) with sync.Once for guaranteed single execution
- Added concurrent regression tests: 10 goroutines for Start/Stop and 10 goroutines for Build, verified with -race -count=5

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Add failing concurrent tests** - `6195181` (test)
2. **Task 1 (GREEN): Fix lazySingleton and Container.Build races** - `e6f84ae` (fix)

_Note: Task 2 tests were written as part of Task 1's TDD RED phase_

## Files Created/Modified
- `di/service.go` - Added s.mu.Lock()/defer s.mu.Unlock() to lazySingleton Start and Stop methods
- `di/container.go` - Added buildOnce sync.Once and buildErr fields; replaced Build() body with buildOnce.Do()
- `di/service_test.go` - Added TestLazySingleton_StartStop_Concurrent with 10+10 goroutines
- `di/container_test.go` - Added TestContainer_Build_Concurrent with 10 goroutines and atomic counter

## Decisions Made
- Used sync.Once for Container.Build() instead of a mutex-guarded flag: eliminates the TOCTOU window entirely and is idiomatic Go for exactly-once initialization
- lazySingleton Start/Stop pattern mirrors eagerSingleton (lines 312-333) for consistency across all singleton types

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- di/ package is race-free for lazySingleton and Container.Build scenarios
- Ready for 49-02 (goroutine closure capture, startup error drain, worker OnStop context)

---
*Phase: 49-fix-critical-concurrency-bugs*
*Completed: 2026-03-29*
