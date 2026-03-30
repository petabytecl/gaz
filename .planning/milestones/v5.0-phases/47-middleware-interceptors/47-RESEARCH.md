# Phase 47: Middleware & Interceptors - Research

**Researched:** 2026-03-06
**Domain:** Connect interceptors, HTTP transport middleware, CORS, OpenTelemetry
**Confidence:** HIGH

## Summary

Phase 47 adds a two-layer middleware stack to the Vanguard server: HTTP transport middleware for cross-cutting concerns (CORS, OTEL) and Connect interceptors for RPC semantics (auth, logging, validation, recovery). Both layers use the same DI auto-discovery + priority-sorting pattern already established in `server/grpc/interceptors.go`.

The gRPC interceptor system (`InterceptorBundle` interface, `collectInterceptors()`, five built-in bundles, priority constants) is the direct template. Connect interceptors differ in one key way: Connect uses a single `connect.Interceptor` type for both unary and streaming, so `ConnectInterceptorBundle.Interceptors()` returns `[]connect.Interceptor` instead of gRPC's `(unary, stream)` pair.

Two new dependencies are required: `connectrpc.com/validate` (proto constraint validation) and `connectrpc.com/otelconnect` (Connect-level OTEL instrumentation). Both must be added to `go.mod` and to the depguard allow lists in `.golangci.yml`. All other dependencies (`rs/cors`, `otelhttp`, `connect`) are already available.

**Primary recommendation:** Mirror the gRPC interceptor bundle pattern exactly for Connect, add TransportMiddleware as a parallel DI-based extension point, and wire both layers in Vanguard's `OnStart` method.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- CORS permissive (AllowAll) in dev mode, strict (explicit origins) in production via existing `DevMode` flag
- CORS settings configurable via CLI flags and config struct, fields inside `vanguard.Config` with `--server-cors-*` prefix
- CORS always enabled, no separate enable/disable flag; uses `rs/cors` library
- `ConnectInterceptorBundle` interface mirrors gRPC `InterceptorBundle`: `Name() string`, `Priority() int`, `Interceptors() []connect.Interceptor`
- Full set of built-in bundles: logging, recovery, auth (opt-in), rate-limit, validation (via `connectrpc.com/validate`)
- Same priority constants as gRPC: PriorityLogging=0, PriorityRateLimit=25, PriorityAuth=50, PriorityValidation=100, PriorityRecovery=1000
- ConnectInterceptorBundle lives in `server/connect/` package alongside existing `Registrar`
- Auto-discovered via `di.ResolveAll[connect.InterceptorBundle]`
- Vanguard server resolves all ConnectInterceptorBundle implementations, builds chain, passes `connect.WithInterceptors()` to Connect handlers
- `Registrar.RegisterConnect()` signature changes to accept `connect.WithInterceptors()` option
- Two transport-level concerns: CORS and OTEL (otelhttp)
- DI-based `TransportMiddleware` interface: `Name() string`, `Priority() int`, `Wrap(http.Handler) http.Handler`
- TransportMiddleware auto-discovered via `di.ResolveAll[TransportMiddleware]`, sorted by priority
- CORS and OTEL are built-in TransportMiddleware implementations
- Middleware ordering: CORS (lowest priority, runs first) -> OTEL -> user middleware -> Vanguard handler
- TransportMiddleware lives in `server/vanguard/` package
- Single OTEL activation: if `*sdktrace.TracerProvider` exists in DI, both otelhttp and otelconnect are wired
- Filter health/reflection endpoints from traces
- otelhttp parent span + otelconnect child span with standard propagation
- New dependencies: `connectrpc.com/validate`, `connectrpc.com/otelconnect`
- Proto validation via `connectrpc.com/validate` interceptor

