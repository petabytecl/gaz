# Roadmap: gaz v5.0

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
- 🚧 **v5.0 Vanguard Unified Server** - Phases 46-48 (in progress)

## Phases

**Phase Numbering:**
- Integer phases (46, 47, 48): Planned milestone work
- Decimal phases (46.1, 46.2): Urgent insertions (marked with INSERTED)

- [x] **Phase 46: Core Vanguard Server** - Single-port server serving gRPC, Connect, gRPC-Web, and REST via Vanguard transcoder with ConnectRegistrar auto-discovery
- [ ] **Phase 47: Middleware & Interceptors** - Two-layer middleware stack with CORS, OTEL observability, Connect interceptor bundles, and proto validation
- [ ] **Phase 48: Server Module & Gateway Removal** - Updated server.NewModule() bundling Vanguard, gateway package removal, standalone HTTP preservation

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
**Plans**: TBD

### Phase 48: Server Module & Gateway Removal
**Goal**: Developer uses updated `server.NewModule()` that bundles Vanguard as the default server, with the legacy gateway cleanly removed and standalone HTTP server preserved
**Depends on**: Phase 47
**Requirements**: SMOD-01, SMOD-02, SMOD-03
**Success Criteria** (what must be TRUE):
  1. `server.NewModule()` provisions Vanguard server + Connect + gRPC as a unified bundle — developer calls one module function to get a complete server
  2. The `server/gateway` package is fully removed from the codebase — no lingering code, imports, or references
  3. The `server/http` package continues to work independently for HTTP-only use cases — existing HTTP-only apps are unaffected
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 46 → 47 → 48

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 46. Core Vanguard Server | 2/2 | Complete    | 2026-03-06 |
| 47. Middleware & Interceptors | 0/? | Not started | - |
| 48. Server Module & Gateway Removal | 0/? | Not started | - |
