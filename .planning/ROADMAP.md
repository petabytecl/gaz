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
- 🚧 **v5.1 Hardening** - Phases 49-53.2 (reopened — 2026-05-09 audit follow-up)

## Phases

**Phase Numbering:**
- Integer phases (46, 47, 48): Planned milestone work
- Decimal phases (46.1, 46.2): Urgent insertions (marked with INSERTED)

- [x] **Phase 46: Core Vanguard Server** - Single-port server serving gRPC, Connect, gRPC-Web, and REST via Vanguard transcoder with ConnectRegistrar auto-discovery
- [x] **Phase 47: Middleware & Interceptors** - Two-layer middleware stack with CORS, OTEL observability, Connect interceptor bundles, and proto validation (completed 2026-03-06)
- [x] **Phase 48: Server Module & Gateway Removal** - Updated server.NewModule() bundling Vanguard, gateway package removal, standalone HTTP preservation (completed 2026-03-06)
- [x] **Phase 49: Fix Critical Concurrency Bugs** - Fix 5 concurrency bugs: goroutine closure capture race (app.go), worker OnStop, lazySingleton race, Container.Build() race, startup error drain (completed 2026-03-29)
- [x] **Phase 50: Fix High-Priority Safety Issues** - Fix 7 safety issues: EventBus race, resolution chain leak, X-Request-ID injection, health path hardcoding, logger issues, Slowloris (completed 2026-03-29)
- [x] **Phase 51: Design and API Improvements** - 11 design improvements: split app.go, context propagation, shutdown errors, validation, timer leaks, backoff jitter (completed 2026-03-30)
- [x] **Phase 52: Test Coverage and Benchmarks** - Vanguard coverage 90%+, hot path benchmarks, cross-package integration tests, t.Parallel() markers (completed 2026-03-30)
- [x] **Phase 53: Tech Debt Cleanup** - Wire logger closer into App shutdown, update OTEL health path filter, fix doc.go references (completed 2026-03-30)
- [x] **Phase 53.1: Critical Review Fixes** (INSERTED) - 6 CRITICAL findings from 2026-05-09 full-repo review: Cobra single-startup-path, eventbus/cron lock-during-blocking-IO, gRPC reflection defaults (x2), DI singleton init deadlock, GitHub Actions SHA pinning (completed 2026-05-09)
- [ ] **Phase 53.2: High and Medium Review Fixes** (INSERTED) - HIGH/MEDIUM cleanup + rule-enforcement automation from 2026-05-09 review: shutdown ctx propagation, mgmt server posture, env-var convention, time.Sleep test sweep, custom linters for Rules 1/6/7/8

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
Phases execute in numeric order: 46 -> 47 -> 48 -> 49 -> 50 -> 51 -> 52 -> 53 -> 53.1 -> 53.2

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 46. Core Vanguard Server | 2/2 | Complete | 2026-03-06 |
| 47. Middleware & Interceptors | 2/2 | Complete | 2026-03-06 |
| 48. Server Module & Gateway Removal | 2/2 | Complete | 2026-03-06 |
| 49. Fix Critical Concurrency Bugs | 2/2 | Complete    | 2026-03-29 |
| 50. Fix High-Priority Safety Issues | 3/3 | Complete    | 2026-03-29 |
| 51. Design and API Improvements | 0/3 | Complete    | 2026-03-30 |
| 52. Test Coverage and Benchmarks | 0/2 | Complete    | 2026-03-30 |
| 53. Tech Debt Cleanup | 0/1 | Complete    | 2026-03-30 |
| 53.1. Critical Review Fixes (INSERTED) | 3/3 | Complete | 2026-05-09 |
| 53.2. High and Medium Review Fixes (INSERTED) | 0/8 | Planned | — |

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
**Plans:** 3/3 plans complete

Plans:
- [x] 50-01-PLAN.md — EventBus Close/Publish race fix + DI resolution chain leak fix
- [x] 50-02-PLAN.md — X-Request-ID validation + ContextHandler chain fix + file handle leak
- [x] 50-03-PLAN.md — Vanguard health path config + Slowloris timeout protection

### Phase 51: Design and API Improvements
**Goal:** 11 design improvements: split app.go, EventBus context propagation, cron context, shutdown error joining, pool size validation, duplicate comment, config panic, dead letter stack trace, async server error, timer leaks, backoff jitter
**Depends on:** Phase 50
**Requirements:** DSGN-01, DSGN-02, DSGN-03, DSGN-04, DSGN-05, DSGN-06, DSGN-07, DSGN-08, DSGN-09, DSGN-10, DSGN-11
**Plans:** 3/3 plans complete

Plans:
- [ ] 51-01-PLAN.md — Pool size validation, config error return, dead letter stack trace, backoff jitter fix
- [ ] 51-02-PLAN.md — EventBus context propagation, HTTP server port bind error detection
- [ ] 51-03-PLAN.md — Split app.go into focused files, cron context, shutdown error join, timer leaks

