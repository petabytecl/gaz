---
phase: 42-refactor-framework-ergonomics
plan: 01
subsystem: api
tags: flags, cobra, module, di
requires: []
provides:
  - Deferred flag registration for modules
  - Recursive flag collection
  - Decoupled App.Use from Cobra
affects: [future-cli-plans]

tech-stack:
  added: []
  patterns: [Deferred Registration]

key-files:
  created: []
  modified:
    - app.go
    - app_use.go
    - module_builder.go
    - cobra.go

key-decisions:
  - "Updated WithCobra to apply pending flags ensures order independence"

patterns-established:
  - "Module flags are stored in App and applied when Cobra is attached"

duration: 15m
completed: 2026-02-03
---

# Phase 42 Plan 01: Deferred Flag Registration Summary

**Decoupled flag registration from Cobra presence allowing App.Use() to work before App.WithCobra()**

## Performance

- **Duration:** 15m
- **Started:** 2026-02-03T00:00:00Z (approx)
- **Completed:** 2026-02-03T00:15:00Z (approx)
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added `flagFns` storage to `App` struct
- Updated `Module.Apply` to register flags recursively
- Decoupled `App.Use` from immediate Cobra flag application
- Ensured flags are applied regardless of `Use`/`WithCobra` order

## Task Commits

1. **Task 1: Add flag storage to App struct** - `03b66e5` (feat)
2. **Task 2: Update Module.Apply to register flags** - `32351a4` (feat)

## Files Created/Modified
- `app.go` - Added `flagFns` and `AddFlagsFn`
- `app_use.go` - Removed flag application logic
- `module_builder.go` - Added flag registration to `Apply`
- `cobra.go` - Added flag application to `WithCobra`

## Decisions Made
- **Updated WithCobra:** Decided to apply stored flags in `WithCobra` to handle cases where `Use` is called before `WithCobra`. This was not in the original plan but is critical for the "deferred" behavior to work.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Updated WithCobra to apply stored flags**
- **Found during:** Task 2 (Implementation)
- **Issue:** Plan specified storing flags but not applying them if `Use` is called before `WithCobra`. Flags would be stored but never used.
- **Fix:** Updated `WithCobra` in `cobra.go` to iterate `flagFns` and apply them to the command.
- **Files modified:** cobra.go
- **Verification:** Verified with `task2_test.go` (deferred registration test case).
- **Committed in:** 32351a4

## Issues Encountered
None.

## Next Phase Readiness
- Framework is ready for more flexible CLI integration patterns.
