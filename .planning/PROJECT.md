# gaz

## What This Is

A unified Go application framework that consolidates dependency injection, application lifecycle management, configuration, and observability into a single cohesive library. Extracts and simplifies the best ideas from internal dibx/gazx libraries into a standalone, potentially open-sourceable package with a convention-over-configuration API.

## Core Value

Simple, type-safe dependency injection with sane defaults — developers register providers and resolve dependencies without fighting configuration options.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Type-safe generic DI container (no interface{} casting)
- [ ] Convention-over-configuration defaults (lazy singletons, minimal options)
- [ ] Flat scope model (no hierarchical scopes)
- [ ] Struct tag injection (`gaz:"inject"`)
- [ ] App builder with Cobra integration
- [ ] Deterministic startup/shutdown (topological ordering)
- [ ] Signal handling (graceful shutdown on SIGTERM/SIGINT)
- [ ] Health check subsystem (readiness/liveness probes)
- [ ] Multi-source config management (files, env vars, flags)
- [ ] Config binding to typed structs
- [ ] slog integration (logger via DI, context propagation)
- [ ] Request-scoped logging with trace IDs

### Out of Scope

- Hierarchical scopes — complexity not worth it for current use cases
- Backward compatibility with dibx/gazx — clean break, fresh API
- Workers/EventBus in v1 — defer to v2 after core is stable
- HTTP server integration — keep framework transport-agnostic

## Context

This is an extraction and redesign of two internal libraries:

**dibx** (Dependency Injection):
- Generic `Provider[T] = func(Injector)(T,error)` pattern
- Multiple service types: lazy, eager, transient, alias
- Hierarchical scopes (being dropped)
- Lifecycle interfaces: Healthchecker, Shutdowner
- Struct tag injection via reflection

**gazx** (Application Framework):
- AppBuilder combining Cobra + dibx
- LifecycleManager for ordered start/stop
- WorkerGroupManager for background tasks
- EventBus for decoupled pub/sub
- HealthManager aggregating health checks

Pain points being addressed:
- Too many configuration options and knobs
- Unclear when to use which service type
- Scope complexity rarely needed
- Two packages to import and coordinate

Target: Internal use first, open source viability later.

## Constraints

- **Language**: Go 1.21+ (generics, slog in stdlib)
- **Dependencies**: Minimize external deps; Cobra required, consider koanf vs viper for config
- **API Surface**: Convention over configuration — sensible defaults, escape hatches when needed
- **Package Structure**: Core gaz package + optional subpackages (health, config, log)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Drop hierarchical scopes | Not used in practice, adds significant complexity | — Pending |
| Clean break from dibx/gazx API | Enables ideal API design without legacy constraints | — Pending |
| Core + subpackages structure | Import only what you need, clear boundaries | — Pending |
| slog over third-party loggers | Stdlib, sufficient for structured logging | — Pending |

---
*Last updated: 2026-01-26 after initialization*
