# Phase 47: Middleware & Interceptors - Context

**Gathered:** 2026-03-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Two-layer middleware stack for the Vanguard server: HTTP transport middleware for cross-cutting concerns (CORS, OTEL) and Connect interceptors for RPC semantics (auth, logging, validation, recovery) — with auto-discovered, priority-sorted chains for both layers. The Vanguard server auto-injects interceptors into all Connect handlers without per-service wiring.

</domain>

<decisions>
## Implementation Decisions

### CORS Configuration
- Permissive (AllowAll) in dev mode, strict (explicit origins required) in production — controlled by existing `DevMode` flag in Vanguard Config
- CORS settings (allowed origins, methods, headers) configurable via CLI flags and config struct
- CORS fields live inside `vanguard.Config` — same struct, same `--server-` prefix (e.g., `--server-cors-origins`)
- Always enabled — no separate enable/disable flag; browser clients always need CORS headers
- Uses `rs/cors` library (already a dependency in go.mod)

### Connect Interceptor Bundle
- `ConnectInterceptorBundle` interface mirrors gRPC `InterceptorBundle`: `Name() string`, `Priority() int`, `Interceptors() []connect.Interceptor`
- Returns `[]connect.Interceptor` (Connect uses a single interceptor type for both unary and streaming, unlike gRPC's separate unary/stream)
- Full set of built-in bundles matching gRPC parity: logging, recovery, auth (opt-in via auth function), rate-limit, validation (via `connectrpc.com/validate`)
- Same priority constants as gRPC: PriorityLogging=0, PriorityRateLimit=25, PriorityAuth=50, PriorityValidation=100, PriorityRecovery=1000
- Lives in `server/connect/` package alongside existing `Registrar`
- Auto-discovered via `di.ResolveAll[connect.InterceptorBundle]` — same pattern as gRPC

### Interceptor Injection
- Vanguard server resolves all `ConnectInterceptorBundle` implementations from DI, builds the interceptor chain, and passes `connect.WithInterceptors()` to Connect handlers
- `ConnectRegistrar.RegisterConnect()` signature changes to accept `connect.WithInterceptors()` option — Vanguard server calls registrars with interceptors pre-configured
- All Connect handlers get the same interceptor chain automatically — no per-service wiring needed

### HTTP Transport Middleware Layer
- Two concerns at transport level: CORS and OTEL (otelhttp) — everything else belongs at interceptor level
- DI-based extensibility via `TransportMiddleware` interface: `Name() string`, `Priority() int`, `Wrap(http.Handler) http.Handler`
- Auto-discovered via `di.ResolveAll[TransportMiddleware]`, sorted by priority, applied in order
- CORS and OTEL are built-in `TransportMiddleware` implementations at fixed priorities
- Middleware ordering: CORS (lowest priority, runs first) -> OTEL -> user middleware -> Vanguard handler
- TransportMiddleware interface lives in `server/vanguard/` package (transport-level concern)

### OTEL Dual-Layer Instrumentation
- Single activation: if `*sdktrace.TracerProvider` exists in DI, both otelhttp (transport) and otelconnect (Connect RPC) are wired automatically
- No separate flags — presence of TracerProvider is the toggle (consistent with how gRPC OTEL works)
- Filter health checks (`/healthz`, `/readyz`, `/livez`) and reflection endpoints from traces — reduces noise
- otelhttp creates parent span at transport level; otelconnect creates child span at RPC level — standard OTEL propagation ensures correlated trace IDs
- Auto-wired by Vanguard server in OnStart — no separate OTEL middleware module needed
- New dependency: `connectrpc.com/otelconnect` for Connect-side instrumentation

### Proto Validation
- Uses `connectrpc.com/validate` interceptor for proto constraint validation at the Connect interceptor level
- Rejects invalid requests before they reach handler logic — same behavior as gRPC's `protovalidate` interceptor
- New dependency: `connectrpc.com/validate`

### Claude's Discretion
- Exact CORS flag names and defaults for strict mode (origin list format, default methods/headers)
- ConnectRegistrar signature evolution — how to pass interceptors while maintaining backward compatibility with existing registrars
- TransportMiddleware priority values for CORS and OTEL built-ins
- Connect auth function type design (Connect uses different auth patterns than gRPC metadata)
- Connect logging interceptor implementation (no go-grpc-middleware equivalent for Connect)
- Connect recovery interceptor implementation (panic recovery for Connect handlers)
- Whether to share priority constants between gRPC and Connect packages or duplicate them

</decisions>

<specifics>
## Specific Ideas

- ConnectInterceptorBundle follows the exact same pattern as gRPC InterceptorBundle — developers who know one immediately understand the other
- TransportMiddleware follows the same Name/Priority/auto-discovery pattern as InterceptorBundle — three consistent DI-based extension points (gRPC interceptors, Connect interceptors, transport middleware)
- CORS in dev mode should be truly permissive (AllowAll) to eliminate CORS friction during development — production requires explicit configuration
- The gateway package already uses `rs/cors` with OTEL — the same patterns move to the Vanguard server, gateway gets removed in Phase 48

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- `server/grpc/InterceptorBundle` interface — direct template for `ConnectInterceptorBundle`
- `server/grpc/collectInterceptors()` — pattern for collecting, sorting, and chaining bundles from DI
- `server/grpc` built-in bundles (LoggingBundle, RecoveryBundle, AuthBundle, RateLimitBundle, ValidationBundle) — templates for Connect equivalents
- `server/grpc/module.go` provider functions — templates for Connect bundle registration
- `rs/cors` already in go.mod — reuse directly in Vanguard transport middleware
- `otelhttp` already in go.mod — reuse directly in Vanguard transport middleware
- `server/otel/module.go` TracerProvider pattern — Vanguard resolves this optionally

### Established Patterns
- Priority-sorted auto-discovery: `di.ResolveAll[Interface]` + `sort.Slice` by Priority() — used for gRPC interceptors, replicated for Connect interceptors and TransportMiddleware
- Optional DI resolution: resolve TracerProvider, health.Manager etc. with fallback if not registered
- Module pattern: `gaz.NewModule("name").Flags().Provide().Build()` — Vanguard module extended with middleware providers
- Config pattern: struct with `Namespace()`, `Flags()`, `SetDefaults()`, `Validate()` — CORS fields added to vanguard.Config

### Integration Points
- `server/vanguard/server.go` OnStart — middleware wrapping happens here, around the transcoder handler
- `server/vanguard/module.go` — extended with TransportMiddleware and Connect interceptor bundle providers
- `server/connect/registrar.go` — ConnectInterceptorBundle interface and built-in bundles added alongside Registrar
- `*sdktrace.TracerProvider` — optional resolution triggers OTEL wiring at both layers
- `server/vanguard/config.go` — CORS fields added to Config struct

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 47-middleware-interceptors*
*Context gathered: 2026-03-06*
