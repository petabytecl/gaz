---
phase: 32-backoff-package
plan: 02
subsystem: backoff
tags: [backoff, retry, context, ticker, permanent-error]

# Dependency graph
requires:
  - phase: 32-01
    provides: BackOff interface, Stop sentinel, simple backoff types
provides:
  - Timer interface for testable sleeps
  - WithContext wrapper for context-aware backoff
  - WithMaxRetries wrapper for retry limiting
  - PermanentError for immediate retry stop
  - Retry/RetryNotify/RetryWithData functions
  - Ticker for periodic backoff-based operations
affects: [32-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wrapper pattern for composable BackOff decorators"
    - "Timer interface abstraction for testable sleeps"
    - "stdlib errors.As for error matching"

key-files:
  created:
    - backoff/timer.go
    - backoff/context.go
    - backoff/tries.go
    - backoff/retry.go
    - backoff/ticker.go
    - backoff/context_test.go
    - backoff/tries_test.go
    - backoff/retry_test.go
    - backoff/ticker_test.go
  modified: []

key-decisions:
  - "Use stdlib errors.As instead of github.com/pkg/errors"
  - "Timer interface enables testing without real delays"
  - "getContext helper extracts context through wrapper chain"

patterns-established:
  - "Wrapper pattern: WithContext, WithMaxRetries compose via embedding"
  - "Context extraction: getContext traverses wrapper chain"

# Metrics
duration: 4min
completed: 2026-02-01
---

# Phase 32 Plan 02: Wrappers and Retry Helpers Summary

**Context-aware wrappers (WithContext, WithMaxRetries), retry helpers (Retry, RetryNotify, PermanentError), and Ticker for periodic backoff-based operations using stdlib errors.As**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-01T20:31:13Z
- **Completed:** 2026-02-01T20:35:52Z
- **Tasks:** 2/2
- **Files created:** 9

## Accomplishments

- Created Timer interface for testable sleep abstractions
- Added WithContext() wrapper that returns Stop when context is cancelled
- Added WithMaxRetries() wrapper that limits retry attempts
- Implemented PermanentError type that stops retry immediately
- Created full Retry/RetryNotify/RetryWithData API with generics support
- Built Ticker for periodic backoff-based operations
- Used stdlib errors.As for PermanentError matching (not pkg/errors)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Timer, Context wrapper, and MaxRetries wrapper** - `f8d32ca` (feat)
2. **Task 2: Create PermanentError, Retry helpers, and Ticker** - `f25f5b9` (feat)

## Files Created/Modified

- `backoff/timer.go` - Timer interface wrapping time.Timer
- `backoff/context.go` - Context-aware backoff wrapper with getContext helper
- `backoff/tries.go` - MaxRetries wrapper limiting retry attempts
- `backoff/retry.go` - Retry helpers and PermanentError type
- `backoff/ticker.go` - Periodic backoff-based ticker
- `backoff/context_test.go` - Tests for WithContext wrapper
- `backoff/tries_test.go` - Tests for WithMaxRetries wrapper
- `backoff/retry_test.go` - Tests for Retry helpers and PermanentError
- `backoff/ticker_test.go` - Tests for Ticker

## Decisions Made

1. **stdlib errors.As for PermanentError** - Reference uses github.com/pkg/errors, but gaz uses stdlib consistently. errors.As works correctly with our PermanentError implementation.

2. **Timer interface design** - Matches reference implementation for drop-in compatibility with testable abstractions.

3. **getContext helper** - Traverses wrapper chain (backOffContext, backOffTries) to extract context. Used by both Ticker and retry functions.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Complete backoff toolkit now available
- Ready for 32-03-PLAN.md: Worker integration (replace jpillora/backoff)
- All exports match must_haves from plan

---
*Phase: 32-backoff-package*
*Completed: 2026-02-01*
