# Phase 40: Observability & Health - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Expose standard health checks and telemetry for production monitoring. Delivers gRPC health endpoint (`grpc.health.v1`), extensible health check pattern with Postgres example, and OpenTelemetry tracing for Gateway → gRPC request flows.

</domain>

<decisions>
## Implementation Decisions

### Health check extensibility
- Discovery pattern via `di.List` — services implement interface, auto-discovered
- Simple `Check()` interface returning status + details (no separate Name/Timeout methods)
- Checks declare themselves as critical or optional
- Critical check failure = NOT_SERVING, optional failure = degraded state

### Health endpoint exposure
- gRPC only (`grpc.health.v1.Health` service) — no HTTP health endpoints
- Aggregate status only — no per-check querying via service parameter
- Three-state status: SERVING / SERVING_DEGRADED / NOT_SERVING
  - SERVING: All checks pass
  - SERVING_DEGRADED: Only optional checks failing
  - NOT_SERVING: Any critical check fails
- Cached checks with periodic background refresh (configurable interval)
- Force-refresh capability available for on-demand evaluation

### Trace propagation
- Full instrumentation: request lifecycle + method calls + DB queries
- W3C Trace Context standard (`traceparent`/`tracestate` headers)
- Parent-based sampling: respect incoming trace decisions
- Probabilistic sampling for root spans (requests without incoming context)
- Configurable sample ratio for root spans

### Configuration surface
- Full config stack: environment vars + module options + CLI flags
- Minimal OTLP config surface: only endpoint URL exposed initially
- OTEL_* env vars work as standard OTEL SDK fallback
- Auto-detect mode: observability enabled if OTEL endpoint configured
- Graceful degradation: log warning and continue if collector unreachable

### Claude's Discretion
- Exact health check interface shape (as long as it's simple with Check returning status+details)
- Background check interval default value
- How degraded status maps to gRPC health proto (may need custom extension)
- OTEL span naming conventions
- Internal span attributes

</decisions>

<specifics>
## Specific Ideas

- Health checks should feel natural with the existing `di.List` discovery pattern from Phase 37
- The Postgres check from success criteria is an example — the pattern should work for any dependency
- Tracing should "just work" when OTEL endpoint is set, without extra code

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 40-observability-health*
*Context gathered: 2026-02-03*
