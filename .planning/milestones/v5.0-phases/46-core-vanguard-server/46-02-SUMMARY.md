---
phase: 46-core-vanguard-server
plan: 02
subsystem: server
tags: [vanguard, grpc, connect, grpc-web, rest, h2c, transcoder, reflection, health]

# Dependency graph
requires:
  - phase: 46-core-vanguard-server/01
    provides: "connect.Registrar interface, gRPC skip-listener mode, *grpc.Server wrapper with GRPCServer() accessor"
provides:
  - "Vanguard server composing gRPC + Connect + gRPC-Web + REST on single h2c port"
  - "Health endpoint auto-mounting (/healthz, /readyz, /livez) via unknown handler"
  - "gRPC reflection v1 and v1alpha registration for grpcurl compatibility"
  - "SetUnknownHandler for custom non-RPC HTTP routes"
  - "NewModule() DI registration for Config (server namespace) and Server (Eager)"
affects: [47-middleware-interceptors, 48-server-module-gateway-removal]

# Tech tracking
tech-stack:
  added: [connectrpc.com/vanguard@v0.4.0, connectrpc.com/grpcreflect@v1.3.0, connectrpc.com/connect@v1.19.1]
  patterns: [vanguardgrpc-transcoder-composition, connect-mux-unknown-handler, h2c-via-http-protocols]

key-files:
  created:
    - server/vanguard/doc.go
    - server/vanguard/config.go
    - server/vanguard/config_test.go
    - server/vanguard/server.go
    - server/vanguard/server_test.go
    - server/vanguard/health.go
    - server/vanguard/module.go
    - server/vanguard/module_test.go
  modified:
    - .golangci.yml
    - go.mod
    - go.sum

key-decisions:
  - "Used vanguardgrpc.NewTranscoder pattern — transcoder wraps gRPC server with Connect mux as unknown handler, rather than composing services into a list"
  - "Health endpoints mounted via buildHealthMux helper on unknown handler mux, not as Vanguard services"
  - "Added connectrpc.com packages to depguard allow lists and vanguard to ireturn exclusion in .golangci.yml"

patterns-established:
  - "Vanguard composition: gRPC transcoder wraps raw grpc.Server, Connect services + reflection + health compose into http.ServeMux as unknown handler"
  - "h2c via Go 1.26 http.Protocols: SetHTTP1(true) + SetUnencryptedHTTP2(true) — no x/net/http2/h2c dependency"
  - "Config Namespace 'server' shared across vanguard server (same prefix as gRPC when in skip-listener mode)"

requirements-completed: [USRV-01, USRV-02, USRV-03, USRV-04, USRV-05, USRV-06, CONN-04, MDDL-05]

# Metrics
duration: 7min
completed: 2026-03-06
---

# Phase 46 Plan 02: Vanguard Server Summary

**Single-port Vanguard server composing gRPC, Connect, gRPC-Web, and REST via vanguardgrpc transcoder with h2c, health auto-mount, and reflection registration**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-06T20:07:00Z
- **Completed:** 2026-03-06T20:14:31Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Created Vanguard config with streaming-safe zero timeouts (ReadTimeout=0, WriteTimeout=0) and "server" namespace
- Built server that composes Connect services + gRPC bridge via `vanguardgrpc.NewTranscoder` on a single h2c port
- Health endpoints (/healthz, /readyz, /livez) auto-mount when health.Manager present via unknown handler pattern
- gRPC reflection v1 and v1alpha registered for grpcurl compatibility
- SetUnknownHandler allows custom non-RPC HTTP routes on the same port
- NewModule() wires Config and Server (Eager) with DI

## Task Commits

Each task was committed atomically:

1. **Task 1: Vanguard config** - `b5cd4eb` (feat) — doc.go, config.go, config_test.go with 20 tests
2. **Task 2: Vanguard server, health, module** - `f19e41d` (feat) — server.go, server_test.go, health.go, module.go, module_test.go with 15+ tests

