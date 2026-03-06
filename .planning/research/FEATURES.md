# Feature Landscape: Unified Server (Vanguard + Connect-Go)

**Domain:** Go DI Framework — Server & Transport Layer (v5.0 Milestone)
**Researched:** Fri Mar 06 2026
**Supersedes:** v4.1 Server & Transport Layer research (Feb 2026)

## Context

This research covers NEW features for the v5.0 milestone: replacing gRPC-Gateway with Vanguard, adding Connect-Go support, and building a unified middleware model. Existing gRPC server capabilities are retained and enhanced.

**Key architectural shift:** From a gRPC server + loopback proxy (gRPC-Gateway) to a single unified HTTP handler (Vanguard Transcoder) that natively speaks REST, Connect, gRPC, and gRPC-Web — all on one port.

---

## Table Stakes

Features users expect. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Connect-Go handler registration** | Core requirement — users need to register Connect services via DI auto-discovery. | Medium | New `ConnectRegistrar` interface; mirrors existing `grpc.Registrar` pattern. |
| **Vanguard transcoder** | Replaces gRPC-Gateway; provides REST/Connect/gRPC/gRPC-Web on single port. | Medium | `vanguard.NewTranscoder()` wraps Connect handlers as `http.Handler`. |
| **REST transcoding from proto annotations** | Users expect `google.api.http` annotations to Just Work™ for REST endpoints. | Low | Vanguard reads from `protoregistry.GlobalFiles` automatically — zero codegen. |
| **gRPC-Web support** | Required for browser clients without Envoy proxy. | Low | Free with Vanguard — no extra config needed. |
| **Single-port serving** | One port for all protocols eliminates ops complexity. | Medium | Requires HTTP/2 without TLS (Go 1.24+ `http.Protocols.SetUnencryptedHTTP2(true)`). |
| **Graceful shutdown** | Prevent dropped connections during deploys. | Low | Already supported by `gaz.App` lifecycle (`di.Starter`/`di.Stopper`). |
| **CORS support** | Required for browser clients (Connect and gRPC-Web). | Low | net/http middleware on the Vanguard handler; `connectrpc.com/cors` for headers. |
| **Health checks** | Required for Kubernetes readiness/liveness. | Low | Already supported by `gaz/health`; wire into unified server. |
| **Connect interceptor injection** | Framework must inject common interceptors (auth, logging) into all handlers. | Medium | `ConnectRegistrar.RegisterConnect(opts ...connect.HandlerOption)` design allows framework to pass interceptors. |
| **gRPC reflection** | Required for debugging with `grpcurl`/Postman. | Low | `connectrpc.com/grpcreflect` for Connect; existing `grpc` reflection for gRPC services. |

## Differentiators