### Claude's Discretion
- Exact CORS flag names and defaults for strict mode (origin list format, default methods/headers)
- ConnectRegistrar signature evolution — how to pass interceptors while maintaining backward compatibility
- TransportMiddleware priority values for CORS and OTEL built-ins
- Connect auth function type design (Connect uses different auth patterns than gRPC metadata)
- Connect logging interceptor implementation (no go-grpc-middleware equivalent for Connect)
- Connect recovery interceptor implementation (panic recovery for Connect handlers)
- Whether to share priority constants between gRPC and Connect packages or duplicate them

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CONN-02 | Framework automatically injects Connect interceptors (auth, logging, validation, OTEL) into all Connect handlers | ConnectInterceptorBundle auto-discovery + Vanguard injection via `connect.WithInterceptors()` passed through modified `Registrar.RegisterConnect()` signature |
| CONN-03 | Developer can create ConnectInterceptorBundle with priority-sorted, auto-discovered interceptor chains | `ConnectInterceptorBundle` interface mirroring gRPC `InterceptorBundle`, `collectConnectInterceptors()` function, priority constants |
| MDDL-01 | Developer can apply CORS middleware at transport level for browser clients | `rs/cors` as built-in `TransportMiddleware` with CORS fields in `vanguard.Config`, AllowAll in dev mode |
| MDDL-02 | Vanguard server uses two-layer middleware model | `TransportMiddleware` interface in `server/vanguard/` (HTTP layer) + `ConnectInterceptorBundle` in `server/connect/` (RPC layer), both wired in Vanguard `OnStart` |
| MDDL-03 | Developer can enable OTEL tracing and metrics for both HTTP transport and Connect RPC layers | `otelhttp` as built-in `TransportMiddleware` + `otelconnect.NewInterceptor()` as built-in `ConnectInterceptorBundle`, activated by presence of `*sdktrace.TracerProvider` in DI |
| MDDL-04 | Developer can enable proto constraint validation via connectrpc.com/validate | `connectrpc.com/validate` interceptor wrapped in `ValidationBundle` ConnectInterceptorBundle |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `connectrpc.com/connect` | v1.19.1 | Connect interceptor types (`connect.Interceptor`, `connect.HandlerOption`) | Already in go.mod; defines interceptor interface |
| `connectrpc.com/validate` | v0.6.0 | Proto constraint validation interceptor for Connect | Official connectrpc.com ecosystem; implements `connect.Interceptor` |
| `connectrpc.com/otelconnect` | v0.9.0 | OpenTelemetry instrumentation for Connect RPC | Official connectrpc.com ecosystem; implements `connect.Interceptor` |
| `github.com/rs/cors` | v1.11.1 | CORS middleware for HTTP handlers | Already in go.mod; used by `server/gateway/` |
| `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` | v0.66.0 | OTEL instrumentation for HTTP transport | Already in go.mod; used by `server/gateway/` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go.opentelemetry.io/otel/sdk/trace` | (in go.mod) | TracerProvider type for DI resolution | Resolved from DI to activate OTEL layers |
| `buf.build/go/protovalidate` | (in go.mod) | Protovalidate validator (used by gRPC, NOT by Connect validate) | Not needed for Connect — `connectrpc.com/validate` handles it internally |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `connectrpc.com/validate` | Manual protovalidate interceptor | `connectrpc.com/validate` wraps protovalidate internally with proper Connect error formatting — hand-rolling loses error detail propagation |
| `connectrpc.com/otelconnect` | Manual OTEL interceptor | otelconnect handles span naming, attribute extraction, and metric recording per Connect conventions — significant effort to replicate correctly |

**Installation:**
```bash
go get connectrpc.com/validate@v0.6.0
go get connectrpc.com/otelconnect@v0.9.0
```

**Depguard additions required** (both `$all` and `non-test files` lists in `.golangci.yml`):
```yaml
- connectrpc.com/validate
- connectrpc.com/otelconnect
```

## Architecture Patterns

### File Structure
```
server/connect/
├── registrar.go          # Existing Registrar interface (signature updated)
├── interceptors.go       # NEW: ConnectInterceptorBundle interface, priority constants, collectConnectInterceptors(), built-in bundles
├── interceptors_test.go  # NEW: Tests for all bundles and collection logic
└── doc.go                # Existing package docs (updated)

