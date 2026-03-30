---
phase: 52-test-coverage-and-benchmarks
plan: "02"
subsystem: testing
tags: [integration-tests, test-optimization, t-parallel, cron-timing]
dependency_graph:
  requires: []
  provides: [cross-package-integration-tests, cron-timing-optimization, parallel-test-markers]
  affects: [cron/internal, di, eventbus, backoff, config]
tech_stack:
  added: []
  patterns: [channel-based-test-synchronization, t-parallel-adoption]
key_files:
  created:
    - integration_test.go
  modified:
    - cron/internal/cron_test.go
    - cron/internal/chain_test.go
    - cron/internal/option_test.go
    - di/lifecycle_test.go
    - di/lifecycle_auto_test.go
    - eventbus/bus_test.go
    - backoff/exponential_test.go
    - config/validation_test.go
    - config/testing_test.go
    - config/accessor_test.go
decisions:
  - "Used t.Parallel in cron/internal to achieve wall-clock target (tests are truly independent with own Cron instances)"
  - "Skipped testify suite tests for t.Parallel (suite manages own lifecycle)"
  - "Used channel-based signaling in cron slow-job tests instead of fixed sleep for reliable timing"
metrics:
  duration: 24m
  completed: "2026-03-30"
---

# Phase 52 Plan 02: Integration Tests, Cron Timing, and Parallel Test Markers Summary

Cross-package integration tests exercising DI + worker + eventbus, cron/internal test suite optimized from 40s to 4s, and t.Parallel() added to 106 independent tests across 4 packages.

## Task 1: Cross-package integration tests and cron timing

**Commit:** 3ec4220

### Integration Tests Created

Created `integration_test.go` in root package with 3 cross-package integration tests:

1. **TestIntegration_WorkerPublishesEvents** -- Registers a worker via DI that publishes events to EventBus on a 100ms ticker. Verifies at least 2 events are received by a subscriber. Tests DI provider registration + worker lifecycle + eventbus pub/sub wiring.

2. **TestIntegration_EventDrivenWorkerChain** -- Producer worker publishes taskCreatedEvent events, a consumer subscriber counts them via atomic counter. Verifies cross-worker communication via eventbus through DI.

3. **TestIntegration_GracefulShutdownDrainsEvents** -- Batch publisher sends 5 events to a slow handler (50ms each). Triggers shutdown and verifies events were processed (eventbus drain). Tests shutdown ordering.

### Cron Timing Optimization

Reduced `cron/internal` test suite from ~40s to ~4s (10x improvement):

- **t.Parallel()** added to all independent test functions (each creates own Cron instance)
- **Channel-based signaling** replaced `time.Sleep(defaultWait)` in panic recovery tests
- **Reduced slow job durations** in TestStopAndWait from 2s to 500-750ms with synchronized start detection
- **Reduced TestAddWhileRunningWithDelay** delay from 5s to 1s
- **Simplified TestSnapshotEntries** from 2s `@every 2s` schedule to 1s `* * * * * ?` schedule

## Task 2: Add t.Parallel() to independent test functions

**Commit:** eb28704

Added `t.Parallel()` to 106 independent test functions across 4 packages:

| Package | File | Count |
|---------|------|-------|
| di | lifecycle_test.go | 3 |
| di | lifecycle_auto_test.go | 4 |
| eventbus | bus_test.go | 18 |
| backoff | exponential_test.go | 14 |
| config | validation_test.go | 27 |
| config | testing_test.go | 20 |
| config | accessor_test.go | 21 |

**Skipped (intentionally):**
- `di/container_test.go`, `di/resolution_test.go` -- testify suites
- `server/vanguard/` -- port-binding tests
- `app_integration_test.go` -- testify suite

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added t.Parallel to cron/internal tests in Task 1**
- **Found during:** Task 1 cron timing optimization
- **Issue:** Reducing individual test durations was insufficient to reach the 15s target (inherent 1s cron resolution with ~20 timing-based tests)
- **Fix:** Added t.Parallel to all cron/internal test functions since each creates its own Cron instance with no shared state
- **Result:** Wall-clock time dropped from ~40s to ~4s

**2. [Rule 1 - Bug] Fixed flaky TestStopAndWait subtests**
- **Found during:** Task 1
- **Issue:** Reducing slow job sleep without synchronization caused race between job start and Stop() call
- **Fix:** Used channel-based signaling so Stop() is called only after confirming slow job is running

**3. [Rule 1 - Bug] Added nolint directives for gocognit/funlen on TestStopAndWait**
- **Found during:** Task 1
- **Issue:** TestStopAndWait exceeded gocognit (28 > 20) and funlen (126 > 100) limits after adding channel synchronization
- **Fix:** Added `//nolint:gocognit,funlen` since this is a vendored test from robfig/cron with inherent complexity

## Known Stubs

None -- all tests are fully wired with real implementations.

## Self-Check: PASSED
