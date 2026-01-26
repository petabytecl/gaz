---
phase: 02-lifecycle-management
plan: 03
subsystem: lifecycle
tags: topological-sort, graph, algorithm
requires:
  - phase: 02-lifecycle-management
    provides: lifecycle hooks (02-01), hook storage (02-02)
provides:
  - Lifecycle ordering engine
affects:
  - 02-04 (integration)
tech-stack:
  added: []
  patterns: [topological-sort, layering]
key-files:
  created: [lifecycle_engine.go, lifecycle_engine_test.go]
  modified: []
key-decisions:
  - "Used Kahn's algorithm for topological sorting to support parallel startup layers"
  - "Filtered out services without hooks to optimize startup/shutdown process"
  - "Sorted layers alphabetically for deterministic behavior"
patterns-established:
  - "Layered startup order allows for future parallel execution"
duration: 2 min
completed: 2026-01-26
---

# Phase 02 Plan 03: Lifecycle Ordering Engine Summary

**Implemented the core "Brain" of the lifecycle system using topological sorting to determine correct startup/shutdown order.**

## Performance

- **Duration:** 2 min
- **Started:** 2026-01-26T18:31:50Z
- **Completed:** 2026-01-26T18:34:04Z
- **Tasks:** 1 (TDD cycle)
- **Files modified:** 2

## Accomplishments
- Implemented `ComputeStartupOrder` using Kahn's algorithm
- Added support for parallel startup layers (independent services grouped together)
- Implemented `ComputeShutdownOrder` (reverse startup order)
- Added cycle detection and reporting
- Optimized output by filtering services with no lifecycle hooks

## Task Commits

Each task was committed atomically:

1. **Task 1: add failing test** - `c2ad0bc` (test)
2. **Task 2: implement engine** - `17351df` (feat)
3. **Task 3: refactor engine** - `522ca86` (refactor)

## Files Created/Modified
- `lifecycle_engine.go` - Core sorting logic and layering
- `lifecycle_engine_test.go` - Comprehensive tests for ordering, cycles, and filtering

## Decisions Made
- **Kahn's Algorithm:** Chosen for its ability to easily produce layers of independent nodes, facilitating parallel startup in the future.
- **Filtering Optimization:** Services without hooks are removed from the execution plan to avoid unnecessary processing steps during startup/shutdown.
- **Determinism:** Layers are internally sorted by name to ensure consistent behavior across runs, which is crucial for debugging and testing.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## Next Phase Readiness
- Core logic is ready.
- Next plan (02-04) will integrate this engine into the `Container.Start()` and `Container.Stop()` methods.
