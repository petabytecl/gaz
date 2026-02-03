# Requirements: gaz

**Defined:** 2026-02-03
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v4.1 Requirements

**Milestone:** v4.1 Server & Transport Layer
**Goal:** Implement production-ready HTTP and gRPC server capabilities with a unified Gateway pattern.

### Core Framework

- [ ] **CORE-01**: Container supports resolving all providers of type T (`di.List[T]`) for discovery

### Transport Layer

- [ ] **TRN-01**: HTTP Server with configurable timeouts (Read/Write/Idle/Header)
- [ ] **TRN-02**: gRPC Server with Interceptors (logging, recovery, OTEL)
- [ ] **TRN-03**: gRPC Reflection enabled by default

### Gateway Integration

- [ ] **GW-01**: Gateway runs on separate HTTP port (no cmux)
- [ ] **GW-02**: Connects to gRPC via loopback client
- [ ] **GW-03**: Dynamic registration of services via DI interface (`GatewayEndpoint`)
- [ ] **GW-04**: CORS support (Origins, Methods, Headers)

### Infrastructure

- [ ] **INF-01**: PGX health check (`jackc/pgx`)
- [ ] **INF-02**: gRPC health check
- [ ] **INF-03**: OpenTelemetry instrumentation

## Out of Scope

| Feature | Reason |
|---------|--------|
| cmux / Single Port | Research indicates fragility and K8s ingress issues |
| HTTP/3 (QUIC) | Defer to v4.2 due to library maturity |
| Distributed Rate Limiting | Requires external store (Redis), out of scope for core |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CORE-01 | — | Pending |
| TRN-01 | — | Pending |
| TRN-02 | — | Pending |
| TRN-03 | — | Pending |
| GW-01 | — | Pending |
| GW-02 | — | Pending |
| GW-03 | — | Pending |
| GW-04 | — | Pending |
| INF-01 | — | Pending |
| INF-02 | — | Pending |
| INF-03 | — | Pending |

**Coverage:**
- v4.1 requirements: 11 total
- Mapped to phases: 0
- Unmapped: 11 ⚠️

---
*Requirements defined: 2026-02-03*
