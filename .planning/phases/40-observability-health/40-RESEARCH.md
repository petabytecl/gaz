# Phase 40: Observability & Health - Research

**Researched:** 2026-02-03
**Domain:** gRPC Health Checks, OpenTelemetry Tracing, PGX Health Checks
**Confidence:** HIGH

## Summary

This phase implements production observability for the gaz framework through three key components: gRPC health checks following the standard `grpc.health.v1` protocol, OpenTelemetry distributed tracing for request flows spanning Gateway to gRPC, and a PGX health check as an example dependency checker.

The gRPC health service is provided by `google.golang.org/grpc/health` with a pre-built `Server` type that manages serving status. The OpenTelemetry Go SDK provides comprehensive instrumentation through `otelgrpc` (gRPC) and `otelhttp` (HTTP/Gateway) packages from the contrib module, with OTLP exporters for trace collection. The pgx pool has a built-in `Ping()` method for health checking, with `exaring/otelpgx` providing OpenTelemetry tracing integration.

**Primary recommendation:** Use grpc-go's built-in health server for gRPC health checks, wrap existing servers with otelgrpc/otelhttp handlers for automatic tracing, and use pgxpool.Ping() for Postgres health checks with otelpgx for DB tracing.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `google.golang.org/grpc/health` | v1.78+ | gRPC health server | Official grpc-go implementation of health checking protocol |
| `google.golang.org/grpc/health/grpc_health_v1` | v1.78+ | Health proto types | Generated types for `grpc.health.v1.Health` service |
| `go.opentelemetry.io/otel` | v1.35+ | OTEL API | Core OpenTelemetry Go API |
| `go.opentelemetry.io/otel/sdk/trace` | v1.35+ | Tracer provider | SDK for trace collection and export |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.35+ | OTLP exporter | gRPC-based OTLP trace exporter |
| `go.opentelemetry.io/otel/propagation` | v1.35+ | Context propagation | W3C Trace Context headers |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` | v0.58+ | gRPC tracing | Automatic gRPC server/client instrumentation |
| `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` | v0.58+ | HTTP tracing | Gateway/HTTP server instrumentation |
| `github.com/exaring/otelpgx` | v0.10+ | PGX tracing | Trace database queries |
| `github.com/jackc/pgx/v5/pgxpool` | v5.8+ | Postgres pool | Connection pool with Ping() for health |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `otlptracegrpc` | `otlptrace/otlptracehttp` | HTTP export if gRPC not available to collector |
| `exaring/otelpgx` | Custom pgx tracer | otelpgx is well-maintained, no need to build custom |
| grpc-go health | Custom health impl | Standard protocol enables load balancer integration |

**Installation:**
```bash
go get google.golang.org/grpc/health@latest
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/sdk/trace@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@latest
go get go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc@latest
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp@latest
go get github.com/exaring/otelpgx@latest
```

## Architecture Patterns

### Recommended Project Structure
```
server/
├── health/                    # gRPC health service integration
│   ├── check.go               # HealthCheck interface
│   ├── grpc.go                # gRPC health server wrapper
│   ├── checker.go             # Aggregated checker with caching
│   └── module.go              # DI registration
├── otel/                      # OpenTelemetry module
│   ├── config.go              # OTEL configuration
│   ├── provider.go            # TracerProvider setup
│   ├── propagator.go          # W3C propagator setup
│   └── module.go              # DI registration
└── checks/                    # Health check implementations
    └── pgx/
        └── check.go           # PGX health check
```

### Pattern 1: gRPC Health Server Integration
**What:** Use grpc-go's built-in health.Server and register it with the gRPC server.
**When to use:** Always - this is the standard gRPC health protocol.
**Example:**
```go
// Source: https://pkg.go.dev/google.golang.org/grpc/health
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Create health server
healthServer := health.NewServer()

// Register with gRPC server
healthpb.RegisterHealthServer(grpcServer, healthServer)

// Set serving status (empty string = overall server health)
healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

