# gaz

## What This Is

A unified Go application framework that consolidates dependency injection, application lifecycle management, configuration, and observability into a single cohesive library. Provides type-safe generics-based DI, background workers, cron scheduling, and event bus for in-process pub/sub — all integrated with a consistent lifecycle model.

## Core Value

Simple, type-safe dependency injection with sane defaults — developers register providers and resolve dependencies without fighting configuration options.

## Current State

**Shipped:** v2.2 Test Coverage (2026-01-29)

The framework now provides:
- **DI Package** (`gaz/di`) — Standalone dependency injection with For[T](), Resolve[T]()
- **Config Package** (`gaz/config`) — Configuration management with Backend interface
- **Workers** — Background workers with lifecycle integration, panic recovery, circuit breaker
- **Cron** — Scheduled tasks wrapping robfig/cron with DI-aware jobs
- **EventBus** — Type-safe pub/sub with Publish[T]/Subscribe[T] generics
- **CLI Integration** — RegisterCobraFlags() exposes ConfigProvider flags to CLI
- **Lifecycle Auto-Detection** — Services implementing Starter/Stopper auto-detected
- **CLI Args Injection** — `gaz.GetArgs(container)` for positional args access
- **gaztest Package** — Test utilities with Builder API and auto-cleanup
- **Service Builder** — `service.New()` for pre-configured production services
- **Module System** — `NewModule(name).Provide().Flags().Build()` for bundled registrations

**Test Coverage:** 92.9% overall (exceeds 90% target)

**Codebase:** ~96,000 lines of Go (including examples)

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
- ✓ Delete deprecated APIs (NewApp, AppOption) — v2.0
- ✓ Remove reflection-based registration (ProvideSingleton, etc.) — v2.0
- ✓ Extract DI to `gaz/di` subpackage — v2.0
- ✓ Extract Config to `gaz/config` subpackage — v2.0
- ✓ Background workers with lifecycle integration — v2.0
- ✓ Cron/scheduled task support — v2.0
- ✓ EventBus with pub/sub pattern — v2.0
- ✓ Cobra CLI flag integration — v2.0
- ✓ Interface auto-detection for Starter/Stopper — v2.1
- ✓ CLI args injection (`gaz.GetArgs()`) — v2.1
- ✓ gaztest package with Builder API — v2.1
- ✓ Service builder for production apps — v2.1
- ✓ ModuleBuilder for bundled registrations — v2.1
- ✓ 90%+ test coverage — v2.2

### Active

(Ready for next milestone)

### Out of Scope

- Hierarchical scopes — complexity not worth it for current use cases
- Backward compatibility with dibx/gazx — clean break, fresh API
- HTTP server integration — keep framework transport-agnostic
- External message queues (Kafka, RabbitMQ) — EventBus is in-process only
- Distributed workers/cron — use asynq for distributed jobs

## Context

Shipped v2.2 on 2026-01-29.
Framework provides unified DI, Lifecycle, Config, Health, Logging, Workers, Cron, and EventBus.
All requirements verified across 5 milestones (v1.0 through v2.2).
Test coverage at 92.9% overall.

This is an extraction and redesign of two internal libraries:

**dibx** (Dependency Injection):
- Generic `Provider[T] = func(Injector)(T,error)` pattern
- Multiple service types: lazy, eager, transient, alias
- Lifecycle interfaces: Healthchecker, Shutdowner
- Struct tag injection via reflection

**gazx** (Application Framework):
- AppBuilder combining Cobra + dibx
- LifecycleManager for ordered start/stop
- WorkerGroupManager for background tasks
- EventBus for decoupled pub/sub
- HealthManager aggregating health checks

Pain points addressed:
- Too many configuration options and knobs → Convention over configuration
- Unclear when to use which service type → For[T]() fluent API
- Two packages to import and coordinate → Single unified gaz package

Target: Internal use first, open source viability later.

## Constraints

- **Language**: Go 1.21+ (generics, slog in stdlib)
- **Dependencies**: Minimize external deps; Cobra, Viper, robfig/cron, jpillora/backoff
- **API Surface**: Convention over configuration — sensible defaults, escape hatches when needed
- **Package Structure**: Core gaz package + subpackages (di, config, worker, cron, eventbus, health, gaztest, service)

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
| For[T]() as sole registration API | Type-safe, no reflection | ✓ Good (v2.0) |
| Renamed NewContainer() → New() | Idiomatic Go constructor | ✓ Good (v2.0) |
| Composed interfaces for Backend | Optional capabilities (Watcher, Writer, EnvBinder) | ✓ Good (v2.0) |
| Circuit breaker hand-rolled | Simple counter+window sufficient for workers | ✓ Good (v2.0) |
| Scheduler implements Worker | Unified lifecycle for cron and workers | ✓ Good (v2.0) |
| Async fire-and-forget EventBus | Non-blocking publish, buffered subscribers | ✓ Good (v2.0) |
| RegisterCobraFlags explicit | CLI flag visibility before Execute() | ✓ Good (v2.0) |
| Reflection for lifecycle interface check | Check both T and *T for Starter/Stopper | ✓ Good (v2.1) |
| gaztest uses t.Cleanup() | Automatic cleanup, no manual Stop() required | ✓ Good (v2.1) |
| Module flags to PersistentFlags | Available to all subcommands | ✓ Good (v2.1) |
| Health auto-registration via interface | Config implements HealthConfigProvider for opt-in | ✓ Good (v2.1) |

---
*Last updated: 2026-01-29 after v2.2 milestone complete*
