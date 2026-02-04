# Project State

**Project:** gaz
**Version:** v4.1
**Core Value:** Simple, type-safe dependency injection with sane defaults

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-02)

**Core value:** Simple, type-safe dependency injection with sane defaults
**Current focus:** v4.1 Server & Transport Layer

## Current Position

- **Milestone:** v4.3 Logger CLI Flags
- **Phase:** 43 - Logger CLI Flags
- **Plan:** 2 of 2 in current phase
- **Status:** Phase complete
- **Last activity:** 2026-02-04 — Completed 43-02-PLAN.md

Progress: [██████████] 100% (Phase 43 complete)

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
| v4.2 | Framework Ergonomics | 42 | 3 | 2026-02-04 |

**Total:** 152 plans across 42+ phases

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
- **gRPC health polling:** Use polling-based status sync (default 5s) for GRPCServer.
- **Empty service name:** Use "" for overall gRPC server health per protocol convention.
- **OTEL graceful degradation:** Return nil TracerProvider when collector unreachable.
- **ParentBased sampling:** Respect incoming trace context, sample 10% of root spans.
- **Health endpoint filtering:** Exclude health endpoints from tracing.
- **Logger fallback:** Use slog.Default() fallback making logger module optional across all server packages.
- **Health Server Signature:** Updated NewManagementServer to accept injected logger.
- **Health Logging:** Replaced direct stderr printing with structured logging in health server.
- **Atomic Handler:** Used atomic.Value for DynamicHandler to ensure zero-lock reads.
- **Naming Consistency:** Renamed ServiceRegistrar to Registrar in gRPC for consistency.
- **ConfigProvider Pattern:** Server modules use standard `Config` structs with `Flags()` method and `gaz.NewModule` builder instead of functional options.
- **Unified Server Module:** `server.NewModule` bundles `gRPC` and `Gateway` (which includes HTTP server) for a complete stack.
- **Native Health Integration:** gRPC server now natively integrates health checks (enabled by default) via internal `healthAdapter`, removing need for `health.WithGRPC()`.
- **Public Health Types:** Exported `StatusUp` and related types from `health` package for better integration.

### v4.2 Decisions (Phase 42)

- **Deferred Flag Registration:** `App.Use` no longer applies flags immediately. Flags are stored and applied when `WithCobra` is attached, ensuring order independence.
- **Cobra Lifecycle Management:** `WithCobra` initializes `App.running` state and `stopCh` in `bootstrap`, ensuring `App.Stop()` works correctly for manual shutdown control even when running under Cobra hooks.
- **Config Defaults:** Removed manual Viper binding in examples favoring framework defaults.
- **Smart Defaults:** Implemented auto-discovery of local gRPC port in Gateway module to improve DX.

### v4.3 Decisions (Phase 43)

- **WithCobra as Option:** `WithCobra(cmd)` is now an Option passed to `gaz.New()` instead of a method, enabling flags to be available before Logger creation.
- **Deferred Logger Init:** Logger, EventBus, WorkerManager, Scheduler are nil until `Build()` is called.
- **Immediate Flag Application:** `AddFlagsFn` applies flags immediately if cobra command is attached, ensuring order-independence.
- **Default LoggerConfig:** LoggerConfig with Info/JSON defaults is set in `New()` for consistent defaults.
- **Early Return in doStop:** `doStop()` returns early if app was never built to prevent nil panics.
- **Logger Module Subpackage:** `logger/module` subpackage used instead of `logger/module.go` to avoid circular import between gaz and logger packages.

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
| 005 | v4.1 Milestone Consistency Review. | 2026-02-03 | 588ea59 | [005-v4-1-milestone-consistency-review](./quick/005-v4-1-milestone-consistency-review/) |
| 006 | Refactor server/module.go remove gaz import. | 2026-02-03 | c06f475 | [006-refactor-server-module-remove-gaz-import](./quick/006-refactor-server-module-remove-gaz-import/) |
| 007 | Run make lint and fix all problems | 2026-02-04 | b9dcff1 | [007-run-make-lint-and-fix-all-problems](./quick/007-run-make-lint-and-fix-all-problems/) |

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
- Plan 39-03 added comprehensive tests achieving 92% coverage
- Phase 39 complete: Gateway package fully implemented and tested
- Phase 40 started: Observability & Health
- Plan 40-01 added gRPC health server wrapper (GRPCServer) with polling-based status sync
- Plan 40-02 added OpenTelemetry TracerProvider with OTLP export and server instrumentation
- Plan 40-03 added PGX health check, auto gRPC health integration, and otel tests
- Phase 40 complete: Observability & Health fully implemented
- Phase 41 added: Refactor server module architecture and consistency
- Plan 41-01 complete: Standardized logger usage across server packages
- Plan 41-02 complete: Fixed Gateway handler race condition and standardized naming
- Plan 41-03 complete: Refactored server modules to use ConfigProvider pattern and simplified API
- Plan 41-04 complete: Integrated native health checks into gRPC server
- Phase 41 complete: Server module architecture refactored and standardized
- Phase 42 added: Refactor Framework Ergonomics
- Plan 42-01 complete: Deferred flag registration decoupled App.Use from Cobra
- Plan 42-02 complete: Update Cobra integration to apply deferred flags and provide default lifecycle management
- Plan 42-03 complete: Refactored examples and fixed server modules to respect default config flags
- Phase 43 started: Logger CLI Flags (format, level, output configuration via CLI)
- Plan 43-01 complete: Restructured App initialization to defer Logger/subsystems until Build()
- Plan 43-02 complete: Created logger/module subpackage with CLI flags (--log-level, --log-format, --log-output, --log-add-source)
- Phase 43 complete: Logger CLI flags fully implemented
- Phase 44 added: Config File CLI Flag (--config flag for config file path)

### Pending Todos

1 todo(s) in `.planning/todos/pending/`

## Session Continuity

Last session: 2026-02-04
Stopped at: Completed 43-02-PLAN.md (Phase 43 complete)
Resume with: Phase 44 (Config File CLI Flag)


---

*For detailed milestone history, see `.planning/MILESTONES.md`*
*For archived milestone details, see `.planning/milestones/`*
