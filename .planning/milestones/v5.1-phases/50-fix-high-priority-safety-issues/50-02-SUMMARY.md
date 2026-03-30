---
phase: 50-fix-high-priority-safety-issues
plan: 02
subsystem: logger
tags: [security, slog, middleware, request-id, file-handle]

requires:
  - phase: none
    provides: existing logger package
provides:
  - X-Request-ID header injection prevention
  - ContextHandler WithAttrs/WithGroup delegation
  - NewLoggerWithCloser for file handle lifecycle management
affects: [logger, app]

tech-stack:
  added: []
  patterns: [request-id-validation, handler-delegation-wrapping, closer-pattern]

key-files:
  created: []
  modified:
    - logger/middleware.go
    - logger/middleware_test.go
    - logger/handler.go
    - logger/handler_test.go
    - logger/provider.go
    - logger/logger_test.go

key-decisions:
  - "Added isValidRequestID with regexp instead of manual char checking for maintainability"
  - "NewLoggerWithCloser as new function preserving backward compatibility of existing NewLogger"
  - "nopCloser type for stdout/stderr instead of returning nil closer"

patterns-established:
  - "Request ID validation: regexp-based allowlist (alphanumeric + dash/underscore/dot, max 64 chars)"
  - "Handler wrapping: WithAttrs/WithGroup return new ContextHandler preserving context propagation"
  - "Closer pattern: NewXxxWithCloser returns (resource, io.Closer) for lifecycle tracking"

requirements-completed: [SAFE-03, SAFE-05, SAFE-06]

duration: 4min
completed: 2026-03-29
---

# Phase 50 Plan 02: Logger Safety Fixes Summary

**X-Request-ID injection prevention via regexp validation, ContextHandler WithAttrs/WithGroup delegation fix, and NewLoggerWithCloser for file handle leak prevention**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-29T20:49:59Z
- **Completed:** 2026-03-29T20:54:04Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- X-Request-ID values validated against injection attacks (max 64 chars, alphanumeric+dash+underscore+dot)
- ContextHandler properly delegates WithAttrs and WithGroup, fixing silent attribute loss in structured logging
- NewLoggerWithCloser API enables file handle tracking and cleanup on shutdown

## Task Commits

Each task was committed atomically:

1. **Task 1: Validate X-Request-ID and fix ContextHandler WithAttrs/WithGroup** - `eba804d` (fix)
2. **Task 2: Fix logger file handle leak** - `aa5d19e` (feat)
3. **Lint fixes** - `5a1653c` (refactor)

## Files Created/Modified
- `logger/middleware.go` - Added isValidRequestID with compiled regexp, reject malformed IDs in RequestIDMiddleware
- `logger/middleware_test.go` - Tests for ID validation (oversized, special chars, valid chars)
- `logger/handler.go` - Added WithAttrs and WithGroup methods on ContextHandler
- `logger/handler_test.go` - Tests for WithAttrs/WithGroup delegation and context propagation preservation
- `logger/provider.go` - Added NewLoggerWithCloser, resolveOutputWithCloser, nopCloser
- `logger/logger_test.go` - Tests for NewLoggerWithCloser with file, stdout, stderr outputs

## Decisions Made
- Used regexp `^[a-zA-Z0-9\-_.]{1,64}$` for request ID validation -- simple, fast, covers common ID formats
- Created NewLoggerWithCloser as new function rather than changing NewLogger signature to preserve backward compatibility
- Used nopCloser struct (not nil) so callers can always call Close() without nil checks

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Split test functions to fix gocognit lint violation**
- **Found during:** Task 1 verification
- **Issue:** TestRequestIDMiddleware had cognitive complexity 25 (limit 20) due to nested subtests
- **Fix:** Split into separate top-level test functions
- **Files modified:** logger/middleware_test.go
- **Verification:** `make lint` passes for logger files
- **Committed in:** 5a1653c

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Test restructuring for lint compliance. No scope creep.

## Deferred Items
- `app.go` uses `logger.NewLogger` -- should be updated to `NewLoggerWithCloser` with closer wired to shutdown. Requires App struct changes (architectural scope).
- Pre-existing nolintlint warnings in 5 module files (cron, eventbus, health, logger, worker) -- not related to this plan

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all implementations are complete.

## Next Phase Readiness
- Logger package safety issues resolved
- NewLoggerWithCloser available for framework integration in a future plan

---
*Phase: 50-fix-high-priority-safety-issues*
*Completed: 2026-03-29*
