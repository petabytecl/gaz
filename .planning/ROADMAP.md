# Roadmap: gaz v3.0 API Harmonization

## Overview

The v3.0 API Harmonization milestone transforms gaz's API patterns to align with industry standards (uber-go/fx, google/wire, samber/do). This is a clean-break refactoring: interface-based lifecycle management, module factory functions, config struct unmarshaling, and consolidated error handling. Seven phases take us from establishing conventions through implementation to polished documentation.

## Milestones

- ✅ **v1.0 MVP** - Phases 1-6 (shipped 2026-01-26)
- ✅ **v1.1 Security & Hardening** - Phases 7-10 (shipped 2026-01-27)
- ✅ **v2.0 Cleanup & Concurrency** - Phases 11-18 (shipped 2026-01-29)
- ✅ **v2.1 API Enhancement** - Phases 19-21 (shipped 2026-01-29)
- ✅ **v2.2 Test Coverage** - Phase 22 (shipped 2026-01-29)
- ✅ **v3.0 API Harmonization** - Phases 23-29 (shipped 2026-02-01)
- ✅ **v3.1 Performance & Stability** - Phase 30 (shipped 2026-02-01)

## Phases

<details>
<summary>✅ v1.0 - v2.2 (Phases 1-22) - SHIPPED</summary>

See `.planning/milestones/` for archived phase details.

</details>

### ✅ v3.0 API Harmonization (Complete)

**Milestone Goal:** Harmonize gaz API with industry-standard patterns for lifecycle, modules, config, and errors.

- [x] **Phase 23: Foundation & Style Guide** - Establish naming conventions before API changes
- [x] **Phase 24: Lifecycle Interface Alignment** - Remove fluent hooks, unify Starter/Stopper patterns
- [x] **Phase 25: Configuration Harmonization** - Add struct-based config unmarshaling
- [x] **Phase 26: Module & Service Consolidation** - Merge service package, standardize NewModule()
- [x] **Phase 27: Error Standardization** - Consolidate and namespace all sentinel errors
- [x] **Phase 28: Testing Infrastructure** - Enhance gaztest and per-package helpers
- [x] **Phase 29: Documentation & Examples** - Complete user documentation and examples

## Phase Details

### Phase 23: Foundation & Style Guide
**Goal**: Naming conventions and API patterns documented for consistent implementation
**Depends on**: Nothing (first phase of v3.0)
**Requirements**: DOC-01
**Success Criteria** (what must be TRUE):
  1. STYLE.md (or equivalent) exists with API naming conventions
  2. Constructor patterns documented (New*() vs builders vs fluent)
  3. Error naming conventions defined (ErrSubsystemAction format)
  4. Module factory function pattern documented (NewModule() returns gaz.Module)
**Plans**: 1 plan

Plans:
- [x] 23-01-PLAN.md — Create STYLE.md with API naming conventions

### Phase 24: Lifecycle Interface Alignment
**Goal**: Unified interface-based lifecycle management across all service types
**Depends on**: Phase 23
**Requirements**: LIF-01, LIF-02, LIF-03 (LIF-03 skipped per user decision - no Adapt() helper)
**Success Criteria** (what must be TRUE):
  1. Services implementing Starter/Stopper are automatically wired without fluent hooks
  2. worker.Worker implementations receive context in OnStart/OnStop and return error
  3. Fluent OnStart/OnStop methods are removed from RegistrationBuilder API
**Plans**: 5 plans in 3 waves

Plans:
- [x] 24-01-PLAN.md — Migrate worker.Worker interface to OnStart(ctx)/OnStop(ctx) error
- [x] 24-02-PLAN.md — Remove fluent hooks from RegistrationBuilder (interface-only lifecycle)
- [x] 24-03-PLAN.md — Update cron.Scheduler and example workers to new interface
- [x] 24-04-PLAN.md — Migrate all remaining fluent hook usages to interfaces
- [x] 24-05-PLAN.md — Update documentation and verify full phase completion

