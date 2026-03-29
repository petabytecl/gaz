# Phase 38: Transport Foundations - Research

**Researched:** 2026-02-03
**Domain:** gRPC and HTTP server infrastructure for Go
**Confidence:** HIGH

## Summary

This phase establishes production-ready gRPC and HTTP transport servers integrated with gaz's DI container and lifecycle management. The standard approach uses `google.golang.org/grpc` for gRPC with `grpc.ChainUnaryInterceptor` and `grpc.ChainStreamInterceptor` for middleware chaining, and Go's standard `net/http` for HTTP servers.

For interceptors, the grpc-ecosystem `go-grpc-middleware/v2` package provides battle-tested logging and recovery interceptors that integrate with slog. gRPC reflection is enabled via `google.golang.org/grpc/reflection` to support grpcurl introspection.

**Primary recommendation:** Use grpc-go's native interceptor chaining (`ChainUnaryInterceptor`/`ChainStreamInterceptor`) with grpc-ecosystem/go-grpc-middleware v2 for logging and recovery interceptors, adapting to gaz's existing slog-based logger.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `google.golang.org/grpc` | v1.73+ | gRPC server implementation | Official Go gRPC implementation, ~263k importers |
| `google.golang.org/grpc/reflection` | (bundled) | gRPC reflection for grpcurl | Official reflection package for runtime service discovery |
| `net/http` | stdlib | HTTP server | Go standard library, proven production-ready |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/grpc-ecosystem/go-grpc-middleware/v2` | v2.3.3 | Interceptor collection | For logging, recovery, and auth interceptors |
| `github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery` | (subpkg) | Panic recovery | Converts panics to gRPC Internal errors |
| `github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging` | (subpkg) | Request logging | Structured logging with slog adapter |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-grpc-middleware | Hand-written interceptors | More control but reinvents wheel; ecosystem has solved this |
| net/http | chi/gin/echo | Unnecessary for this phase - raw HTTP mux is sufficient for Gateway foundation |

**Installation:**
```bash
go get google.golang.org/grpc@v1.73.0
go get github.com/grpc-ecosystem/go-grpc-middleware/v2@v2.3.3
```

## Architecture Patterns

### Recommended Project Structure
```
server/
├── config.go              # ServersConfig struct (grpc + http)
├── doc.go                 # Package documentation
├── grpc/
│   ├── config.go          # GRPCConfig struct
│   ├── server.go          # GRPCServer with Starter/Stopper
│   ├── interceptors.go    # Logging, recovery interceptors
│   ├── registrar.go       # ServiceRegistrar interface
│   └── module.go          # DI registration
└── http/
    ├── config.go          # HTTPConfig struct
    ├── server.go          # HTTPServer with Starter/Stopper
    └── module.go          # DI registration
```

### Pattern 1: gRPC Server with ChainedInterceptors
**What:** Native gRPC interceptor chaining without external middleware chaining library
**When to use:** Always - grpc-go includes ChainUnaryInterceptor/ChainStreamInterceptor since v1.28
**Example:**
```go
// Source: https://pkg.go.dev/google.golang.org/grpc
import (
    "google.golang.org/grpc"
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
)

func NewGRPCServer(cfg GRPCConfig, logger *slog.Logger) *grpc.Server {
    // Create slog adapter for logging interceptor
    loggerAdapter := InterceptorLogger(logger)
    
    // Recovery handler with stack trace logging
    recoveryHandler := recovery.WithRecoveryHandlerContext(
        func(ctx context.Context, p any) error {
            logger.ErrorContext(ctx, "panic recovered", "panic", p)
            return status.Errorf(codes.Internal, "internal error")
        },
    )
    
    return grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            logging.UnaryServerInterceptor(loggerAdapter),
            recovery.UnaryServerInterceptor(recoveryHandler),
        ),
        grpc.ChainStreamInterceptor(
            logging.StreamServerInterceptor(loggerAdapter),
            recovery.StreamServerInterceptor(recoveryHandler),
        ),
    )
}
```

### Pattern 2: gRPC Reflection Registration
**What:** Enable runtime service discovery via grpcurl
**When to use:** Always enabled by default for dev/debugging; can be disabled in production
**Example:**
```go
// Source: https://github.com/grpc/grpc-go/blob/master/Documentation/server-reflection-tutorial.md
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
)

func enableReflection(server *grpc.Server) {
    reflection.Register(server)
}

// Verification with grpcurl:
// $ grpcurl -plaintext localhost:50051 list
// $ grpcurl -plaintext localhost:50051 describe helloworld.Greeter
```

### Pattern 3: Service Auto-Discovery via DI
**What:** Use gaz.ResolveAll to discover all gRPC services implementing a registrar interface
**When to use:** For automatic gRPC service registration without explicit wiring
**Example:**
```go
// Based on gaz discovery pattern in examples/discovery/main.go

