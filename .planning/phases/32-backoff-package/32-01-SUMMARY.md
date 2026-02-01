---
phase: 32-backoff-package
plan: 01
subsystem: backoff
tags: [backoff, retry, exponential, jitter, math/rand/v2]

# Dependency graph
requires: []
provides:
  - BackOff interface with NextBackOff() and Reset() methods
  - Stop sentinel constant (-1)
  - Simple backoffs (ZeroBackOff, StopBackOff, ConstantBackOff)
  - ExponentialBackOff with functional options
  - Clock interface for testable time
affects: [32-02, 32-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Functional options pattern for ExponentialBackOff configuration"
    - "Clock interface abstraction for testing"
    - "Interface compliance assertions (var _ BackOff = (*Type)(nil))"

key-files:
  created:
    - backoff/doc.go
    - backoff/backoff.go
    - backoff/backoff_test.go
    - backoff/exponential.go
    - backoff/exponential_test.go
  modified: []

key-decisions:
  - "Use math/rand/v2 for thread-safe jitter without mutex"
  - "Worker-appropriate defaults: 1s initial, 2.0 multiplier, 5min max"
  - "MaxElapsedTime defaults to 0 (disabled) - worker controls via circuit breaker"

patterns-established:
  - "BackOff interface pattern: NextBackOff() time.Duration + Reset()"
  - "Stop sentinel (-1) for signaling no more retries"

# Metrics
duration: 3min
completed: 2026-02-01
---

# Phase 32 Plan 01: Core BackOff Types Summary

**BackOff interface with Stop sentinel, simple backoff types (Zero, Stop, Constant), and ExponentialBackOff with functional options and thread-safe jitter using math/rand/v2**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-01T20:24:29Z
- **Completed:** 2026-02-01T20:27:36Z
- **Tasks:** 2/2
- **Files modified:** 5 created

## Accomplishments

- Created BackOff interface defining the core retry delay contract
- Implemented Stop sentinel (-1) for signaling retry exhaustion
- Added simple backoff types: ZeroBackOff, StopBackOff, ConstantBackOff
- Built ExponentialBackOff with functional options for full configurability
- Used math/rand/v2 for thread-safe jitter (no mutex needed)
- Implemented overflow protection to clamp at MaxInterval

## Task Commits

Each task was committed atomically:

1. **Task 1: Create backoff package with interface and simple types** - `dafd87d` (feat)
2. **Task 2: Create ExponentialBackOff with functional options** - `1474c4b` (feat)

## Files Created/Modified

- `backoff/doc.go` - Package documentation with usage examples
- `backoff/backoff.go` - BackOff interface, Stop constant, simple types
- `backoff/backoff_test.go` - Tests for simple backoff types
- `backoff/exponential.go` - ExponentialBackOff with functional options
- `backoff/exponential_test.go` - Comprehensive tests including overflow and jitter

## Decisions Made

1. **math/rand/v2 for jitter** - Top-level functions are thread-safe and auto-seeded, eliminating need for per-instance rand with mutex
2. **Worker-appropriate defaults** - 1s initial (not 100ms), 2.0 multiplier (not 1.5), 5min max - better suited for worker restarts than API retries
3. **MaxElapsedTime = 0 by default** - Disabled because worker supervisor controls retry limits via circuit breaker, not elapsed time

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Core backoff types complete and tested
- Ready for 32-02-PLAN.md: Wrappers and retry helpers (Context, MaxRetries, Retry, Ticker)
- ExponentialBackOff provides foundation for worker integration in 32-03

---
*Phase: 32-backoff-package*
*Completed: 2026-02-01*
