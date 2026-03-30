---
phase: 51-design-and-api-improvements
plan: 01
subsystem: api
tags: [worker, backoff, config, safety, observability]

requires:
  - phase: 48-server-module-gateway-removal
    provides: stable codebase post v5.0

provides:
  - Pool size upper bound validation (MaxPoolSize=1024)
  - Dead letter stack trace capture (LastPanicStack)
  - Safe SetEnvKeyReplacer error return (no panic)
  - Correct backoff jitter calculation (no off-by-one)

affects: [worker, config, backoff]

tech-stack:
  added: []
  patterns: [clamp-validation, error-return-over-panic]

key-files:
  created: []
  modified:
    - worker/options.go
    - worker/supervisor.go
    - config/backend.go
    - config/viper/backend.go
    - config/manager.go
    - backoff/exponential.go

key-decisions:
  - "Clamp pool size to MaxPoolSize instead of rejecting with error (simpler API)"
  - "SetEnvKeyReplacer returns error instead of panicking (safer API contract)"
  - "Store panic stack trace as string in supervisor struct (simple, no extra allocation on happy path)"

patterns-established:
  - "Clamp-validation: options that exceed bounds are silently clamped rather than errored"

requirements-completed: [DSGN-05, DSGN-07, DSGN-08, DSGN-11]

duration: 5min
completed: 2026-03-30
---

# Phase 51 Plan 01: Design & API Improvements Summary

**Pool size bounds (MaxPoolSize=1024), dead letter stack traces, SetEnvKeyReplacer error return, backoff jitter off-by-one fix**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-30T00:22:22Z
- **Completed:** 2026-03-30T00:27:20Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- WithPoolSize now clamps values above 1024 (MaxPoolSize constant)
- DeadLetterInfo.LastPanicStack captures goroutine stack trace on worker panic
- SetEnvKeyReplacer returns error instead of panicking for non-*strings.Replacer
- Backoff jitter calculation no longer exceeds MaxInterval (removed off-by-one +1)

## Task Commits

Each task was committed atomically:

1. **Task 1: Pool size validation, config error return, dead letter stack trace**
   - `be0f5a1` (test: RED - failing tests)
   - `8b0d83d` (feat: GREEN - implementation)
2. **Task 2: Fix backoff jitter off-by-one**
   - `3ee6949` (test: RED - failing test)
   - `1a7ed4c` (fix: GREEN - remove +1)
3. **Lint fix** - `d921473` (fix: perfsprint warnings)

## Files Created/Modified
- `worker/options.go` - Added MaxPoolSize constant, clamp in WithPoolSize, LastPanicStack in DeadLetterInfo
- `worker/supervisor.go` - Store panic stack trace, pass to dead letter handler
- `worker/options_test.go` - Tests for pool size bounds
- `worker/supervisor_test.go` - Tests for LastPanicStack (panic vs error cases)
- `config/backend.go` - Updated EnvBinder interface: SetEnvKeyReplacer returns error
- `config/viper/backend.go` - Error return instead of panic
- `config/manager.go` - Handle SetEnvKeyReplacer error in Load()
- `config/viper/backend_test.go` - Test for non-*strings.Replacer error return
- `backoff/exponential.go` - Fixed jitter off-by-one
- `backoff/exponential_test.go` - Precise jitter boundary tests

## Decisions Made
- Clamp pool size to MaxPoolSize instead of rejecting with error (simpler API, consistent with existing WithPoolSize behavior for n<=0)
- SetEnvKeyReplacer returns error instead of panicking (safer API contract, breaking change to interface)
- Store panic stack as string field on supervisor struct (simple, no allocation on happy path)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed perfsprint lint warnings**
- **Found during:** Post-task verification
- **Issue:** `fmt.Errorf` used for static error strings, flagged by perfsprint linter
- **Fix:** Changed to `errors.New` in config/viper/backend.go and worker/supervisor_test.go
- **Files modified:** config/viper/backend.go, worker/supervisor_test.go
- **Committed in:** d921473

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Lint compliance fix, no scope creep.

## Issues Encountered
- Pre-existing gocognit warning on config/manager.go Load() (cognitive complexity 22 > 20) - not introduced by this plan, not fixed (out of scope)

## Next Phase Readiness
- All four fixes complete and tested
- No blockers

---
*Phase: 51-design-and-api-improvements*
*Completed: 2026-03-30*
