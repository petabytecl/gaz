# Stack Research

**Domain:** Go application framework — unified HTTP/gRPC server with Vanguard multiplexer
**Researched:** 2026-03-06
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `connectrpc.com/connect` | v1.19.1 | Connect protocol server/client, handler generation | Industry standard for modern gRPC-compatible APIs. 4,556 importers on pkg.go.dev. Serves gRPC, gRPC-Web, and Connect protocols simultaneously from native `http.Handler` implementations. Eliminates the need for separate gRPC-Gateway proxy. |
| `connectrpc.com/vanguard` | v0.4.0 | Unified HTTP multiplexer — REST transcoding, protocol translation | Replaces `grpc-gateway/v2` entirely. Wraps existing `*grpc.Server` via `vanguardgrpc.NewTranscoder()` to serve gRPC, Connect, gRPC-Web, AND REST (via google.api.http annotations) on a single port through a single `http.Handler`. Published 2026-03-04. |
| `connectrpc.com/otelconnect` | v0.9.0 | OpenTelemetry instrumentation for Connect handlers | Drop-in OTEL interceptor for Connect. Supports custom `TracerProvider` and `MeterProvider` via options. Replaces `otelgrpc` for Connect-protocol handlers. Published 2026-01-05. |
| `connectrpc.com/validate` | v0.6.0 | Protovalidate interceptor for Connect handlers | Uses same `buf.build/go/protovalidate` engine already in go.mod. Replaces `go-grpc-middleware/v2/interceptors/protovalidate` for Connect handlers. Published 2025-09-27. |

### Supporting Libraries (Already in go.mod — Keep)

| Library | Version | Purpose | Reason to Keep |
|---------|---------|---------|----------------|
| `google.golang.org/grpc` | v1.79.1 | Core gRPC server implementation | Vanguard wraps `*grpc.Server` — existing gRPC services, interceptors, and reflection continue to work unchanged through the transcoder. |
| `buf.build/go/protovalidate` | v1.1.3 | Protobuf validation engine | Shared by both gRPC `ValidationBundle` and Connect `connectrpc.com/validate` interceptor. Single validation engine, two protocol entry points. |
| `rs/cors` | v1.11.1 | CORS middleware | Still needed for the unified Vanguard handler. Connect and REST clients from browsers require CORS. Wraps the Vanguard `http.Handler`. |
| `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` | v0.66.0 | HTTP-level OTEL instrumentation | Wraps the Vanguard handler at the HTTP layer. Complements `otelconnect` (RPC-level) and `otelgrpc` (gRPC-native level). |
| `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` | v0.66.0 | gRPC-level OTEL instrumentation | Still active on the underlying `*grpc.Server` for native gRPC protocol requests routed through Vanguard. |
| `google.golang.org/genproto/googleapis/api` | v0.0.0-... | `google.api.http` annotations | Vanguard reads these annotations from protobuf descriptors to perform REST transcoding. Already a dependency. |
| `grpc-ecosystem/go-grpc-middleware/v2` | v2.3.3 | gRPC interceptor chains | Still used by gRPC `InterceptorBundle` system (logging, recovery, auth, ratelimit, validation). Interceptors run on the `*grpc.Server` that Vanguard wraps. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `buf` CLI | Protobuf management, code generation | Generate both `protoc-gen-go-grpc` (gRPC stubs) and `protoc-gen-connect-go` (Connect handlers). Use `buf.gen.yaml` to configure both plugins. |
| `protoc-gen-connect-go` | Connect handler code generation | Generates `(path string, handler http.Handler)` tuples from `.proto` files. Install via `go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest`. |

## Installation