// ServiceRegistrar is implemented by gRPC services that want to be registered
type ServiceRegistrar interface {
    RegisterService(server grpc.ServiceRegistrar)
}

// In server startup, discover all services
func (s *GRPCServer) OnStart(ctx context.Context) error {
    // Auto-discover all services implementing ServiceRegistrar
    registrars, err := gaz.ResolveAll[ServiceRegistrar](s.container)
    if err != nil {
        return fmt.Errorf("discover services: %w", err)
    }
    
    for _, r := range registrars {
        r.RegisterService(s.server)
    }
    
    // Enable reflection after all services registered
    reflection.Register(s.server)
    
    // Start serving
    go func() {
        if err := s.server.Serve(s.listener); err != nil {
            s.logger.Error("gRPC serve error", "error", err)
        }
    }()
    
    return nil
}
```

### Pattern 4: Graceful Shutdown
**What:** Use grpc.Server.GracefulStop() with context timeout
**When to use:** Always implement for production servers
**Example:**
```go
// Source: https://github.com/grpc/grpc-go/blob/master/examples/features/gracefulstop/README.md

func (s *GRPCServer) OnStop(ctx context.Context) error {
    // Create a channel to signal when GracefulStop completes
    done := make(chan struct{})
    
    go func() {
        s.server.GracefulStop()
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-ctx.Done():
        // Force stop if graceful didn't complete in time
        s.server.Stop()
        return ctx.Err()
    }
}
```

### Pattern 5: HTTP Server with Configurable Timeouts
**What:** Standard http.Server with explicit timeout configuration
**When to use:** Always - prevents slow loris and similar attacks
**Example:**
```go
// Based on gaz health/server.go pattern

func NewHTTPServer(cfg HTTPConfig) *http.Server {
    return &http.Server{
        Addr:              fmt.Sprintf(":%d", cfg.Port),
        Handler:           mux,
        ReadTimeout:       cfg.ReadTimeout,       // e.g., 10s
        WriteTimeout:      cfg.WriteTimeout,      // e.g., 30s
        IdleTimeout:       cfg.IdleTimeout,       // e.g., 120s
        ReadHeaderTimeout: cfg.ReadHeaderTimeout, // e.g., 5s
    }
}
```

### Anti-Patterns to Avoid
- **Single interceptor registration:** Don't use `grpc.UnaryInterceptor()` when you need multiple interceptors - use `ChainUnaryInterceptor` instead
- **Blocking OnStart:** Don't block in OnStart - spawn goroutine for server.Serve()
- **Missing port binding error handling:** Always check listener creation error before returning from OnStart
- **Interceptor ordering:** Recovery interceptor should be LAST so panics don't skip logging

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Panic recovery | Custom recover() wrapper | go-grpc-middleware/v2/interceptors/recovery | Handles both unary and streaming, proper gRPC error conversion |
| Request logging | Custom logging interceptor | go-grpc-middleware/v2/interceptors/logging | Duration tracking, field extraction, slog integration |
| Interceptor chaining | Custom chain function | grpc.ChainUnaryInterceptor | Native to grpc-go since v1.28, handles edge cases |
| gRPC reflection | Manual service listing | google.golang.org/grpc/reflection | Standard implementation, works with grpcurl |
| Request ID propagation | Custom metadata handling | gRPC metadata package | Standard pattern for gRPC context propagation |

**Key insight:** grpc-ecosystem/go-grpc-middleware v2 was specifically designed for Go gRPC middleware patterns. It's used by 45.6k projects and actively maintained. The v2 rewrite removed the chain utilities (now in grpc-go core) and modernized the API.

## Common Pitfalls

### Pitfall 1: Interceptor Ordering
**What goes wrong:** Recovery interceptor runs before logging, causing panics to not be logged
**Why it happens:** Interceptors execute in registration order for request, reverse for response
**How to avoid:** Always order: logging → auth → recovery (recovery last in chain, first to handle panic)
**Warning signs:** Panics appear without corresponding log entries

### Pitfall 2: Blocking Server Startup
**What goes wrong:** OnStart blocks on server.Serve(), preventing lifecycle from progressing
**Why it happens:** server.Serve() blocks until the server stops
**How to avoid:** Spawn goroutine for Serve(), return immediately from OnStart after listener binds
**Warning signs:** Application hangs after gRPC server starts, HTTP server never starts

### Pitfall 3: Port Binding Without Error Check
**What goes wrong:** Server returns success but port is already in use
**Why it happens:** net.Listen error not checked before returning from OnStart
**How to avoid:** Create listener in OnStart, verify it succeeds, then spawn Serve goroutine
**Warning signs:** "address already in use" in logs but application continues

### Pitfall 4: Missing Graceful Shutdown Timeout
**What goes wrong:** GracefulStop hangs forever with stuck connections
**Why it happens:** GracefulStop waits for all RPCs to complete, which may never happen
**How to avoid:** Use context timeout, fallback to Stop() if graceful exceeds timeout
**Warning signs:** Application hangs on shutdown, requiring SIGKILL

### Pitfall 5: Forgetting HTTP ReadHeaderTimeout
**What goes wrong:** Slow loris attacks can exhaust server resources
**Why it happens:** Default http.Server has no ReadHeaderTimeout
**How to avoid:** Always set ReadHeaderTimeout (typically 5-10 seconds)
**Warning signs:** gosec/staticcheck warnings about missing timeout

### Pitfall 6: Wrong Shutdown Order
**What goes wrong:** HTTP gateway fails during shutdown because gRPC backend is already down
**Why it happens:** Shutting down gRPC before HTTP when HTTP depends on gRPC
**How to avoid:** Shutdown in reverse startup order (HTTP first, then gRPC)
**Warning signs:** Gateway errors during graceful shutdown period

## Code Examples

Verified patterns from official sources and project conventions:

### Logging Interceptor with slog Adapter
```go
// Source: go-grpc-middleware logging examples pattern
// Adapted for gaz's slog-based logger

