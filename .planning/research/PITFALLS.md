# Domain Pitfalls: Server & Transport Layer

**Domain:** Server Framework
**Researched:** Mon Feb 02 2026

## Critical Pitfalls

### Pitfall 1: The "Cmux" Trap
**What goes wrong:** Developers try to run gRPC and HTTP on the same port using `cmux`.
**Why it happens:** Desire for "simplicity" (one port).
**Consequences:** Weird connection drops, HTTP/2 compatibility issues, L7 load balancer failures, complex TLS configuration.
**Prevention:** **Enforce separate ports** by default.
**Detection:** If using `cmux`, watch for "connection closed" errors or HTTP/1.1 vs HTTP/2 negotiation failures.

### Pitfall 2: Circular Dependencies in Gateway
**What goes wrong:** The `Gateway` depends on `ServiceA` (to register it), but `ServiceA` depends on `Gateway` (for some reason) or they both depend on a shared component that causes a cycle.
**Why it happens:** Eager resolution during discovery.
**Prevention:**
- Use `di.List` inside `OnStart` (runtime), not in the provider (build time).
- Ensure `GatewayEndpoint` implementations are lightweight and don't block `OnStart`.

### Pitfall 3: Startup Race Conditions
**What goes wrong:** Gateway starts accepting traffic before gRPC server is ready or before handlers are registered.
**Consequences:** 502 Bad Gateway or 404 Not Found errors during startup.
**Prevention:**
1.  Start gRPC Server first.
2.  Register all handlers.
3.  Start Gateway Listener last.
**Implementation:** Use `gaz` startup order (dependency graph). `Gateway` should depend on `GrpcServer` (implicitly or explicitly via `ClientConn`).

## Moderate Pitfalls

### Pitfall 4: Missing CORS
**What goes wrong:** Browser clients fail to connect to Gateway.
**Prevention:** Include default CORS middleware in Gateway.

### Pitfall 5: IPv6/IPv4 Binding
**What goes wrong:** Server binds to `localhost` (IPv6 `::1`) but client dials `127.0.0.1`, or vice versa.
**Prevention:** Use explicit binding (e.g., `0.0.0.0` or consistent interface).

## Sources

- gRPC Go Community Issues
- Kubernetes Best Practices