**Plan metadata:** (pending — final docs commit)

_Note: TDD tasks — both followed RED/GREEN pattern_

## Files Created/Modified
- `server/vanguard/doc.go` — Package documentation
- `server/vanguard/config.go` — Config struct with streaming-safe defaults, Namespace returns "server"
- `server/vanguard/config_test.go` — 20 config tests (defaults, flags, validation, namespace)
- `server/vanguard/server.go` — Server with OnStart/OnStop lifecycle, vanguardgrpc bridge, Connect mux, reflection, health
- `server/vanguard/server_test.go` — 15 server tests (start/stop, connect discovery, health auto-mount, unknown handler, port binding)
- `server/vanguard/health.go` — buildHealthMux helper for health endpoint mounting
- `server/vanguard/module.go` — DI module with provideConfig, provideServer (Eager), NewModule()
- `server/vanguard/module_test.go` — 3 module tests
- `.golangci.yml` — Added connectrpc.com packages to depguard allow lists, vanguard to ireturn exclusion
- `go.mod` — Added connectrpc.com/vanguard, grpcreflect, connect dependencies
- `go.sum` — Updated checksums

## Decisions Made
- **vanguardgrpc.NewTranscoder pattern:** The plan assumed `NewTranscoder` returns `[]*vanguard.Service` to append, but it actually returns a self-contained `*vanguard.Transcoder`. Adapted architecture: transcoder wraps gRPC server with Connect mux as unknown handler.
- **Health via buildHealthMux:** Separated health endpoint logic into `health.go` with `buildHealthMux` helper that creates a dedicated mux, mounted on the server's unknown handler mux.
- **depguard updates:** Added `connectrpc.com/connect`, `connectrpc.com/vanguard`, `connectrpc.com/grpcreflect` to both deprecated and non-test allow lists in `.golangci.yml`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated depguard allow lists in .golangci.yml**
- **Found during:** Task 2 (server implementation)
- **Issue:** connectrpc.com packages not in depguard allow lists, lint would fail
- **Fix:** Added connectrpc.com/connect, connectrpc.com/vanguard, connectrpc.com/grpcreflect to both deprecated and non-test allow lists
- **Files modified:** .golangci.yml
- **Verification:** golangci-lint run passes clean
- **Committed in:** f19e41d (Task 2 commit)

**2. [Rule 3 - Blocking] Added vanguard to ireturn exclusion**
- **Found during:** Task 2 (module implementation)
- **Issue:** NewModule() returns gaz.Module interface, ireturn linter would flag it
- **Fix:** Added `server/vanguard` to existing ireturn exclusion pattern
- **Files modified:** .golangci.yml
- **Verification:** golangci-lint run passes clean
- **Committed in:** f19e41d (Task 2 commit)

**3. [Rule 1 - Bug] Adapted vanguardgrpc.NewTranscoder API**
- **Found during:** Task 2 (transcoder composition)
- **Issue:** Plan assumed NewTranscoder returns services to append, but it returns a self-contained transcoder
- **Fix:** Used transcoder-wraps-gRPC-with-connect-mux-as-unknown-handler pattern
- **Files modified:** server/vanguard/server.go
- **Verification:** All tests pass including connect service discovery
- **Committed in:** f19e41d (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking)
**Impact on plan:** All auto-fixes necessary for correctness and lint compliance. No scope creep.

## Issues Encountered
None — all issues handled via deviation rules above.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- Vanguard server package complete with all core functionality
- Ready for Phase 47 (Middleware & Interceptors) — CORS, OTEL, Connect interceptor bundles
- Ready for Phase 48 (Server Module & Gateway Removal) — server.NewModule() bundling

---
*Phase: 46-core-vanguard-server*
*Completed: 2026-03-06*

## Self-Check: PASSED

All 8 source files exist, both task commits verified (b5cd4eb, f19e41d), SUMMARY.md created.
