---
phase: 002-add-tests-to-examples-coverage
plan: 002
subsystem: testing
tags: [testing, examples, coverage, cobra, microservice, eventbus]

# Dependency graph
requires: []
provides:
  - "Refactored all examples to be testable (extracted run/execute functions)"
  - "Added main_test.go to all examples covering basic functionality"
  - "Fixed circular dependency in microservice example using lazy resolution"
  - "Fixed EventBus panic on unsubscribe after close"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: 
    - "Lazy DI resolution for workers to avoid circular dependencies"
    - "Testable main pattern (run/execute function)"

key-files:
  created:
    - examples/basic/main_test.go
    - examples/config-loading/main_test.go
    - examples/modules/main_test.go
    - examples/http-server/main_test.go
    - examples/lifecycle/main_test.go
    - examples/background-workers/main_test.go
    - examples/microservice/main_test.go
    - examples/cobra-cli/main_test.go
    - examples/system-info-cli/main_test.go
  modified:
    - examples/microservice/main.go
    - eventbus/bus.go

key-decisions:
  - "Refactored example main() functions to run() returning error for testability"
  - "Used lazy resolution in microservice workers to break circular dependency with EventBus"
  - "Allowed EventBus unsubscribe to be idempotent if bus is already closed"

patterns-established:
  - "Example tests use short timeouts (100ms) to verify startup without blocking"
  - "CLI examples expose execute(io.Writer) for output capture testing"

# Metrics
duration: 9m
completed: 2026-02-03
---

# Phase 002: Add Tests to Examples Coverage Summary

**Refactored all examples for testability, added comprehensive tests, and fixed critical circular dependency and panic bugs in microservice example and EventBus.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-03T00:11:11Z
- **Completed:** 2026-02-03T00:20:32Z
- **Tasks:** 3
- **Files modified:** 20+

## Accomplishments
- Refactored `main` packages in all 9 examples to expose testable `run()` or `execute()` functions.
- Created `main_test.go` for all examples, ensuring they are covered by CI/tests.
- Fixed a circular dependency issue in `examples/microservice` where workers depended on EventBus which (as a worker) was resolved during worker discovery.
- Fixed a panic in `EventBus` when subscribers unsubscribe after the bus has been closed.

## Task Commits

Each task was committed atomically:

1. **Task 1: Test simple non-blocking examples** - `41da6ce` (feat)
2. **Task 2: Test blocking/server examples** - `e39b303` (feat)
   - Also includes fix for EventBus panic: `f284475` (fix)
3. **Task 3: Test CLI examples** - `c843733` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `examples/*/main.go` - Refactored to extract logic into `run()`/`execute()`
- `examples/*/main_test.go` - Added tests
- `eventbus/bus.go` - Fixed panic on unsubscribe after close
- `examples/microservice/main.go` - Implemented lazy resolution for workers

## Decisions Made
- **Lazy Resolution for Microservice Workers:** To solve the "circular dependency detected" error in the microservice example (caused by EventBus being both an instance dependency and a worker being discovered), I switched `OrderProcessor`, `OrderSimulator`, and `NotificationSubscriber` to use lazy resolution of `EventBus` in `OnStart` instead of constructor injection. This breaks the resolution cycle during `app.Build()`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed EventBus panic on unsubscribe**
- **Found during:** Task 2 (microservice example verification)
- **Issue:** `EventBus` stops and closes channels. Then subscribers stop and call `Unsubscribe`. `Unsubscribe` attempted to close the channel again (or close a closed channel), causing panic.
- **Fix:** Added check `if b.closed { return }` in `unsubscribe`.
- **Files modified:** `eventbus/bus.go`
- **Verification:** Microservice example runs and shuts down cleanly without panic.
- **Committed in:** `f284475`

**2. [Rule 1 - Bug] Fixed circular dependency in microservice example**
- **Found during:** Task 2 (microservice example verification)
- **Issue:** `gaz/di` detected a circular dependency when resolving `OrderProcessor` -> `EventBus`. Likely due to `EventBus` being resolved during worker discovery while `OrderProcessor` (eager) was also being built.
- **Fix:** Refactored workers to take `*gaz.Container` and resolve `EventBus` lazily in `OnStart`.
- **Files modified:** `examples/microservice/main.go`
- **Verification:** `go run examples/microservice/main.go` starts successfully.
- **Committed in:** `e39b303`

**Total deviations:** 2 auto-fixed (bugs blocking example correctness).
**Impact on plan:** Essential fixes to ensure examples work as intended.

## Issues Encountered
- `examples/system-info-cli` dependencies were missing/untidy. Ran `go mod tidy` to fix.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Examples are now covered by tests, reducing risk of regression.
- EventBus is more robust.
