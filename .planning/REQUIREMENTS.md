# Requirements: gaz

**Defined:** 2026-02-03
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v4.1 Requirements

**Milestone:** v4.1 Server & Transport Layer
**Goal:** Implement production-ready HTTP and gRPC server capabilities with a unified Gateway pattern.

### Core Framework

- [x] **CORE-01**: Container supports resolving all providers of type T (`di.List[T]`) for discovery

### Transport Layer

- [x] **TRN-01**: HTTP Server with configurable timeouts (Read/Write/Idle/Header)
- [x] **TRN-02**: gRPC Server with Interceptors (logging, recovery, OTEL)
- [x] **TRN-03**: gRPC Reflection enabled by default

### Gateway Integration

- [x] **GW-01**: Gateway runs on separate HTTP port (no cmux)
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
| CORE-01 | Phase 37 | Complete |
| TRN-01 | Phase 38 | Complete |
| TRN-02 | Phase 38 | Complete |
| TRN-03 | Phase 38 | Complete |
| GW-01 | Phase 38 | Complete |
| GW-02 | Phase 39 | Pending |
| GW-03 | Phase 39 | Pending |
| GW-04 | Phase 39 | Pending |
| INF-01 | Phase 40 | Pending |
| INF-02 | Phase 40 | Pending |
| INF-03 | Phase 40 | Pending |

**Coverage:**
- v4.1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-03*
