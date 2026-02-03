# Quick Task 004 Summary: Create v4.1 Milestone Requirements

**Objective:** Capture requirements for v4.1 milestone (HTTP/gRPC/Gateway support) in a structured specification document.

## Outcome
Created `specs/v4.1-requirements.md` detailing the specifications for the upcoming v4.1 milestone.

## Key Decisions
- **Gateway Pattern:** Adapted the `_tmp_trust/grpcx/gateway.go` pattern (separate ports for HTTP/gRPC, `Service` interface for dynamic registration).
- **Standard Lib:** Chose `net/http` + Go 1.22 `ServeMux` over external routers like `gin` or `echo` to align with `gaz`'s "sane defaults" philosophy.
- **Port Separation:** Explicit separation of HTTP and gRPC ports (e.g., 8081/8082) to avoid protocol multiplexing complexity.
- **Infrastructure:** Included requirements for `pgx` support in health checks.

## Key Files
- `specs/v4.1-requirements.md`: The single source of truth for v4.1 implementation.

## Metrics
- **Duration:** < 5 minutes
- **Completed:** 2026-02-03
