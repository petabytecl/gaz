---
phase: 02-lifecycle-management
plan: 01
subsystem: lifecycle
tags: [dependency-injection, graph, lifecycle]

requires:
  - phase: 01-core-di-container
    provides: "Core container with resolution chains"
provides:
  - "Dependency graph storage and automatic recording"
affects:
  - 02-02-lifecycle-interfaces
  - 02-03-startup-shutdown

tech-stack:
  added: []
  patterns: [adjacency-list]

key-files:
  created:
    - container_graph_test.go
  modified:
    - container.go
    - service.go

key-decisions:
  - "Used separate graphMu RWMutex for granular locking"
  - "Return deep copy from getGraph() for thread safety"

metrics:
  duration: 22 min
  completed: 2026-01-26
---

# Phase 02 Plan 01: Dependency Graph Recording Summary

**Automatic dependency graph recording during service resolution using adjacency lists.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-01-26T18:00:00Z
- **Completed:** 2026-01-26T18:22:01Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Implemented `dependencyGraph` storage in `Container`
- Added thread-safe `recordDependency` and `getGraph` methods
- Updated `resolveByName` to automatically capture parent->child relationships
- Fixed blocking build issue with `serviceWrapper` interface compliance

## Task Commits

1. **Task 1: Add dependency graph storage** - `853eba8` (feat)
2. **Fix: Implement missing serviceWrapper methods** - `cec8b0e` (fix)
3. **Task 2: Capture dependencies** - `92c3c25` (feat)
4. **Fix: Align service constructor signatures** - `a273042` (fix)

## Files Created/Modified
- `container.go` - Added graph storage and recording logic
- `container_graph_test.go` - Added graph verification tests
- `service.go` - Implemented missing interface methods and updated constructor
- `registration.go` - Updated calls to match new signatures
- `service_test.go` - Updated tests to match new signatures

## Decisions Made
- **Separate Mutex for Graph:** Used `graphMu` instead of reusing `mu` to allow graph inspections without blocking service resolution/registration.
- **Deep Copy Graph:** `getGraph()` returns a full deep copy to ensure the internal state cannot be corrupted by consumers.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed missing serviceWrapper methods and constructor signatures**
- **Found during:** Task 2 Verification
- **Issue:** `serviceWrapper` interface defined `start`, `stop`, `hasLifecycle` but implementations did not implement them. Also, `newEagerSingleton` and `newInstanceService` signatures in `service.go` expected hooks (from a previous partial refactor) but callers in `registration.go` and `service_test.go` were not passing them.
- **Fix:** Implemented dummy/stub methods for all service types in `service.go`. Updated `newLazySingleton` to accept hooks. Updated all callers to pass hook arguments.
- **Files modified:** `service.go`, `registration.go`, `service_test.go`
- **Verification:** `go test ./...` passed
- **Committed in:** `cec8b0e`, `a273042`

**Total deviations:** 1 auto-fixed (blocking).
**Impact:** Essential for build success. No scope creep.

## Next Phase Readiness
- Dependency graph is now populated during resolution.
- Next steps: Implement `LifecycleManager` to use this graph for topological sorting (Plan 02-03).