// On shutdown
healthServer.Shutdown() // Sets all to NOT_SERVING
```

### Pattern 2: OpenTelemetry Provider Setup
**What:** Initialize TracerProvider at app startup with OTLP exporter.
**When to use:** When OTEL_EXPORTER_OTLP_ENDPOINT is configured.
**Example:**
```go
// Source: https://github.com/open-telemetry/opentelemetry-go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer(ctx context.Context, endpoint string) (*trace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(endpoint),
        otlptracegrpc.WithInsecure(), // For local dev
    )
    if err != nil {
        return nil, err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
        trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(0.1))),
    )

    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return tp, nil
}
```

### Pattern 3: gRPC Server Instrumentation
**What:** Add OpenTelemetry stats handler to gRPC server for automatic tracing.
**When to use:** When OTEL is enabled.
**Example:**
```go
// Source: https://github.com/open-telemetry/opentelemetry-go-contrib
import (
    "google.golang.org/grpc"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// Create gRPC server with OTEL instrumentation
server := grpc.NewServer(
    grpc.StatsHandler(otelgrpc.NewServerHandler(
        otelgrpc.WithFilter(func(info *otelgrpc.InterceptorInfo) bool {
            // Skip tracing health checks
            return info.Method != "/grpc.health.v1.Health/Check"
        }),
    )),
    // ... other options
)
```

### Pattern 4: HTTP/Gateway Instrumentation
**What:** Wrap HTTP handler with otelhttp for automatic tracing.
**When to use:** For Gateway HTTP endpoints.
**Example:**
```go
// Source: https://github.com/open-telemetry/opentelemetry-go-contrib
import (
    "net/http"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Wrap handler with OTEL instrumentation
handler := otelhttp.NewHandler(mux, "gateway",
    otelhttp.WithFilter(func(r *http.Request) bool {
        // Skip health check endpoints
        return r.URL.Path != "/health"
    }),
)
```

### Pattern 5: Health Check Interface with Discovery
**What:** Define a simple health check interface discoverable via di.ResolveAll.
**When to use:** To allow services to register their own health checks.
**Example:**
```go
// HealthCheck is implemented by services that provide health checks.
type HealthCheck interface {
    // Check performs the health check.
    // Returns nil if healthy, error describing the issue if unhealthy.
    Check(ctx context.Context) error
}

// HealthCheckMeta provides metadata about a health check.
type HealthCheckMeta interface {
    HealthCheck
    // Name returns the check name (e.g., "postgres", "redis").
    Name() string
    // Critical returns true if this check failure should cause NOT_SERVING.
    Critical() bool
}
```

### Pattern 6: PGX Health Check
**What:** Use pgxpool.Ping() for Postgres connectivity check.
**When to use:** When Postgres database is a dependency.
**Example:**
```go
// Source: https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool
import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

type PostgresCheck struct {
    pool *pgxpool.Pool
}

func (c *PostgresCheck) Check(ctx context.Context) error {
    return c.pool.Ping(ctx)
}

func (c *PostgresCheck) Name() string { return "postgres" }
func (c *PostgresCheck) Critical() bool { return true }
```

### Pattern 7: PGX OpenTelemetry Tracing
**What:** Use otelpgx tracer for database query tracing.
**When to use:** When OTEL is enabled and using pgx.
**Example:**
```go
// Source: https://github.com/exaring/otelpgx
import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/exaring/otelpgx"
)

cfg, err := pgxpool.ParseConfig(connString)
if err != nil {
    return nil, err
}

// Add OTEL tracer
cfg.ConnConfig.Tracer = otelpgx.NewTracer()

pool, err := pgxpool.NewWithConfig(ctx, cfg)
```

### Anti-Patterns to Avoid
- **Custom health protocol:** Don't invent a custom gRPC health protocol. Use `grpc.health.v1` - load balancers understand it.
- **Health checks in interceptors:** Don't perform health checks on every request. Use cached background checks.
- **Blocking OTEL init:** Don't block app startup waiting for OTEL collector. Fail gracefully if unreachable.
- **Tracing everything:** Don't trace health check endpoints - they're high-frequency and add noise.
- **Synchronous shutdown:** Don't shutdown TracerProvider immediately. Use deferred shutdown with timeout.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| gRPC health protocol | Custom health service | `google.golang.org/grpc/health` | Standard protocol, load balancer compatible |
| Trace context propagation | Manual header parsing | `otel/propagation.TraceContext{}` | W3C standard, handles edge cases |
| gRPC tracing | Custom interceptors | `otelgrpc.NewServerHandler()` | Handles all RPC types, proper span naming |
| HTTP tracing | Custom middleware | `otelhttp.NewHandler()` | Request attributes, status codes, proper spans |
| PGX tracing | Custom query logging | `otelpgx.NewTracer()` | Query sanitization, proper span hierarchy |
| OTLP export | Manual HTTP/gRPC calls | `otlptracegrpc.New()` | Batching, retry, backoff built-in |

**Key insight:** OpenTelemetry and gRPC health are standards with official implementations. Custom solutions break interoperability.

## Common Pitfalls

### Pitfall 1: Health Checks Blocking Startup
**What goes wrong:** App waits for all dependencies before starting health server.
**Why it happens:** Attempting to validate all checks before reporting any status.
**How to avoid:** Start health server immediately with UNKNOWN status, then update as checks complete.
**Warning signs:** Slow startup, health endpoint unavailable during initialization.

### Pitfall 2: Health Check Thundering Herd
**What goes wrong:** Multiple health check requests all trigger simultaneous dependency checks.
**Why it happens:** Each request runs checks synchronously.
**How to avoid:** Cache check results, run checks in background on a timer.
**Warning signs:** Database connection spikes during health probes.

### Pitfall 3: OTEL Collector Blocking Startup
**What goes wrong:** App fails to start if OTEL collector is unreachable.
**Why it happens:** OTLP exporter creation fails hard.
**How to avoid:** Create exporter with retry/backoff, log warning but continue without tracing.
**Warning signs:** App won't start in environments without collector.

### Pitfall 4: Trace Sampling Confusion
**What goes wrong:** Some traces appear, others don't, seemingly random.
**Why it happens:** Misunderstanding of sampling configuration.
**How to avoid:** Use ParentBased sampler - respect incoming trace decisions, sample probabilistically for root spans.
**Warning signs:** Incomplete traces, "missing" spans.

### Pitfall 5: Degraded Status Mapping
**What goes wrong:** gRPC health proto only has SERVING/NOT_SERVING, no "degraded" state.
**Why it happens:** Standard health proto lacks intermediate states.
**How to avoid:** Map SERVING_DEGRADED to SERVING (it can still serve), but include degraded checks in health response metadata or logs.
**Warning signs:** Can't distinguish between fully healthy and partially healthy.

### Pitfall 6: Health Check Timeout Mismatch
**What goes wrong:** Health checks timeout before completing, returning false negatives.
**Why it happens:** Kubernetes probe timeout < check execution time.
**How to avoid:** Keep individual check timeouts short (1-3s), use background caching.
**Warning signs:** Intermittent health failures, container restarts.

## Code Examples

Verified patterns from official sources:

### gRPC Health Server Setup
```go
// Source: https://pkg.go.dev/google.golang.org/grpc/health
package health

import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Server wraps grpc-go's health.Server with status management.
type Server struct {
    health *health.Server
}

func NewServer() *Server {
    return &Server{
        health: health.NewServer(),
    }
}

// Register adds the health service to the gRPC server.
func (s *Server) Register(srv *grpc.Server) {
    healthpb.RegisterHealthServer(srv, s.health)
}

// SetStatus updates the serving status for a service.
// Use empty string for overall server health.
func (s *Server) SetStatus(service string, status healthpb.HealthCheckResponse_ServingStatus) {
    s.health.SetServingStatus(service, status)
}

// Shutdown sets all services to NOT_SERVING.
func (s *Server) Shutdown() {
    s.health.Shutdown()
}

// Resume sets all services to SERVING.
func (s *Server) Resume() {
    s.health.Resume()
}
```

### OTEL TracerProvider with OTLP
```go
// Source: https://github.com/open-telemetry/opentelemetry-go
package otel

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/propagation"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Config struct {
    Endpoint    string  // OTLP endpoint (e.g., "localhost:4317")
    ServiceName string  // Service name for traces
    SampleRatio float64 // Sampling ratio for root spans (0.0-1.0)
    Insecure    bool    // Use insecure connection
}

