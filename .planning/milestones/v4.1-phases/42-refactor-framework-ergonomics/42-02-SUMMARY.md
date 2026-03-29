---
phase: 42-refactor-framework-ergonomics
plan: 02
subsystem: api
tags: cobra, lifecycle, shutdown, flags
requires:
  - phase: 42-refactor-framework-ergonomics
    provides: "Deferred flag registration"
provides:
  - "WithCobra default RunE loop"
  - "Zero-config CLI lifecycle management"
affects: [future-cli-plans]

tech-stack:
  added: []
  patterns: [Default Lifecycle Injection]

key-files:
  created: []
  modified:
    - cobra.go
    - cobra_test.go

key-decisions:
  - "Inject default RunE only if both Run and RunE are nil"
  - "Initialize App.running state in Cobra bootstrap to support manual Stop() calls"

patterns-established:
  - "Cobra commands manage App lifecycle via PreRunE (Start) and PostRunE (Stop)"

duration: 10m
completed: 2026-02-03
---

# Phase 42 Plan 02: Cobra Lifecycle Enhancement Summary

**Enhanced WithCobra to provide zero-config lifecycle management by injecting a default RunE loop and ensuring deferred flags are applied.**

## Performance

- **Duration:** 10m
- **Started:** 2026-02-03T22:38:00Z (approx)
- **Completed:** 2026-02-03T22:48:00Z (approx)
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Injected default `RunE` loop that waits for shutdown signal if user doesn't provide one
- Verified deferred flags are applied correctly (building on Plan 01)
- Fixed `App.Stop()` behavior when running under Cobra by properly initializing run state

## Task Commits

1. **feat(42-02): inject default RunE and verify deferred flags** - `b57561c` (feat)

## Files Created/Modified
- `cobra.go` - Added default RunE injection and updated bootstrap logic
- `cobra_test.go` - Added verification tests

## Decisions Made
- **Lifecycle Management:** Decided to manage `running` state and `stopCh` within `WithCobra`'s `bootstrap` to match `App.Run` behavior. This ensures `App.Stop()` works correctly even when the app is started via Cobra hooks.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Fixed App.Stop behavior with Cobra**
- **Found during:** Task 2 verification
- **Issue:** `waitForShutdownSignal` relies on `stopCh` being initialized, but `WithCobra` (via `bootstrap`) only called `App.Start` (which doesn't init `stopCh`), causing the default Run loop to hang during shutdown.
- **Fix:** Updated `bootstrap` in `cobra.go` to initialize `stopCh` and `running=true`, and updated `PersistentPostRunE` to reset `running=false`.
- **Files modified:** `cobra.go`
- **Verification:** `TestWithCobraInjectsDefaultRunE` passes.
- **Committed in:** `b57561c`

## Issues Encountered
None.

## Next Phase Readiness
- Ready for simplified example apps that don't need manual RunE implementation.
