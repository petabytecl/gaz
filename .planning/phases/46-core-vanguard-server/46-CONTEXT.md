# Phase 46: Core Vanguard Server - Context

**Gathered:** 2026-03-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Single-port Vanguard server serving gRPC, Connect, gRPC-Web, and REST via Vanguard transcoder with ConnectRegistrar auto-discovery. This phase delivers the core server infrastructure — middleware, interceptors, CORS, and OTEL integration belong to Phase 47; server module bundling and gateway removal belong to Phase 48.

</domain>

<decisions>
## Implementation Decisions

### ConnectRegistrar Interface
- Single method interface: `RegisterConnect() (string, http.Handler)` — returns the service path and handler
- Mirrors the simplicity of the existing gRPC `Registrar` interface (single method, auto-discovered via `di.List`)
- Lives in new `server/connect/` package
- Auto-discovered via `di.ResolveAll[ConnectRegistrar]` in the Vanguard server's `OnStart`
- gRPC reflection implemented as a Connect handler using `connectrpc.com/grpcreflect` — registered as a built-in ConnectRegistrar when reflection is enabled

### Vanguard Server Architecture
- New `server/vanguard/` package owns the Vanguard transcoder and the h2c `http.Server`
- Creates its OWN `http.Server` — does NOT reuse `server/http` package (different lifecycle and protocol needs)
- Wraps the existing `*grpc.Server` from `server/grpc` via `vanguardgrpc.NewTranscoder` to bridge gRPC services into the Vanguard handler
- gRPC server gets a "skip listener" mode: still creates `*grpc.Server` and registers services via Registrars, but does NOT bind a port or call `server.Serve()` — Vanguard handles all incoming connections
- Vanguard transcoder is built during `OnStart` (one-shot construction) — not in the provider function
- h2c enabled via Go 1.26+ native `http.Protocols` configuration on `http.Server`

### gRPC Server Skip-Listener Mode
- Add a config flag `SkipListener bool` to gRPC `Config`
- When `SkipListener` is true, `OnStart` still discovers registrars, registers services, enables reflection, wires health — but skips `net.Listen` and `server.Serve()`
- `OnStop` with `SkipListener` calls `GracefulStop()` directly (no listener to close)
- Vanguard server resolves `*grpc.Server` from DI and passes it to `vanguardgrpc.NewTranscoder`
- CLI flag: `--grpc-skip-listener` (default: false for backward compatibility; Vanguard module sets it to true)

### Non-RPC Handler Mounting
- Single fallback handler via Vanguard's unknown handler mechanism — NOT a full HTTP mux
- Health endpoints (`/healthz`, `/readyz`, `/livez`) auto-registered if `health.Manager` is present in DI container
- Health responses use IETF Health Check Response format (`application/health+json`) — same as existing `health.Manager` output
- `SetUnknownHandler(h http.Handler)` method on the Vanguard server for user-defined custom HTTP routes
- The unknown handler receives any request that doesn't match a gRPC, Connect, or REST (proto annotation) route

### Config & CLI Flags
- Flag prefix: `server-` (NOT `vanguard-`) — this IS the server from the user's perspective
- Default port: 8080
- Streaming-safe timeout defaults: `ReadTimeout=0`, `WriteTimeout=0`, `ReadHeaderTimeout=5s`, `IdleTimeout=120s`
- Zero read/write timeouts are intentional — streaming RPCs can last arbitrarily long; the header timeout prevents slowloris
- `--server-dev-mode` flag: enables human-readable logging, relaxed timeouts, and dev-friendly defaults
- Config struct follows existing pattern: `Flags()`, `SetDefaults()`, `Validate()`, `Namespace()` methods

### Claude's Discretion
- Internal package structure within `server/vanguard/` (single file vs multiple files)
- Exact error messages and log field names (follow existing conventions)
- Whether to expose Vanguard-specific options (like service codecs) via config or keep them internal
- Test helper design in `gaztest` for Vanguard server testing

</decisions>

<specifics>
## Specific Ideas

- ConnectRegistrar follows the exact same auto-discovery pattern as gRPC Registrar — developers who know one immediately understand the other
- The gRPC server's `GRPCServer()` accessor already exists and returns `*grpc.Server` — Vanguard server uses this to get the server instance for transcoding
- `vanguardgrpc.NewTranscoder` is the bridge — it takes a `*grpc.Server` and produces a Vanguard service handler that the transcoder can dispatch to
- Health check mounting on the Vanguard server should feel automatic — if you have `health.Manager` in DI, you get `/healthz` on the server port without any extra config

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `server/grpc.Registrar` interface and `di.ResolveAll[Registrar]` auto-discovery pattern — directly cloned for ConnectRegistrar
- `server/grpc.Server.GRPCServer()` accessor — Vanguard server resolves this to get the `*grpc.Server` for transcoding
- `server/grpc.Config` with `Flags()/SetDefaults()/Validate()/Namespace()` — template for Vanguard config
- `server/grpc.healthAdapter` — pattern for bridging `health.Manager` to protocol-specific health endpoints
- `server/http.Server` — lifecycle pattern (OnStart/OnStop with graceful shutdown) to replicate in Vanguard server

### Established Patterns
- Module pattern: `gaz.NewModule("name").Flags().Provide().Build()` — Vanguard module will follow this
- Provider with DI resolution: `func NewServer(cfg Config, logger *slog.Logger, container *di.Container, tp *sdktrace.TracerProvider) *Server` — Vanguard server provider follows same signature pattern
- Auto-discovery: `di.ResolveAll[Interface]` in `OnStart` — not in constructor
- Interceptor bundles: priority-sorted, auto-discovered — Phase 47 will add Connect equivalent
- Health adapter: resolves `*health.Manager` optionally, registers if found

### Integration Points
- `server/grpc.Server` — Vanguard resolves this from DI to get the `*grpc.Server` for transcoding; gRPC server must be in "skip listener" mode
- `health.Manager` — Optional DI resolution for auto-wiring health endpoints on Vanguard port
- `di.Container` — ConnectRegistrar and gRPC Registrar both use `di.ResolveAll` for service discovery
- `*sdktrace.TracerProvider` — Optional OTEL integration (Phase 47 adds full OTEL, but provider resolution pattern exists)

</code_context>

<deferred>
## Deferred Ideas

- CORS middleware for browser clients — Phase 47 (MDDL-01)
- Connect interceptor bundles (auth, logging, validation) — Phase 47 (CONN-02, CONN-03)
- OTEL instrumentation for Connect RPC layer — Phase 47 (MDDL-03)
- Proto constraint validation interceptor — Phase 47 (MDDL-04)
- Updated `server.NewModule()` bundling — Phase 48 (SMOD-01)
- Gateway package removal — Phase 48 (SMOD-02)
- Migration guide and examples — Deferred (MIGR-01, MIGR-02, EXMP-01)

</deferred>

---

*Phase: 46-core-vanguard-server*
*Context gathered: 2026-03-06*
