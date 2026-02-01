---
phase: 32-backoff-package
plan: 03
subsystem: backoff
tags: [backoff, worker, migration, dependency-removal]

# Dependency graph
requires:
  - phase: 32-02
    provides: Complete internal backoff package with wrappers and retry helpers
provides:
  - Worker package migrated to internal backoff
  - jpillora/backoff dependency removed from go.mod
  - All tests pass with internal backoff implementation
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "API migration: Duration() → NextBackOff()"
    - "Config wrapper delegates to internal backoff options"

key-files:
  created: []
  modified:
    - worker/supervisor.go
    - worker/backoff.go
    - worker/backoff_test.go
    - worker/doc.go
    - go.mod
    - go.sum

key-decisions:
  - "Keep BackoffConfig for API compatibility - users may configure via options"
  - "Jitter: true maps to RandomizationFactor 0.5, false maps to 0"

patterns-established:
  - "Dependency migration: update imports, change types, update method calls"

# Metrics
duration: 4min
completed: 2026-02-01
---

# Phase 32 Plan 03: Worker Integration Summary

**Worker package migrated to internal backoff, jpillora/backoff dependency removed from go.mod**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-01T20:38:35Z
- **Completed:** 2026-02-01T20:42:23Z
- **Tasks:** 2/2
- **Files modified:** 6

## Accomplishments

- Updated worker/supervisor.go to import internal backoff package
- Changed supervisor backoff type from `*backoff.Backoff` to `*backoff.ExponentialBackOff`
- Replaced `Duration()` calls with `NextBackOff()` (API mapping from RESEARCH.md)
- Simplified BackoffConfig.NewBackoff() to return internal ExponentialBackOff
- Removed jpillora/backoff from go.mod via `go mod tidy`
- Updated worker/doc.go comment to reference internal backoff
- All tests pass including race detector

## Task Commits

Each task was committed atomically:

1. **Task 1: Update worker package to use internal backoff** - `4bd6d01` (feat)
2. **Task 2: Remove jpillora/backoff from go.mod and verify** - `6e98f72` (chore)

## Files Created/Modified

- `worker/supervisor.go` - Import updated, type changed, NextBackOff() used
- `worker/backoff.go` - Simplified to use internal backoff package
- `worker/backoff_test.go` - Tests updated for new API
- `worker/doc.go` - Comment updated to reference internal backoff
- `go.mod` - jpillora/backoff removed from require block
- `go.sum` - Checksums updated

## Decisions Made

1. **Keep BackoffConfig wrapper** - Preserved for API compatibility. Users can still configure via WithBackoffMin, WithBackoffMax, etc. The NewBackoff() method now returns the internal ExponentialBackOff.

2. **Jitter mapping** - BackoffConfig.Jitter (bool) maps to RandomizationFactor:
   - `true` → 0.5 (default, ±50% jitter)
   - `false` → 0 (no randomization)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated worker/doc.go comment**
- **Found during:** Task 2 (grep found reference to jpillora/backoff)
- **Issue:** Comment still referenced "jpillora/backoff"
- **Fix:** Updated to reference "internal backoff.ExponentialBackOff"
- **Files modified:** worker/doc.go
- **Verification:** grep confirms no references remain
- **Committed in:** 6e98f72

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Minor documentation update required for complete migration.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 32 (Backoff Package) complete: 3/3 plans done
- Internal backoff package fully implemented and integrated
- jpillora/backoff dependency removed from project
- Ready for Phase 33 (Tint Package)

---
*Phase: 32-backoff-package*
*Completed: 2026-02-01*