func InitTracer(ctx context.Context, cfg Config, logger *slog.Logger) (*sdktrace.TracerProvider, error) {
    if cfg.Endpoint == "" {
        return nil, nil // OTEL disabled
    }

    opts := []otlptracegrpc.Option{
        otlptracegrpc.WithEndpoint(cfg.Endpoint),
    }
    if cfg.Insecure {
        opts = append(opts, otlptracegrpc.WithInsecure())
    }

    exporter, err := otlptracegrpc.New(ctx, opts...)
    if err != nil {
        logger.Warn("failed to create OTLP exporter, tracing disabled",
            slog.Any("error", err))
        return nil, nil // Graceful degradation
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String(cfg.ServiceName),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("create resource: %w", err)
    }

    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
        sdktrace.WithSampler(sdktrace.ParentBased(
            sdktrace.TraceIDRatioBased(cfg.SampleRatio),
        )),
    )

    // Set global providers
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{},
        propagation.Baggage{},
    ))

    return tp, nil
}

func ShutdownTracer(ctx context.Context, tp *sdktrace.TracerProvider) error {
    if tp == nil {
        return nil
    }
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    return tp.Shutdown(ctx)
}
```

### Cached Health Aggregator
```go
package health

import (
    "context"
    "sync"
    "time"
)

// Status represents aggregated health status.
type Status int

const (
    StatusServing Status = iota
    StatusDegraded
    StatusNotServing
)

