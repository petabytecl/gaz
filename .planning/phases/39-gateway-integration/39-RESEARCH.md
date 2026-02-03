# Phase 39: Gateway Integration - Research

**Researched:** 2026-02-03
**Domain:** gRPC-Gateway HTTP/REST proxy for gRPC services
**Confidence:** HIGH

## Summary

This phase implements a Gateway layer that unifies HTTP and gRPC by using grpc-gateway to translate RESTful HTTP/JSON requests into gRPC calls. The Gateway auto-discovers services implementing a `GatewayRegistrar` interface via the existing DI pattern (`di.ResolveAll`), connects to the gRPC server via loopback, and applies CORS middleware for cross-origin requests.

The standard approach uses grpc-gateway v2.x with `runtime.ServeMux` as the core HTTP handler, wrapping it with rs/cors middleware for CORS support. Error responses are customized to RFC 7807 Problem Details format using `runtime.WithErrorHandler`. The Gateway follows the same lifecycle patterns as the existing `server/grpc` and `server/http` packages.

**Primary recommendation:** Use grpc-gateway v2.27.x with rs/cors middleware, custom error handler for RFC 7807, and `di.ResolveAll[GatewayRegistrar]` for auto-discovery.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/grpc-ecosystem/grpc-gateway/v2` | v2.27.7 | HTTP-to-gRPC proxy | Official grpc-ecosystem project, 19.8k stars, actively maintained |
| `github.com/rs/cors` | latest | CORS middleware | 2.9k stars, 55.8k dependents, standard Go CORS solution |
| `google.golang.org/grpc` | v1.74.2 | gRPC client connection | Already in go.mod, required for loopback client |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `google.golang.org/grpc/credentials/insecure` | (part of grpc) | Insecure transport for loopback | Same-process localhost connections |
| `google.golang.org/protobuf/encoding/protojson` | (part of protobuf) | JSON marshaling options | Customizing JSON output format |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| rs/cors | grpc-gateway built-in | grpc-gateway has no built-in CORS; rs/cors is standard |
| Custom error handler | Default grpc-gateway errors | Default uses gRPC status format, not RFC 7807 |
| Loopback connection | In-process bufconn | Loopback is simpler, bufconn requires more setup |

**Installation:**
```bash
go get github.com/grpc-ecosystem/grpc-gateway/v2@v2.27.7
go get github.com/rs/cors
```

## Architecture Patterns

### Recommended Project Structure
```
server/
├── grpc/           # Existing gRPC server (Phase 38)
├── http/           # Existing HTTP server (Phase 38)
└── gateway/        # NEW: Gateway integration
    ├── gateway.go      # Gateway struct, lifecycle
    ├── config.go       # Configuration (port, CORS, etc.)
    ├── module.go       # DI module registration
    ├── errors.go       # RFC 7807 error handler
    ├── headers.go      # Header matcher configuration
    └── doc.go          # Package documentation
```

### Pattern 1: GatewayRegistrar Interface
**What:** Interface for services to register their HTTP handlers with the Gateway.
**When to use:** Every gRPC service that wants HTTP exposure.
**Example:**
```go
// Source: Context7 /grpc-ecosystem/grpc-gateway - runtime.ServeMux registration pattern
package gateway

import (
    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc"
)

// GatewayRegistrar is implemented by services that want HTTP exposure via Gateway.
// Services call their generated RegisterXXXHandler function in this method.
type GatewayRegistrar interface {
    RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}

// Example service implementation:
type GreeterService struct {
    pb.UnimplementedGreeterServer
}

func (s *GreeterService) RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
    return pb.RegisterGreeterHandler(ctx, mux, conn)
}
```

### Pattern 2: grpc.NewClient for Loopback Connection
**What:** Create a virtual gRPC client connection to localhost for proxying.
**When to use:** Gateway connecting to same-process gRPC server.
**Example:**
```go
// Source: Context7 /grpc/grpc-go - anti-patterns.md (grpc.NewClient is recommended over grpc.Dial)
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// Create loopback connection to gRPC server
target := fmt.Sprintf("localhost:%d", grpcPort)
conn, err := grpc.NewClient(target,
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
if err != nil {
    return fmt.Errorf("gateway: create grpc client: %w", err)
}
// conn is a virtual connection - no immediate dial, connects on first RPC
```

### Pattern 3: runtime.ServeMux with Custom Options
**What:** Create the HTTP mux with error handler, header matcher, and marshaler options.
**When to use:** Gateway initialization.
**Example:**
```go
// Source: Context7 /grpc-ecosystem/grpc-gateway - customizing_your_gateway.md
mux := runtime.NewServeMux(
    runtime.WithErrorHandler(problemDetailsErrorHandler),
    runtime.WithIncomingHeaderMatcher(customHeaderMatcher),
    runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
        MarshalOptions: protojson.MarshalOptions{
            EmitUnpopulated: true,
        },
        UnmarshalOptions: protojson.UnmarshalOptions{
            DiscardUnknown: true,
        },
    }),
)
```

### Pattern 4: CORS Middleware Wrapping
**What:** Wrap runtime.ServeMux with rs/cors handler.
**When to use:** Gateway HTTP handler setup.
**Example:**
```go
// Source: GitHub rs/cors README
import "github.com/rs/cors"