import (
    "log/slog"
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

// InterceptorLogger adapts slog.Logger to logging.Logger interface
func InterceptorLogger(l *slog.Logger) logging.Logger {
    return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
        l.Log(ctx, slog.Level(lvl), msg, fields...)
    })
}

// Usage
logger := slog.Default()
logOpts := []logging.Option{
    logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
}
interceptor := logging.UnaryServerInterceptor(InterceptorLogger(logger), logOpts...)
```

### Recovery Interceptor with Stack Trace
```go
// Source: go-grpc-middleware recovery documentation

import (
    "runtime/debug"
    "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func recoveryHandler(logger *slog.Logger, devMode bool) recovery.Option {
    return recovery.WithRecoveryHandlerContext(
        func(ctx context.Context, p any) error {
            // Log full stack trace
            logger.ErrorContext(ctx, "panic recovered",
                "panic", p,
                "stack", string(debug.Stack()),
            )
            
            // Return error details only in dev mode
            if devMode {
                return status.Errorf(codes.Internal, "panic: %v", p)
            }
            return status.Error(codes.Internal, "internal server error")
        },
    )
}
```

### Config Structures (following gaz conventions)
```go
// Nested under servers key per CONTEXT.md decisions

type ServersConfig struct {
    GRPC GRPCConfig `mapstructure:"grpc"`
    HTTP HTTPConfig `mapstructure:"http"`
}

type GRPCConfig struct {
    Port              int           `mapstructure:"port" validate:"min=1,max=65535"`
    ReflectionEnabled bool          `mapstructure:"reflection_enabled"`
    ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}

type HTTPConfig struct {
    Port              int           `mapstructure:"port" validate:"min=1,max=65535"`
    ReadTimeout       time.Duration `mapstructure:"read_timeout"`
    WriteTimeout      time.Duration `mapstructure:"write_timeout"`
    IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
    ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
    ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}

func DefaultServersConfig() ServersConfig {
    return ServersConfig{
        GRPC: GRPCConfig{
            Port:              50051,
            ReflectionEnabled: true,
            ShutdownTimeout:   30 * time.Second,
        },
        HTTP: HTTPConfig{
            Port:              8080,
            ReadTimeout:       10 * time.Second,
            WriteTimeout:      30 * time.Second,
            IdleTimeout:       120 * time.Second,
            ReadHeaderTimeout: 5 * time.Second,
            ShutdownTimeout:   30 * time.Second,
        },
    }
}
```

### gRPC Server Lifecycle Integration
```go
// Follows gaz health/server.go pattern for Starter/Stopper

type GRPCServer struct {
    config    GRPCConfig
    server    *grpc.Server
    listener  net.Listener
    container *gaz.Container
    logger    *slog.Logger
}

func (s *GRPCServer) OnStart(ctx context.Context) error {
    // Bind port first (fail fast if already in use)
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
    if err != nil {
        return fmt.Errorf("bind gRPC port %d: %w", s.config.Port, err)
    }
    s.listener = lis
    
    // Auto-discover and register services
    registrars, err := gaz.ResolveAll[ServiceRegistrar](s.container)
    if err != nil {
        return fmt.Errorf("discover gRPC services: %w", err)
    }
    
    for _, r := range registrars {
        r.RegisterService(s.server)
    }
    
    // Enable reflection
    if s.config.ReflectionEnabled {
        reflection.Register(s.server)
    }
    
    s.logger.Info("gRPC server starting", "port", s.config.Port)
    
    // Spawn serve goroutine (non-blocking)
    go func() {
        if err := s.server.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
            s.logger.Error("gRPC server error", "error", err)
        }
    }()
    
    return nil
}

