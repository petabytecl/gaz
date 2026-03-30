---
phase: 47-middleware-interceptors
plan: 01
subsystem: server
tags: [connect, interceptors, middleware, auth, ratelimit, validation, recovery, logging]

# Dependency graph
requires:
  - phase: 46-core-vanguard
    provides: "Vanguard server, connect.Registrar interface, DI container"
provides:
  - "ConnectInterceptorBundle interface with Name(), Priority(), Interceptors()"
  - "5 built-in bundles: LoggingBundle, RecoveryBundle, AuthBundle, RateLimitBundle, ValidationBundle"
  - "collectConnectInterceptors() for DI auto-discovery with priority sorting"
  - "ConnectAuthFunc and ConnectLimiter types for auth/rate-limit extension points"
  - "AlwaysPassLimiter default implementation"
  - "Registrar.RegisterConnect(opts ...connect.HandlerOption) updated signature"
affects: [47-02-PLAN, server/vanguard]

# Tech tracking
tech-stack:
  added: [connectrpc.com/validate]
  patterns: [ConnectInterceptorBundle auto-discovery, priority-sorted interceptor chain, http.Header-based auth]

key-files:
  created:
    - server/connect/interceptors.go
  modified:
    - server/connect/interceptors_test.go
    - server/connect/registrar.go
    - server/connect/registrar_test.go
    - server/connect/doc.go
    - .golangci.yml
    - go.mod
    - go.sum

key-decisions:
  - "ConnectAuthFunc uses http.Header+connect.Spec instead of connect.AnyRequest — AnyRequest has unexported methods preventing external implementation"
  - "ConnectLimiter uses http.Header+connect.Spec for same reason — uniform interface for both unary and streaming"
  - "Added connectrpc.com/validate dependency for ValidationBundle"

patterns-established:
  - "ConnectInterceptorBundle: auto-discovered via di.ResolveAll, sorted by Priority(), flattened Interceptors()"
  - "Auth/RateLimit use http.Header+connect.Spec for unary/streaming uniformity"
  - "WrapStreamingClient always pass-through (server-side only bundles)"

requirements-completed: [CONN-02, CONN-03, MDDL-04]

# Metrics
duration: 8min
completed: 2026-03-06
---

# Phase 47 Plan 01: Connect Interceptor Bundles Summary

**ConnectInterceptorBundle interface with 5 built-in bundles (logging, recovery, auth, rate-limit, validation) using priority-sorted auto-discovery from DI, plus Registrar updated to accept connect.HandlerOption for interceptor injection**

## Performance

- **Duration:** 8 min
- **Started:** 2026-03-06T21:37:03Z
- **Completed:** 2026-03-06T21:45:17Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- ConnectInterceptorBundle interface with 5 built-in bundles mirroring gRPC InterceptorBundle pattern
- collectConnectInterceptors() for DI auto-discovery with priority-based sorting and flattening
- ConnectAuthFunc using http.Header+connect.Spec (works for both unary and streaming without AnyRequest)
- Registrar.RegisterConnect() updated to accept variadic connect.HandlerOption for interceptor injection
- Comprehensive test suite with 24 tests covering all bundles, collection logic, priority ordering

## Task Commits

Each task was committed atomically:

1. **Task 1: ConnectInterceptorBundle interface, priority constants, built-in bundles, and collection logic** — TDD
   - `cc9f9b4` (test: add failing tests — RED phase)
   - `b394823` (feat: implement bundles and fix tests — GREEN phase)
2. **Task 2: Update Registrar interface signature and tests, update doc.go** — `f41c5a2` (feat)

## Files Created/Modified
- `server/connect/interceptors.go` — ConnectInterceptorBundle interface, priority constants, 5 built-in bundles, collectConnectInterceptors(), ConnectAuthFunc, ConnectLimiter, AlwaysPassLimiter
- `server/connect/interceptors_test.go` — 19 tests: interface compliance, priority ordering, collection logic, recovery panic handling, rate limiting, auth
- `server/connect/registrar.go` — Updated Registrar.RegisterConnect(opts ...connect.HandlerOption)
- `server/connect/registrar_test.go` — 5 tests including handler options forwarding and no-options variadic
- `server/connect/doc.go` — Package docs with Registrar and ConnectInterceptorBundle documentation
- `.golangci.yml` — Added connectrpc.com/validate to depguard allow lists
- `go.mod` / `go.sum` — Added connectrpc.com/validate dependency

## Decisions Made
- **ConnectAuthFunc uses http.Header+connect.Spec instead of connect.AnyRequest:** The connect.AnyRequest interface has unexported methods (internalOnly(), setRequestMethod()) that prevent external implementation. This means we cannot create an adapter for streaming handlers. Using http.Header+connect.Spec provides a uniform interface that works for both unary (extracting from req.Header()) and streaming (extracting from conn.RequestHeader()) handlers. Auth tokens are always in HTTP headers anyway, making this a natural fit.
- **ConnectLimiter follows same http.Header+connect.Spec pattern:** For consistency with ConnectAuthFunc and for the same unexported-methods constraint.
- **Added connectrpc.com/validate dependency:** ValidationBundle wraps validate.NewInterceptor() for protobuf message validation via buf's protovalidate rules.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed ConnectAuthFunc/ConnectLimiter signature to use http.Header instead of connect.AnyRequest**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** connect.AnyRequest has unexported methods (internalOnly(), setRequestMethod()) preventing external implementation. The streamAuthRequest adapter could not satisfy the interface.
- **Fix:** Changed ConnectAuthFunc from `func(ctx, connect.AnyRequest) (context.Context, error)` to `func(ctx, http.Header, connect.Spec) (context.Context, error)`. Same change for ConnectLimiter.Limit(). Removed streamAuthRequest type entirely.
- **Files modified:** server/connect/interceptors.go, server/connect/interceptors_test.go
- **Verification:** All 24 tests pass with race detection
- **Committed in:** b394823

**2. [Rule 3 - Blocking] Added connectrpc.com/validate dependency and depguard config**
- **Found during:** Task 1 (GREEN phase)
- **Issue:** connectrpc.com/validate not in go.mod; depguard blocked the import
- **Fix:** Ran `go get connectrpc.com/validate@v0.6.0`, added to both depguard allow lists in .golangci.yml
- **Files modified:** go.mod, go.sum, .golangci.yml
- **Verification:** go vet passes, linter accepts the import
- **Committed in:** b394823

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both necessary for correctness. AnyRequest constraint is a fundamental Connect library design — the http.Header approach is cleaner anyway. No scope creep.

## Issues Encountered
None beyond the deviations documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- ConnectInterceptorBundle interface ready for Vanguard server integration (Plan 02)
- Registrar signature updated — Plan 02 must update Vanguard server's call from `reg.RegisterConnect()` to `reg.RegisterConnect(opts...)`
- Note: `go build ./...` at project level may fail until Plan 02 updates the Vanguard server consumer

---
*Phase: 47-middleware-interceptors*
*Completed: 2026-03-06*
