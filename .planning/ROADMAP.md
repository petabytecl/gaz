# Roadmap: gaz

## Overview

gaz delivers a unified Go application framework through 6 phases: starting with a type-safe generic DI container as the foundation, adding lifecycle management for deterministic startup/shutdown, then building the fluent App API with Cobra integration. Optional production-ready subsystems (config, health checks, logging) complete the framework. Phases 1-3 are sequential (dependencies); phases 4-6 can parallelize.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

- [x] **Phase 1: Core DI Container** - Type-safe dependency injection with generics
- [x] **Phase 1.1: Update Test Framework for Testify** - Migrate to testify assertions (INSERTED)
- [x] **Phase 1.2: Create Makefile** - Testing, coverage, formatting, linting (INSERTED)
- [x] **Phase 2: Lifecycle Management** - Deterministic startup/shutdown with hooks
- [x] **Phase 2.1: Improve Code Quality** - Validate linter config (INSERTED)
- [x] **Phase 3: App Builder + Cobra** - Fluent API and CLI integration
- [x] **Phase 4: Config System** - Multi-source configuration loading
- [x] **Phase 4.1: Refactor configuration** - Urgent work (INSERTED)
- [x] **Phase 5: Health Checks** - Readiness/liveness probes for production
- [ ] **Phase 6: Logging (slog)** - Structured logging with context propagation

## Phase Details

### Phase 1: Core DI Container
**Goal**: Developers can register and resolve dependencies with type-safe generics
**Depends on**: Nothing (first phase)
**Requirements**: DI-01, DI-02, DI-03, DI-04, DI-05, DI-06, DI-07, DI-08, DI-09
**Success Criteria** (what must be TRUE):
  1. Developer can register a provider with `Register[T](provider)` and resolve with `Resolve[T]()`
  2. Services instantiate lazily on first resolution by default; eager services instantiate at startup
  3. Errors from providers propagate through the dependency chain with clear context
  4. Developer can register multiple named implementations of the same type and resolve by name
  5. Developer can inject dependencies into struct fields tagged with `gaz:"inject"`
**Plans**: 6 plans

Plans:
- [x] 01-01-PLAN.md — Foundation (errors, types, container) ✓
- [x] 01-02-PLAN.md — Service wrappers (lazy, transient, eager, instance) ✓
- [x] 01-03-PLAN.md — Registration API (For[T], fluent builder) ✓
- [x] 01-04-PLAN.md — Resolution & cycle detection (Resolve[T]) ✓
- [x] 01-05-PLAN.md — Struct tag injection (gaz:"inject") ✓
- [x] 01-06-PLAN.md — Build phase & integration tests ✓

### Phase 1.1: Update Test Framework for Testify (INSERTED)
**Goal:** All 6 test files migrated to testify suite pattern with require/assert assertions
**Depends on**: Phase 1
**Plans:** 3 plans

Plans:
- [x] 01.1-01-PLAN.md — Setup testify + migrate types/registration/service tests ✓
- [x] 01.1-02-PLAN.md — Migrate inject and resolution tests ✓
- [x] 01.1-03-PLAN.md — Migrate container tests + final verification ✓

**Details:**
Migrate test framework to use https://github.com/stretchr/testify for improved assertions and test utilities.

### Phase 1.2: Create Makefile (INSERTED)
**Goal:** Development workflow automation with Makefile targets and GitHub Actions CI
**Depends on**: Phase 1
**Plans:** 1 plan

Plans:
- [x] 01.2-01-PLAN.md — Create Makefile + GitHub Actions CI workflow ✓

**Details:**
Create Makefile with targets for testing, coverage, formatting, linting, and other development workflow tasks.

### Phase 2: Lifecycle Management
**Goal**: App startup and shutdown are deterministic and graceful
**Depends on**: Phase 1 (hooks register services in container)
**Requirements**: LIFE-01, LIFE-02, LIFE-03, LIFE-04, LIFE-05, LIFE-06, LIFE-07, LIFE-08
**Success Criteria** (what must be TRUE):
  1. Developer can register OnStart/OnStop hooks that execute during app lifecycle
  2. App shuts down gracefully when receiving SIGTERM or SIGINT signals
  3. Lifecycle hooks receive context for cancellation and respect timeout configuration
  4. Services start in topological order based on dependencies and stop in LIFO order
**Plans**: 4 plans

Plans:
- [x] 02-01-PLAN.md — Enable container to record dependency graph during resolution ✓
- [x] 02-02-PLAN.md — Define lifecycle interfaces and update builder API ✓
- [x] 02-03-PLAN.md — Implement lifecycle ordering logic (TDD) ✓
- [x] 02-04-PLAN.md — Implement gaz.App wrapper with run/stop/signals ✓

### Phase 2.1: Improve Code Quality (INSERTED)
**Goal:** All linter warnings resolved, `make lint` passes cleanly
**Depends on**: Phase 2
**Plans:** 5 plans

Plans:
- [x] 02.1-01-PLAN.md — Configure linter and auto-fix formatting issues ✓
- [x] 02.1-02-PLAN.md — Fix production code issues (app.go, container.go, registration.go) ✓
- [x] 02.1-03-PLAN.md — Fix production code issues (service.go, types.go, lifecycle_engine.go) ✓
- [x] 02.1-04-PLAN.md — Fix testifylint and unused-parameter in test files ✓
- [x] 02.1-05-PLAN.md — Fix errcheck, shadow, unused, intrange in test files ✓