Features that set `gaz` apart. Not expected, but high value.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Auto-discovery of Connect services** | Register a `ConnectRegistrar` anywhere in DI → auto-wired into unified server. Zero manual wiring. | Medium | Uses `di.ResolveAll[ConnectRegistrar]()` — same pattern as existing `grpc.Registrar`. |
| **Two-layer middleware model** | Clear separation: net/http middleware for transport (CORS, request ID, OTEL HTTP) and Connect interceptors for RPC semantics (auth, validation, logging). | High | Avoids the "unified interceptor" trap. Each layer has clear responsibilities. |
| **InterceptorBundle for Connect** | Priority-sorted, auto-discovered Connect interceptor bundles — mirrors existing gRPC `InterceptorBundle` pattern. | Medium | `ConnectInterceptorBundle` interface: `Name()`, `Priority()`, `Interceptors() []connect.Interceptor`. |
| **Vanguard + existing gRPC bridge** | Existing `grpc.Registrar` services automatically bridged via `vanguardgrpc.NewTranscoder(grpcServer)`. | Medium | Allows incremental migration: old gRPC services work alongside new Connect services. |
| **Built-in OTEL observability** | Automatic OpenTelemetry tracing/metrics for both transport (HTTP) and RPC (Connect) layers. | Medium | `connectrpc.com/otelconnect` interceptor + `otelhttp` middleware. |
| **Proto validation interceptor** | Automatic request validation from proto constraints via `connectrpc.com/validate`. | Low | Single interceptor, massive DX improvement. |
| **Unknown handler for 404s** | Custom handling for unmatched routes (health endpoints, metrics, static files). | Low | `vanguard.WithUnknownHandler(handler)` — mux non-RPC routes. |
| **Module-level flags & config** | Server config (address, timeouts, TLS) via `gaz.NewModule("vanguard").Flags().Build()` pattern. | Low | Mirrors existing `server/grpc` config pattern with `Namespace()`, `Validate()`, `Flags()`. |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Unified gRPC ↔ Connect interceptor type** | Fundamentally different signatures and semantics (`grpc.UnaryServerInterceptor` vs `connect.Interceptor`). Bridging creates a leaky abstraction that satisfies neither well. | Separate `InterceptorBundle` (gRPC) and `ConnectInterceptorBundle` (Connect) — both auto-discovered, both priority-sorted. Consistent DX without false unification. |
| **`cmux` protocol multiplexing** | Hides protocol routing complexity, adds runtime fragility, and is unnecessary with Vanguard. | Vanguard handles all protocol dispatch natively as a single `http.Handler`. |
| **gRPC-Gateway continuation** | Adds codegen step, requires loopback gRPC connection (latency + complexity), and Vanguard is a strict superset. | Deprecate `server/gateway`; new `server/vanguard` module. Provide migration guide. |
| **Custom REST routing DSL** | Proto `google.api.http` annotations are the standard. A custom DSL fragments the ecosystem. | Use proto annotations; Vanguard reads them automatically from the global proto registry. |
| **Automatic TLS certificate management** | Complex, many valid approaches (cert-manager, Let's Encrypt, mTLS), and most deployments use a reverse proxy. | Document TLS config options; let users BYO certs or use platform-level TLS termination. |
| **WebSocket support** | Different paradigm from RPC; Vanguard doesn't support it. Mixing concerns. | Use a separate WebSocket server/module if needed. Connect's streaming covers most real-time use cases. |
| **Multi-port serving** | Adds operational complexity. The whole point of Vanguard is protocol unification on one port. | Single port with Vanguard. If users need separate ports (e.g., admin vs public), they can create multiple server module instances. |

## Feature Dependencies

```
ConnectRegistrar Interface
├── Requires: di.ResolveAll[ConnectRegistrar]() (existing DI capability)
└── Produces: (string, http.Handler) pairs for Vanguard

Vanguard Transcoder (server/vanguard)
├── Requires: ConnectRegistrar services (Connect-Go handlers)
├── Requires: grpc.Registrar services (optional, via vanguardgrpc bridge)
├── Requires: Go 1.24+ http.Protocols for H2C
├── Requires: Proto files registered in protoregistry.GlobalFiles (for REST annotations)
└── Produces: Single http.Handler serving REST/Connect/gRPC/gRPC-Web

ConnectInterceptorBundle
├── Requires: connect.Interceptor interface (connectrpc.com/connect)
├── Mirrors: existing grpc InterceptorBundle pattern
└── Injected via: ConnectRegistrar.RegisterConnect(opts ...connect.HandlerOption)

Two-Layer Middleware
├── Layer 1 (Transport): net/http middleware wrapping Vanguard handler
│   ├── CORS (connectrpc.com/cors helpers)
│   ├── Request ID
│   ├── OTEL HTTP (otelhttp)
│   └── Recovery (net/http level)
├── Layer 2 (RPC): Connect interceptors injected into handlers
│   ├── Auth
│   ├── Logging
│   ├── Validation (connectrpc.com/validate)
│   ├── OTEL RPC (connectrpc.com/otelconnect)
│   └── Recovery (connect.WithRecover())
└── Ordering: Transport middleware wraps the Vanguard handler; RPC interceptors are per-handler

Unified Server Module (server/vanguard/module.go)
├── Requires: Vanguard Transcoder
├── Requires: ConnectInterceptorBundle auto-discovery
├── Requires: Config struct (address, timeouts)
├── Implements: di.Starter / di.Stopper
└── Registered as: Eager() singleton in DI
```

## ConnectRegistrar Interface Design

**Critical design decision:** The interface must allow the framework to inject interceptors while remaining compatible with Connect-Go's generated handler factories.

```go
// ConnectRegistrar is implemented by services that register Connect-Go handlers.
// It mirrors the grpc.Registrar pattern for auto-discovery via DI.
type ConnectRegistrar interface {
    // RegisterConnect returns the path and handler for a Connect service.
    // The provided HandlerOptions include framework-level interceptors
    // (auth, logging, validation, OTEL) that MUST be passed to the
    // generated NewXxxServiceHandler() factory.
    RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler)
}
```

**Usage by service implementors:**

```go
func (s *GreetService) RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler) {
    // Pass framework opts + any service-specific opts
    return greetv1connect.NewGreetServiceHandler(s, opts...)
}
```

**Why this works:**
- Connect-Go generated factories return `(string, http.Handler)` — exact match.
- `connect.HandlerOption` is variadic — framework and service options compose naturally.
- Vanguard accepts `(string, http.Handler)` via `vanguard.NewService(path, handler)` — seamless wiring.

## Middleware Strategy

### Layer 1: Transport (net/http)

Applied to the Vanguard `http.Handler` before protocol dispatch. Sees raw HTTP requests.

| Middleware | Source | Purpose |
|------------|--------|---------|
| CORS | `rs/cors` + `connectrpc.com/cors` | Browser access headers |
| Request ID | Custom | Inject/propagate X-Request-ID |
| OTEL HTTP | `otelhttp` | HTTP-level traces and metrics |
| Panic Recovery | Custom | Catch panics in handler chain |
| Access Log | Custom | HTTP access logging (method, path, status, duration) |

### Layer 2: RPC (Connect Interceptors)

Injected into each Connect handler via `ConnectRegistrar`. Operates on typed RPC requests/responses.

| Interceptor | Source | Purpose |
|-------------|--------|---------|
| OTEL Connect | `connectrpc.com/otelconnect` | RPC-level traces with proto metadata |
| Validation | `connectrpc.com/validate` | Proto constraint validation |
| Auth | Custom | Token verification, context injection |
| Logging | Custom | Structured RPC logging (service, method, duration, error) |
| Recovery | `connect.WithRecover()` | Built-in panic recovery at RPC level |

### Why Two Layers?

1. **Transport middleware sees ALL requests** — including health checks, metrics endpoints, and unknown routes. RPC interceptors only see valid RPC calls.
2. **Different semantics** — HTTP middleware works with `http.Request`/`http.ResponseWriter`; Connect interceptors work with typed `connect.Request[T]`/`connect.Response[T]`.
3. **Connect interceptors survive transcoding** — per Vanguard FAQ, interceptors work whether the incoming request is REST, Connect, gRPC, or gRPC-Web.

## Key Libraries

| Library | Version | Purpose | Confidence |
|---------|---------|---------|------------|
| `connectrpc.com/connect` | v1.18.1 | Connect-Go RPC framework | HIGH |
| `connectrpc.com/vanguard` | v0.4.0 | Protocol transcoding (REST/Connect/gRPC/gRPC-Web) | MEDIUM (alpha) |
| `connectrpc.com/otelconnect` | latest | OpenTelemetry for Connect | HIGH |
| `connectrpc.com/validate` | latest | Proto validation interceptor | HIGH |
| `connectrpc.com/grpcreflect` | latest | gRPC reflection for Connect | HIGH |
| `connectrpc.com/cors` | latest | CORS header helpers for Connect protocols | HIGH |
| `buf.build/gen/go/...` | latest | Buf BSR generated code (optional) | MEDIUM |

## MVP Recommendation

### Phase 1: Core Unified Server

Priority: Ship the core Connect + Vanguard integration.

1. **`ConnectRegistrar` interface** — in `server/vanguard/` package
2. **`ConnectInterceptorBundle` interface** — mirrors gRPC pattern
3. **Vanguard server** — auto-discovers registrars, assembles transcoder, serves on single port
4. **H2C support** — Go 1.24+ `http.Protocols` for gRPC without TLS
5. **Basic config** — address, read/write timeouts
6. **Module wiring** — `gaz.NewModule("vanguard")` with DI registration

### Phase 2: Middleware & Observability

Priority: Production-ready middleware stack.

1. **OTEL integration** — `otelconnect` interceptor bundle + `otelhttp` transport middleware
2. **Validation bundle** — `connectrpc.com/validate` as default interceptor
3. **CORS middleware** — transport-level with `connectrpc.com/cors` helpers
4. **Auth interceptor pattern** — reference implementation
5. **Access logging** — both transport and RPC layers

### Phase 3: Polish & Migration

Priority: Migration path and documentation.

1. **gRPC bridge** — `vanguardgrpc.NewTranscoder` for existing `grpc.Registrar` services
2. **Deprecation of `server/gateway`** — deprecation notices, migration guide
3. **Updated `server/module.go`** — unified module bundles Vanguard instead of Gateway
4. **Examples** — reference implementation in `examples/`

### Defer

- **WebSocket support** — different paradigm, out of scope
- **Custom REST routing** — use proto annotations
- **TLS management** — document, don't automate

## Sources

- Context7: `connectrpc.com/connect` documentation (v1.18.1) — HIGH confidence
- Context7: `connectrpc.com/vanguard` documentation (v0.4.0) — HIGH confidence
- Context7: `buf.build/blog` Vanguard REST demo — MEDIUM confidence
- Existing `gaz` codebase: `server/grpc/`, `server/gateway/`, `di/` — direct analysis
- `.planning/PROJECT.md` — project constraints and milestone definition
