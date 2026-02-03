---
phase: 40-observability-health
plan: 02
subsystem: observability
tags: [opentelemetry, otel, tracing, otlp, otelgrpc, otelhttp, instrumentation]

# Dependency graph
requires:
  - phase: 38
    provides: "gRPC and HTTP servers"
  - phase: 39
    provides: "Gateway with CORS support"
provides:
  - OpenTelemetry TracerProvider with OTLP export
  - gRPC server instrumentation with otelgrpc
  - Gateway HTTP instrumentation with otelhttp
  - Graceful degradation when collector unavailable
affects: [40-03, future observability phases]

# Tech tracking
tech-stack:
  added:
    - go.opentelemetry.io/otel
    - go.opentelemetry.io/otel/sdk/trace
    - go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
    - go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
    - go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
  patterns:
    - ParentBased sampling with TraceIDRatioBased for root spans
    - Optional instrumentation via nil TracerProvider pattern
    - Health endpoint filtering from traces

key-files:
  created:
    - server/otel/doc.go
    - server/otel/config.go
    - server/otel/provider.go
    - server/otel/module.go
  modified:
    - server/grpc/server.go
    - server/grpc/module.go
    - server/gateway/gateway.go
    - server/gateway/module.go
    - .golangci.yml

key-decisions:
  - "Graceful degradation: return nil TracerProvider when collector unreachable"
  - "Filter health endpoints from tracing to reduce noise"
  - "Use ParentBased sampler to respect incoming trace context"
  - "Pass TracerProvider as optional parameter (nil = disabled)"

patterns-established:
  - "Optional instrumentation: pass nil TracerProvider to disable OTEL"
  - "Health endpoint filtering: exclude /health, /healthz from HTTP and grpc.health.v1 from gRPC"
  - "OTEL_EXPORTER_OTLP_ENDPOINT env var fallback for configuration"

# Metrics
duration: 13min
completed: 2026-02-03
---

# Phase 40 Plan 02: OpenTelemetry Tracing Summary

**OpenTelemetry tracing infrastructure with TracerProvider, OTLP export, and instrumentation of gRPC server and Gateway for distributed tracing**

## Performance

- **Duration:** 13 min
- **Started:** 2026-02-03T21:38:58Z
- **Completed:** 2026-02-03T21:51:28Z
- **Tasks:** 3
- **Files modified:** 12

## Accomplishments

- Created server/otel package with TracerProvider initialization and OTLP gRPC exporter
- Instrumented gRPC server with otelgrpc stats handler (filtered health checks)
- Instrumented Gateway with otelhttp wrapper (filtered health endpoints)
- Added graceful degradation when OTEL collector is unreachable
- Updated depguard to allow OTEL imports across the project

## Task Commits

Each task was committed atomically:

1. **Task 1: Create OTEL TracerProvider package** - `68ece22` (feat)
2. **Task 2: Instrument gRPC server with otelgrpc** - `6261bca` (feat)
3. **Task 3: Instrument Gateway with otelhttp** - `93b26e9` (feat)

## Files Created/Modified

### Created
- `server/otel/doc.go` - Package documentation explaining OpenTelemetry integration
- `server/otel/config.go` - Config struct with endpoint, service name, sample ratio, insecure flag
- `server/otel/provider.go` - InitTracer and ShutdownTracer with OTLP exporter setup
- `server/otel/module.go` - DI module with options and env var fallback

### Modified
- `server/grpc/server.go` - Added otelgrpc stats handler when TracerProvider available
- `server/grpc/module.go` - Resolve optional TracerProvider from DI
- `server/grpc/server_test.go` - Updated tests to pass nil TracerProvider
- `server/gateway/gateway.go` - Wrapped handler with otelhttp when enabled
- `server/gateway/module.go` - Resolve optional TracerProvider from DI
- `server/gateway/gateway_test.go` - Updated tests to pass nil TracerProvider
- `go.mod` / `go.sum` - Added OTEL dependencies
- `.golangci.yml` - Added OTEL to depguard allow list

## Decisions Made

1. **Graceful degradation pattern:** If OTLP exporter creation fails, log a warning and continue without tracing rather than failing startup. This ensures applications work in environments without collectors.

2. **ParentBased sampling:** Uses ParentBased(TraceIDRatioBased(0.1)) sampler - respects incoming trace context decisions, samples 10% of root spans by default.

3. **Health endpoint filtering:** Both gRPC and HTTP health check endpoints are excluded from tracing to reduce noise:
   - gRPC: `/grpc.health.v1.Health/Check` and `/grpc.health.v1.Health/Watch`
   - HTTP: `/health` and `/healthz`

4. **Optional TracerProvider pattern:** Pass nil TracerProvider to disable instrumentation. This allows servers to work identically with or without OTEL configured.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- **otelgrpc Filter signature change:** The otelgrpc package's Filter type now uses `*stats.RPCTagInfo` instead of `*otelgrpc.InterceptorInfo`. Fixed by using `info.FullMethodName` field instead of `info.Method`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- OTEL infrastructure complete with TracerProvider and server instrumentation
- Ready for 40-03-PLAN.md (PGX health check and comprehensive tests)
- W3C Trace Context propagation enabled for Gateway -> gRPC flows
- Applications can now be traced when OTEL_EXPORTER_OTLP_ENDPOINT is set

---
*Phase: 40-observability-health*
*Completed: 2026-02-03*
