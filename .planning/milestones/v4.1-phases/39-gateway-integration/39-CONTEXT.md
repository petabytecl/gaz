# Phase 39: Gateway Integration - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Unify HTTP and gRPC via a dynamic Gateway that auto-discovers services implementing `GatewayEndpoint` and proxies HTTP requests to the gRPC server via loopback. Uses grpc-gateway as the underlying HTTP-to-gRPC translation layer.

</domain>

<decisions>
## Implementation Decisions

### GatewayEndpoint interface
- Use grpc-gateway's native registration system (proto annotations define HTTP mappings)
- Define `GatewayRegistrar` interface with `RegisterWithMux(*runtime.ServeMux, grpc.ClientConn)` method
- Services call their generated `RegisterXXXHandler` via this interface
- Registration happens during service provider function: `gaz.Resolve[*Gateway](c).Register(...)`
- Gateway uses `di.List[GatewayRegistrar]` for auto-discovery of all registered services

### HTTP-to-gRPC mapping
- Gateway connects to gRPC server via in-process loopback (`localhost:grpcPort`)
- Default loopback derived from gRPC port configuration
- Explicit `--gateway-grpc-target` flag available for distributed deployments (gRPC in different pod)
- Gateway auto-starts after gRPC server is ready (DI dependency ordering)
- Explicit header allowlist for HTTP-to-gRPC metadata forwarding (not grpc-gateway defaults)

### CORS configuration
- Global CORS policy with per-route overrides capability
- Dev mode: wide-open CORS (AllowAll=true)
- Prod mode: strict, requires explicit AllowOrigins configuration
- Dev mode inherited from gRPC server's `--dev-mode` flag
- Allowed origins configurable via module options, with config/env override at runtime

### Error response format
- Use RFC 7807 Problem Details format: `{"type", "title", "status", "detail", "instance"}`
- `instance` field contains correlation ID for request tracing
- Dev mode includes stack traces and debug details in responses
- Prod mode strips internal details

### Claude's Discretion
- gRPC status code to HTTP status code mapping (use standard grpc-gateway mapping or sensible custom)
- Header allowlist defaults (x-request-id, authorization, etc.)
- Exact grpc-gateway runtime.ServeMux options

</decisions>

<specifics>
## Specific Ideas

- grpc-gateway is the foundation; don't reinvent the HTTP-gRPC mapping
- Support running gRPC and Gateway in same process (typical) or separate pods (distributed)
- RFC 7807 is the standard for HTTP API error responses
- Dev mode should be convenient for local development (wide-open CORS, debug info)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 39-gateway-integration*
*Context gathered: 2026-02-03*