server/vanguard/
├── config.go             # Updated: CORS fields added to Config
├── server.go             # Updated: middleware/interceptor wiring in OnStart
├── module.go             # Updated: TransportMiddleware + ConnectInterceptorBundle providers
├── middleware.go          # NEW: TransportMiddleware interface, built-in middleware (CORS, OTEL)
├── middleware_test.go     # NEW: Tests for middleware chain
└── health.go             # Existing (unchanged)
```

### Pattern 1: ConnectInterceptorBundle Interface
**What:** Mirrors the gRPC `InterceptorBundle` with Connect-specific return type.
**When to use:** Any Connect interceptor that should be auto-discovered and chained.
**Source:** `server/grpc/interceptors.go:58-72`

```go
// ConnectInterceptorBundle provides Connect interceptors for auto-discovery.
type ConnectInterceptorBundle interface {
    Name() string
    Priority() int
    Interceptors() []connect.Interceptor
}
```

Key difference from gRPC: returns `[]connect.Interceptor` (slice) because Connect uses a single type for both unary and streaming. A bundle MAY return multiple interceptors (e.g., validation might want request + response validation as separate interceptors).

### Pattern 2: collectConnectInterceptors()
**What:** Discovers, sorts, and flattens all ConnectInterceptorBundle implementations into a single interceptor slice.
**Source template:** `server/grpc/interceptors.go:76-106`

```go
func collectConnectInterceptors(container *di.Container, logger *slog.Logger) []connect.Interceptor {
    bundles, err := di.ResolveAll[ConnectInterceptorBundle](container)
    if err != nil {
        logger.Warn("failed to resolve connect interceptor bundles", slog.Any("error", err))
        return nil
    }

    sort.Slice(bundles, func(i, j int) bool {
        return bundles[i].Priority() < bundles[j].Priority()
    })

    var interceptors []connect.Interceptor
    for _, b := range bundles {
        interceptors = append(interceptors, b.Interceptors()...)
        logger.Debug("registered connect interceptor bundle",
            slog.String("name", b.Name()),
            slog.Int("priority", b.Priority()),
        )
    }
    return interceptors
}
```

### Pattern 3: TransportMiddleware Interface
**What:** DI-based HTTP middleware chain for transport-level concerns.
**When to use:** Cross-cutting concerns that apply to ALL HTTP traffic (not just RPC).

```go
// TransportMiddleware wraps the Vanguard HTTP handler for transport-level concerns.
type TransportMiddleware interface {
    Name() string
    Priority() int
    Wrap(http.Handler) http.Handler
}
```

### Pattern 4: Registrar Signature Evolution
**What:** Update `RegisterConnect()` to accept `connect.HandlerOption` options for interceptor injection.
**Design decision (Claude's discretion):**

**Recommended approach:** Change signature to accept variadic `connect.HandlerOption`:
```go
type Registrar interface {
    RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler)
}
```

This is a breaking change from Phase 46, but the Vanguard server is the only consumer and Phase 46 was just completed. The `opts` parameter passes through to Connect-Go generated `NewXxxServiceHandler(impl, opts...)`. This is the cleanest design — services don't need to know about interceptors, they just forward options.

### Pattern 5: Connect Auth Function Design
**What:** Auth function type for Connect interceptors.
**Design decision (Claude's discretion):**

Connect doesn't use gRPC metadata for auth. Instead, Connect interceptors operate on `connect.AnyRequest` which provides headers via `req.Header()`. The auth function for Connect should work with standard HTTP headers:

```go
// ConnectAuthFunc validates requests and returns an enriched context.
// Extract credentials from request headers (e.g., Authorization).
type ConnectAuthFunc func(ctx context.Context, req connect.AnyRequest) (context.Context, error)
```

This type lives in `server/connect/` and is registered in DI just like gRPC's `AuthFunc`. The auth bundle checks `gaz.Has[ConnectAuthFunc]` — same opt-in pattern.

### Pattern 6: Connect Logging Interceptor
**What:** Custom logging interceptor for Connect (no go-grpc-middleware equivalent exists).
**Design decision (Claude's discretion):**

Must be hand-written as a `connect.UnaryInterceptorFunc` / custom `connect.Interceptor` implementation. Log: procedure name, duration, error status, peer address. Use `slog.Logger` consistent with gRPC logging bundle.

```go
type loggingInterceptor struct {
    logger *slog.Logger
}

func (l *loggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
    return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
        start := time.Now()
        resp, err := next(ctx, req)
        l.logger.InfoContext(ctx, "connect rpc",
            slog.String("procedure", req.Spec().Procedure),
            slog.Duration("duration", time.Since(start)),
            slog.Bool("error", err != nil),
        )
        return resp, err
    }
}

func (l *loggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
    return next // Server-side only — pass through client streaming.
}