**Details:**
Fix 162 remaining linter issues across production code and test files.

### Phase 3: App Builder + Cobra
**Goal**: Developers can build and run applications with a fluent API
**Depends on**: Phase 2 (builder orchestrates container + lifecycle)
**Requirements**: APP-01, APP-02, APP-03, APP-04, APP-05, APP-06
**Success Criteria** (what must be TRUE):
  1. Developer can create app with `gaz.New()` and start with `.Run()`
  2. Developer can add providers fluently with `.Provide()` method chain
  3. Developer can compose related services into modules via `.Module()`
  4. Developer can integrate app with cobra.Command for CLI subcommands
**Plans**: 4 plans

Plans:
- [x] 03-01-PLAN.md — Core App refactor (gaz.New(), provider methods, Build validation) ✓
- [x] 03-02-PLAN.md — Module composition (Module() method, duplicate detection) ✓
- [x] 03-03-PLAN.md — Cobra integration (WithCobra(), FromContext()) ✓
- [x] 03-04-PLAN.md — End-to-end integration tests ✓

### Phase 4: Config System
**Goal**: Applications load configuration from multiple sources
**Depends on**: Phase 3 (config integrates with App builder and Cobra flags)
**Requirements**: CONF-01, CONF-02, CONF-03
**Success Criteria** (what must be TRUE):
  1. Developer can load config from environment variables
  2. Developer can load config from files (YAML, JSON, TOML)
  3. Developer can load config from CLI flags with Cobra integration
**Plans**: 2 plans

Plans:
- [x] 04-01-PLAN.md — Core Config Infrastructure (Viper, Interfaces, App integration) ✓
- [x] 04-02-PLAN.md — Cobra Flag Binding & Profile Support ✓

### Phase 4.1: Refactor configuration (INSERTED)

**Goal:** Decouple configuration logic into ConfigManager with Functional Options
**Depends on:** Phase 4
**Plans:** 2 plans

Plans:
- [x] 04.1-01-PLAN.md — Implement ConfigManager and Options
- [x] 04.1-02-PLAN.md — Integrate ConfigManager into App and Cobra

**Details:**
Refactors the App struct to remove direct Viper dependency, introducing a dedicated ConfigManager and replacing the config struct options with the Functional Options pattern.

### Phase 5: Health Checks
**Goal**: Applications expose production-ready health endpoints
**Depends on**: Phase 3 (health checks register in container, execute in lifecycle)
**Requirements**: HLTH-01, HLTH-02, HLTH-03, HLTH-04, HLTH-05
**Success Criteria** (what must be TRUE):
  1. Developer can implement HealthChecker interface for custom health checks
  2. Framework distinguishes readiness probes (ready to serve) from liveness probes (still running)
  3. Health checkers registered in DI are auto-discovered and aggregated
  4. Health checks execute concurrently with configurable timeout
**Plans**: 3 plans

Plans:
- [x] 05-01-PLAN.md — Core Infrastructure (interfaces, registry, shutdown hook) ✓
- [x] 05-02-PLAN.md — Handlers & IETF Formatting (JSON output, status codes) ✓
- [x] 05-03-PLAN.md — Management Server (dedicated port, app integration) ✓
- [x] 05-04-SUMMARY.md — Fix linter issues (43 issues resolved) ✓

### Phase 6: Logging (slog)
**Goal**: Applications have structured logging with context propagation
**Depends on**: Phase 3 (logger registers in container, used by framework)
**Requirements**: LOG-01, LOG-02, LOG-03, LOG-04
**Success Criteria** (what must be TRUE):
  1. Framework provides pre-configured slog.Logger via DI
  2. Logger propagates through context.Context
  3. Framework logs its own events (startup, shutdown, errors) via slog
  4. Developer can provide custom slog.Handler for formatting
**Plans**: 4 plans

Plans:
- [ ] 06-01-PLAN.md — Core Infrastructure (slog, tint, ContextHandler)
- [ ] 06-02-PLAN.md — Context Middleware (RequestID, context helpers)
- [ ] 06-03-PLAN.md — Framework Integration (App, Lifecycle)
- [ ] 06-04-PLAN.md — Cleanup & Linting

## Progress

**Execution Order:**
Phases 1-3 sequential, phases 4-6 can parallelize after phase 3.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Core DI Container | 6/6 | Complete | 2026-01-26 |
| 1.1 Update Test Framework | 3/3 | Complete | 2026-01-26 |
| 1.2 Create Makefile | 1/1 | Complete | 2026-01-26 |
| 2. Lifecycle Management | 4/4 | Complete | 2026-01-26 |
| 2.1 Improve Code Quality | 5/5 | Complete | 2026-01-26 |
| 3. App Builder + Cobra | 4/4 | Complete | 2026-01-26 |
| 4. Config System | 2/2 | Complete | 2026-01-26 |
| 4.1 Refactor config | 2/2 | Complete | 2026-01-26 |
| 5. Health Checks | 3/3 | Complete | 2026-01-27 |
| 6. Logging (slog) | 0/? | Not started | - |

---
*Created: 2026-01-26*
*Last updated: 2026-01-26*
