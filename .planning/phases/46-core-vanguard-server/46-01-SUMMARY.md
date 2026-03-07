---
phase: 46-core-vanguard-server
plan: 01
subsystem: server
tags: [connect-go, grpc, di, auto-discovery, skip-listener]

# Dependency graph
requires: []
provides:
  - connect.Registrar interface for Connect-Go service auto-discovery
  - gRPC server skip-listener mode for Vanguard integration
  - registerServices() shared helper for service/health/reflection setup
affects: [46-02-PLAN, vanguard-server]

# Tech tracking
tech-stack:
  added: []
  patterns: [registrar-interface, skip-listener-mode, di-auto-discovery]

key-files:
  created:
    - server/connect/doc.go
    - server/connect/registrar.go
    - server/connect/registrar_test.go
  modified:
    - server/grpc/config.go
    - server/grpc/server.go
    - server/grpc/server_test.go

key-decisions:
  - "Renamed ConnectRegistrar to connect.Registrar to avoid golangci-lint stutter (matches grpc.Registrar pattern)"
  - "Extracted registerServices() helper to eliminate duplication between OnStart and onStartSkipListener"

patterns-established:
  - "Registrar interface pattern: single method returning (string, http.Handler) for Connect-Go auto-discovery"
  - "Skip-listener mode: server registers services/health/reflection without binding a port"

requirements-completed: [CONN-01]

# Metrics
duration: 5min
completed: 2026-03-06
---

# Phase 46 Plan 01: Connect Registrar & Skip-Listener Summary

**Connect-Go Registrar interface in server/connect/ with RegisterConnect() signature, and gRPC skip-listener mode that registers services without binding a port**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-06T19:55:53Z
- **Completed:** 2026-03-06T20:01:21Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Created `server/connect/` package with `Registrar` interface (`RegisterConnect() (string, http.Handler)`) for auto-discovery of Connect-Go services via DI
- Added `SkipListener` config field, `--grpc-skip-listener` CLI flag, and skip-listener mode to gRPC server (services register but no port binds)
- Extracted `registerServices()` helper to eliminate code duplication between normal and skip-listener startup paths

## Task Commits

Each task was committed atomically:

1. **Task 1: Create ConnectRegistrar interface** — TDD
   - `1fbfd39` (feat) — RED+GREEN: interface + compliance tests
2. **Task 2: Add skip-listener mode to gRPC server** — TDD
   - `77945d2` (test) — RED: failing skip-listener tests
   - `f1ac747` (feat) — GREEN: implement skip-listener mode
   - `f78ac8c` (refactor) — REFACTOR: extract registerServices()

## Files Created/Modified
- `server/connect/doc.go` — Package documentation for Connect-Go integration
- `server/connect/registrar.go` — `Registrar` interface with `RegisterConnect() (string, http.Handler)`
- `server/connect/registrar_test.go` — Interface compliance tests (3 tests)
- `server/grpc/config.go` — Added `SkipListener` field, CLI flag, validation skip
- `server/grpc/server.go` — Added `onStartSkipListener()`, `registerServices()` helper, SkipListener branch in `OnStop`
- `server/grpc/server_test.go` — Added 5 skip-listener tests (start/stop, reflection, no port binding, config validation, config flag)

## Decisions Made
- **Renamed ConnectRegistrar to Registrar:** `golangci-lint` flagged `connect.ConnectRegistrar` as stuttering. Renamed to `connect.Registrar` to match the `grpc.Registrar` pattern already in the codebase. The plan's `must_haves` reference `ConnectRegistrar` but `connect.Registrar` is the Go-idiomatic equivalent.
- **Extracted registerServices() helper:** During TDD REFACTOR, identified that `OnStart` and `onStartSkipListener` duplicated service discovery, health registration, and reflection logic. Extracted into a shared `registerServices(ctx) (int, error)` method.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Renamed ConnectRegistrar to Registrar for lint compliance**
- **Found during:** Task 1 (ConnectRegistrar interface creation)
- **Issue:** `golangci-lint` flagged `connect.ConnectRegistrar` as stuttering (package name repeated in type name)
- **Fix:** Renamed to `connect.Registrar`, matching the existing `grpc.Registrar` convention
- **Files modified:** `server/connect/registrar.go`, `server/connect/registrar_test.go`
- **Verification:** `golangci-lint run ./server/connect/...` passes with 0 issues
- **Committed in:** `1fbfd39` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 linter compliance)
**Impact on plan:** Naming change only. Interface contract (`RegisterConnect() (string, http.Handler)`) unchanged. No scope creep.

## Issues Encountered
None

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `connect.Registrar` interface ready for Plan 02 (Vanguard server) to use with `di.ResolveAll[connect.Registrar]`
- gRPC `SkipListener` mode ready for Plan 02 to configure when Vanguard handles all connections
- `registerServices()` helper provides clean separation for Vanguard to call gRPC registration without port binding

---
*Phase: 46-core-vanguard-server*
*Completed: 2026-03-06*
