---
phase: 14-workers
plan: 01
subsystem: worker
tags: [worker, backoff, jpillora, lifecycle, goroutine]

# Dependency graph
requires:
  - phase: 13-config
    provides: Package extraction patterns (options, doc.go structure)
provides:
  - Worker interface with Start(), Stop(), Name() methods
  - WorkerOptions for registration configuration
  - BackoffConfig wrapping jpillora/backoff
  - jpillora/backoff dependency in go.mod
affects: [14-02 WorkerManager, 14-03 App integration]

# Tech tracking
tech-stack:
  added: [github.com/jpillora/backoff v1.0.0]
  patterns: [Option function pattern, Interface-based workers]

key-files:
  created:
    - worker/worker.go
    - worker/options.go
    - worker/backoff.go
    - worker/doc.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Worker interface matches CONTEXT.md locked decision (Start/Stop/Name)"
  - "BackoffConfig defaults from RESEARCH.md (1s min, 5m max, factor 2, jitter)"
  - "WorkerOptions defaults from RESEARCH.md (30s stable, 5 restarts, 10m window)"

patterns-established:
  - "Worker interface: non-blocking Start(), signaling Stop(), required Name()"
  - "Option functions for WorkerOptions and BackoffConfig following config/options.go"

# Metrics
duration: 4min
completed: 2026-01-28
---

# Phase 14 Plan 01: Worker Foundation Summary

**Worker interface with Start/Stop/Name lifecycle, registration options with pool/critical settings, and BackoffConfig wrapping jpillora/backoff with sensible defaults**

## Performance

- **Duration:** 4 min
- **Started:** 2026-01-28T20:25:46Z
- **Completed:** 2026-01-28T20:28:01Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Worker interface defined with Start(), Stop(), Name() methods per CONTEXT.md specification
- WorkerOptions with PoolSize, Critical, StableRunPeriod, MaxRestarts, CircuitWindow
- BackoffConfig wrapping jpillora/backoff with sensible defaults from RESEARCH.md
- Complete package documentation with usage examples

## Task Commits

Each task was committed atomically:

1. **Task 1: Create Worker interface and package documentation** - `7fbf3a8` (feat)
2. **Task 2: Create registration options and backoff configuration** - `06ae0bc` (feat)

**Plan metadata:** (pending)

## Files Created/Modified

- `worker/worker.go` - Worker interface with Start(), Stop(), Name() methods
- `worker/doc.go` - Package documentation with usage examples
- `worker/options.go` - WorkerOptions struct and option functions
- `worker/backoff.go` - BackoffConfig wrapping jpillora/backoff
- `go.mod` - Added jpillora/backoff dependency
- `go.sum` - Updated checksums

## Decisions Made

- Worker interface matches CONTEXT.md locked decision exactly (Start/Stop/Name, not Run(ctx))
- BackoffConfig defaults match RESEARCH.md: 1s min, 5m max, factor 2, jitter true
- WorkerOptions defaults match RESEARCH.md: 30s stable period, 5 max restarts, 10m circuit window
- Option functions follow existing pattern from config/options.go

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- Worker interface ready for WorkerManager implementation (14-02)
- Options and backoff ready for supervisor integration
- Package compiles successfully with jpillora/backoff dependency

---
*Phase: 14-workers*
*Completed: 2026-01-28*
