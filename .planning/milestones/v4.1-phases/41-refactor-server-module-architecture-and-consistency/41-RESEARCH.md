# Phase 41: Refactor server module architecture and consistency - Research

**Researched:** Tue Feb 03 2026
**Domain:** Server Architecture & Lifecycle
**Confidence:** HIGH

## Summary

This research focuses on refactoring the `gaz` server modules (`server/grpc`, `server/http`, `server/gateway`, `health`) to ensure architectural consistency, robust lifecycle management, and alignment with recent project decisions (v4.1).

The current implementation splits responsibilities into distinct packages, but reveals potential inconsistencies in logging initialization, naming conventions, and lifecycle coordination—specifically between the `Gateway` (which builds handlers dynamically) and the `http.Server` (which serves them).

**Primary recommendation:** Standardize constructor patterns (explicit logger fallback), unify discovery mechanisms via `di.ResolveAll`, and resolve the "Gateway-to-Server" lifecycle race condition by ensuring handlers are ready before traffic acceptance.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` | Go 1.25+ | HTTP Server | Standard library, robust. |
| `google.golang.org/grpc` | Latest | gRPC Server | Industry standard for Go gRPC. |
| `grpc-ecosystem/grpc-gateway` | v2 | HTTP-to-gRPC Proxy | Standard way to expose gRPC as JSON/HTTP. |
| `log/slog` | Go 1.25+ | Structured Logging | Built-in, zero-dependency. |
| `go.opentelemetry.io/otel` | Latest | Observability | Industry standard for tracing/metrics. |

### Architecture Patterns

### Recommended Project Structure
```
gaz/
├── server/
│   ├── grpc/       # gRPC Server implementation
│   ├── http/       # Generic HTTP Server implementation
│   └── gateway/    # HTTP-to-gRPC Gateway logic
├── health/         # Health check management & server
└── di/             # Dependency Injection (lifecycle management)
```

### Pattern 1: Auto-Discovery via DI
Services register themselves by implementing a specific interface (`ServiceRegistrar` for gRPC, `Registrar` for Gateway). The server module discovers them at startup.

**Example (gRPC):**
```go
// In server/grpc/server.go
registrars, err := di.ResolveAll[ServiceRegistrar](container)
for _, r := range registrars {
    r.RegisterService(s.server)
}
```

### Pattern 2: Lifecycle Hooks (`OnStart`/`OnStop`)
All servers implement `di.Starter` and `di.Stopper` to integrate with the `gaz` application lifecycle.

### Pattern 3: Port Separation
- **gRPC Server:** Typically port 9090.
- **Gateway/HTTP:** Typically port 8080.
- **Health/Management:** Dedicated port (optional) or merged.

## Refactoring Opportunities & Inconsistencies

### 1. Logger Initialization consistency
- **Current State:**
  - `server/http`: `NewServer` explicitly checks `if logger == nil { logger = slog.Default() }`.
  - `server/gateway`: `NewGateway` explicitly checks `if logger == nil`.
  - `server/grpc`: `NewServer` *accepts* a logger but does NOT explicitly check for `nil` before passing it to interceptors, potentially leading to panics or usage of nil pointers if not strictly controlled.
  - `health`: `OnStart` prints to `stderr` because "we don't have a configured logger yet".
- **Recommendation:** specific `slog.Default()` fallback in ALL constructors/methods.

### 2. Naming Consistency
- `server/grpc` uses `ServiceRegistrar`.
- `server/gateway` uses `Registrar` (intended to be `GatewayEndpoint` in some plans).
- **Recommendation:** Align naming if possible, or accept `[Package]Registrar` pattern. `ServiceRegistrar` is good for gRPC (matches standard). `GatewayRegistrar` might be clearer than just `Registrar` in the gateway package to avoid ambiguity when imported.

### 3. Gateway-HTTP Lifecycle Race
- **Problem:** `Gateway.OnStart` performs service discovery and builds the `http.Handler`. `http.Server.OnStart` starts listening.
- If `http.Server` depends on `Gateway` to get the handler *before* starting, the dependency injection graph must enforce `Gateway` -> `http.Server`.
- However, `Gateway.OnStart` runs at "start time". If `http.Server` starts serving immediately in its own `OnStart`, and `Gateway` hasn't finished `OnStart` (or they run in parallel if not strictly dependent in the *lifecycle* engine), there's a risk.
- **Current Fix:** `http.Server` has `SetHandler` which panics if called after start.
- **Refactor:** Ensure `Gateway` initializes its handler *during construction* (if possible) or use a "Not Ready" handler until `Gateway.OnStart` completes, utilizing `atomic.Value` for the handler in `http.Server` to allow hot-swapping or late-binding without panics (safe dynamic handler).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| gRPC <-> JSON | Custom transcoding | `grpc-gateway` | Complex mapping rules, standard annotations. |
| Logging | Custom wrapper | `log/slog` | Built-in, high performance. |
| Health Checks | Custom endpoints | `grpc.health.v1` | Standard gRPC health protocol. |

## Common Pitfalls

### Pitfall 1: Late Handler Binding
**What:** Setting the HTTP handler after the server has started accepting connections.
**Risk:** Race conditions, 404s for early requests, or panics (as currently enforced).
**Fix:** Use an atomic `http.Handler` wrapper that can be swapped safely, or ensure rigorous lifecycle ordering (Gateway `OnStart` completes -> `http.Server` `OnStart` begins).

### Pitfall 2: Logger Panic
**What:** Passing `nil` logger to a component that expects a pointer.
**Fix:** Always normalize to `slog.Default()` in constructors.

### Pitfall 3: Port Conflicts
**What:** Running gRPC and Gateway on the same port without a cmux (Connection Multiplexer).
**Fix:** The chosen "Port Separation" strategy (8080 vs 9090) avoids this complexity. Stick to it.

## Code Examples

### Robust Constructor (Recommended)
```go
func NewServer(cfg Config, logger *slog.Logger) *Server {
    if logger == nil {
        logger = slog.Default()
    }
    // ...
}
```

### Safe Dynamic Handler (for Gateway integration)
```go
type DynamicHandler struct {
    handler atomic.Value // stores http.Handler
}

func (h *DynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if v := h.handler.Load(); v != nil {
        v.(http.Handler).ServeHTTP(w, r)
        return
    }
    http.Error(w, "Service Not Ready", http.StatusServiceUnavailable)
}
```

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| `grpc.Dial` | `grpc.NewClient` | Non-blocking, modern API. |
| `log` / `zap` | `log/slog` | Standard library, structured. |
| Manual Registration | `di.ResolveAll` | Decoupled, plugin-friendly. |

## Open Questions

1. **Lifecycle Ordering:** Does `gaz`'s DI engine guarantee `OnStart` order based on dependency graph?
   - *Assumption:* Yes, usually.
   - *Action:* Verify if `http.Server` depends on `Gateway` struct. If so, `Gateway` is built first. But `OnStart` might still be concurrent or strictly sequential. Explicitly checking `di` lifecycle logic would confirm if "dependent's OnStart runs after dependency's OnStart".

## Sources

### Primary (HIGH confidence)
- Codebase analysis (`server/grpc`, `server/http`, `server/gateway`, `di`).
- `net/http` and `grpc` standard patterns.

### Secondary (MEDIUM confidence)
- Project "Prior Decisions" context.