func (s *GRPCServer) OnStop(ctx context.Context) error {
    s.logger.Info("gRPC server stopping")
    
    done := make(chan struct{})
    go func() {
        s.server.GracefulStop()
        close(done)
    }()
    
    select {
    case <-done:
        s.logger.Info("gRPC server stopped gracefully")
        return nil
    case <-ctx.Done():
        s.server.Stop()
        s.logger.Warn("gRPC server force stopped")
        return ctx.Err()
    }
}
```

### Module Registration Pattern
```go
// Follows gaz health/module.go pattern

func Module(c *gaz.Container) error {
    // Register config (assumes config.Manager is available)
    if err := gaz.For[GRPCConfig](c).Provider(func(c *gaz.Container) (GRPCConfig, error) {
        mgr, err := gaz.Resolve[*config.Manager](c)
        if err != nil {
            return GRPCConfig{}, err
        }
        cfg := DefaultGRPCConfig()
        if err := mgr.UnmarshalKey("servers.grpc", &cfg); err != nil {
            return GRPCConfig{}, fmt.Errorf("unmarshal grpc config: %w", err)
        }
        return cfg, nil
    }); err != nil {
        return fmt.Errorf("register grpc config: %w", err)
    }
    
    // Register server (eager to participate in lifecycle)
    if err := gaz.For[*GRPCServer](c).Eager().Provider(NewGRPCServer); err != nil {
        return fmt.Errorf("register grpc server: %w", err)
    }
    
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| go-grpc-middleware v1 chain | grpc.ChainUnaryInterceptor | grpc-go v1.28 (2020) | No need for external chain utility |
| grpc-prometheus (separate) | go-grpc-middleware/providers/prometheus | v2.0 (2023) | Consolidated, better API |
| OpenTracing | OpenTelemetry | 2022+ | otelgrpc is the standard now |
| grpc_ctxtags | logging.InjectFields | v2.0 (2023) | Simplified API |
| DialContext | NewClient | grpc-go v1.63 (2024) | Dial* deprecated |

**Deprecated/outdated:**
- `grpc.Dial`, `grpc.DialContext`: Deprecated since v1.63, use `grpc.NewClient` for clients
- go-grpc-middleware v1 chain utilities: Now in grpc-go core
- grpc_ctxtags: Removed in v2, use logging.InjectFields

## Open Questions

Things that couldn't be fully resolved:

1. **Request ID propagation pattern for gRPC**
   - What we know: gRPC uses metadata for headers, X-Request-ID is HTTP convention
   - What's unclear: Best practice for extracting X-Request-ID from gRPC-Gateway calls
   - Recommendation: Use grpc.metadata, check for "x-request-id" key, fallback to UUID generation

2. **TLS configuration details**
   - What we know: CONTEXT.md says TLS disabled by default, enabled via config
   - What's unclear: Exact config structure for TLS (cert paths vs inline)
   - Recommendation: Simple path-based config (cert_file, key_file), defer detailed TLS to future phase

## Sources

### Primary (HIGH confidence)
- `/grpc/grpc-go` via Context7 - interceptor setup, reflection, graceful shutdown
- `pkg.go.dev/google.golang.org/grpc` - ChainUnaryInterceptor, ChainStreamInterceptor documentation
- `pkg.go.dev/github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery` - recovery interceptor API
- `pkg.go.dev/github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging` - logging interceptor API

### Secondary (MEDIUM confidence)
- `github.com/grpc-ecosystem/go-grpc-middleware` README - interceptor ordering, v2 migration notes
- grpc-go examples/features/gracefulstop - graceful shutdown pattern

### Project-Specific (HIGH confidence)
- `/home/coto/dev/petabyte/gaz/health/server.go` - existing gaz server lifecycle pattern
- `/home/coto/dev/petabyte/gaz/health/module.go` - existing gaz module registration pattern
- `/home/coto/dev/petabyte/gaz/examples/discovery/main.go` - ResolveAll pattern for service discovery
- `/home/coto/dev/petabyte/gaz/logger/` - existing gaz slog integration

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - grpc-go and grpc-ecosystem are the definitive choices, verified via Context7 and pkg.go.dev
- Architecture: HIGH - follows existing gaz patterns from health module
- Pitfalls: HIGH - derived from official documentation and common gRPC Go issues
- Code examples: HIGH - adapted from official sources to gaz conventions

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable domain, 30 days)
