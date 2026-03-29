---
phase: 20-testing-utilities
plan: 01
subsystem: testing
tags: [testing, test-utilities, fxtest-pattern, tdd]

# Dependency graph
requires:
  - phase: 19-interface-auto-detection
    provides: Lifecycle auto-detection for Starter/Stopper interfaces
provides:
  - gaztest.New(t) builder for test apps
  - Builder.WithTimeout() for custom timeout
  - Builder.WithApp() for pre-configured apps
  - Builder.Replace() for mock injection
  - App.RequireStart()/RequireStop() with t.Fatal()
  - Automatic cleanup via t.Cleanup()
affects: [21-service-builder]

# Tech tracking
tech-stack:
  added: []
  patterns: [fxtest-pattern, builder-pattern, t.Cleanup]

key-files:
  created:
    - gaztest/doc.go
    - gaztest/builder.go
    - gaztest/app.go
    - gaztest/gaztest_test.go
  modified: []

key-decisions:
  - "Use non-embedded *gaz.App field to avoid linter warnings"
  - "Replace() uses reflection type inference - works with concrete types only"
  - "WithApp() allows testing with pre-built apps and their services"

patterns-established:
  - "fxtest pattern: Builder → Build → RequireStart → test → RequireStop"
  - "Automatic cleanup via t.Cleanup() registration at Build() time"
  - "Idempotent RequireStart/RequireStop for safe repeated calls"

# Metrics
duration: 8min
completed: 2026-01-29
---

# Phase 20 Plan 01: Core gaztest Package Summary

**Test-friendly gaz wrapper with builder API, automatic cleanup, and mock injection via t.Cleanup()**

## Performance

- **Duration:** 8 min
- **Started:** 2026-01-29T21:45:32Z
- **Completed:** 2026-01-29T21:53:31Z
- **Tasks:** 3 (TDD RED → GREEN → Verify)
- **Files created:** 4

## Accomplishments

- Implemented `gaztest.New(t)` builder with fluent API
- Added `WithTimeout()` for custom timeout override (default 5s)
- Added `Replace()` for mock injection with reflection-based type inference
- Implemented `RequireStart()`/`RequireStop()` that fail tests on error
- Registered automatic cleanup via `t.Cleanup()` at Build() time
- Full TDD test coverage for all requirements

## Task Commits

Each task was committed atomically:

1. **Task 1: RED - Write failing tests** - `0b3e5ce` (test)
2. **Task 2: GREEN - Implement gaztest** - `e602b73` (feat)

_TDD REFACTOR phase was included in GREEN commit (linter fixes)_

## Files Created/Modified

- `gaztest/doc.go` - Package documentation with usage examples
- `gaztest/builder.go` - TB interface, Builder struct, New, WithTimeout, Replace, Build
- `gaztest/app.go` - App wrapper with RequireStart, RequireStop, cleanup
- `gaztest/gaztest_test.go` - Comprehensive tests for all requirements

## Decisions Made

1. **Non-embedded *gaz.App field**: Changed from embedded `*gaz.App` to named `app *gaz.App` field to satisfy linter (QF1008 warning about embedded field removal from selectors). This is cleaner as it avoids confusion between our `App` type and the embedded `gaz.App`.

2. **Replace() uses reflection type inference**: Per RESEARCH.md pitfall, `Replace(mock)` infers type from the concrete type of mock instance. This means replacing interface types requires registering and resolving by concrete type. Documented in code.

3. **WithApp() for pre-configured apps**: Added `WithApp()` method to allow testing with apps that have services already registered. This is required for `Replace()` to work (type must be registered first).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- Core gaztest package complete with all TEST-01 through TEST-05 requirements
- Ready for Phase 20 Plan 02: Integration tests and examples
- Package exports: `TB`, `Builder`, `App`, `New`, `DefaultTimeout`

---
*Phase: 20-testing-utilities*
*Completed: 2026-01-29*
