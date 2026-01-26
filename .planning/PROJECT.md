# gaz

## What This Is

A unified Go application framework that consolidates dependency injection, application lifecycle management, configuration, and observability into a single cohesive library. Extracts and simplifies the best ideas from internal dibx/gazx libraries into a standalone, potentially open-sourceable package with a convention-over-configuration API.

## Core Value

Simple, type-safe dependency injection with sane defaults — developers register providers and resolve dependencies without fighting configuration options.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [x] Type-safe generic DI container (no interface{} casting)
- [x] Convention-over-configuration defaults (lazy singletons, minimal options)
- [x] Flat scope model (no hierarchical scopes)
- [x] Struct tag injection (`gaz:"inject"`)
- [x] App builder with Cobra integration
- [x] Deterministic startup/shutdown (topological ordering)
- [x] Signal handling (graceful shutdown on SIGTERM/SIGINT)
- [x] Multi-source config management (files, env vars, flags)
- [x] Config binding to typed structs
- [ ] Health check subsystem (readiness/liveness probes)
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
| Drop hierarchical scopes | Not used in practice, adds significant complexity | Done |
| Clean break from dibx/gazx API | Enables ideal API design without legacy constraints | Done |
| Core + subpackages structure | Import only what you need, clear boundaries | Done |
| slog over third-party loggers | Stdlib, sufficient for structured logging | Pending |
| Use `spf13/viper` (instance mode) | Integrates natively with Cobra, avoid global state | Done |
| `Defaulter`/`Validator` interfaces | Prefer logic over tags for robust config lifecycle | Done |
| Config as Singleton Instance | Config object accessible via DI injection | Done |
| Auto-bind Env Vars | Reflection-based binding ensures Env overrides work without explicit keys | Done |

---
*Last updated: 2026-01-26 after initialization*
