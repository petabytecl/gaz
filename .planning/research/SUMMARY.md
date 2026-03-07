# Project Research Summary

**Project:** gaz v5.0 — Vanguard Unified Server
**Domain:** Go application framework — unified HTTP/gRPC/Connect server with Vanguard multiplexer
**Researched:** 2026-03-06
**Confidence:** HIGH

## Executive Summary

Gaz v5.0 replaces the multi-port gRPC + gRPC-Gateway architecture with a single-port Vanguard server that natively serves gRPC, Connect, gRPC-Web, and REST (via `google.api.http` annotations) — all through one `http.Handler`. The recommended approach is to wrap the existing `*grpc.Server` with Vanguard's transcoder (`vanguardgrpc.NewTranscoder`), add a new `ConnectRegistrar` interface mirroring the existing `grpc.Registrar` pattern, and compose everything behind Vanguard as the unified HTTP entry point. This eliminates the grpc-gateway loopback connection, removes code generation for REST transcoding, and reduces operational complexity from 2-3 ports to one.

The stack is well-validated: Connect-Go (v1.19.1) is production-stable with 4,556 importers, and the supporting libraries (otelconnect, validate) are thin wrappers with low API surface risk. The primary risk is Vanguard itself at v0.4.0 (pre-stable, 22 importers). This is mitigated by wrapping Vanguard behind gaz's own `server/vanguard` interfaces — the same insulation pattern used for grpc-gateway in v4.1 — and keeping the deprecated gateway module available as a fallback during migration. Known Vanguard issues (#165 JSON codec, #170 special characters, #184 error codec) need regression tests.

The most critical technical challenges are: (1) interceptor incompatibility between gRPC and Connect (different type signatures require parallel `InterceptorBundle` and `ConnectInterceptorBundle` interfaces with shared transport-agnostic logic), (2) Vanguard's one-shot transcoder construction requiring careful DI lifecycle ordering (build in `OnStart`, not in provider), and (3) h2c configuration for single-port gRPC that may break existing clients (requires Go 1.26 `http.Protocols` API and thorough multi-client testing). A phased approach — core server first, then middleware/interceptors, then migration tooling — aligns with dependency ordering and risk mitigation.

## Key Findings

### Recommended Stack

The v5.0 stack extends gaz's existing dependencies with four new packages from the Connect ecosystem, all maintained by the Buf team. Existing gRPC infrastructure (`grpc-go`, `go-grpc-middleware`, `otelgrpc`) is retained because Vanguard wraps the existing `*grpc.Server` — no gRPC code needs to change.

**Core new technologies:**
- **`connectrpc.com/connect` v1.19.1:** Connect protocol server/client — stable (v1.x), 4,556 importers, generates `(path, http.Handler)` tuples from proto files
- **`connectrpc.com/vanguard` v0.4.0:** Unified HTTP multiplexer — wraps `*grpc.Server` via `vanguardgrpc.NewTranscoder()` to serve gRPC, Connect, gRPC-Web, and REST on single port with zero codegen. **Alpha — wrap behind gaz interfaces**
- **`connectrpc.com/otelconnect` v0.9.0:** Drop-in OpenTelemetry interceptor for Connect handlers, supporting custom TracerProvider/MeterProvider
- **`connectrpc.com/validate` v0.6.0:** Protovalidate interceptor for Connect, uses same `buf.build/go/protovalidate` engine already in go.mod

**Critical version requirement:** Go 1.26+ for native h2c via `http.Protocols.SetUnencryptedHTTP2(true)` — eliminates need for `golang.org/x/net/http2/h2c` package.

**What NOT to use:** grpc-gateway (for new development), cmux (fragile byte-sniffing), multi-port serving (defeats the milestone goal).

See: [STACK.md](STACK.md)

### Expected Features

