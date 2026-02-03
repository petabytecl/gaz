# Technology Stack: Server & Transport Layer

**Project:** gaz (v4.1)
**Researched:** Mon Feb 02 2026

## Recommended Stack

### Transport Layer
| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `net/http` | std (Go 1.25) | HTTP Server | Go 1.22+ `ServeMux` is powerful enough for most HTTP needs; avoid heavy frameworks (Gin/Echo) unless necessary. |
| `google.golang.org/grpc` | v1.70+ | gRPC Server | Industry standard. Native Go implementation. |
| `grpc-ecosystem/grpc-gateway` | v2.26+ | REST Gateway | Generates reverse proxy from proto definitions. Essential for backward compatibility and web clients. |
| `connect-go` | (Alternative) | gRPC-compatible | *Consider* for simpler setups, but stick to `grpc-gateway` per requirements. |

### Database
| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/jackc/pgx/v5` | v5.7+ | PostgreSQL Driver | High performance, native types, robust `pgxpool`. Superior to `lib/pq` (maintenance mode). |

### Middleware & Interceptors
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/grpc-ecosystem/go-grpc-middleware/v2` | v2.2+ | Interceptor Chains | Essential for chaining logging, recovery, auth, and validation interceptors. |
| `github.com/bufbuild/protovalidate-go` | v0.5+ | Request Validation | Modern replacement for `protoc-gen-validate`. Defines validation rules in `.proto` files. |
| `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` | v0.59+ | Tracing/Metrics | Standard OTel instrumentation for gRPC. |
| `github.com/rs/cors` | v1.11+ | CORS Middleware | Required if Gateway is called from browser frontends. |

## Tooling (Dev Experience)
| Tool | Purpose | Recommendation |
|------|---------|----------------|
| `buf` | Proto Management | Use `buf` instead of raw `protoc`. Handles dependency management, linting, and generation much cleaner. |
| `protoc-gen-openapiv2` | API Docs | Generate Swagger/OpenAPI spec from gRPC definitions for the Gateway. |

## Integration Strategy

### Architecture: Separate Ports (Recommended)
While `cmux` or `h2c` allows single-port operation, **separate ports** are recommended for production Kubernetes environments:
- **Port 9090 (gRPC):** Pure HTTP/2. Native performance.
- **Port 8080 (HTTP):** HTTP/1.1 + HTTP/2. Serves Gateway + other HTTP routes (health, metrics).
- **Reason:** Ingress controllers (ALB, Nginx) often handle protocols differently. `cmux` adds fragility and complexity (L7 matching).

### DI Integration (gaz)
Register distinct providers for each component:

1.  **gRPC Server Provider:**
    -   Inputs: `*grpc.ServerOption` (interceptors), registered services.
    -   Outputs: `*grpc.Server`.
    -   Lifecycle: `OnStart` (Listen & Serve), `OnStop` (`GracefulStop`).

2.  **Gateway Provider:**
    -   Inputs: `*grpc.ClientConn` (loopback to gRPC server), `*runtime.ServeMuxOption`.
    -   Outputs: `*runtime.ServeMux`.
    -   Lifecycle: None (stateless handler).

3.  **HTTP Server Provider:**
    -   Inputs: `*runtime.ServeMux` (as handler), Port config.
    -   Outputs: `*http.Server`.
    -   Lifecycle: `OnStart` (ListenAndServe), `OnStop` (`Shutdown`).

4.  **Database Provider:**
    -   Inputs: Config struct.
    -   Outputs: `*pgxpool.Pool`.
    -   Lifecycle: `OnStart` (Ping), `OnStop` (`Close`).

## Anti-Patterns to Avoid

-   **`cmux` for everything:** Avoid unless single-port is a hard constraint (e.g., restrictive firewall). It complicates debugging.
-   **`protoc-gen-validate` (legacy):** Deprecated in favor of `protovalidate`. Do not start new projects with legacy PGV.
-   **Global State:** Do not rely on `http.DefaultServeMux` or global `grpc.Server`. Always inject instances.
-   **Ignoring `MaxConnectionAge`:** In gRPC, this causes load balancer imbalances. Configure `KeepaliveParams`.

## Version Compatibility Check
-   `grpc-gateway/v2` requires `google.golang.org/grpc` v1.64+ (Verified: v1.70+ is safe).
-   `protovalidate-go` requires `google.golang.org/protobuf` v1.34+ (Verified: v1.36+ is safe).

## Installation

```bash
# Core
go get google.golang.org/grpc@latest
go get google.golang.org/protobuf@latest
go get github.com/grpc-ecosystem/grpc-gateway/v2@latest
go get github.com/jackc/pgx/v5@latest

# Middleware & Validation
go get github.com/grpc-ecosystem/go-grpc-middleware/v2@latest
go get github.com/bufbuild/protovalidate-go@latest
go get go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc@latest

# CORS (if needed)
go get github.com/rs/cors@latest
```