// Aggregator runs health checks in the background and caches results.
type Aggregator struct {
    checks   []HealthCheckMeta
    results  map[string]error
    status   Status
    mu       sync.RWMutex
    interval time.Duration
    stop     chan struct{}
}

func NewAggregator(checks []HealthCheckMeta, interval time.Duration) *Aggregator {
    return &Aggregator{
        checks:   checks,
        results:  make(map[string]error),
        interval: interval,
        stop:     make(chan struct{}),
    }
}

func (a *Aggregator) Start(ctx context.Context) {
    // Run initial check
    a.runChecks(ctx)

    go func() {
        ticker := time.NewTicker(a.interval)
        defer ticker.Stop()
        for {
            select {
            case <-ticker.C:
                a.runChecks(ctx)
            case <-a.stop:
                return
            case <-ctx.Done():
                return
            }
        }
    }()
}

func (a *Aggregator) Stop() {
    close(a.stop)
}

func (a *Aggregator) runChecks(ctx context.Context) {
    results := make(map[string]error)
    var criticalFailed, optionalFailed bool

    for _, check := range a.checks {
        checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
        err := check.Check(checkCtx)
        cancel()

        results[check.Name()] = err
        if err != nil {
            if check.Critical() {
                criticalFailed = true
            } else {
                optionalFailed = true
            }
        }
    }

    a.mu.Lock()
    a.results = results
    if criticalFailed {
        a.status = StatusNotServing
    } else if optionalFailed {
        a.status = StatusDegraded
    } else {
        a.status = StatusServing
    }
    a.mu.Unlock()
}

func (a *Aggregator) Status() Status {
    a.mu.RLock()
    defer a.mu.RUnlock()
    return a.status
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gRPC interceptors for tracing | `grpc.StatsHandler` with otelgrpc | 2023 | Cleaner API, better streaming support |
| `grpc.Dial()` | `grpc.NewClient()` | grpc-go v1.63 | Dial is deprecated, use NewClient |
| Custom health endpoints | `grpc.health.v1` standard | Always | Load balancer compatibility |
| OTEL trace.SimpleSpanProcessor | trace.WithBatcher() | Always | Better performance in production |
| JAEGER\_\* env vars | OTEL\_\* env vars | 2022 | OTEL standardization |

**Deprecated/outdated:**
- `grpc.Dial()`: Deprecated in favor of `grpc.NewClient()` in v1.63
- `otelgrpc.UnaryServerInterceptor()`: Use `otelgrpc.NewServerHandler()` stats handler instead
- JAEGER exporter: Use OTLP exporter with Jaeger's OTLP receiver

## Open Questions

Things that couldn't be fully resolved:

1. **Degraded status in gRPC health proto**
   - What we know: gRPC health proto only has UNKNOWN, SERVING, NOT_SERVING, SERVICE_UNKNOWN
   - What's unclear: How to communicate "degraded" state to clients
   - Recommendation: Map SERVING_DEGRADED to SERVING (still functional), log degraded state, consider custom metadata in response

2. **Background check interval default**
   - What we know: Must balance freshness vs. database load
   - What's unclear: No universal "right" value
   - Recommendation: Default to 10 seconds (Kubernetes default probe interval), make configurable

3. **TracerProvider lifecycle with DI**
   - What we know: TracerProvider needs shutdown on app exit
   - What's unclear: Integration with gaz lifecycle ordering
   - Recommendation: Register TracerProvider shutdown as final stopper, ensure flushes complete

## Sources

### Primary (HIGH confidence)
- `/grpc/grpc-go` Context7 - gRPC health service documentation
- `/open-telemetry/opentelemetry-go` Context7 - OTEL SDK initialization, exporters
- `/open-telemetry/opentelemetry-go-contrib` Context7 - otelgrpc, otelhttp instrumentation
- `/jackc/pgx` Context7 - pgxpool.Ping(), QueryTracer interface
- https://pkg.go.dev/google.golang.org/grpc/health - Official health server API
- https://pkg.go.dev/google.golang.org/grpc/health/grpc_health_v1 - Health proto types
- https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool - Pool with Ping() method

### Secondary (MEDIUM confidence)
- https://github.com/exaring/otelpgx - PGX OpenTelemetry tracer (v0.10.0, Jan 2026)

### Tertiary (LOW confidence)
- None - all findings verified with authoritative sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All packages from official sources with verified documentation
- Architecture: HIGH - Patterns from official examples and contrib packages
- Pitfalls: HIGH - Well-documented issues in OTEL and gRPC communities
- Code examples: HIGH - Sourced from Context7 and official pkg.go.dev

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - stable libraries)
