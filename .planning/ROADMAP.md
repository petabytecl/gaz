# Roadmap: gaz v5.1

## Milestones

- ✅ **v1.0 MVP** - Phases 1-6 (shipped 2026-01-26)
- ✅ **v1.1 Security & Hardening** - Phases 7-10 (shipped 2026-01-27)
- ✅ **v2.0 Cleanup & Concurrency** - Phases 11-18 (shipped 2026-01-29)
- ✅ **v2.1 API Enhancement** - Phases 19-21 (shipped 2026-01-29)
- ✅ **v2.2 Test Coverage** - Phase 22 (shipped 2026-01-29)
- ✅ **v3.0 API Harmonization** - Phases 23-29 (shipped 2026-02-01)
- ✅ **v3.1 Performance & Stability** - Phase 30 (shipped 2026-02-01)
- ✅ **v3.2 Feature Maturity** - Phase 31 (shipped 2026-02-01)
- ✅ **v4.0 Dependency Reduction** - Phases 32-36 (shipped 2026-02-02)
- ✅ **v4.1 Server & Transport Layer** - Phases 37-45 (shipped 2026-02-04)
- ✅ **v5.0 Vanguard Unified Server** - Phases 46-48 (shipped 2026-03-06)
- 🚧 **v5.1 Hardening** - Phases 49-52 (in progress)

## Phases

**Phase Numbering:**
- Integer phases (46, 47, 48): Planned milestone work
- Decimal phases (46.1, 46.2): Urgent insertions (marked with INSERTED)

- [x] **Phase 46: Core Vanguard Server** - Single-port server serving gRPC, Connect, gRPC-Web, and REST via Vanguard transcoder with ConnectRegistrar auto-discovery
- [x] **Phase 47: Middleware & Interceptors** - Two-layer middleware stack with CORS, OTEL observability, Connect interceptor bundles, and proto validation (completed 2026-03-06)
- [x] **Phase 48: Server Module & Gateway Removal** - Updated server.NewModule() bundling Vanguard, gateway package removal, standalone HTTP preservation (completed 2026-03-06)
- [x] **Phase 49: Fix Critical Concurrency Bugs** - Fix 5 concurrency bugs: goroutine closure capture race, worker OnStop, lazySingleton race, Container.Build() race, startup error drain (completed 2026-03-29)
- [ ] **Phase 50: Fix High-Priority Safety Issues** - Fix 7 safety issues: EventBus race, resolution chain leak, X-Request-ID injection, health path hardcoding, logger issues, Slowloris
- [ ] **Phase 51: Design and API Improvements** - 11 design improvements: split app.go, context propagation, shutdown errors, validation, timer leaks, backoff jitter
- [ ] **Phase 52: Test Coverage and Benchmarks** - Vanguard coverage 90%+, hot path benchmarks, cross-package integration tests, t.Parallel() markers

## Phase Details

### Phase 46: Core Vanguard Server
**Goal**: Developer can create a single-port Vanguard server that serves gRPC, Connect, gRPC-Web, and REST protocols with auto-discovered Connect handlers and REST transcoding from proto annotations
**Depends on**: Phase 45 (v4.1 complete)
**Requirements**: USRV-01, USRV-02, USRV-03, USRV-04, USRV-05, USRV-06, CONN-01, CONN-04, MDDL-05
**Success Criteria** (what must be TRUE):
  1. Developer can register Connect-Go services via `ConnectRegistrar` and they are auto-discovered through `di.List` — same pattern as existing gRPC `Registrar`
  2. All four protocols (gRPC, Connect, gRPC-Web, REST) are served on a single port via h2c, verified by making requests from gRPC client, Connect client, browser gRPC-Web client, and curl REST client
  3. REST endpoints work from proto `google.api.http` annotations without any codegen — developer only writes proto files and Connect handlers
  4. Non-RPC HTTP routes (health, metrics, static files) are mountable on the same port via unknown handler configuration
  5. Server address, timeouts, and Vanguard options are configurable via CLI flags and config struct, with streaming-safe timeout defaults
**Plans:** 2/2 plans complete
  - [x] 46-01-PLAN.md — ConnectRegistrar interface + gRPC skip-listener mode
  - [x] 46-02-PLAN.md — Vanguard server config, server lifecycle, health, reflection, and module