**Must have (table stakes):**
- Connect-Go handler registration via `ConnectRegistrar` interface with DI auto-discovery
- Vanguard transcoder serving REST/Connect/gRPC/gRPC-Web on single port
- REST transcoding from `google.api.http` proto annotations (zero codegen)
- gRPC-Web support (free with Vanguard)
- Single-port serving with h2c
- Connect interceptor injection (auth, logging, validation)
- CORS support for browser clients
- gRPC reflection (via `connectrpc.com/grpcreflect`)
- Health checks wired into unified server

**Should have (differentiators):**
- Auto-discovery of Connect services via `di.ResolveAll[ConnectRegistrar]`
- Two-layer middleware model: HTTP middleware (CORS, request ID, OTEL) wrapping Vanguard + Connect interceptors for RPC semantics (auth, validation, logging)
- `ConnectInterceptorBundle` with priority-sorted auto-discovery (mirrors gRPC `InterceptorBundle`)
- Vanguard + existing gRPC bridge for incremental migration
- Built-in OTEL observability across both transport and RPC layers
- Proto validation interceptor (`connectrpc.com/validate`)
- Unknown handler for non-RPC routes (health, metrics, static)

**Defer (v6.0+):**
- WebSocket support — different paradigm, not RPC
- Custom REST routing DSL — use proto annotations
- Automatic TLS certificate management — document, don't automate
- Unified gRPC ↔ Connect interceptor type — leaky abstraction, keep separate

See: [FEATURES.md](FEATURES.md)

### Architecture Approach

The target architecture replaces the three-listener model (gRPC :50051, Gateway :8080, HTTP :8081) with a single h2c-enabled `http.Server` on :8080 that serves a composed handler chain: `CORS → OTEL HTTP → Vanguard Transcoder`. The Vanguard transcoder composes gRPC services (via `vanguardgrpc.NewTranscoder(grpcServer)`) and Connect services (via `vanguard.NewService(path, handler)` per registrar) into one `http.Handler`. Protocol detection happens at the HTTP layer via Content-Type and path matching — no TCP-level multiplexing.

**Major components:**
1. **`server/vanguard/`** — Owns the Vanguard transcoder, h2c-enabled HTTP server, CORS middleware, and single-port lifecycle. This is the new "glue" replacing `server/gateway/`.
2. **`server/connect/`** — New package defining `ConnectRegistrar` interface, `ConnectInterceptorBundle`, and DI module for auto-discovering Connect handlers.
3. **`server/grpc/`** — Modified: still owns `*grpc.Server` creation and service registration, but no longer owns a listener. The gRPC server is wrapped by Vanguard.
4. **`server/module.go`** — Top-level module bundling vanguard + grpc + connect with correct startup ordering via DI.
5. **`server/gateway/`** — Deprecated, kept temporarily for migration fallback.

**Key patterns:** Dual Registrar (gRPC + Connect), Vanguard as Composition Root, h2c via Go 1.26 `http.Protocols`, HTTP middleware for transport concerns / Connect interceptors for RPC concerns.

See: [v5.0-ARCHITECTURE.md](v5.0-ARCHITECTURE.md)

### Critical Pitfalls

1. **Interceptor incompatibility (gRPC vs Connect)** — Type signatures are fundamentally different. Do NOT build a unified interceptor type. Create parallel `InterceptorBundle` (gRPC) and `ConnectInterceptorBundle` (Connect) with shared transport-agnostic logic underneath (e.g., `AuthChecker` interface with gRPC/Connect adapters).

2. **Vanguard one-shot transcoder construction** — `vanguard.NewTranscoder()` takes all services at construction. Build it in `OnStart` lifecycle hook (after DI is fully resolved), not in a DI provider. Mark as Eager. Add startup validation comparing registered vs transcoded services.

3. **h2c breaks existing gRPC clients** — `http.Server` serving h2c has subtly different HTTP/2 behavior than native `grpc.Server`. Test with ALL client types (Go gRPC, grpcurl, curl, non-Go clients). HIGH recovery cost if discovered late.

