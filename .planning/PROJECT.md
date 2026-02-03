# gaz

## What This Is

A unified Go application framework that consolidates dependency injection, application lifecycle management, configuration, and observability into a single cohesive library. Provides type-safe generics-based DI, background workers, cron scheduling, and event bus for in-process pub/sub — all integrated with a consistent lifecycle model.

## Core Value

Simple, type-safe dependency injection with sane defaults — developers register providers and resolve dependencies without fighting configuration options.

## Current Milestone: v4.1 Server & Transport Layer

**Goal:** Implement production-ready HTTP and gRPC server capabilities with a unified Gateway pattern.

**Target features:**
- HTTP Server (`net/http`, `http.ServeMux`, `Starter`/`Stopper`)
- gRPC Server (`google.golang.org/grpc`, Interceptors, Reflection)
- gRPC-Gateway (HTTP proxy to gRPC, dynamic registration)
- Infrastructure (PGX support in health checks, gRPC health checks)

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
- ✓ Internal backoff package with exponential backoff, jitter, context — v4.0
- ✓ Internal tint slog handler with TTY detection — v4.0
- ✓ Internal cron engine with 5-field parser, @descriptors, timezone support — v4.0
- ✓ Internal health package with parallel checks, IETF format — v4.0
- ✓ Builtin health checks (SQL, Redis, HTTP, TCP, DNS, Runtime, Disk) — v4.0
- ✓ Core Discovery (multi-binding, ResolveAll, ResolveGroup) — v4.1

### Active

- [ ] HTTP Server (Standard Library)
- [ ] gRPC Server (google.golang.org/grpc)
- [ ] gRPC-Gateway (Unified proxy)
- [ ] Database Health Checks (pgx)
- [ ] gRPC Health Checks

### Out of Scope

- Hierarchical scopes — complexity not worth it for current use cases
- Backward compatibility with dibx/gazx — clean break, fresh API
- External message queues (Kafka, RabbitMQ) — EventBus is in-process only
- Distributed workers/cron — use asynq for distributed jobs

## Context

Shipped v4.0 on 2026-02-02.
Framework now has minimal external dependencies (Cobra, Viper, gopsutil, valkey-go).
All critical infrastructure (backoff, logging, cron, health) uses internal implementations.
Test coverage at 91.7% overall.

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

- **Language**: Go 1.25+ (generics, slog in stdlib)
- **Dependencies**: Minimal external deps; Cobra and Viper only. For v4.1: net/http, grpc, pgx.
- **API Surface**: Convention over configuration — sensible defaults, escape hatches when needed
- **Package Structure**: Core gaz package + subpackages (di, config, worker, cron, eventbus, health, gaztest, service, backoff, logger/tint, transport/http, transport/grpc, transport/gateway)

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
| Internal backoff package | Full control, no external dep for retry logic | ✓ Good (v4.0) |
| Internal tint slog handler | Full control, no external dep for logging | ✓ Good (v4.0) |
| Internal cron engine | Full control, no external dep for scheduling | ✓ Good (v4.0) |
| Internal health package | Full control, IETF format built-in | ✓ Good (v4.0) |
| Builtin health checks | Production-ready checks for common infra | ✓ Good (v4.0) |
| Implicit Collection for DI | Multiple `For` calls append, `Replace` overwrites | ✓ Good (v4.1) |
| Discovery via Groups | `InGroup` tagging and `ResolveGroup` | ✓ Good (v4.1) |

---
*Last updated: 2026-02-02 after Phase 37 completion*