func (l *loggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
    return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
        start := time.Now()
        err := next(ctx, conn)
        l.logger.InfoContext(ctx, "connect stream",
            slog.String("procedure", conn.Spec().Procedure),
            slog.Duration("duration", time.Since(start)),
            slog.Bool("error", err != nil),
        )
        return err
    }
}
```

### Pattern 7: Connect Recovery Interceptor
**What:** Panic recovery for Connect handlers.
**Design decision (Claude's discretion):**

Same approach as gRPC recovery: catch panics, log stack trace, return `connect.CodeInternal` error. Dev mode exposes panic details, production returns generic error.

```go
func newRecoveryInterceptor(logger *slog.Logger, devMode bool) connect.Interceptor {
    return &recoveryInterceptor{logger: logger, devMode: devMode}
}

// WrapUnary wraps with panic recovery.
func (r *recoveryInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
    return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
        defer func() {
            if p := recover(); p != nil {
                r.logger.ErrorContext(ctx, "panic recovered in connect handler",
                    slog.Any("panic", p),
                    slog.String("stack", string(debug.Stack())),
                )
                if r.devMode {
                    err = connect.NewError(connect.CodeInternal, fmt.Errorf("panic: %v", p))
                } else {
                    err = connect.NewError(connect.CodeInternal, errors.New("internal server error"))
                }
            }
        }()
        return next(ctx, req)
    }
}
```

### Anti-Patterns to Avoid
- **Shared interceptor types between gRPC and Connect:** The type signatures are fundamentally different (`grpc.UnaryServerInterceptor` vs `connect.Interceptor`). Don't try to bridge them.
- **Injecting interceptors at server level (like gRPC):** Connect interceptors must be passed at handler construction time via `connect.WithInterceptors()`, not at server level. The Registrar signature change is the correct approach.
- **CORS at interceptor level:** CORS must be at HTTP transport level. Connect interceptors only see deserialized RPC messages, not raw HTTP headers needed for CORS preflight.

### Priority Constants Decision
**Design decision (Claude's discretion):**

**Recommended:** Duplicate priority constants in `server/connect/`. Sharing constants between packages creates an import dependency that doesn't exist naturally. The values are identical (0, 25, 50, 100, 1000) but each package owns its own:

```go
// In server/connect/interceptors.go
const (
    PriorityLogging    = 0
    PriorityRateLimit  = 25
    PriorityAuth       = 50
    PriorityValidation = 100
    PriorityRecovery   = 1000
)
```

### TransportMiddleware Priority Values Decision
**Design decision (Claude's discretion):**

**Recommended:**
```go
const (
    PriorityCORS = 0    // CORS runs first (outermost handler).
    PriorityOTEL = 100  // OTEL wraps after CORS.
)
```

User-defined transport middleware should use values between 1 and 99 to run between CORS and OTEL, or above 100 to run after OTEL. Lower priority = runs first (outermost in the handler chain).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Proto validation for Connect | Custom protovalidate interceptor | `connectrpc.com/validate` v0.6.0 | Handles Connect error formatting, detail attachment, and supports both request/response validation |
| Connect OTEL instrumentation | Manual span creation in interceptor | `connectrpc.com/otelconnect` v0.9.0 | Correct span naming, attribute extraction, metric recording per OTEL semantic conventions for RPC |
| CORS handling | Manual header manipulation | `rs/cors` v1.11.1 | Preflight caching, credential handling, origin matching edge cases are surprisingly complex |
| HTTP OTEL instrumentation | Manual otelhttp wrapper | `otelhttp` v0.66.0 | Proper span propagation, metric recording, request/response attribute extraction |

**Key insight:** The interceptor implementations (logging, recovery, auth, rate-limit) are the only hand-written pieces. Everything else has a well-tested library.

## Common Pitfalls

### Pitfall 1: Connect Interceptors at Wrong Level
**What goes wrong:** Trying to apply CORS or OTEL at the Connect interceptor level instead of HTTP transport level.
**Why it happens:** Intuition from frameworks where middleware and interceptors are interchangeable.
**How to avoid:** Two-layer model — transport middleware wraps `http.Handler`, interceptors wrap Connect RPCs. CORS and OTEL go at transport level.
**Warning signs:** CORS preflight requests failing, missing parent spans in traces.

### Pitfall 2: Interceptor Injection Timing
**What goes wrong:** Trying to add interceptors to Connect handlers after construction.
**Why it happens:** gRPC allows adding interceptors at server level via `grpc.ChainUnaryInterceptor()`.
**How to avoid:** Pass `connect.WithInterceptors()` at handler construction time in `RegisterConnect()`.
**Warning signs:** Interceptors not firing on Connect handlers.

### Pitfall 3: otelconnect.NewInterceptor Returns Error
**What goes wrong:** Ignoring the error return from `otelconnect.NewInterceptor()`.
**Why it happens:** Most interceptor constructors don't return errors.
**How to avoid:** `otelconnect.NewInterceptor(opts...)` returns `(*Interceptor, error)` — handle the error.
**Warning signs:** Compile error or nil pointer panic.

### Pitfall 4: CORS AllowAll with Credentials
**What goes wrong:** Using `cors.AllowAll()` with `AllowCredentials: true` — browsers reject this combination.
**Why it happens:** Dev mode sets `AllowedOrigins: ["*"]`.
**How to avoid:** Dev mode uses `AllowCredentials: false`. Production requires explicit origins to enable credentials.
**Warning signs:** Browser CORS errors despite CORS being "enabled".

### Pitfall 5: Missing Depguard Allow Lists
**What goes wrong:** Lint failures when importing `connectrpc.com/validate` or `connectrpc.com/otelconnect`.
**Why it happens:** New dependencies not added to `.golangci.yml` depguard lists.
**How to avoid:** Add both packages to BOTH `$all` and `non-test files` allow lists before writing any code.
**Warning signs:** `golangci-lint run` fails with depguard violations.

### Pitfall 6: Health/Reflection Endpoint Trace Noise
**What goes wrong:** Traces flooded with health check and reflection requests.
**Why it happens:** Kubernetes probes hit `/healthz` every few seconds, reflection tools poll continuously.
**How to avoid:** Filter `/healthz`, `/readyz`, `/livez` and reflection paths from both otelhttp and otelconnect.
**Warning signs:** Trace backend overwhelmed with non-business traffic.

## Code Examples

### connectrpc.com/validate Usage
**Source:** connectrpc.com/validate pkg.go.dev documentation
```go
import "connectrpc.com/validate"