```bash
# New direct dependencies for v5.0
go get connectrpc.com/connect@v1.19.1
go get connectrpc.com/vanguard@v0.4.0
go get connectrpc.com/otelconnect@v0.9.0
go get connectrpc.com/validate@v0.6.0

# Code generation tool (for consumers, not gaz itself)
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| `connectrpc.com/vanguard` (single-port transcoder) | `grpc-gateway/v2` (two-port proxy) | Never for new projects. Gateway requires a loopback gRPC connection, separate port, and generates HTTP client code. Vanguard does the same transcoding natively on one port with zero code generation. |
| `connectrpc.com/vanguard` (transcoder wrapping `*grpc.Server`) | Pure Connect handlers only (no `*grpc.Server`) | If you never need native gRPC protocol support (e.g., internal services called only by browser/mobile via Connect). Gaz needs both because existing users have gRPC `Registrar` implementations. |
| `connectrpc.com/validate` | `go-grpc-middleware/v2/interceptors/protovalidate` | Only for pure-gRPC interceptor chains. Both use the same `buf.build/go/protovalidate` engine. The gRPC middleware version stays for the `*grpc.Server` interceptor chain; the Connect version is for Connect handler interceptors. |
| `connectrpc.com/otelconnect` (Connect interceptor) | `otelhttp` only (HTTP-level tracing) | `otelhttp` gives HTTP-level spans but lacks RPC-level metadata (procedure name, protocol, error codes). Use `otelconnect` for Connect handlers to get proper RPC semantics in traces. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `grpc-gateway/v2` (for new development) | Requires separate port, loopback gRPC connection, and generated gateway code. Vanguard replaces all of this with zero generated code, single port, and wraps the existing `*grpc.Server` directly. | `connectrpc.com/vanguard` with `vanguardgrpc.NewTranscoder()` |
| `cmux` (connection multiplexing) | Fragile byte-sniffing, breaks with TLS termination at load balancer, known K8s ingress issues. Already rejected in v4.1 research. | Vanguard — protocol detection at HTTP/2 layer, not TCP byte sniffing. |
| `grpc.NewClient` loopback in Gateway | No longer needed. Vanguard wraps `*grpc.Server` in-process — no network roundtrip, no connection management, no port coordination. | `vanguardgrpc.NewTranscoder(grpcServer)` |
| `connect.WithGRPC()` client option for server-side | This is for Connect *clients* calling gRPC servers. Don't confuse with server-side protocol support. Vanguard handles protocol negotiation server-side. | Vanguard transcoder handles all protocols automatically. |

## Integration Architecture

### How Vanguard Wraps the Existing gRPC Server

```
                          +-----------------------+
                          |   net/http.Server      |
                          |   (single port :8080)  |
                          +----------+------------+
                                     |
                          +----------v------------+
                          |   otelhttp.Handler     |
                          +----------+------------+
                                     |
                          +----------v------------+
                          |   cors.Handler         |
                          +----------+------------+
                                     |
                    +----------------v------------------+
                    |        http.ServeMux               |
                    +---+----------+----------+---------+
                        |          |          |
              +---------v--+  +---v------+  +v---------+
              | /api/...   |  | /grpc/...|  | /health  |
              | Vanguard   |  | Connect  |  | Plain    |
              | Transcoder |  | Handlers |  | HTTP     |
              +-----+------+  +----------+  +----------+
                    |
          +---------v---------+
          |  *grpc.Server     |
          |  (with existing   |
          |   interceptors)   |
          +-------------------+