### Phase 25: Configuration Harmonization
**Goal**: Struct-based config resolution via unmarshaling
**Depends on**: Phase 24
**Requirements**: CFG-01
**Success Criteria** (what must be TRUE):
  1. ProviderValues has Unmarshal(namespace, &target) method that populates config structs
  2. Existing LoadInto() pattern continues to work (no breaking change there)
  3. Config namespacing enables module isolation (each module's config is prefixed)
**Plans**: 2 plans in 2 waves

Plans:
- [x] 25-01-PLAN.md — Add ErrKeyNotFound sentinel, viper gaz tag methods, and ProviderValues Unmarshal/UnmarshalKey
- [x] 25-02-PLAN.md — Add gaz tag to validator, comprehensive tests for Unmarshal functionality

### Phase 26: Module & Service Consolidation
**Goal**: Simplified module system with consistent NewModule() patterns
**Depends on**: Phase 25
**Requirements**: MOD-01, MOD-02, MOD-03, MOD-04
**Success Criteria** (what must be TRUE):
  1. gaz.App provides all functionality previously in service.Builder
  2. gaz/service package is removed (import path no longer exists)
  3. Subsystem packages (worker, cron, health, eventbus, config) export NewModule()
  4. Import relationship between di and gaz packages is documented clearly
  5. All existing tests pass with consolidated module system
**Plans**: 6 plans in 4 waves

Plans:
- [x] 26-01-PLAN.md — Service Builder migration + removal (MOD-01, MOD-02)
- [x] 26-02-PLAN.md — Health NewModule() with functional options (MOD-03)
- [x] 26-03-PLAN.md — Worker & Cron NewModule() (MOD-03)
- [x] 26-04-PLAN.md — EventBus & Config NewModule() (MOD-03)
- [x] 26-05-PLAN.md — Documentation + final verification (MOD-04)
- [x] 26-06-PLAN.md — Gap closure: Fix return types to di.Module (MOD-03)

### Phase 27: Error Standardization
**Goal**: Predictable, contextual error handling with namespaced sentinels
**Depends on**: Phase 26
**Requirements**: ERR-01, ERR-02, ERR-03
**Success Criteria** (what must be TRUE):
  1. All sentinel errors are defined in gaz/errors.go (single source of truth)
  2. Error names include subsystem prefix (ErrDINotFound, ErrConfigNotFound, ErrWorkerStopped)
  3. All error wrapping uses consistent "pkg: context: %w" format
  4. errors.Is/As work correctly for all gaz error types
**Plans**: 4 plans in 3 waves

Plans:
- [x] 27-01-PLAN.md — Consolidate sentinel errors + typed errors in gaz/errors.go
- [x] 27-02-PLAN.md — Migrate DI package to gaz errors
- [x] 27-03-PLAN.md — Migrate Config package to gaz errors
- [x] 27-04-PLAN.md — Migrate Worker/Cron + final verification

### Phase 28: Testing Infrastructure
**Goal**: Comprehensive test support for v3 patterns
**Depends on**: Phase 27
**Requirements**: TST-01, TST-02, TST-03
**Success Criteria** (what must be TRUE):
  1. gaztest builder API fully supports v3 patterns (no deprecated methods remain)
  2. Each subsystem has testing.go with test helpers (health, worker, config, cron, eventbus)
  3. Testing guide documentation explains common testing patterns
  4. Example tests demonstrate all v3 patterns
**Plans**: 4 plans in 2 waves

Plans:
- [x] 28-01-PLAN.md — Enhance gaztest Builder API (WithModules, WithConfigMap, RequireResolve)
- [x] 28-02-PLAN.md — Per-subsystem testing.go for health, worker, cron
- [x] 28-03-PLAN.md — Per-subsystem testing.go for config, eventbus
- [x] 28-04-PLAN.md — Testing guide documentation and example tests

### Phase 29: Documentation & Examples
**Goal**: Complete user-facing documentation for v3
**Depends on**: Phase 28
**Requirements**: DOC-02, DOC-03
**Success Criteria** (what must be TRUE):
  1. README includes getting started guide for new users
  2. godoc examples exist for all major public APIs
  3. All example code uses v3 patterns exclusively
  4. Tutorials cover common use cases (DI setup, lifecycle, modules, config)
**Plans**: 5 plans in 2 waves

Plans:
- [x] 29-01-PLAN.md — Core Package Examples (di + config)
- [x] 29-02-PLAN.md — Health & EventBus Examples
- [x] 29-03-PLAN.md — Worker & Cron Examples
- [x] 29-04-PLAN.md — Tutorial Example Apps (background-workers, microservice)
- [x] 29-05-PLAN.md — Documentation Finalization (troubleshooting, v3 verification)

### ✅ v3.1 Performance & Stability (Complete)

**Milestone Goal:** Address critical performance and stability issues identified in GAZ_REVIEW.md.

- [x] **Phase 30: DI Performance & Stability** - Remove goroutine ID hack, fix config discovery side-effects

## Phase Details (v3.1)

### Phase 30: DI Performance & Stability
**Goal**: Remove runtime.Stack() hack for cycle detection and fix config discovery side-effects
**Depends on**: Phase 29 (v3.0 complete)
**Requirements**: PERF-01, STAB-01
**Success Criteria** (what must be TRUE):
  1. getGoroutineID() removed from di/container.go
  2. ResolveByName uses explicit chain []string parameter for cycle detection
  3. Provider signature accepts resolution context for chain propagation
  4. collectProviderConfigs no longer fully instantiates services
  5. All existing tests pass with new cycle detection approach
**Plans**: 2 plans in 1 wave

Plans:
- [x] 30-01-PLAN.md — Remove goroutine ID hack, use explicit chain parameter (PERF-01)
- [x] 30-02-PLAN.md — Fix config discovery to check type before instantiation (STAB-01)

## Progress

**Execution Order:** Phases 23 → 24 → 25 → 26 → 27 → 28 → 29

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 23. Foundation & Style Guide | v3.0 | 1/1 | Complete | 2026-01-30 |
| 24. Lifecycle Interface Alignment | v3.0 | 5/5 | Complete | 2026-01-30 |
| 25. Configuration Harmonization | v3.0 | 2/2 | Complete | 2026-01-30 |
| 26. Module & Service Consolidation | v3.0 | 6/6 | Complete | 2026-01-31 |
| 27. Error Standardization | v3.0 | 4/4 | Complete | 2026-01-31 |
| 28. Testing Infrastructure | v3.0 | 4/4 | Complete | 2026-02-01 |
| 29. Documentation & Examples | v3.0 | 5/5 | Complete | 2026-02-01 |
| 30. DI Performance & Stability | v3.1 | 2/2 | Complete | 2026-02-01 |

---
*Roadmap created: 2026-01-29*
*Milestone: v3.0 API Harmonization - COMPLETE*
*Milestone: v3.1 Performance & Stability - COMPLETE*
