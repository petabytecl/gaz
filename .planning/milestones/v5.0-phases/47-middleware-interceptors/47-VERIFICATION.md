---
phase: 47-middleware-interceptors
verified: 2026-03-06T19:10:00Z
status: passed
score: 5/5 must-haves verified
must_haves:
  truths:
    - "Browser clients can access Connect and gRPC-Web services with correct CORS headers — preflight and actual requests work across origins"
    - "Connect interceptors (auth, logging, validation) are automatically injected into all Connect handlers without per-service wiring"
    - "ConnectInterceptorBundle supports priority-sorted, auto-discovered interceptor chains via DI — same pattern as gRPC InterceptorBundle"
    - "OpenTelemetry traces span both HTTP transport layer (otelhttp) and Connect RPC layer (otelconnect), with correlated trace IDs across the boundary"
    - "Proto constraint validation rejects invalid requests at the interceptor level via connectrpc.com/validate before reaching handler logic"
  artifacts:
    - path: "server/connect/interceptors.go"
      provides: "ConnectInterceptorBundle interface, 5 built-in bundles, CollectConnectInterceptors, ConnectAuthFunc, ConnectLimiter, AlwaysPassLimiter"
    - path: "server/connect/interceptors_test.go"
      provides: "19 tests covering all bundles, collection logic, priority ordering"
    - path: "server/connect/registrar.go"
      provides: "Registrar interface with RegisterConnect(opts ...connect.HandlerOption)"
    - path: "server/connect/registrar_test.go"
      provides: "5 tests including handler options forwarding"
    - path: "server/connect/doc.go"
      provides: "Package documentation for Registrar and ConnectInterceptorBundle"
    - path: "server/vanguard/middleware.go"
      provides: "TransportMiddleware interface, CORSMiddleware, OTELMiddleware, OTELConnectBundle, collectTransportMiddleware"
    - path: "server/vanguard/middleware_test.go"
      provides: "12 tests covering middleware chain, CORS config, OTEL activation, priority ordering"
    - path: "server/vanguard/config.go"
      provides: "CORSConfig struct, DefaultCORSConfig, 6 CORS flags"
    - path: "server/vanguard/server.go"
      provides: "OnStart wiring of Connect interceptors and transport middleware"
    - path: "server/vanguard/module.go"
      provides: "8 provider functions for middleware and interceptor bundles"
  key_links:
    - from: "server/connect/interceptors.go"
      to: "connectrpc.com/connect"
      via: "connect.Interceptor type"
    - from: "server/connect/interceptors.go"
      to: "connectrpc.com/validate"
      via: "validate.NewInterceptor()"
    - from: "server/connect/interceptors.go"
      to: "github.com/petabytecl/gaz/di"
      via: "di.ResolveAll[ConnectInterceptorBundle]"
    - from: "server/vanguard/server.go"
      to: "server/connect/interceptors.go"
      via: "connectpkg.CollectConnectInterceptors"
    - from: "server/vanguard/server.go"
      to: "server/vanguard/middleware.go"
      via: "collectTransportMiddleware"
    - from: "server/vanguard/server.go"
      to: "server/connect/registrar.go"
      via: "reg.RegisterConnect(handlerOpts...)"
    - from: "server/vanguard/middleware.go"
      to: "github.com/rs/cors"
      via: "cors.New()/cors.AllowAll()"
    - from: "server/vanguard/middleware.go"
      to: "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
      via: "otelhttp.NewMiddleware()"
    - from: "server/vanguard/middleware.go"
      to: "connectrpc.com/otelconnect"
      via: "otelconnect.NewInterceptor()"
requirements:
  - id: CONN-02
    status: satisfied
  - id: CONN-03
    status: satisfied
  - id: MDDL-01
    status: satisfied
  - id: MDDL-02
    status: satisfied
  - id: MDDL-03
    status: satisfied
  - id: MDDL-04
    status: satisfied
---

# Phase 47: Middleware & Interceptors Verification Report