// Create validation interceptor.
interceptor := validate.NewInterceptor()
// interceptor implements connect.Interceptor

// Use in handler options:
path, handler := greetv1connect.NewGreeterServiceHandler(svc,
    connect.WithInterceptors(interceptor),
)
```

`validate.NewInterceptor()` returns `*validate.Interceptor` (not an error). It uses protovalidate internally. Options available: `validate.WithValidator()`, `validate.WithValidateResponses()`.

### connectrpc.com/otelconnect Usage
**Source:** connectrpc.com/otelconnect pkg.go.dev documentation
```go
import "connectrpc.com/otelconnect"

// Create OTEL interceptor — NOTE: returns error.
otelInterceptor, err := otelconnect.NewInterceptor(
    otelconnect.WithTracerProvider(tp),
    otelconnect.WithMeterProvider(mp),
    otelconnect.WithFilter(func(_ context.Context, spec connect.Spec) bool {
        // Filter out health check RPCs.
        return spec.Procedure != "/grpc.health.v1.Health/Check"
    }),
)
if err != nil {
    return fmt.Errorf("create otelconnect interceptor: %w", err)
}
```

### rs/cors AllowAll Pattern
**Source:** `server/gateway/gateway.go:117-125` (existing codebase)
```go
// Dev mode — wide-open CORS.
corsHandler := cors.AllowAll()

// Production — explicit configuration.
corsHandler := cors.New(cors.Options{
    AllowedOrigins:   cfg.CORS.AllowedOrigins,
    AllowedMethods:   cfg.CORS.AllowedMethods,
    AllowedHeaders:   cfg.CORS.AllowedHeaders,
    ExposedHeaders:   cfg.CORS.ExposedHeaders,
    AllowCredentials: cfg.CORS.AllowCredentials,
    MaxAge:           cfg.CORS.MaxAge,
})
```

### TransportMiddleware Chain Application
```go
// In Vanguard OnStart, after building the transcoder handler:
middlewares, _ := di.ResolveAll[TransportMiddleware](s.container)
sort.Slice(middlewares, func(i, j int) bool {
    return middlewares[i].Priority() < middlewares[j].Priority()
})