### Phase 52: Test Coverage and Benchmarks
**Goal:** Improve test infrastructure: vanguard coverage (74.4% -> 90%+), add benchmarks for hot paths, cross-package integration tests, investigate cron timing, add t.Parallel() markers
**Depends on:** Phase 51
**Requirements:** TEST-01, TEST-02, TEST-03, TEST-04, TEST-05
**Plans:** 2/2 plans complete

Plans:
- [ ] 52-01-PLAN.md — Vanguard coverage 90%+ and hot-path benchmarks (DI, EventBus, Backoff)
- [ ] 52-02-PLAN.md — Cross-package integration tests, cron timing fix, t.Parallel() markers

## Backlog

(empty)

### Phase 53: Tech Debt Cleanup

**Goal:** Wire logger.NewLoggerWithCloser into App shutdown lifecycle, update OTEL middleware trace filter to use health.Config paths, fix doc.go health path references
**Requirements**: SAFE-06 (partial closure)
**Depends on:** Phase 52
**Plans:** 1/1 plans complete

Plans:
- [ ] 53-01-PLAN.md — Logger closer wiring + OTEL health path filter + doc.go fix

### Phase 53.1: Critical Review Fixes (INSERTED)

**Goal:** Resolve the 6 CRITICAL findings from the 2026-05-09 full-repo review — restore Rule 1, 2, 4, 7, 8 compliance and fix the DI singleton init-under-lock deadlock.
**Origin:** Full-repo code review 2026-05-09 (every codified rule had at least one active violation).
**Depends on:** Phase 53
**Context:** `.planning/milestones/v5.1-phases/53.1-critical-review-fixes/CONTEXT.md`
**Requirements:** ITEM-01, ITEM-02, ITEM-03, ITEM-04, ITEM-05, ITEM-06, ITEM-07
**Plans:** 3/3 plans complete

Plans:
- [x] 53.1-01-PLAN.md — Cobra single startup path (Rule 2) + EventBus lock-during-blocking-IO (Rule 1)
- [x] 53.1-02-PLAN.md — Cron lock-during-channel-ops (Rule 1) + DI singleton init-under-lock (Rule 1)
- [x] 53.1-03-PLAN.md — Reflection defaults to false (Rule 7) + GitHub Actions SHA pinning (Rule 8)

Items (summary — see CONTEXT.md for fix sketches):
1. Cobra single startup path — `cobra.go:179-214` reimplements lifecycle; extract `startServices(ctx)` shared helper (Rule 2)
2. EventBus Publish lock-during-channel-send — `eventbus/bus.go:155-191` (Rule 1)
3. Cron internal scheduler lock-during-channel x 4 — `cron/internal/cron.go:136,155,181,302` (Rule 1)
4. gRPC reflection default true — `server/grpc/config.go:62` (Rule 7, BREAKING)
5. Vanguard reflection default true — `server/vanguard/config.go:106` (Rule 7, BREAKING)
6. DI singleton GetInstance runs provider while holding mu — `di/service.go:151,297` (Rule 1 + cross-goroutine cycle hang)
7. GitHub Actions not SHA-pinned — `.github/workflows/ci.yml` (Rule 8)

### Phase 53.2: High and Medium Review Fixes (INSERTED)

**Goal:** Clear the HIGH/MEDIUM review findings, pay down the 103-site `time.Sleep` test debt, and ship rule-enforcement automation so Rules 1/5/6/7/8 cannot regress silently again.
**Origin:** Full-repo code review 2026-05-09.
**Depends on:** Phase 53.1
**Context:** `.planning/milestones/v5.1-phases/53.2-high-medium-review-fixes/CONTEXT.md`
**Requirements:** A1-A17, B1-B21, C1-C3 (41 items from CONTEXT.md)
**Plans:** 8 plans

Plans:
- [ ] 53.2-01-PLAN.md — Shutdown context propagation (A1-A5, A6-verify, A13)
- [ ] 53.2-02-PLAN.md — DI container hardening (A7-A9, B1-verify, B3, B4, B14)
- [ ] 53.2-03-PLAN.md — Server security defaults + config fixes (A10-A12, A14-A16, B10)
- [ ] 53.2-04-PLAN.md — DI + app cleanup (B2, B5, B6, B16-B18)
- [ ] 53.2-05-PLAN.md — Server fixes: gRPC deferred construction, health adapter, Connect error code (B7-B9)
- [ ] 53.2-06-PLAN.md — time.Sleep sweep (A17, 103 sites across 18 files)
- [ ] 53.2-07-PLAN.md — Rule enforcement automation + CHANGELOG (C1-C3)
- [ ] 53.2-08-PLAN.md — Health, logger, cron, worker, backoff, gaztest cleanup (B11-B13, B15, B19-B21)