### Phase 47: Middleware & Interceptors
**Goal**: Developer has a complete two-layer middleware stack — HTTP transport middleware for cross-cutting concerns and Connect interceptors for RPC semantics — with auto-discovered, priority-sorted interceptor chains
**Depends on**: Phase 46
**Requirements**: CONN-02, CONN-03, MDDL-01, MDDL-02, MDDL-03, MDDL-04
**Success Criteria** (what must be TRUE):
  1. Browser clients can access Connect and gRPC-Web services with correct CORS headers — preflight and actual requests work across origins
  2. Connect interceptors (auth, logging, validation) are automatically injected into all Connect handlers without per-service wiring
  3. `ConnectInterceptorBundle` supports priority-sorted, auto-discovered interceptor chains via DI — same pattern as gRPC `InterceptorBundle`
  4. OpenTelemetry traces span both HTTP transport layer (otelhttp) and Connect RPC layer (otelconnect), with correlated trace IDs across the boundary
  5. Proto constraint validation rejects invalid requests at the interceptor level via `connectrpc.com/validate` before reaching handler logic
**Plans:** 2/2 plans complete
Plans:
  - [x] 47-01-PLAN.md — ConnectInterceptorBundle interface, built-in bundles, Registrar signature update
  - [ ] 47-02-PLAN.md — TransportMiddleware, CORS config, Vanguard wiring, module extension

### Phase 48: Server Module & Gateway Removal
**Goal**: Developer uses updated `server.NewModule()` that bundles Vanguard as the default server, with the legacy gateway cleanly removed and standalone HTTP server preserved
**Depends on**: Phase 47
**Requirements**: SMOD-01, SMOD-02, SMOD-03
**Success Criteria** (what must be TRUE):
  1. `server.NewModule()` provisions Vanguard server + Connect + gRPC as a unified bundle — developer calls one module function to get a complete server
  2. The `server/gateway` package is fully removed from the codebase — no lingering code, imports, or references
  3. The `server/http` package continues to work independently for HTTP-only use cases — existing HTTP-only apps are unaffected
**Plans:** 2/2 plans complete
Plans:
  - [ ] 48-01-PLAN.md — Server module update + gateway deletion + dependency cleanup
  - [ ] 48-02-PLAN.md — Vanguard example creation + README update

## Progress

**Execution Order:**
Phases execute in numeric order: 46 → 47 → 48 → 49 → 50 → 51 → 52

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 46. Core Vanguard Server | 2/2 | Complete | 2026-03-06 |
| 47. Middleware & Interceptors | 2/2 | Complete | 2026-03-06 |
| 48. Server Module & Gateway Removal | 2/2 | Complete | 2026-03-06 |
| 49. Fix Critical Concurrency Bugs | 2/2 | Complete    | 2026-03-29 |
| 50. Fix High-Priority Safety Issues | 0/3 | Planning | — |
| 51. Design and API Improvements | 0/0 | Pending | — |
| 52. Test Coverage and Benchmarks | 0/0 | Pending | — |

### Phase 49: Fix Critical Concurrency Bugs
**Goal:** Fix 5 concurrency bugs found in full codebase review: goroutine closure capture race (app.go), worker OnStop cancelled context, lazySingleton Start/Stop race, Container.Build() race, startup error drain
**Depends on:** Phase 48 (v5.0 complete)
**Requirements:** CONC-01, CONC-02, CONC-03, CONC-04, CONC-05
**Plans:** 2/2 plans complete

Plans:
- [x] 49-01-PLAN.md — Fix lazySingleton Start/Stop race + Container.Build() race (di/)
- [x] 49-02-PLAN.md — Fix goroutine closure capture, startup error drain, worker OnStop context

### Phase 50: Fix High-Priority Safety Issues
**Goal:** Fix 7 safety issues: EventBus close/publish race, resolution chain leak, X-Request-ID injection, Vanguard health path hardcoding, logger ContextHandler chain break, logger file handle leak, Slowloris timeout
**Depends on:** Phase 49
**Requirements:** SAFE-01, SAFE-02, SAFE-03, SAFE-04, SAFE-05, SAFE-06, SAFE-07
**Plans:** 3 plans

Plans:
- [ ] 50-01-PLAN.md — EventBus Close/Publish race fix + DI resolution chain leak fix
- [ ] 50-02-PLAN.md — X-Request-ID validation + ContextHandler chain fix + file handle leak
- [ ] 50-03-PLAN.md — Vanguard health path config + Slowloris timeout protection

### Phase 51: Design and API Improvements
**Goal:** 11 design improvements: split app.go, EventBus context propagation, cron context, shutdown error joining, pool size validation, duplicate comment, config panic, dead letter stack trace, async server error, timer leaks, backoff jitter
**Depends on:** Phase 50
**Requirements:** TBD
**Plans:** 0 plans

Plans:
- [ ] TBD

### Phase 52: Test Coverage and Benchmarks
**Goal:** Improve test infrastructure: vanguard coverage (74.4% → 90%+), add benchmarks for hot paths, cross-package integration tests, investigate cron timing, add t.Parallel() markers
**Depends on:** Phase 51
**Requirements:** TBD
**Plans:** 0 plans

Plans:
- [ ] TBD

## Backlog

(empty)
