# Feature Landscape: Server & Transport Layer

**Domain:** Server Framework (gRPC + HTTP Gateway)
**Researched:** Mon Feb 02 2026

## Table Stakes

Features users expect from a modern Go server framework.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **gRPC Server** | Core requirement for type-safe APIs. | Medium | Use `google.golang.org/grpc`. |
| **HTTP/JSON Gateway** | Required for frontend/browser support. | Medium | Use `grpc-ecosystem/grpc-gateway`. |
| **Dynamic Registration** | Adding a service should auto-expose endpoints. | Medium | Requires DI enhancement (`di.List`). |
| **Graceful Shutdown** | Prevent dropped connections during deploys. | Low | Already supported by `gaz.App`. |
| **CORS Support** | Required for browser clients. | Low | Middleware on Gateway. |
| **Health Checks** | Required for Kubernetes. | Low | Already supported by `gaz/health`. |

## Differentiators

Features that improve Developer Experience (DX) in `gaz`.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Auto-Discovery** | No manual wiring of handlers; just implement interface. | High | Leverages `gaz` DI power. |
| **Unified Middleware** | Apply middleware (auth, log) once for both protocols? | High | Hard due to different stacks. Focus on separate but consistent middleware. |
| **Swagger/OpenAPI** | Auto-generation of API docs from proto. | Medium | tooling/build step. |

## Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Magic Ports** | Random ports are hard to debug/configure. | Explicit config or standard defaults. |
| **Hidden Multiplexing** | `cmux` hides protocol complexity but adds runtime fragility. | Explicit ports. |

## Feature Dependencies

```
Gateway Feature
└── Requires: gRPC Server (target)
└── Requires: DI Discovery (implementation mechanism)
```

## MVP Recommendation

1.  **DI Update:** Implement `di.List[T]`.
2.  **Gateway Service:** Basic implementation with separate ports.
3.  **Discovery Logic:** Wiring the two together.
4.  **Config:** Port configuration for both servers.

## Sources

- `gaz` Requirements
- gRPC Ecosystem Patterns
