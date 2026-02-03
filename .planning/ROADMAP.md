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
**Plans:** 3/3 plans complete

- [x] 38-01-PLAN.md — gRPC Server with interceptors, reflection, service discovery
- [x] 38-02-PLAN.md — HTTP Server with configurable timeouts
- [x] 38-03-PLAN.md — Unified module and tests

**Status:** Complete (2026-02-03)

**Success Criteria:**
1. Application starts both a gRPC server and an HTTP server on configured ports.
2. gRPC Reflection is available and queryable via `grpcurl`.
3. Servers shut down gracefully when the application stops.
4. Basic interceptors (logging/recovery) are active on the gRPC server.

### Phase 38.1: gRPC and HTTP Server CLI Flags (INSERTED)

**Goal:** Enable gRPC and HTTP servers to register CLI flags for configuring ports and core settings.
**Dependencies:** Phase 38
**Requirements:** Derived from TRN-01, TRN-02
**Plans:** 1/1 plan complete

- [x] 38.1-01-PLAN.md — Add NewModuleWithFlags returning gaz.Module with CLI flags

**Status:** Complete (2026-02-03)

**Success Criteria:**
1. gRPC server port configurable via `--grpc-port` flag.
2. HTTP server port configurable via `--http-port` flag.
3. gRPC reflection and dev mode configurable via flags.
4. Existing `server.NewModule()` API preserved for backward compatibility.
5. Flag values correctly applied at runtime (not at module creation time).

**Scope Note:** HTTP timeouts (read, write, idle) remain configurable via module options only. CLI flags for timeouts may be added in a future quick task if needed.

### Phase 39: Gateway Integration
**Goal:** Unify HTTP and gRPC via a dynamic, auto-discovering Gateway layer.
**Dependencies:** Phase 37 (Discovery), Phase 38 (Servers)
**Requirements:** GW-02, GW-03, GW-04
**Plans:** 3/3 plans complete

Plans:
- [x] 39-01-PLAN.md — Core Gateway package (config, headers, gateway, errors)
- [x] 39-02-PLAN.md — Gateway module with DI and CLI flags
- [x] 39-03-PLAN.md — Comprehensive tests (92% coverage)

**Status:** Complete (2026-02-03)

**Success Criteria:**
1. Gateway automatically detects services implementing `Registrar`.
2. HTTP requests to Gateway port are proxied to the gRPC server via loopback.
3. CORS headers are correctly applied to Gateway responses.
4. Adding a new service requires no manual Gateway wiring code.

### Phase 40: Observability & Health
**Goal:** Expose standard health checks and telemetry for production monitoring.
**Dependencies:** Phase 38 (Servers)
**Requirements:** INF-01, INF-02, INF-03
**Plans:** 3 plans

Plans:
- [x] 40-01-PLAN.md — Health check interface, aggregator, gRPC health server
- [x] 40-02-PLAN.md — OpenTelemetry TracerProvider and server instrumentation
- [x] 40-03-PLAN.md — PGX health check and comprehensive tests

**Status:** Complete (2026-02-03)

**Success Criteria:**
1. Standard gRPC Health Verification endpoint returns serving status.
2. App reports "Unhealthy" if the configured Postgres database is unreachable.
3. OpenTelemetry traces are generated for requests spanning Gateway -> gRPC.

## Progress

| Phase | Goal | Status | Requirements |
|-------|------|--------|--------------|
| 37 | Core Discovery | **Complete** | CORE-01 |
| 38 | Transport Foundations | **Complete** | TRN-01, TRN-02, TRN-03, GW-01 |
| 38.1 | gRPC/HTTP CLI Flags | **Complete** | TRN-01, TRN-02 |
| 39 | Gateway Integration | **Complete** | GW-02, GW-03, GW-04 |
| 40 | Observability & Health | **Complete** | INF-01, INF-02, INF-03 |
