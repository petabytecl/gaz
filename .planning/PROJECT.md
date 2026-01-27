# gaz

## What This Is

A unified Go application framework that consolidates dependency injection, application lifecycle management, configuration, and observability into a single cohesive library. Extracts and simplifies the best ideas from internal dibx/gazx libraries into a standalone, potentially open-sourceable package with a convention-over-configuration API.

## Core Value

Simple, type-safe dependency injection with sane defaults — developers register providers and resolve dependencies without fighting configuration options.

## Requirements

### Validated

- ✓ Type-safe generic DI container (no interface{} casting) — v1.0
- ✓ Convention-over-configuration defaults (lazy singletons, minimal options) — v1.0
- ✓ Flat scope model (no hierarchical scopes) — v1.0
- ✓ Struct tag injection (`gaz:"inject"`) — v1.0
- ✓ App builder with Cobra integration — v1.0
- ✓ Deterministic startup/shutdown (topological ordering) — v1.0
- ✓ Signal handling (graceful shutdown on SIGTERM/SIGINT) — v1.0
- ✓ Multi-source config management (files, env vars, flags) — v1.0
- ✓ Config binding to typed structs — v1.0
- ✓ Health check subsystem (readiness/liveness probes) — v1.0
- ✓ slog integration (logger via DI, context propagation) — v1.0

### Active

- [ ] Request-scoped logging with trace IDs

### Out of Scope

- Hierarchical scopes — complexity not worth it for current use cases
- Backward compatibility with dibx/gazx — clean break, fresh API
- Workers/EventBus in v1 — defer to v2 after core is stable
- HTTP server integration — keep framework transport-agnostic

## Context

Shipped v1.0 MVP on 2026-01-26.
Framework consolidated DI, Lifecycle, Config, Health, and Logging into a single cohesive package.
Core DI container verified with 100% test coverage.

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
| Drop hierarchical scopes | Not used in practice, adds significant complexity | Done (v1.0) |
| Clean break from dibx/gazx API | Enables ideal API design without legacy constraints | Done (v1.0) |
| Core + subpackages structure | Import only what you need, clear boundaries | Done (v1.0) |
| slog over third-party loggers | Stdlib, sufficient for structured logging | Done (v1.0) |
| Use `spf13/viper` (instance mode) | Integrates natively with Cobra, avoid global state | Done (v1.0) |
| `Defaulter`/`Validator` interfaces | Prefer logic over tags for robust config lifecycle | Done (v1.0) |
| Config as Singleton Instance | Config object accessible via DI injection | Done (v1.0) |
| Auto-bind Env Vars | Reflection-based binding ensures Env overrides work without explicit keys | Done (v1.0) |
| Integrated logger into App struct | Reduces cognitive complexity, no separate LifecycleEngine | Done (v1.0) |

---
*Last updated: 2026-01-26 after v1.0 milestone completion*
