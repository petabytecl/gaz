# Phase 38: Transport Foundations - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish independent, production-ready gRPC and HTTP listeners on configurable ports. This phase delivers the server infrastructure (both servers running, reflection enabled, graceful shutdown, basic interceptors) that downstream Gateway and Observability phases will build on.

</domain>

<decisions>
## Implementation Decisions

### Configuration design
- Config structure: nested under `servers` key (e.g., `servers.grpc.port`, `servers.http.port`)
- Default ports: gRPC on 50051, HTTP on 8080
- CLI flags (--grpc-port, --http-port) override config file values
- TLS disabled by default, can be enabled via config

### Server lifecycle integration
- Startup order: gRPC first, then HTTP (Gateway will depend on gRPC being up)
- Shutdown order: HTTP first, then gRPC (reverse of startup)
- Graceful shutdown timeout: 30 seconds default, configurable
- Port binding failure: fail fast with clear error message (no retry, no fallback)

### Interceptor behavior
- Request logging: use gaz logger's existing format and defaults
- Request IDs: accept X-Request-ID header if present, generate UUID if not
- Panic recovery: log full stack trace; return error details in dev mode only, generic Internal error in production
- Logging verbosity: configurable (can log all, errors only, or errors + slow requests)

### Module structure
- Package location: `server/` with subpackages (`server/grpc`, `server/http`)
- gRPC service registration: auto-discovery via `di.List` pattern (services implement a registrar interface)
- gRPC reflection: enabled by default when module is imported

### Claude's Discretion
- Whether to use unified transport module or separate grpc/http modules
- Exact interface name for service registration (e.g., `grpc.ServiceRegistrar`)
- Specific config struct field names within the `servers` namespace
- How to detect dev vs production mode for error detail exposure

</decisions>

<specifics>
## Specific Ideas

- Follow gRPC convention for default port (50051) to feel familiar to gRPC users
- gRPC reflection should "just work" so grpcurl can query services out of the box
- Startup/shutdown order explicitly considers Gateway dependency (Phase 39)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 38-transport-foundations*
*Context gathered: 2026-02-03*
