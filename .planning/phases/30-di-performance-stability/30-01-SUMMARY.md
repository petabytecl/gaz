---
phase: 30-di-performance-stability
plan: 01
subsystem: di
tags: [goid, goroutine, performance, cycle-detection]

# Dependency graph
requires:
  - phase: 29-documentation-examples
    provides: v3.0 complete with stable APIs
provides:
  - Fast goroutine ID tracking via goid package
  - Proper chain propagation in Resolve[T]
affects: []

# Tech tracking
tech-stack:
  added: [github.com/petermattis/goid]
  patterns: []

key-files:
  created: []
  modified:
    - di/container.go
    - di/resolution.go
    - go.mod
    - go.sum

key-decisions:
  - "Use github.com/petermattis/goid for efficient goroutine ID access"
  - "Resolve[T] retrieves chain from Container via getChain() for nested resolution"

patterns-established: []

# Metrics
duration: 6min
completed: 2026-02-01
---

# Phase 30 Plan 01: Remove Goroutine ID Hack Summary

**Replace runtime.Stack() goroutine ID parsing with efficient goid.Get() and enable proper chain propagation through Resolve[T]**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-01T15:49:00Z
- **Completed:** 2026-02-01T15:55:57Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- Removed runtime.Stack() based goroutine ID parsing (allocates buffer, parses string)
- Added github.com/petermattis/goid for fast, stable goroutine ID access
- Updated Resolve[T] to use c.getChain() for proper chain propagation
- All existing tests pass with new implementation

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace runtime.Stack with goid package** - `4e81398` (perf)
2. **Task 2: Update Resolve[T] to use Container's current chain** - `fcff86d` (feat)
3. **Task 3: Run full test suite and verify** - No code changes (verification only)

## Files Created/Modified

- `di/container.go` - Replaced getGoroutineID() implementation, removed decimalBase constant
- `di/resolution.go` - Resolve[T] now calls c.getChain() before ResolveByName
- `go.mod` - Added github.com/petermattis/goid dependency
- `go.sum` - Updated checksums

## Decisions Made

- **goid package over manual parsing:** The goid package uses runtime linkname for direct access to goroutine ID, which is fast (no allocation), stable (maintained package with Go version compatibility), and safe.
- **Preserve goroutine-local chain tracking:** Kept the per-goroutine chain storage pattern rather than passing chain through all provider signatures, maintaining API compatibility.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- Plan 30-01 complete - goroutine ID hack removed
- Ready for remaining v3.1 plans

---
*Phase: 30-di-performance-stability*
*Completed: 2026-02-01*