4. **Server timeouts kill streaming RPCs** — `http.Server.ReadTimeout`/`WriteTimeout` apply per-connection, not per-request. Set both to 0, use `ReadHeaderTimeout` for DoS protection, handle timeouts per-request via context deadlines.

5. **Health/reflection state divergence** — gRPC and Connect health services can report different statuses. Wire a single `health.Manager` to both. Use `connectrpc.com/grpcreflect` exclusively when all traffic goes through Vanguard.

See: [v5.0-VANGUARD-PITFALLS.md](v5.0-VANGUARD-PITFALLS.md)

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Core Vanguard Server
**Rationale:** Foundation that everything else depends on. Must validate h2c, single-port, and Vanguard wrapping before building interceptors or migration tooling.
**Delivers:** Working single-port server that serves gRPC (via Vanguard-wrapped `*grpc.Server`) and Connect handlers on one port, with REST transcoding from proto annotations.
**Addresses features:** Vanguard transcoder, single-port serving, h2c setup, `ConnectRegistrar` interface, REST transcoding, basic config (address, timeouts), DI module wiring, health check integration, gRPC reflection, unknown handler for non-RPC routes.
**Avoids pitfalls:** Vanguard registration ordering (build in `OnStart`), h2c client compatibility (integration tests with all client types), server timeout configuration (streaming-safe defaults), health/reflection state divergence (single source of truth), Vanguard v0.x instability (version pinning, abstraction layer, regression tests for known issues).

### Phase 2: Middleware & Interceptors
**Rationale:** Production readiness requires auth, logging, validation, and observability. Depends on Phase 1's Connect handler registration to have something to intercept. Interceptor incompatibility pitfall must be addressed here before Connect handlers go to production.
**Delivers:** Complete two-layer middleware stack (HTTP transport + Connect RPC interceptors), `ConnectInterceptorBundle` with priority-sorted auto-discovery, built-in OTEL observability.
**Addresses features:** Connect interceptor injection, `ConnectInterceptorBundle` interface, CORS middleware, OTEL integration (otelconnect + otelhttp), proto validation interceptor, auth interceptor pattern, access logging.
**Avoids pitfalls:** Interceptor incompatibility (parallel bundle interfaces with shared logic), auth bypass on Connect path (security tests verifying rejection across all protocols), CORS misconfiguration (transport-level only, not on gRPC).

### Phase 3: Migration & Polish
**Rationale:** Migration tooling and gateway deprecation depend on Phases 1-2 being stable. This phase focuses on the transition path for existing gaz users and framework completeness.
**Delivers:** gRPC bridge for existing services, gateway deprecation with migration guide, updated `server/module.go`, reference examples.
**Addresses features:** `vanguardgrpc` bridge for existing `grpc.Registrar` services, `server/gateway` deprecation, `server.NewModule()` using Vanguard by default + `server.NewLegacyModule()` for old behavior, documentation and examples.
**Avoids pitfalls:** Gateway registrar contract breakage (deprecation warnings, no forced interface changes), REST transcoding behavioral differences from grpc-gateway (compatibility test suite), different error formats between protocols (standardized error handler).

### Phase Ordering Rationale

- **Phase 1 before 2:** Connect interceptors need Connect handlers to exist first. The `ConnectRegistrar` interface and Vanguard server must be working before interceptors can be injected into handlers.
- **Phase 2 before 3:** Migration requires the full middleware stack to be in place — you can't migrate production services to the new server without auth, logging, and observability.
- **Dependency chain:** Vanguard transcoder → Connect registrar → Connect interceptors → gRPC bridge → gateway deprecation. This is a strict dependency chain that dictates phase ordering.
- **Risk-first:** Phase 1 tackles the highest-risk items (h2c compatibility, Vanguard stability, registration ordering). If Phase 1 reveals blocking issues, Phases 2-3 can be redesigned without wasted work.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 1:** h2c behavior — needs integration testing with specific client libraries (Java gRPC, Python gRPC, grpc-web) before committing to single-port. The `http.Protocols` API is new in Go 1.25/1.26 and may have edge cases.
- **Phase 2:** Connect interceptor ordering semantics — need to verify that interceptor execution order is deterministic when using `connect.WithInterceptors(...)` with multiple interceptors from different bundles.
- **Phase 3:** REST transcoding compatibility — need a comprehensive diff between grpc-gateway and Vanguard behavior for edge cases (FieldMask, oneof, additional_bindings, enum serialization).