**Phase Goal:** Developer has a complete two-layer middleware stack — HTTP transport middleware for cross-cutting concerns and Connect interceptors for RPC semantics — with auto-discovered, priority-sorted interceptor chains
**Verified:** 2026-03-06T19:10:00Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Browser clients can access Connect and gRPC-Web services with correct CORS headers — preflight and actual requests work across origins | ✓ VERIFIED | `CORSMiddleware` in `server/vanguard/middleware.go:76-113` wraps handler with `cors.AllowAll()` (dev) or `cors.New(cors.Options{...})` (prod). Tests `TestCORSMiddleware_DevModeAllowsAll` and `TestCORSMiddleware_ProductionRespectsConfig` verify preflight and origin enforcement. 6 `--server-cors-*` flags in `config.go:124-129`. |
| 2 | Connect interceptors (auth, logging, validation) are automatically injected into all Connect handlers without per-service wiring | ✓ VERIFIED | `server.go:78` calls `collectpkg.CollectConnectInterceptors()`, `server.go:81` wraps into `connect.WithInterceptors()`, `server.go:97` passes `handlerOpts...` to every `reg.RegisterConnect()`. Module registers all bundles as DI providers (module.go:232-239). |
| 3 | `ConnectInterceptorBundle` supports priority-sorted, auto-discovered interceptor chains via DI — same pattern as gRPC InterceptorBundle | ✓ VERIFIED | Interface at `interceptors.go:52-71` with `Name()`, `Priority()`, `Interceptors()`. `CollectConnectInterceptors()` at line 75 uses `di.ResolveAll`, sorts by priority, flattens. Test `TestCollectConnectInterceptors_SortsByPriority` confirms ordering low→mid→high. 5 built-in bundles with priority constants: Logging(0), RateLimit(25), Auth(50), Validation(100), Recovery(1000). |
| 4 | OpenTelemetry traces span both HTTP transport layer (otelhttp) and Connect RPC layer (otelconnect), with correlated trace IDs across the boundary | ✓ VERIFIED | `OTELMiddleware` at `middleware.go:120-159` uses `otelhttp.NewMiddleware()` with `WithTracerProvider(tp)`. `OTELConnectBundle` at `middleware.go:166-201` uses `otelconnect.NewInterceptor()` with same `TracerProvider`. Both filter health/reflection endpoints. Module conditionally registers both when `TracerProvider` exists in DI (module.go:60-97). |
| 5 | Proto constraint validation rejects invalid requests at the interceptor level via `connectrpc.com/validate` before reaching handler logic | ✓ VERIFIED | `ValidationBundle` at `interceptors.go:424-445` wraps `validate.NewInterceptor()`. Priority 100 places it after auth (50) but before recovery (1000). Module registers via `provideConnectValidationBundle` (module.go:125-131). `connectrpc.com/validate v0.6.0` in go.mod and depguard allow lists. |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `server/connect/interceptors.go` | ConnectInterceptorBundle interface, 5 bundles, collection logic | ✓ VERIFIED | 446 lines. Interface, priority constants, 5 bundles (Logging, Recovery, Auth, RateLimit, Validation), CollectConnectInterceptors, ConnectAuthFunc, ConnectLimiter, AlwaysPassLimiter. All exported. |
| `server/connect/interceptors_test.go` | Tests for all bundles and collection | ✓ VERIFIED | 465 lines (>200 min). 19 test methods in testify suite. Tests priority ordering, interface compliance, panic recovery (dev+prod), rate limiting, auth, empty container, flattening. |
| `server/connect/registrar.go` | Updated Registrar with HandlerOption | ✓ VERIFIED | `RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler)` at line 29. Doc comment shows usage pattern. |
| `server/connect/registrar_test.go` | Updated tests for new signature | ✓ VERIFIED | 76 lines. 5 tests including `TestRegisterConnectReceivesHandlerOptions` verifying opts forwarding and `TestRegisterConnectNoOptions` verifying variadic empty call. |
| `server/connect/doc.go` | Package documentation | ✓ VERIFIED | 54 lines. Documents both Registrar and ConnectInterceptorBundle interfaces with examples and built-in bundle list. |
| `server/vanguard/middleware.go` | TransportMiddleware, CORS, OTEL, OTELConnect | ✓ VERIFIED | 202 lines. TransportMiddleware interface (Name/Priority/Wrap), PriorityCORS(0)/PriorityOTEL(100), collectTransportMiddleware, CORSMiddleware, OTELMiddleware, OTELConnectBundle. |
| `server/vanguard/middleware_test.go` | Tests for middleware chain, CORS, OTEL | ✓ VERIFIED | 276 lines (>150 min). 12 tests: priority constants, CORS dev/prod, OTEL interface/wrapping, OTELConnect interface/interceptors, middleware ordering, empty container, DefaultCORSConfig dev/prod. Compile-time interface checks. |
| `server/vanguard/config.go` | CORSConfig struct, CORS flags | ✓ VERIFIED | CORSConfig struct (lines 68-89), CORS field in Config (line 64), DefaultCORSConfig (lines 135-154), DefaultCORSMaxAge constant (line 21), 6 CORS flags (lines 124-129). |
| `server/vanguard/server.go` | Middleware and interceptor wiring | ✓ VERIFIED | OnStart step 0 collects interceptors (line 78), builds handlerOpts with WithInterceptors (line 81), passes to RegisterConnect (line 97), step 8.5 applies transport middleware (line 148). Logs interceptor count (line 176). |
| `server/vanguard/module.go` | 8 middleware/bundle providers | ✓ VERIFIED | 8 provider functions (lines 45-176): CORS, OTEL transport, OTELConnect, Logging, Recovery, Validation, Auth (opt-in), RateLimit. All wired in NewModule builder (lines 232-239). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `server/connect/interceptors.go` | `connectrpc.com/connect` | `connect.Interceptor` type | ✓ WIRED | 10+ references to `connect.Interceptor` throughout file |
| `server/connect/interceptors.go` | `connectrpc.com/validate` | `validate.NewInterceptor()` | ✓ WIRED | Line 430: `validate.NewInterceptor()` in ValidationBundle constructor |
| `server/connect/interceptors.go` | `github.com/petabytecl/gaz/di` | `di.ResolveAll[ConnectInterceptorBundle]` | ✓ WIRED | Line 76: `di.ResolveAll[ConnectInterceptorBundle](container)` |
| `server/vanguard/server.go` | `server/connect/interceptors.go` | `CollectConnectInterceptors` | ✓ WIRED | Line 78: `connectpkg.CollectConnectInterceptors(s.container, s.logger)` |
| `server/vanguard/server.go` | `server/vanguard/middleware.go` | `collectTransportMiddleware` | ✓ WIRED | Line 148: `collectTransportMiddleware(s.container, s.logger, handler)` |
| `server/vanguard/server.go` | `server/connect/registrar.go` | `RegisterConnect(opts...)` | ✓ WIRED | Line 81: `connect.WithInterceptors(connectInterceptors...)`, Line 97: `reg.RegisterConnect(handlerOpts...)` |
| `server/vanguard/middleware.go` | `github.com/rs/cors` | `cors.New()/AllowAll()` | ✓ WIRED | Line 86: `cors.AllowAll()`, Line 88: `cors.New(cors.Options{...})` |
| `server/vanguard/middleware.go` | `otelhttp` | `otelhttp.NewMiddleware()` | ✓ WIRED | Line 143: `otelhttp.NewMiddleware("vanguard", ...)` |
| `server/vanguard/middleware.go` | `connectrpc.com/otelconnect` | `otelconnect.NewInterceptor()` | ✓ WIRED | Line 190: `otelconnect.NewInterceptor(...)` |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| CONN-02 | 47-01 | Framework automatically injects Connect interceptors (auth, logging, validation, OTEL) into all Connect handlers | ✓ SATISFIED | `server.go:78-97` collects interceptors via DI and passes to every `RegisterConnect()` call. Module registers all bundle providers. |
| CONN-03 | 47-01 | Developer can create ConnectInterceptorBundle with priority-sorted, auto-discovered interceptor chains | ✓ SATISFIED | `ConnectInterceptorBundle` interface at `interceptors.go:52-71`. `CollectConnectInterceptors` sorts by priority. Custom bundles can be registered with any priority value. |
| MDDL-01 | 47-02 | Developer can apply CORS middleware at transport level for browser clients | ✓ SATISFIED | `CORSMiddleware` wraps Vanguard handler. AllowAll in dev mode, configurable in prod. 6 `--server-cors-*` flags for CLI configuration. |
| MDDL-02 | 47-02 | Vanguard server uses two-layer middleware model | ✓ SATISFIED | Transport middleware (`collectTransportMiddleware` at server.go:148) wraps HTTP handler. Connect interceptors (`CollectConnectInterceptors` at server.go:78) are injected per-service via `WithInterceptors`. |
| MDDL-03 | 47-02 | Developer can enable OpenTelemetry tracing for both HTTP transport and Connect RPC layers | ✓ SATISFIED | `OTELMiddleware` uses `otelhttp.NewMiddleware()`. `OTELConnectBundle` uses `otelconnect.NewInterceptor()`. Both share same `TracerProvider` for correlated traces. Both conditionally registered via `gaz.Has[*sdktrace.TracerProvider]`. |
| MDDL-04 | 47-01 | Developer can enable proto constraint validation via connectrpc.com/validate | ✓ SATISFIED | `ValidationBundle` wraps `validate.NewInterceptor()`. Always registered by module. Runs at priority 100 (before recovery, after auth). |

