# Requirements: gaz v5.0

**Defined:** 2026-03-06
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v5.0 Requirements

Requirements for the Vanguard Unified Server milestone. Each maps to roadmap phases.

### Unified Server

- [x] **USRV-01**: Developer can create a Vanguard server that serves gRPC, Connect, gRPC-Web, and REST on a single http.Handler
- [x] **USRV-02**: Developer can serve all protocols on a single port using HTTP/2 cleartext (h2c) via Go native http.Protocols
- [x] **USRV-03**: Developer can expose REST endpoints from proto google.api.http annotations without codegen
- [x] **USRV-04**: Browser clients can access services via gRPC-Web without an external proxy
- [x] **USRV-05**: Developer can mount custom HTTP handlers for non-RPC routes (health, metrics, static) via unknown handler
- [x] **USRV-06**: Developer can configure Vanguard server address, timeouts, and options via CLI flags and config struct

### Connect Integration

- [x] **CONN-01**: Developer can register Connect-Go services via ConnectRegistrar interface with auto-discovery through di.List
- [ ] **CONN-02**: Framework automatically injects Connect interceptors (auth, logging, validation, OTEL) into all Connect handlers
- [ ] **CONN-03**: Developer can create ConnectInterceptorBundle with priority-sorted, auto-discovered interceptor chains
- [x] **CONN-04**: Developer can enable gRPC reflection for Connect services via connectrpc.com/grpcreflect

### Middleware

- [ ] **MDDL-01**: Developer can apply CORS middleware at transport level for browser clients accessing Connect and gRPC-Web
- [ ] **MDDL-02**: Vanguard server uses two-layer middleware model: HTTP transport middleware wrapping handler, Connect interceptors per-service
- [ ] **MDDL-03**: Developer can enable OpenTelemetry tracing and metrics for both HTTP transport (otelhttp) and Connect RPC (otelconnect) layers
- [ ] **MDDL-04**: Developer can enable proto constraint validation via connectrpc.com/validate interceptor
- [x] **MDDL-05**: Health checks are wired into the unified Vanguard server

### Server Module

- [ ] **SMOD-01**: Developer can use updated server.NewModule() that bundles Vanguard instead of gRPC-Gateway
- [ ] **SMOD-02**: Existing server/gateway package is removed (clean break, no backward compatibility)
- [ ] **SMOD-03**: Existing server/http package is preserved for standalone HTTP-only use cases

## Future Requirements

### Deferred

- **MIGR-01**: vanguardgrpc.NewTranscoder bridge for existing grpc.Registrar services
- **MIGR-02**: Migration guide from gRPC-Gateway to Vanguard
- **EXMP-01**: Reference implementation examples in examples/

## Out of Scope

| Feature | Reason |
|---------|--------|
| Unified gRPC/Connect interceptor type | Fundamentally different signatures (grpc.UnaryServerInterceptor vs connect.Interceptor); bridging creates leaky abstraction |
| cmux protocol multiplexing | Unnecessary — Vanguard handles all protocol dispatch natively |
| Custom REST routing DSL | Proto google.api.http annotations are the standard; custom DSL fragments ecosystem |
| Automatic TLS management | Complex, many approaches; users BYO certs or platform-level TLS termination |
| WebSocket support | Different paradigm; Vanguard doesn't support it; Connect streaming covers real-time use cases |
| Multi-port serving | Contradicts single-port goal; users can create multiple module instances if needed |
| gRPC-Gateway backward compatibility | Clean break — Vanguard is a strict superset; no bridge needed |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| USRV-01 | Phase 46 | Complete |
| USRV-02 | Phase 46 | Complete |
| USRV-03 | Phase 46 | Complete |
| USRV-04 | Phase 46 | Complete |
| USRV-05 | Phase 46 | Complete |
| USRV-06 | Phase 46 | Complete |
| CONN-01 | Phase 46 | Complete |
| CONN-02 | Phase 47 | Pending |
| CONN-03 | Phase 47 | Pending |
| CONN-04 | Phase 46 | Complete |
| MDDL-01 | Phase 47 | Pending |
| MDDL-02 | Phase 47 | Pending |
| MDDL-03 | Phase 47 | Pending |
| MDDL-04 | Phase 47 | Pending |
| MDDL-05 | Phase 46 | Complete |
| SMOD-01 | Phase 48 | Pending |
| SMOD-02 | Phase 48 | Pending |
| SMOD-03 | Phase 48 | Pending |

**Coverage:**
- v5.0 requirements: 18 total
- Mapped to phases: 18
- Unmapped: 0 ✓

---
*Requirements defined: 2026-03-06*
*Last updated: 2026-03-06 after roadmap creation*