Phases with standard patterns (skip research-phase):
- **Phase 2 (partial):** CORS, OTEL HTTP middleware, and validation interceptor are well-documented with official examples from Connect ecosystem.
- **Phase 1 (partial):** DI module wiring follows established gaz patterns (`gaz.NewModule`, `di.ResolveAll`, Eager registration). No research needed for the DI parts.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | **HIGH** | All packages verified on pkg.go.dev with exact versions. Connect-Go is stable (v1.x, 4,556 importers). Vanguard is alpha but version-pinned and wrapped. |
| Features | **HIGH** | Feature set derived from existing codebase analysis, official Connect/Vanguard documentation, and gaz's established patterns. Clear table stakes vs differentiators. |
| Architecture | **HIGH** | Architecture validated against Vanguard's official examples, Connect-Go handler API, and existing gaz server packages. Data flows verified with actual API signatures. |
| Pitfalls | **HIGH** | Pitfalls sourced from official docs, GitHub issues (#165, #170, #184), codebase analysis (interceptor types, lifecycle ordering), and Go HTTP/2 documentation. Recovery strategies costed. |

**Overall confidence:** HIGH

### Gaps to Address

- **Vanguard v0.4.0 edge cases:** Known issues (#165, #170, #184) need to be tested against gaz's specific use cases. Write regression tests early in Phase 1 and file upstream issues if new problems are found.
- **h2c with non-Go gRPC clients:** Research covers the theory but lacks empirical testing with Java/Python/C++ gRPC clients through Go's `http.Protocols` h2c. Needs integration testing in Phase 1.
- **Connect bidi streaming over h2c:** Connect only supports bidi streaming over HTTP/2. Verify this works correctly through `http.Server` with h2c, not just through native `grpc.Server`.
- **Error format standardization:** gRPC status codes, Connect error codes, and HTTP status codes all map differently. Need explicit error mapping configuration in Phase 2 to ensure consistent client experience across protocols.
- **Vanguard + grpc-web:** Pitfalls research flagged uncertainty about whether Vanguard handles grpc-web natively or if it requires Connect's built-in grpc-web support. Needs validation in Phase 1.

## Sources

### Primary (HIGH confidence)
- `connectrpc.com/connect` — pkg.go.dev (v1.19.1, published 2025-10-07)
- `connectrpc.com/vanguard` — pkg.go.dev (v0.4.0, published 2026-03-04)
- `connectrpc.com/otelconnect` — pkg.go.dev (v0.9.0, published 2026-01-05)
- `connectrpc.com/validate` — pkg.go.dev (v0.6.0, published 2025-09-27)
- Context7: `/connectrpc/connect-go` — interceptor patterns, handler creation, protocol support
- Context7: `/connectrpc/vanguard-go` — transcoder API, gRPC wrapping, service registration
- Context7: `/connectrpc/connectrpc.com` — observability, validation, architecture overview
- Connect-Go official docs — deployment, interceptors, routing, errors, migration from grpc-go
- Go 1.25/1.26 docs — `http.Protocols` API, native h2c support
- Existing gaz codebase — `server/grpc/`, `server/gateway/`, `server/http/`, `di/` packages (direct code review)

### Secondary (MEDIUM confidence)
- Vanguard GitHub issues (#165, #170, #184) — known transcoding edge cases
- Buf blog — Vanguard REST demo and architecture overview

### Tertiary (LOW confidence)
- h2c behavior with non-Go gRPC clients — theoretical analysis, needs empirical validation

---
*Research completed: 2026-03-06*
*Ready for roadmap: yes*