func (g *Gateway) buildHandler(mux *runtime.ServeMux) http.Handler {
    corsHandler := cors.New(cors.Options{
        AllowedOrigins:   g.config.CORS.AllowedOrigins,
        AllowedMethods:   g.config.CORS.AllowedMethods,
        AllowedHeaders:   g.config.CORS.AllowedHeaders,
        AllowCredentials: g.config.CORS.AllowCredentials,
        MaxAge:           g.config.CORS.MaxAge,
        Debug:            g.devMode,
    })
    return corsHandler.Handler(mux)
}
```

### Anti-Patterns to Avoid
- **Using grpc.Dial instead of grpc.NewClient:** `grpc.Dial` is deprecated. Use `grpc.NewClient` which creates a virtual connection without immediate dialing.
- **Manual service registration:** Don't manually wire services; use `di.ResolveAll[GatewayRegistrar]` for auto-discovery.
- **Hardcoding CORS origins:** Use configuration for origins; dev mode can use `AllowOriginFunc: func(origin string) bool { return true }`.
- **Exposing gRPC error details in production:** Custom error handler must strip internal details in prod mode.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP-to-gRPC translation | Custom HTTP handlers calling gRPC | grpc-gateway | Handles path params, body parsing, streaming, all edge cases |
| gRPC status to HTTP status | Custom mapping | `runtime.HTTPStatusFromCode` | Standard mapping per Google API design guide |
| CORS preflight handling | Custom OPTIONS handler | rs/cors | Handles preflight, credentials, expose headers correctly |
| JSON marshaling | encoding/json | grpc-gateway's runtime.JSONPb | Handles protobuf well-known types, enums, oneofs |
| Header forwarding | Manual header copying | `runtime.WithIncomingHeaderMatcher` | Handles permanent headers, metadata prefixing |

**Key insight:** grpc-gateway handles dozens of edge cases in HTTP-gRPC translation (path parameters, query strings, body mapping, streaming, trailers). Custom solutions will miss many of these.

## Common Pitfalls

### Pitfall 1: Starting Gateway Before gRPC Server
**What goes wrong:** Gateway tries to connect to gRPC server that isn't ready yet.
**Why it happens:** DI registration order doesn't guarantee startup order.
**How to avoid:** Use DI dependency injection - Gateway depends on `*grpc.Server`. The lifecycle engine starts dependencies first.
**Warning signs:** "connection refused" errors on startup.

### Pitfall 2: Using grpc.WithBlock on NewClient
**What goes wrong:** Blocks startup waiting for connection, timeouts, or hangs.
**Why it happens:** Old patterns from grpc.Dial era.
**How to avoid:** Use `grpc.NewClient` without blocking options. It creates a virtual connection that connects on first use.
**Warning signs:** Slow startup, timeout errors during initialization.

### Pitfall 3: CORS AllowedOrigins "*" with AllowCredentials true
**What goes wrong:** Security vulnerability; browsers block this combination anyway.
**Why it happens:** Developer tries to allow all origins with credentials.
**How to avoid:** rs/cors prevents this combination by default. If needed for dev, use `AllowOriginFunc` with explicit acknowledgment.
**Warning signs:** Browsers reject credentials, security scanner warnings.

### Pitfall 4: Exposing Stack Traces in Production Errors
**What goes wrong:** Information disclosure vulnerability.
**Why it happens:** Dev mode error handler used in production.
**How to avoid:** Check devMode flag in error handler; strip details for production.
**Warning signs:** Error responses contain Go stack traces or internal paths.

### Pitfall 5: Not Forwarding Important Headers
**What goes wrong:** Authentication, request ID, tracing context lost.
**Why it happens:** grpc-gateway has a restrictive default header matcher.
**How to avoid:** Use `WithIncomingHeaderMatcher` with explicit allowlist including: `authorization`, `x-request-id`, `x-correlation-id`, `x-forwarded-*`.
**Warning signs:** Auth fails, distributed tracing breaks, request IDs not propagated.

### Pitfall 6: Not Setting ReadHeaderTimeout
**What goes wrong:** Slow loris attack vulnerability.
**Why it happens:** Gateway uses http.Server without timeout configuration.
**How to avoid:** Reuse existing `server/http.Server` which has `ReadHeaderTimeout: 5s` by default.
**Warning signs:** Security scanner reports slow loris vulnerability.

## Code Examples

Verified patterns from official sources:

### Gateway Struct with Lifecycle
```go
// Gateway is an HTTP-to-gRPC gateway with auto-discovery and CORS support.
// Implements di.Starter and di.Stopper for lifecycle integration.
type Gateway struct {
    config    Config
    mux       *runtime.ServeMux
    conn      *grpc.ClientConn
    container *di.Container
    logger    *slog.Logger
    devMode   bool
}

