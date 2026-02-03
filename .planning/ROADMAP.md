# Roadmap: v4.1 Server & Transport Layer

**Milestone:** v4.1
**Status:** Active
**Started:** 2026-02-03

## Overview

Milestone v4.1 transforms `gaz` into a production-ready application server by implementing a dual-port architecture (gRPC + HTTP Gateway). This roadmap delivers the "Dynamic Gateway" pattern, where HTTP traffic is transparently proxied to gRPC services via auto-discovery. This requires a fundamental upgrade to the Core DI engine (`di.List[T]`) to support service enumeration, followed by the implementation of the transport and observability layers.

## Phases

### Phase 37: Core Discovery
**Goal:** Enable the container to resolve all registered providers of a type to support auto-discovery patterns.
**Dependencies:** None
**Requirements:** CORE-01
**Plans:** 2/2 plans complete

- [x] 37-01-PLAN.md — Refactor Core DI engine & Implement Discovery API
- [x] 37-02-PLAN.md — Verify Discovery with Plugin Example

**Status:** Complete (2026-02-02)

**Success Criteria:**
1. User can register multiple providers for the same interface/type.
2. User can resolve `[]T` (or `di.List[T]`) to retrieve all instances.
3. Container detects and prevents cycles within list resolution.
4. Existing single-resolution (`Resolve[T]`) remains unaffected.

### Phase 38: Transport Foundations
**Goal:** Establish independent, production-ready gRPC and HTTP listeners on configurable ports.
**Dependencies:** Phase 37
**Requirements:** TRN-01, TRN-02, TRN-03, GW-01
**Plans:** 3 plans

Plans:
- [ ] 38-01-PLAN.md — gRPC Server with interceptors, reflection, service discovery
- [ ] 38-02-PLAN.md — HTTP Server with configurable timeouts
- [ ] 38-03-PLAN.md — Unified module and tests

**Success Criteria:**
1. Application starts both a gRPC server and an HTTP server on configured ports.
2. gRPC Reflection is available and queryable via `grpcurl`.
3. Servers shut down gracefully when the application stops.
4. Basic interceptors (logging/recovery) are active on the gRPC server.

### Phase 39: Gateway Integration
**Goal:** Unify HTTP and gRPC via a dynamic, auto-discovering Gateway layer.
**Dependencies:** Phase 37 (Discovery), Phase 38 (Servers)
**Requirements:** GW-02, GW-03, GW-04

**Success Criteria:**
1. Gateway automatically detects services implementing `GatewayEndpoint`.
2. HTTP requests to Gateway port are proxied to the gRPC server via loopback.
3. CORS headers are correctly applied to Gateway responses.
4. Adding a new service requires no manual Gateway wiring code.

### Phase 40: Observability & Health
**Goal:** Expose standard health checks and telemetry for production monitoring.
**Dependencies:** Phase 38 (Servers)
**Requirements:** INF-01, INF-02, INF-03

**Success Criteria:**
1. Standard gRPC Health Verification endpoint returns serving status.
2. App reports "Unhealthy" if the configured Postgres database is unreachable.
3. OpenTelemetry traces are generated for requests spanning Gateway -> gRPC.

## Progress

| Phase | Goal | Status | Requirements |
|-------|------|--------|--------------|
| 37 | Core Discovery | **Complete** | CORE-01 |
| 38 | Transport Foundations | Pending | TRN-01, TRN-02, TRN-03, GW-01 |
| 39 | Gateway Integration | Pending | GW-02, GW-03, GW-04 |
| 40 | Observability & Health | Pending | INF-01, INF-02, INF-03 |
