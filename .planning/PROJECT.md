# gaz

## What This Is

A unified Go application framework that consolidates dependency injection, application lifecycle management, configuration, and observability into a single cohesive library. Extracts and simplifies the best ideas from internal dibx/gazx libraries into a standalone, potentially open-sourceable package with a convention-over-configuration API.

## Core Value

Simple, type-safe dependency injection with sane defaults — developers register providers and resolve dependencies without fighting configuration options.

## Current Milestone: v2.0 Cleanup & Concurrency

**Goal:** Clean up codebase, extract DI to standalone package, add concurrency primitives.

**Target features:**
- Delete deprecated code (NewApp, AppOption, reflection-based registration)
- Extract DI to `gaz/di` for standalone use
- Background workers with graceful shutdown
- Worker pool for queued processing
- Cron/scheduled tasks
- In-app EventBus (pub/sub)

## Current State

**Shipped:** v1.1 Security & Hardening (2026-01-27)

The framework now provides production-grade robustness:
- Config validation at startup (struct tags, early exit)
- Shutdown hardening (timeout enforcement, blame logging)
- Provider config registration (service-level config)
- Comprehensive documentation and examples

**Codebase:** 11,319 lines of Go

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
- ✓ Config validation (struct tags, early exit) — v1.1
- ✓ Shutdown hardening (timeout enforcement, blame logging) — v1.1
- ✓ Provider config registration — v1.1
- ✓ Comprehensive documentation and examples — v1.1

### Active

- [ ] Delete deprecated APIs (NewApp, AppOption)
- [ ] Remove reflection-based registration (ProvideSingleton, etc.)
- [ ] Extract DI to `gaz/di` subpackage
- [ ] Background workers with lifecycle integration
- [ ] Worker pool for queued processing
- [ ] Cron/scheduled task support
- [ ] EventBus with pub/sub pattern

### Out of Scope

- Hierarchical scopes — complexity not worth it for current use cases
- Backward compatibility with dibx/gazx — clean break, fresh API
- HTTP server integration — keep framework transport-agnostic
- External message queues (Kafka, RabbitMQ) — EventBus is in-process only

## Context

Shipped v1.1 on 2026-01-27.
Framework now includes DI, Lifecycle, Config, Health, Logging, Validation, Shutdown Hardening, and Provider Config.
All v1.1 requirements verified with comprehensive tests (100% coverage).

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
| Drop hierarchical scopes | Not used in practice, adds significant complexity | ✓ Good (v1.0) |
| Clean break from dibx/gazx API | Enables ideal API design without legacy constraints | ✓ Good (v1.0) |
| Core + subpackages structure | Import only what you need, clear boundaries | ✓ Good (v1.0) |
| slog over third-party loggers | Stdlib, sufficient for structured logging | ✓ Good (v1.0) |
| Use `spf13/viper` (instance mode) | Integrates natively with Cobra, avoid global state | ✓ Good (v1.0) |
| `Defaulter`/`Validator` interfaces | Prefer logic over tags for robust config lifecycle | ✓ Good (v1.0) |
| Config as Singleton Instance | Config object accessible via DI injection | ✓ Good (v1.0) |
| Auto-bind Env Vars | Reflection-based binding ensures Env overrides work without explicit keys | ✓ Good (v1.0) |
| Integrated logger into App struct | Reduces cognitive complexity, no separate LifecycleEngine | ✓ Good (v1.0) |
| go-playground/validator for validation | Industry-standard, cross-field support | ✓ Good (v1.1) |
| Per-hook timeout with blame logging | Debugging hung shutdowns | ✓ Good (v1.1) |
| exitFunc global for test injection | Necessary for shutdown testing, has nolint | ✓ Good (v1.1) |
| Skip transient services in config collection | Avoid side effects during Build() | ✓ Good (v1.1) |

---
*Last updated: 2026-01-27 after v2.0 milestone started*