```

### Key Integration Points

1. **Vanguard Transcoder** (`vanguardgrpc.NewTranscoder`): Takes the existing `*grpc.Server` and produces an `http.Handler`. All registered gRPC services are auto-discovered from the server's reflection info. REST transcoding uses `google.api.http` annotations from proto descriptors.

2. **Connect Handlers**: Generated via `protoc-gen-connect-go`, return `(path, http.Handler)` tuples. Mount directly on `http.ServeMux`. Interceptors applied per-handler via `connect.WithInterceptors()`.

3. **Unified `http.ServeMux`**: The Go 1.22+ enhanced `ServeMux` routes to Vanguard transcoder, Connect handlers, and plain HTTP handlers by path prefix.

4. **H2C for gRPC**: Go 1.26 `http.Protocols` API with `SetUnencryptedHTTP2(true)` enables HTTP/2 cleartext (required for gRPC protocol over plain HTTP/2). No need for `golang.org/x/net/http2/h2c` package.

### Interceptor/Middleware Mapping

| Concern | gRPC (existing) | Connect (new) | HTTP (existing) |
|---------|-----------------|---------------|-----------------|
| Logging | `LoggingBundle` via `go-grpc-middleware` | Connect `Interceptor` (new) | `otelhttp` |
| Recovery | `RecoveryBundle` via `go-grpc-middleware` | Connect `Interceptor` with `recover()` | HTTP middleware |
| Validation | `ValidationBundle` via `go-grpc-middleware` | `connectrpc.com/validate` interceptor | N/A |
| Auth | `AuthBundle` via `go-grpc-middleware` | Connect `Interceptor` (new) | HTTP middleware |
| OTEL | `otelgrpc` stats handler | `otelconnect` interceptor | `otelhttp` handler |
| Rate Limit | `RateLimitBundle` via `go-grpc-middleware` | Connect `Interceptor` (new) | HTTP middleware |
| CORS | N/A (binary protocol) | `rs/cors` on `http.Handler` | `rs/cors` on `http.Handler` |

### What Stays Unchanged

- `server/grpc.Registrar` interface and auto-discovery pattern.
- `server/grpc.InterceptorBundle` interface and all built-in bundles.
- `server/grpc.Server.GRPCServer()` method — returns `*grpc.Server` for Vanguard wrapping.
- `server/http.Server` — kept for HTTP-only use cases (standalone health endpoints, metrics).
- All existing gRPC service implementations (no code changes needed).

### What Gets Replaced

| v4.1 Component | v5.0 Replacement | Migration |
|----------------|------------------|-----------|
| `server/gateway.Gateway` | `server/vanguard.Server` (new) | Gateway's REST transcoding done by Vanguard transcoder. No more loopback connection. |
| `server/gateway.Registrar` | Removed entirely | Services no longer need `RegisterGateway()` method. Vanguard auto-discovers from `*grpc.Server`. |
| `server/gateway.Config` (CORS, GRPCTarget) | CORS config moves to vanguard module. GRPCTarget eliminated. | Config simplified — no port coordination. |
| Separate gRPC port + Gateway port | Single Vanguard port | One port serves all protocols. |

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| `connectrpc.com/connect@v1.19.1` | `google.golang.org/grpc@v1.79.1` | Connect servers can receive gRPC protocol requests natively. Vanguard bridges the two. |
| `connectrpc.com/vanguard@v0.4.0` | `google.golang.org/grpc@v1.79.1` | `vanguardgrpc.NewTranscoder()` accepts `*grpc.Server` from any v1.x grpc-go. |
| `connectrpc.com/vanguard@v0.4.0` | `connectrpc.com/connect@v1.19.1` | Vanguard can also route to Connect handlers via `vanguard.NewService()`. |
| `connectrpc.com/otelconnect@v0.9.0` | `go.opentelemetry.io/otel@v1.41.0` | Supports OTEL SDK v1. Compatible with existing TracerProvider/MeterProvider. |
| `connectrpc.com/validate@v0.6.0` | `buf.build/go/protovalidate@v1.1.3` | Same validation engine as existing `ValidationBundle`. |
| Go 1.26 `http.Protocols` | All HTTP handlers | Native H2C support — `SetUnencryptedHTTP2(true)` eliminates need for `x/net/http2/h2c`. |

## Stability Assessment

| Package | Status | Risk | Mitigation |
|---------|--------|------|------------|
| `connectrpc.com/connect@v1.19.1` | **Stable** (v1.x, 4,556 importers) | LOW | Production-ready. Backed by Buf team. |
| `connectrpc.com/vanguard@v0.4.0` | **Alpha** (v0.x, 22 importers) | MEDIUM | API may change before v1.0. Pin exact version. Wrap behind gaz's own interfaces so internal consumers are insulated. Vanguard's core transcoding logic is mature (same engine as Buf's hosted services). |
| `connectrpc.com/otelconnect@v0.9.0` | **Pre-stable** (v0.x) | LOW | API is simple (one constructor + options). Unlikely to break. |
| `connectrpc.com/validate@v0.6.0` | **Pre-stable** (v0.x) | LOW | Thin wrapper over `protovalidate`. API is one function. |

**Vanguard v0.4.0 risk mitigation**: Gaz wraps Vanguard behind `server/vanguard.Server` with its own `Config`, `Registrar` interfaces, and DI module. If Vanguard's API changes in v0.5+, only the wrapper needs updating — no consumer code changes. This is the same pattern used for `grpc-gateway/v2` in v4.1.

## Sources

- `connectrpc.com/connect` — pkg.go.dev (v1.19.1, published 2025-10-07, verified 2026-03-06)
- `connectrpc.com/vanguard` — pkg.go.dev (v0.4.0, published 2026-03-04, verified 2026-03-06)
- `connectrpc.com/otelconnect` — pkg.go.dev (v0.9.0, published 2026-01-05, verified 2026-03-06)
- `connectrpc.com/validate` — pkg.go.dev (v0.6.0, published 2025-09-27, verified 2026-03-06)
- Context7: `/connectrpc/connect-go` — interceptor patterns, handler creation, protocol support
- Context7: `/connectrpc/vanguard-go` — transcoder API, gRPC wrapping, service registration
- Context7: `/connectrpc/connectrpc.com` — observability, validation, architecture overview
- Existing codebase: `server/grpc/`, `server/gateway/`, `server/http/` packages

---
*Stack research for: gaz v5.0 Vanguard Unified Server*
*Researched: 2026-03-06*
