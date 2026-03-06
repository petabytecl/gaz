---
phase: 47-middleware-interceptors
plan: 02
subsystem: server
tags: [cors, otel, otelhttp, otelconnect, middleware, interceptors, vanguard, connect]

# Dependency graph
requires:
  - phase: 47-middleware-interceptors/01
    provides: ConnectInterceptorBundle interface, built-in bundles, collectConnectInterceptors
provides:
  - TransportMiddleware interface with priority-based chaining
  - CORSMiddleware (AllowAll dev / strict prod) with --server-cors-* flags
  - OTELMiddleware wrapping otelhttp when TracerProvider available
  - OTELConnectBundle wrapping otelconnect when TracerProvider available
  - Full middleware and interceptor wiring in Vanguard OnStart
  - Module providers for CORS, OTEL, and all Connect interceptor bundles
affects: [48-finalization]

# Tech tracking
tech-stack:
  added: [connectrpc.com/otelconnect v0.9.0]
  patterns: [TransportMiddleware interface, priority-based middleware chain, optional DI provider pattern]

key-files:
  created:
    - server/vanguard/middleware.go
    - server/vanguard/middleware_test.go
  modified:
    - server/vanguard/config.go
    - server/vanguard/server.go
    - server/vanguard/module.go
    - server/connect/interceptors.go
    - server/connect/interceptors_test.go
    - .golangci.yml
    - go.mod
    - go.sum

key-decisions:
  - "Exported CollectConnectInterceptors for cross-package use from vanguard"
  - "OTELConnectBundle placed in vanguard package (not connect) since it depends on sdktrace"
  - "Transport middleware applied after transcoder build, before h2c config"
  - "CORS middleware always registered; OTEL middleware and OTELConnect bundle are conditional on TracerProvider"

patterns-established:
  - "TransportMiddleware: Name()/Priority()/Wrap(http.Handler) with priority-sorted chain"
  - "Optional DI providers: use gaz.Has[T] before Resolve, skip silently when absent"
  - "buildTranscoder() extraction to keep OnStart cognitive complexity under gocognit threshold"

requirements-completed: [MDDL-01, MDDL-02, MDDL-03]

# Metrics
duration: 13min
completed: 2026-03-06
---

# Phase 47 Plan 02: HTTP Transport Middleware and Vanguard Wiring Summary

**CORS and OTEL transport middleware with priority-based chaining, OTELConnect interceptor bundle, and full middleware/interceptor wiring in Vanguard OnStart with 8 DI module providers**

## Performance

- **Duration:** 13 min
- **Started:** 2026-03-06T21:51:06Z
- **Completed:** 2026-03-06T22:04:39Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- TransportMiddleware interface with priority-based chain (CORS=0, OTEL=100) auto-discovered from DI
- CORSMiddleware with AllowAll dev mode, strict prod config, and 6 `--server-cors-*` CLI flags
- OTELMiddleware and OTELConnectBundle activated only when TracerProvider registered in DI
- Vanguard OnStart collects Connect interceptors → passes via WithInterceptors to RegisterConnect, then wraps handler with transport middleware chain
- Module extended with 8 providers: CORS, OTEL transport, OTELConnect, Logging, Recovery, Validation, Auth (opt-in), RateLimit

## Task Commits

Each task was committed atomically:

1. **Task 1: Add HTTP transport middleware and OTEL Connect bundle** — `64770a9` (feat)
2. **Task 2: Wire middleware and interceptors into vanguard server and module** — `fad0850` (feat)

## Files Created/Modified
- `server/vanguard/middleware.go` — TransportMiddleware interface, CORSMiddleware, OTELMiddleware, OTELConnectBundle, collectTransportMiddleware, priority constants
- `server/vanguard/middleware_test.go` — 12 tests covering all middleware components, priority ordering, dev/prod CORS
- `server/vanguard/config.go` — CORSConfig struct, DefaultCORSConfig, CORS field in Config, 6 CORS flags, DefaultCORSMaxAge constant
- `server/vanguard/server.go` — Interceptor collection (step 0), handlerOpts to RegisterConnect, middleware wrapping, buildTranscoder extraction
- `server/vanguard/server_test.go` — Updated mockConnectRegistrar.RegisterConnect signature for HandlerOption
- `server/vanguard/module.go` — 8 new provider functions, updated NewModule builder, extended doc comment
- `server/connect/interceptors.go` — Exported CollectConnectInterceptors for cross-package access
- `server/connect/interceptors_test.go` — Updated 4 call sites to exported name
- `.golangci.yml` — Added connectrpc.com/otelconnect to both depguard allow lists
- `go.mod` / `go.sum` — Added connectrpc.com/otelconnect v0.9.0

## Decisions Made
- **Exported CollectConnectInterceptors:** Plan 01 created it as unexported `collectConnectInterceptors` for same-package use in gRPC. Vanguard needs cross-package access, so exported it with uppercase name.
- **OTELConnectBundle in vanguard package:** Unlike other Connect bundles (in `server/connect/`), the OTEL bundle depends on `sdktrace.TracerProvider` which is a vanguard-level concern. Kept in vanguard to avoid circular dependencies.
- **Middleware applied after transcoder:** Transport middleware wraps the fully-built transcoder handler, ensuring CORS/OTEL see the final HTTP handler.
- **Optional providers pattern:** OTEL middleware, OTELConnect bundle, and Auth bundle use `gaz.Has[T]` to check for dependencies before resolving, skipping silently when absent.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Exported collectConnectInterceptors for cross-package access**
- **Found during:** Task 2 (server wiring)
- **Issue:** `collectConnectInterceptors` in `server/connect/` was unexported, but vanguard server needs to call it from `server/vanguard/`
- **Fix:** Renamed to `CollectConnectInterceptors` (exported), updated 4 test call sites
- **Files modified:** `server/connect/interceptors.go`, `server/connect/interceptors_test.go`
- **Verification:** All connect and vanguard tests pass
- **Committed in:** `fad0850` (Task 2 commit)

**2. [Rule 1 - Bug] Extracted buildTranscoder to keep OnStart under gocognit threshold**
- **Found during:** Task 2 (server wiring)
- **Issue:** Adding interceptor collection and middleware wiring pushed OnStart cognitive complexity above gocognit threshold of 20
- **Fix:** Extracted `buildTranscoder()` helper method from OnStart
- **Files modified:** `server/vanguard/server.go`
- **Verification:** `golangci-lint run ./server/vanguard/...` passes with no gocognit warnings
- **Committed in:** `fad0850` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both necessary for correctness. No scope creep.

## Issues Encountered

- 7 pre-existing lint warnings in `server/connect/interceptors.go` from Plan 01 (3 revive stutter, 2 wrapcheck, 1 nonamedreturns, 1 perfsprint). These are out of scope — documented in `deferred-items.md`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 47 (Middleware & Interceptors) is now complete — both plans executed
- Vanguard server has full middleware stack: CORS + OTEL transport middleware, 6 Connect interceptor bundles
- Ready for Phase 48 (Finalization) — the last phase of v5.0

## Self-Check: PASSED

- All 11 claimed files verified on disk
- Both task commits verified: `64770a9` (Task 1), `fad0850` (Task 2)

---
*Phase: 47-middleware-interceptors*
*Completed: 2026-03-06*
