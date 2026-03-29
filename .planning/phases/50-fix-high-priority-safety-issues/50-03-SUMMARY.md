---
phase: 50-fix-high-priority-safety-issues
plan: 03
subsystem: server
tags: [vanguard, health, slowloris, security, timeout]

requires:
  - phase: 46-core-vanguard-server
    provides: Vanguard server with buildHealthMux and Config
provides:
  - Dynamic health endpoint paths from health.Config in Vanguard server
  - Slowloris protection via WriteTimeout validation with explicit opt-out
affects: [server-vanguard, health]

tech-stack:
  added: []
  patterns:
    - "Explicit opt-in for dangerous defaults (AllowZeroWriteTimeout pattern)"

key-files:
  created: []
  modified:
    - server/vanguard/health.go
    - server/vanguard/server.go
    - server/vanguard/config.go
    - server/vanguard/config_test.go
    - server/vanguard/server_test.go
    - server/vanguard/module_test.go

key-decisions:
  - "buildHealthMux takes *health.Config parameter; nil falls back to health.DefaultConfig()"
  - "Added mountHealthEndpoints for direct registration on unknownMux in OnStart"
  - "AllowZeroWriteTimeout defaults to false; forces explicit choice for streaming"
  - "DefaultConfig() keeps WriteTimeout=0 but Validate() rejects it without opt-in"

patterns-established:
  - "Explicit opt-in for security-sensitive zero values: AllowZeroX bool field + Validate() gate"

requirements-completed: [SAFE-04, SAFE-07]

duration: 5min
completed: 2026-03-29
---

# Phase 50 Plan 03: Vanguard Health Path and Slowloris Fix Summary

**Dynamic health paths from health.Config replace hardcoded /healthz /readyz /livez, and WriteTimeout=0 now requires explicit AllowZeroWriteTimeout opt-in**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-29T20:50:18Z
- **Completed:** 2026-03-29T20:55:16Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Health endpoints now use paths from health.Config (/live, /ready, /startup by default) instead of hardcoded K8s paths
- Fixed bug where /healthz mapped to NewReadinessHandler instead of NewLivenessHandler
- Added Slowloris protection: Validate() rejects WriteTimeout=0 unless AllowZeroWriteTimeout=true
- Added startup path handler support (previously missing from buildHealthMux)

## Task Commits

Each task was committed atomically:

1. **Task 1: Use health.Config paths in buildHealthMux** - `b440afa` (fix)
2. **Task 2: Add Slowloris protection with AllowZeroWriteTimeout** - `6d765d9` (fix)

## Files Created/Modified
- `server/vanguard/health.go` - buildHealthMux accepts *health.Config; added mountHealthEndpoints helper
- `server/vanguard/server.go` - Resolves health.Config from DI; uses mountHealthEndpoints in OnStart
- `server/vanguard/config.go` - AllowZeroWriteTimeout field; Validate() Slowloris check; updated doc comments
- `server/vanguard/config_test.go` - Tests for zero/positive WriteTimeout with/without opt-in
- `server/vanguard/server_test.go` - Updated buildHealthMux tests for new signature; added custom path tests
- `server/vanguard/module_test.go` - Updated test fixture to opt in to zero WriteTimeout

## Decisions Made
- buildHealthMux receives *health.Config (nil = use health.DefaultConfig defaults)
- Added separate mountHealthEndpoints function for direct unknownMux registration (avoids nested mux delegation)
- AllowZeroWriteTimeout defaults to false -- forces developer to make an explicit streaming decision
- Used errors.New instead of fmt.Errorf for static Slowloris error message (linter compliance)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed module_test.go config validation failure**
- **Found during:** Task 2 (WriteTimeout validation)
- **Issue:** TestProvideConfig_RegistersConfig used DefaultConfig() which now fails Validate() with zero WriteTimeout
- **Fix:** Added AllowZeroWriteTimeout=true to test fixture
- **Files modified:** server/vanguard/module_test.go
- **Verification:** go test -race ./server/vanguard/ passes
- **Committed in:** 6d765d9 (Task 2 commit)

**2. [Rule 1 - Bug] Fixed fmt.Errorf lint error for static string**
- **Found during:** Task 2 verification (make lint)
- **Issue:** perfsprint linter rejects fmt.Errorf for strings without format verbs
- **Fix:** Changed to errors.New
- **Files modified:** server/vanguard/config.go
- **Verification:** make lint passes with 0 issues
- **Committed in:** 6d765d9 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for test/lint correctness. No scope creep.

## Deferred Items

- OTEL middleware (middleware.go:148) still has hardcoded health path checks (/healthz, /readyz, /livez) for trace filtering. This is cosmetic (trace noise reduction) not functional, and does not affect correctness. Should be updated in a follow-up to use health.Config paths.
- Doc comments in doc.go reference old paths (/healthz, /readyz, /livez). Should be updated in a documentation pass.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Vanguard health paths are now config-driven and consistent with the health management server
- WriteTimeout validation enforces explicit streaming opt-in for security
- OTEL trace filter paths should be updated in a follow-up plan to use dynamic health.Config paths

---
*Phase: 50-fix-high-priority-safety-issues*
*Completed: 2026-03-29*
