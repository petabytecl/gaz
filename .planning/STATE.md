# Project State

**Project:** gaz
**Version:** v4.1
**Core Value:** Simple, type-safe dependency injection with sane defaults

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-02)

**Core value:** Simple, type-safe dependency injection with sane defaults
**Current focus:** v4.1 Server & Transport Layer

## Current Position

- **Milestone:** v4.1 Server & Transport Layer
- **Phase:** 39 - Gateway Integration
- **Plan:** 2 of 3 in current phase
- **Status:** In progress
- **Last activity:** 2026-02-03 — Completed 39-02-PLAN.md

Progress: [███████░░░] 67% (Plan 39-02 complete, ready for 39-03)

## Milestones Shipped

| Version | Name | Phases | Plans | Shipped |
|---------|------|--------|-------|---------|
| v1.0 | MVP | 1-6 | 35 | 2026-01-26 |
| v1.1 | Security & Hardening | 7-10 | 12 | 2026-01-27 |
| v2.0 | Cleanup & Concurrency | 11-18 | 34 | 2026-01-29 |
| v2.1 | API Enhancement | 19-21 | 8 | 2026-01-29 |
| v2.2 | Test Coverage | 22 | 4 | 2026-01-29 |
| v3.0 | API Harmonization | 23-29 | 27 | 2026-02-01 |
| v3.1 | Performance & Stability | 30 | 2 | 2026-02-01 |
| v3.2 | Feature Maturity | 31 | 2 | 2026-02-01 |
| v4.0 | Dependency Reduction | 32-36 | 18 | 2026-02-02 |
| v4.1 (Partial) | Core Discovery | 37 | 2 | 2026-02-02 |

**Total:** 146 plans across 37+ phases

## Accumulated Context

### Decisions (Cumulative)

All key decisions documented in PROJECT.md Key Decisions table.

### v4.1 Decisions

- **Port Separation:** Running Gateway and gRPC on separate ports (e.g., 8080/9090) to avoid `cmux` complexity.
- **Auto-Discovery:** Gateway will use `di.List[GatewayEndpoint]` to find services rather than manual registration.
- **Implicit Collection:** Allowed Register to append duplicates instead of returning error.
- **Ambiguity Handling:** Resolve returns ErrAmbiguous if multiple services registered.
- **Plugin Pattern:** Use `gaz.ResolveAll` to discover services implementing an interface.
- **Group Resolution:** Use `gaz.ResolveGroup` for categorized discovery (e.g., "system" vs "user" plugins).
- **HTTP Default Port 8080:** Standard HTTP port for Gateway, separate from health (9090) and gRPC (50051).
- **ReadHeaderTimeout 5s:** Prevents slow loris attacks per security research.
- **Registrar interface:** Renamed from GatewayRegistrar to avoid stutter.
- **grpc.NewClient:** Used instead of deprecated grpc.Dial for loopback connection.

### Research Summary

See: .planning/research/SUMMARY.md

### Blockers/Concerns

None.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 001 | Do a full review of all the package. | 2026-02-02 | b215f5a | [001-full-review-code-quality-security-docs](./quick/001-full-review-code-quality-security-docs/) |
| 002 | Add tests to examples and refactor for coverage. | 2026-02-02 | 26a4106 | [002-add-tests-to-examples-coverage](./quick/002-add-tests-to-examples-coverage/) |
| 003 | Improve test coverage to >90%. | 2026-02-03 | 4f00dec | [003-improve-test-coverage-to-90](./quick/003-improve-test-coverage-to-90/) |
| 004 | Create v4.1 Milestone Requirements. | 2026-02-03 | 13ce1bb | [004-create-v4-1-milestone-requirements](./quick/004-create-v4-1-milestone-requirements/) |

### Roadmap Evolution

- v4.0 complete: All 4 external dependencies replaced with internal implementations
- Phase 36 added builtin health checks for common infrastructure
- Quick Task 002 ensured examples are tested and fixed EventBus bugs.
- Quick Task 003 improved total test coverage to >90%.
- Quick Task 004 defined specs for v4.1 (HTTP/gRPC/Gateway).
- Roadmap v4.1 created with 4 phases (37-40).
- Phase 37 complete (Discovery).
- Phase 38 complete (Transport Foundations).
- Plan 38-01 added gRPC server with interceptors.
- Plan 38-02 added HTTP server with timeout protection.
- Plan 38-03 added unified server module and comprehensive tests.
- Phase 38.1 inserted after Phase 38: gRPC and HTTP servers should register flags to pass the port and other settings via CLI flags (URGENT)
- Phase 38.1 complete: NewModuleWithFlags() adds CLI flag support for server configuration
- Phase 39 started: Gateway Integration
- Plan 39-01 added core gateway package (config, headers, gateway, errors)
- Plan 39-02 added Gateway DI module with options and CLI flags

### Pending Todos

0 todo(s) in `.planning/todos/pending/`

## Session Continuity

Last session: 2026-02-03
Stopped at: Completed 39-02-PLAN.md
Resume with: 39-03-PLAN.md (Comprehensive tests)

---

*For detailed milestone history, see `.planning/MILESTONES.md`*
*For archived milestone details, see `.planning/milestones/`*