No orphaned requirements. All 6 requirement IDs from the phase (CONN-02, CONN-03, MDDL-01, MDDL-02, MDDL-03, MDDL-04) are accounted for across Plan 01 and Plan 02.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | None found | — | — |

No TODOs, FIXMEs, placeholders, stubs, or empty implementations detected in any phase artifact.

### Human Verification Required

### 1. CORS Preflight End-to-End

**Test:** Start Vanguard server in dev mode, send OPTIONS preflight from a different origin to a Connect endpoint, then send actual POST request.
**Expected:** Preflight returns `Access-Control-Allow-Origin: *` with correct headers. Actual request succeeds with CORS headers.
**Why human:** Requires running server and making real HTTP requests to verify full CORS flow including browser behavior.

### 2. OTEL Trace Correlation

**Test:** Start Vanguard server with TracerProvider, make a Connect RPC call, inspect exported spans.
**Expected:** Both `otelhttp` transport span and `otelconnect` RPC span exist with the same trace ID. Health endpoints produce no spans.
**Why human:** Requires running OTEL collector or in-memory exporter and inspecting trace tree structure.

### 3. Validation Rejection

**Test:** Send a Connect RPC request with a protobuf message that violates `buf.validate` constraints.
**Expected:** Request is rejected at interceptor level with validation error before reaching handler logic.
**Why human:** Requires protobuf service with validation rules defined and a real Connect client.

### Gaps Summary

No gaps found. All 5 success criteria from ROADMAP.md are verified:

1. **CORS headers** — CORSMiddleware with AllowAll/strict modes, 6 flags, tested with preflight
2. **Auto-injected interceptors** — DI auto-discovery, priority sorting, WithInterceptors wiring
3. **ConnectInterceptorBundle** — Interface with 5 built-in bundles, same pattern as gRPC
4. **OTEL dual-layer tracing** — otelhttp transport + otelconnect RPC, shared TracerProvider, health filtering
5. **Proto validation** — connectrpc.com/validate interceptor at priority 100

Both plans executed successfully. Full project builds cleanly (`go build ./...`). All 24 connect tests and 25+ vanguard tests pass with race detection.

---

_Verified: 2026-03-06T19:10:00Z_
_Verifier: Claude (gsd-verifier)_
