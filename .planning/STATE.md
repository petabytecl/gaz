# Project State

**Project:** gaz
**Core Value:** Simple, type-safe dependency injection with sane defaults

## Project Reference

See: .planning/PROJECT.md (updated 2026-01-29)

**Core value:** Simple, type-safe dependency injection with sane defaults
**Current focus:** v3.0 API Harmonization - Phase 29: Documentation & Examples

## Current Position

- **Phase:** 29 of 29 (Documentation & Examples)
- **Plan:** 3 of 5 in current phase
- **Status:** In progress
- **Last activity:** 2026-02-01 — Completed 29-03-PLAN.md (Worker & Cron Examples)

Progress: [█████████░] 94% (Phase 29: 3/5 plans complete)

## Milestones Shipped

| Version | Name | Phases | Plans | Shipped |
|---------|------|--------|-------|---------|
| v1.0 | MVP | 1-6 | 35 | 2026-01-26 |
| v1.1 | Security & Hardening | 7-10 | 12 | 2026-01-27 |
| v2.0 | Cleanup & Concurrency | 11-18 | 34 | 2026-01-29 |
| v2.1 | API Enhancement | 19-21 | 8 | 2026-01-29 |
| v2.2 | Test Coverage | 22 | 4 | 2026-01-29 |

**Total:** 99 plans across 24 phases

## v3.0 Phase Overview

| Phase | Name | Requirements |
|-------|------|--------------|
| 23 | Foundation & Style Guide | DOC-01 ✓ |
| 24 | Lifecycle Interface Alignment | LIF-01 ✓, LIF-02 ✓, LIF-03 skipped |
| 25 | Configuration Harmonization | CFG-01 ✓ |
| 26 | Module & Service Consolidation | MOD-01 ✓, MOD-02 ✓, MOD-03 ✓, MOD-04 ✓ |
| 27 | Error Standardization | ERR-01 ✓, ERR-02 ✓, ERR-03 ✓ (all via re-export pattern) |
| 28 | Testing Infrastructure | TST-01 ✓, TST-02 ✓, TST-03 ✓ |
| 29 | Documentation & Examples | DOC-02, DOC-03 |

## Accumulated Context

### Decisions (Cumulative)

All key decisions documented in PROJECT.md Key Decisions table.

**Phase 26-01 additions:**
- health module uses di package directly to break import cycle with gaz
- health.WithHealthChecks() removed - superseded by HealthConfigProvider pattern
- service package removed completely with no deprecation period (v3 clean break)

**Phase 26-02 additions:**
- di.Module interface added to di package (breaks import cycle for subsystem modules)
- gaz.App.UseDI() method added to accept di.Module from subsystems
- health.NewModule() returns di.Module, not gaz.Module (import cycle constraint)

**Phase 26-04 additions:**
- eventbus.NewModule() and config.NewModule() use di package (same import cycle issue)
- NewModule returns func(*di.Container) error, not gaz.Module interface
- All five subsystem packages now have consistent NewModule() API

**Phase 26-03 additions:**
- worker.NewModule() and cron.NewModule() use di package (consistent with health pattern)
- Subsystem modules use di.Container not gaz.Container to avoid import cycles

**Phase 26-05 additions:**
- di/doc.go now explains "When to Use di vs gaz" for new users
- All re-exported types listed in di package documentation

**Phase 26-06 additions (gap closure):**
- All 4 subsystem modules (worker, cron, eventbus, config) now return di.Module
- Use di.NewModuleFunc() wrapper matching health.NewModule() pattern
- MOD-03 fully satisfied: all NewModule() functions have consistent return type

**Phase 27-01 additions:**
- Consolidated 16 sentinel errors in errors.go with ErrSubsystemAction naming
- Typed errors (ResolutionError, LifecycleError, ValidationError) added
- Backward compat aliases point to di.Err* until migration complete

**Phase 27-02 additions:**
- Standardized di error messages to 'di: action' format
- gaz.ErrDI* re-export di.Err* instead of being independent sentinels
- Error wrapping uses 'di: context: %w' format consistently
- Pattern established: subsystem defines errors, gaz re-exports with ErrSubsystem* naming

**Phase 27-03 additions:**
- gaz.ErrConfig* re-export config.Err* for errors.Is compatibility
- ValidationError and FieldError are type aliases to config package types
- config/errors.go stays as canonical source (import cycle constraint)

**Phase 27-04 additions:**
- cron/errors.go created with ErrNotRunning as canonical sentinel
- gaz.ErrWorker* re-export worker.Err* for errors.Is compatibility
- gaz.ErrCronNotRunning re-exports cron.ErrNotRunning
- All four subsystems (di, config, worker, cron) now use consistent re-export pattern
- ERR-01/02/03 requirements fully satisfied via re-export architecture

**Phase 28-02 additions:**
- health.TestConfig() and NewTestConfig() for safe test defaults
- health.MockRegistrar with testify/mock for mocking Registrar interface
- worker.MockWorker and SimpleWorker for worker testing
- cron.MockJob, SimpleJob, and MockResolver for cron testing
- TestManager/TestScheduler factories with discard loggers
- All Require* assertion helpers use testing.TB and t.Helper()

**Phase 28-03 additions:**
- config.MapBackend: in-memory Backend implementation for testing
- config.TestManager() factory for creating test managers
- eventbus.TestBus() factory for creating test EventBus
- eventbus.TestSubscriber[T] with WaitFor synchronization for async testing
- All helpers use testing.TB and tb.Helper() for proper reporting

**Phase 28-01 additions:**
- gaztest.WithModules(m ...di.Module) for module registration in test apps
- gaztest.WithConfigMap(map[string]any) for config injection in tests
- gaztest.RequireResolve[T](tb, app) generic helper for type-safe resolution
- gaz.App.MergeConfigMap() method for config injection
- WithApp and WithModules are mutually exclusive (panic on both)

**Phase 28-04 additions:**
- gaztest/README.md testing guide with Quick Reference and patterns
- gaztest/examples_test.go with v3 pattern examples (WithModules, RequireResolve, subsystem helpers)
- gaztest/doc.go updated to reference v3 patterns
- Phase 28 complete: all testing infrastructure in place

**Phase 29-02 additions:**
- health/doc.go: Package-level documentation for health package
- health/example_test.go: 13 godoc examples for health package APIs
- eventbus/example_test.go: 14 godoc examples for eventbus package APIs
- DOC-03 (godoc examples) partially complete for health and eventbus

**Phase 29-03 additions:**
- worker/example_test.go: 13 godoc examples for worker package APIs
- cron/example_test.go: 14 godoc examples for cron package APIs
- DOC-03 (godoc examples) now covers health, eventbus, worker, cron

### Blockers/Concerns

None - Phase 29 in progress.

### Pending Todos

1 todo(s) in `.planning/todos/pending/`

## Session Continuity

Last session: 2026-02-01 04:02
Stopped at: Completed 29-03-PLAN.md (Worker & Cron Examples)
Resume file: None - ready for 29-04 (Tutorial Example Apps)

---

*For detailed milestone history, see `.planning/MILESTONES.md`*
*For archived milestone details, see `.planning/milestones/`*