func NewGateway(cfg Config, logger *slog.Logger, container *di.Container, devMode bool) *Gateway {
    return &Gateway{
        config:    cfg,
        container: container,
        logger:    logger,
        devMode:   devMode,
    }
}
```

### OnStart with Auto-Discovery
```go
// Source: Context7 /grpc-ecosystem/grpc-gateway - adding_annotations.md + gaz patterns
func (g *Gateway) OnStart(ctx context.Context) error {
    // Create loopback connection to gRPC server
    target := g.config.GRPCTarget
    if target == "" {
        target = fmt.Sprintf("localhost:%d", g.config.GRPCPort)
    }
    
    conn, err := grpc.NewClient(target,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return fmt.Errorf("gateway: create grpc client: %w", err)
    }
    g.conn = conn
    
    // Create ServeMux with options
    g.mux = runtime.NewServeMux(
        runtime.WithErrorHandler(g.errorHandler),
        runtime.WithIncomingHeaderMatcher(g.headerMatcher),
    )
    
    // Auto-discover and register services
    registrars, err := di.ResolveAll[GatewayRegistrar](g.container)
    if err != nil {
        conn.Close()
        return fmt.Errorf("gateway: discover registrars: %w", err)
    }
    
    for _, r := range registrars {
        if err := r.RegisterGateway(ctx, g.mux, conn); err != nil {
            conn.Close()
            return fmt.Errorf("gateway: register service: %w", err)
        }
    }
    
    g.logger.InfoContext(ctx, "Gateway initialized",
        slog.Int("services", len(registrars)),
        slog.String("grpc_target", target),
    )
    
    return nil
}
```

### RFC 7807 Problem Details Error Handler
```go
// Source: grpc-gateway runtime/errors.go + RFC 7807 spec
type ProblemDetails struct {
    Type     string `json:"type"`               // URI reference identifying problem type
    Title    string `json:"title"`              // Human-readable summary
    Status   int    `json:"status"`             // HTTP status code
    Detail   string `json:"detail,omitempty"`   // Human-readable explanation
    Instance string `json:"instance,omitempty"` // URI reference to specific occurrence (correlation ID)
    // Dev mode extensions
    Code       string `json:"code,omitempty"`        // gRPC code name
    StackTrace string `json:"stack_trace,omitempty"` // Only in dev mode
}

func (g *Gateway) errorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
    s := status.Convert(err)
    httpStatus := runtime.HTTPStatusFromCode(s.Code())
    
    problem := ProblemDetails{
        Type:     fmt.Sprintf("https://grpc.io/docs/guides/status-codes/#%s", strings.ToLower(s.Code().String())),
        Title:    s.Code().String(),
        Status:   httpStatus,
        Instance: r.Header.Get("X-Request-ID"),
    }
    
    if g.devMode {
        problem.Detail = s.Message()
        problem.Code = s.Code().String()
        // Add stack trace if available in error details
    } else {
        // Sanitize message for production
        problem.Detail = httpStatusText(httpStatus)
    }
    
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(httpStatus)
    json.NewEncoder(w).Encode(problem)
}
```

### Custom Header Matcher
```go
// Source: Context7 /grpc-ecosystem/grpc-gateway - customizing_your_gateway.md
func (g *Gateway) headerMatcher(key string) (string, bool) {
    // Allowlist of headers to forward as gRPC metadata
    switch strings.ToLower(key) {
    case "authorization":
        return key, true
    case "x-request-id", "x-correlation-id":
        return key, true
    case "x-forwarded-for", "x-forwarded-host":
        return key, true
    case "accept-language":
        return key, true
    default:
        // Fall back to default behavior for standard headers
        return runtime.DefaultHeaderMatcher(key)
    }
}
```

### CORS Configuration
```go
// Source: GitHub rs/cors README
type CORSConfig struct {
    AllowedOrigins   []string `json:"allowed_origins" yaml:"allowed_origins"`
    AllowedMethods   []string `json:"allowed_methods" yaml:"allowed_methods"`
    AllowedHeaders   []string `json:"allowed_headers" yaml:"allowed_headers"`
    ExposedHeaders   []string `json:"exposed_headers" yaml:"exposed_headers"`
    AllowCredentials bool     `json:"allow_credentials" yaml:"allow_credentials"`
    MaxAge           int      `json:"max_age" yaml:"max_age"`
}