// Apply in reverse order so lowest priority wraps outermost.
h := handler
for i := len(middlewares) - 1; i >= 0; i-- {
    h = middlewares[i].Wrap(h)
    s.logger.Debug("applied transport middleware",
        slog.String("name", middlewares[i].Name()),
        slog.Int("priority", middlewares[i].Priority()),
    )
}
s.httpServer.Handler = h
```

### CORS Flag Names Decision
**Design decision (Claude's discretion):**

Following the existing gateway pattern and the `--server-` prefix:
```go
fs.StringSliceVar(&c.CORS.AllowedOrigins, "server-cors-origins", c.CORS.AllowedOrigins, "Allowed CORS origins")
fs.StringSliceVar(&c.CORS.AllowedMethods, "server-cors-methods", c.CORS.AllowedMethods, "Allowed CORS methods")
fs.StringSliceVar(&c.CORS.AllowedHeaders, "server-cors-headers", c.CORS.AllowedHeaders, "Allowed CORS headers")
fs.StringSliceVar(&c.CORS.ExposedHeaders, "server-cors-exposed-headers", c.CORS.ExposedHeaders, "Exposed CORS headers")
fs.BoolVar(&c.CORS.AllowCredentials, "server-cors-credentials", c.CORS.AllowCredentials, "Allow CORS credentials")
fs.IntVar(&c.CORS.MaxAge, "server-cors-max-age", c.CORS.MaxAge, "CORS preflight max age (seconds)")
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gRPC-Gateway with separate CORS middleware | Vanguard transcoder with transport middleware chain | v5.0 (this milestone) | CORS moves from gateway package to vanguard transport middleware |
| Manual OTEL wiring per handler | connectrpc.com/otelconnect v0.9.0 | 2024 | Standardized Connect OTEL instrumentation with proper semantic conventions |
| Manual protovalidate interceptor | connectrpc.com/validate v0.6.0 | 2024 | Drop-in Connect interceptor for proto constraint validation |

**Deprecated/outdated:**
- `server/gateway/` CORS pattern: Still works, but will be removed in Phase 48. Vanguard's transport middleware replaces it.

## Open Questions

1. **Rate Limit Interface for Connect**
   - What we know: gRPC uses `ratelimit.Limiter` from go-grpc-middleware which takes `context.Context`.
   - What's unclear: Whether to reuse the same `Limiter` type or define a Connect-specific one that also receives `connect.AnyRequest`.
   - Recommendation: Define a `ConnectLimiter` interface with `Limit(ctx context.Context, req connect.AnyRequest) error` for richer rate-limiting decisions (e.g., per-procedure, per-user). If no `ConnectLimiter` is registered, use an `AlwaysPassLimiter` (same pattern as gRPC).

2. **otelconnect Filter Signature**
   - What we know: `otelconnect.WithFilter()` takes `func(context.Context, connect.Spec) bool`.
   - What's unclear: Exact procedures to filter (health check service paths via Connect).
   - Recommendation: Filter reflection and health procedures. The exact procedure paths need to be determined during implementation, but they follow the pattern `/{package}.{Service}/{Method}`.

## Sources

### Primary (HIGH confidence)
- `server/grpc/interceptors.go` — InterceptorBundle interface, priority constants, all 5 built-in bundles, collectInterceptors()
- `server/grpc/module.go` — Bundle provider registration pattern (provideLoggingBundle, provideAuthBundle opt-in, etc.)
- `server/grpc/server.go` — Interceptor chaining, OTEL stats handler wiring
- `server/connect/registrar.go` — Current Registrar interface
- `server/vanguard/server.go` — Current OnStart flow, handler construction
- `server/vanguard/config.go` — Config struct with DevMode flag
- `server/gateway/gateway.go` — CORS + otelhttp wrapping pattern
- `server/gateway/config.go` — CORSConfig struct template
- `connectrpc.com/connect` Context7 — connect.Interceptor interface, HandlerOption, WithInterceptors()
- `connectrpc.com/validate` pkg.go.dev — v0.6.0 API: NewInterceptor() returns *Interceptor
- `connectrpc.com/otelconnect` pkg.go.dev — v0.9.0 API: NewInterceptor() returns (*Interceptor, error)

### Secondary (MEDIUM confidence)
- `.golangci.yml` depguard section — verified current allow lists (lines ~435-503)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries verified via Context7, pkg.go.dev, and existing go.mod
- Architecture: HIGH — directly mirrors established gRPC interceptor patterns in the codebase
- Pitfalls: HIGH — derived from actual API differences between gRPC and Connect verified against source code

**Research date:** 2026-03-06
**Valid until:** 2026-04-06 (stable libraries, low churn risk)
