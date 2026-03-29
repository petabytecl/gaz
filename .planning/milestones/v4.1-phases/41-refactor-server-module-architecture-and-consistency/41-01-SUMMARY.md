---
phase: 41-refactor-server-module-architecture-and-consistency
plan: 01
subsystem: logging
tags: slog, logging, consistency, refactor
requires:
  - phase: 40-observability-health
    provides: health module, grpc server, otel provider
provides:
  - consistent logger handling across server packages
  - structured logging for health server errors
affects:
  - future server modules
tech-stack:
  added: []
  patterns:
    - "Nil logger fallback to slog.Default()"
key-files:
  created: []
  modified:
    - health/server.go
    - health/grpc.go
    - server/grpc/server.go
    - server/otel/provider.go
key-decisions:
  - "Use slog.Default() as fallback for all server components when logger is nil"
  - "Update NewManagementServer signature to inject logger directly"
  - "Remove direct stderr printing in favor of structured logging"
---

# Phase 41 Plan 01: Standardize Logger Usage Summary

**Standardized logger usage across server packages with consistent slog.Default() fallback and fixed health logging.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-02-03
- **Completed:** 2026-02-03
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Implemented consistent `nil` logger fallback to `slog.Default()` in `health`, `server/grpc`, and `server/otel` packages.
- Updated `NewManagementServer` to accept an injected logger, enabling proper structured logging.
- Replaced direct `fmt.Fprintf(os.Stderr)` in health server with `logger.ErrorContext`, ensuring logs follow the application's log format (JSON/Text).
- Verified `server/http` already adhered to the standard.

## Task Commits

1. **Task 1 & 2 (Health):** `10220d6` (feat: standardize logger fallback in health package)
2. **Task 1 (gRPC):** `d5351af` (feat: standardize logger fallback in grpc server)
3. **Task 1 (OTEL):** `69354cc` (feat: standardize logger fallback in otel provider)

## Files Created/Modified
- `health/server.go` - Added logger field, updated constructor, fixed logging
- `health/grpc.go` - Added logger fallback
- `server/grpc/server.go` - Added logger fallback
- `server/otel/provider.go` - Added logger fallback, removed redundant checks
- `health/module.go` - Updated module to pass logger to server
- `health/module_test.go` - Updated tests
- `health/server_test.go` - Updated tests

## Decisions Made
- **Inject Logger in Health Server:** Changed `NewManagementServer` signature to explicitly accept a logger. This is a breaking change for the internal API but necessary for consistent observability.
- **Graceful Fallback:** All server components now safely handle `nil` loggers by falling back to `slog.Default()`, improving developer experience and reducing nil panics.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated `health/module.go` and tests**
- **Found during:** Task 1 (Health server update)
- **Issue:** Changing `NewManagementServer` signature broke the module and tests.
- **Fix:** Updated `health/module.go` to resolve and pass the logger. Updated tests to pass `nil` or logger as needed.
- **Files modified:** `health/module.go`, `health/module_test.go`, `health/server_test.go`
- **Verification:** `make test` passed.
- **Committed in:** `10220d6`

## Issues Encountered
None.

## Next Phase Readiness
- Server packages now have consistent logging patterns.
- Ready for further refactoring or feature additions in Phase 41.

---
*Phase: 41-refactor-server-module-architecture-and-consistency*
*Completed: 2026-02-03*