func DefaultCORSConfig(devMode bool) CORSConfig {
    if devMode {
        return CORSConfig{
            AllowedOrigins:   []string{"*"},
            AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
            AllowedHeaders:   []string{"*"},
            AllowCredentials: false, // Cannot use * with credentials
            MaxAge:           86400,
        }
    }
    return CORSConfig{
        AllowedOrigins:   []string{}, // Must be explicitly configured
        AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
        AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
        ExposedHeaders:   []string{"X-Request-ID"},
        AllowCredentials: true,
        MaxAge:           86400,
    }
}
```

## gRPC Status to HTTP Status Mapping

Standard mapping from grpc-gateway (use as-is):

| gRPC Code | HTTP Status | Notes |
|-----------|-------------|-------|
| OK | 200 | Success |
| Canceled | 499 | Client closed request |
| Unknown | 500 | Internal Server Error |
| InvalidArgument | 400 | Bad Request |
| DeadlineExceeded | 504 | Gateway Timeout |
| NotFound | 404 | Not Found |
| AlreadyExists | 409 | Conflict |
| PermissionDenied | 403 | Forbidden |
| Unauthenticated | 401 | Unauthorized |
| ResourceExhausted | 429 | Too Many Requests |
| FailedPrecondition | 400 | Bad Request |
| Aborted | 409 | Conflict |
| OutOfRange | 400 | Bad Request |
| Unimplemented | 501 | Not Implemented |
| Internal | 500 | Internal Server Error |
| Unavailable | 503 | Service Unavailable |
| DataLoss | 500 | Internal Server Error |

**Recommendation:** Use `runtime.HTTPStatusFromCode` directly; don't implement custom mapping unless specific requirements exist.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| grpc.Dial | grpc.NewClient | gRPC-Go v1.63 | Non-blocking, no deprecated options |
| runtime.HTTPError | mux.errorHandler | grpc-gateway v2 | Customization via ServeMux options |
| runtime.DisallowUnknownFields | UnmarshalOptions.DiscardUnknown | grpc-gateway v2 | Via JSONPb marshaler options |

**Deprecated/outdated:**
- `grpc.Dial`, `grpc.DialContext`: Use `grpc.NewClient` instead
- `grpc.WithBlock`, `grpc.WithTimeout`: Not supported in NewClient
- `runtime.DisallowUnknownFields`: Use marshaler options instead

## Open Questions

Things that couldn't be fully resolved:

1. **In-process connection vs localhost loopback**
   - What we know: Both work; loopback is simpler, bufconn avoids network stack
   - What's unclear: Performance difference for high-throughput scenarios
   - Recommendation: Start with loopback (simpler), optimize if needed

2. **Streaming support complexity**
   - What we know: grpc-gateway supports server-side streaming via Server-Sent Events
   - What's unclear: Whether SSE is sufficient or if WebSocket upgrade needed
   - Recommendation: Start with built-in streaming support, evaluate later

## Sources

### Primary (HIGH confidence)
- Context7 `/grpc-ecosystem/grpc-gateway` - ServeMux creation, handler registration, error handling, header matching, marshaler options
- Context7 `/grpc/grpc-go` - grpc.NewClient usage, deprecated grpc.Dial, status codes
- GitHub `grpc-ecosystem/grpc-gateway/runtime/errors.go` - HTTPStatusFromCode, DefaultHTTPErrorHandler
- GitHub `rs/cors` README - CORS middleware options and usage

### Secondary (MEDIUM confidence)
- grpc-gateway releases page - v2.27.7 is latest as of Jan 2026
- gaz codebase - existing patterns in `server/grpc`, `server/http`, `di.ResolveAll`

### Tertiary (LOW confidence)
- RFC 7807 implementation - verified spec structure, implementation pattern inferred from common usage

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Context7 and official docs verified
- Architecture: HIGH - Based on existing gaz patterns and verified grpc-gateway patterns
- Pitfalls: HIGH - Documented in official anti-patterns guide
- Error handling: MEDIUM - RFC 7807 structure verified, exact grpc-gateway integration pattern is inference

**Research date:** 2026-02-03
**Valid until:** 30 days (grpc-gateway is stable, rs/cors is stable)
