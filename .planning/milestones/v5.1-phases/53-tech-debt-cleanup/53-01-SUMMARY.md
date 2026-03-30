---
phase: 53-tech-debt-cleanup
plan: 01
subsystem: infra
tags: [logger, otel, health, lifecycle, middleware]

requires:
  - phase: 48-server-module-gateway-removal
    provides: Vanguard server module with OTEL middleware and health endpoints
provides:
  - Logger file handle closing on App shutdown (SAFE-06 gap closed)
  - OTEL middleware trace filter using configurable health.Config paths
  - Correct doc.go health path documentation
affects: [server, logger, app-lifecycle]

tech-stack:
  added: []
  patterns:
    - "NewLoggerWithCloser returns (logger, io.Closer) for resource cleanup"
    - "OTEL middleware accepts health.Config for path-based trace filtering"

key-files:
  created: []
  modified:
    - app.go
    - logger/provider.go
    - server/vanguard/middleware.go
    - server/vanguard/module.go
    - server/vanguard/doc.go

key-decisions:
  - "nopCloser type for stdout/stderr instead of returning nil closer"
  - "Logger closer called after service shutdown but before Run exit signal"
  - "provideOTELMiddleware falls back to health.DefaultConfig when health module not registered"

patterns-established:
  - "NewLoggerWithCloser for callers that need to manage logger output lifetime"

requirements-completed: []

duration: 9min
completed: 2026-03-30
---

# Phase 53 Plan 01: Tech Debt Cleanup Summary

**Logger file handle leak fix, OTEL health path filter using health.Config, and doc.go path correction**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-30T01:45:20Z
- **Completed:** 2026-03-30T01:54:12Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Logger file handles are now closed during App shutdown via NewLoggerWithCloser + logCloser field
- OTEL middleware trace filter uses health.Config paths (/live, /ready, /startup) instead of hardcoded old paths (/healthz, /readyz, /livez)
- doc.go health endpoint references corrected to match health.DefaultConfig() values

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire logger closer + OTEL health paths** - `1fd6a54` (test: RED) + `d7b13ab` (feat: GREEN)
2. **Task 2: Fix doc.go health path references** - `e0c427e` (docs)

## Files Created/Modified
- `app.go` - Added logCloser field, io import, closer cleanup in doStop
- `logger/provider.go` - Added NewLoggerWithCloser, nopCloser, resolveOutputWithCloser
- `logger/logger_test.go` - Tests for NewLoggerWithCloser (stdout, stderr, file, empty)
- `app_test.go` - Tests for logCloser lifecycle (file output, stdout, Build storage)
- `server/vanguard/middleware.go` - OTELMiddleware uses health.Config for path filtering
- `server/vanguard/middleware_test.go` - Tests for health path filtering (default, old, custom)
- `server/vanguard/module.go` - provideOTELMiddleware resolves health.Config from DI
- `server/vanguard/module_test.go` - Tests for provideOTELMiddleware with/without health.Config
- `server/vanguard/doc.go` - Health path references updated to /ready, /live, /startup

## Decisions Made
- Used nopCloser type for stdout/stderr instead of returning nil -- avoids nil checks at every call site
- Logger closer is called after all services and workers are stopped but before signaling Run to exit -- services may still log during shutdown
- provideOTELMiddleware resolves health.Config from DI with fallback to health.DefaultConfig() -- works both with and without health module

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created NewLoggerWithCloser function**
- **Found during:** Task 1
- **Issue:** Plan references logger.NewLoggerWithCloser but the function did not exist in the codebase
- **Fix:** Implemented NewLoggerWithCloser, nopCloser, and resolveOutputWithCloser in logger/provider.go
- **Files modified:** logger/provider.go
- **Verification:** All NewLoggerWithCloser tests pass
- **Committed in:** d7b13ab

**2. [Rule 3 - Blocking] No separate app_build.go/app_shutdown.go files**
- **Found during:** Task 1
- **Issue:** Plan references app_build.go and app_shutdown.go but all code is in app.go
- **Fix:** Applied changes to initializeLogger and doStop methods in app.go
- **Files modified:** app.go
- **Verification:** All app tests pass
- **Committed in:** d7b13ab

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both auto-fixes necessary for implementation. No scope creep.

## Issues Encountered
- gci linter required import grouping fix in module_test.go -- resolved by moving sdktrace import into third-party group

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 3 tech debt items resolved
- Logger handles properly cleaned up during shutdown
- OTEL trace filtering stays in sync with health endpoint configuration
- All tests pass, lint clean

---
*Phase: 53-tech-debt-cleanup*
*Completed: 2026-03-30*
