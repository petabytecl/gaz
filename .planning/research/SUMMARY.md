# Research Summary: Server & Transport Layer

**Synthesized:** Mon Feb 02 2026
**Sources:** STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md

## Executive Summary

The research concludes that `gaz` should implement a **dual-port server architecture** supporting both gRPC (native) and HTTP/JSON (via Gateway). The industry-standard `grpc-ecosystem/grpc-gateway` is the recommended approach for the HTTP translation layer, as it offers superior compatibility and tooling compared to alternatives.

A critical architectural decision is to **avoid `cmux`** (multiplexing on a single port) in favor of running the gRPC server (e.g., :9090) and HTTP Gateway (e.g., :8080) on separate ports. This simplifies configuration, improves compatibility with L7 load balancers, and enhances observability.

To align with `gaz`'s developer experience goals, the system relies on **Dynamic Interface Discovery**. Services should not manually register with the server; instead, the Gateway will auto-discover all implementations of a `GatewayEndpoint` interface using a new `di.List[T]` capability. This requires a fundamental enhancement to the core Dependency Injection container before the server layer can be fully realized.

## Key Findings

### Stack & Technology
- **Core:** `net/http` (Std Lib) + `google.golang.org/grpc` (v1.70+).
- **Gateway:** `grpc-ecosystem/grpc-gateway` (v2.26+) for REST translation.
- **Validation:** `bufbuild/protovalidate-go` (modern replacement for legacy validator).
- **Database:** `jackc/pgx/v5` is the standard for PostgreSQL.

### Architecture
- **Port Separation:** Run gRPC and HTTP on distinct ports to avoid the "cmux trap."
- **Discovery Pattern:** Use Dependency Injection to find services. The Gateway iterates over `di.List[GatewayEndpoint]` at startup.
- **Loopback:** The Gateway connects to the gRPC server via a local client connection.

### Features
- **Table Stakes:** gRPC Server, JSON Gateway, CORS, Health Checks, Graceful Shutdown.
- **Differentiators:** Auto-discovery of endpoints (Zero-config registration for developers).
- **Anti-Features:** "Magic Ports" (random assignment) and Hidden Multiplexing (`cmux`).

### Critical Pitfalls
- **The "Cmux" Trap:** Complexity of multiplexing leads to fragility in Kubernetes/Service Mesh environments.
- **Circular Dependencies:** Eager resolution of services in the Gateway constructor can cause cycles. Discovery must happen in `OnStart`.
- **Startup Races:** Gateway must not accept traffic until the backend gRPC server is healthy.

## Implications for Roadmap

The research strongly suggests a phased approach starting with the Core DI engine.

### Phase 1: Core DI Enhancement
**Rationale:** The auto-discovery feature (a key differentiator) depends on `di.List[T]`, which does not currently exist.
- **Deliverables:** `di.List[T](c)` implementation, generic type discovery in Container.
- **Pitfalls:** Ensure thread safety and avoid breaking existing resolution logic.

### Phase 2: Server Foundations & Loopback
**Rationale:** Establish the runtime environment before adding user features.
- **Deliverables:** `GrpcServer` module, `Gateway` module, Port configuration, Internal Loopback connection.
- **Architecture:** Separate ports (e.g., 9090/8080).

### Phase 3: Dynamic Registration & Middleware
**Rationale:** Connect the DI capabilities (Phase 1) with the Server (Phase 2).
- **Deliverables:** `GatewayEndpoint` interface, Discovery logic in `Gateway.OnStart`, Middleware chains (logging, recovery, CORS).
- **Pitfalls:** Handle startup ordering carefully to prevent 502s.

## Research Flags

- **Standard Patterns:** The gRPC/Gateway stack is well-understood. No further research needed for Phase 2.
- **Implementation Detail:** Phase 1 (`di.List`) requires careful design within `gaz`'s existing reflection code but does not require external research.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| **Stack** | High | Industry standard choices (gRPC, pgx, grpc-gateway). |
| **Features** | High | Clear distinction between table stakes and differentiators. |
| **Architecture** | High | Strong consensus on Port Separation vs. Multiplexing. |
| **Pitfalls** | High | Specific, actionable warnings about concurrency and networking. |
