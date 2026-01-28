# Project State

**Project:** gaz
**Core Value:** Simple, type-safe dependency injection with sane defaults

## Project Reference

See: .planning/PROJECT.md (updated 2026-01-27)

**Core value:** Simple, type-safe dependency injection with sane defaults
**Current focus:** v2.0 Cleanup & Concurrency - Phase 14 Complete

## Current Position

- **Phase:** 14.3 of 16 (Flag-Based Config Registration)
- **Plan:** 1 of 1 in current phase
- **Status:** Phase complete
- **Last activity:** 2026-01-28 — Completed 14.3-01-PLAN.md

Progress: [████████████████] 100% (17/17 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 17 (v2.0)
- Average duration: 13 min
- Total execution time: 3.6 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 11. Cleanup | 2/2 | 50 min | 25 min |
| 12. DI Package | 4/4 | 100 min | 25 min |
| 13. Config Package | 4/4 | 26 min | 7 min |
| 14. Workers | 4/4 | 14 min | 3.5 min |
| 14.1 Cleanup Re-exports | 2/2 | 6 min | 3 min |
| 14.3 Flag-Based Config | 1/1 | 1 min | 1 min |

**Previous Milestones:**
- v1.0 MVP: 35 plans, 1 day
- v1.1 Hardening: 12 plans, 2 days

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Generic fluent API (`For[T](c).Provider(...)`) is the sole registration API ✓ DONE
- Reflection-based registration removed (CLN-04 to CLN-09) ✓ DONE in 11-01
- registerInstance() and instanceServiceAny retained for internal use (WithConfig, Logger)
- CHANGELOG uses Keep a Changelog format with semver
- DI should work standalone without full gaz framework (DI-08) ✓ DONE in 12-01
- Order: Cleanup → DI → Config → Workers/Cron/EventBus
- Renamed NewContainer() → New() for idiomatic Go constructor (12-01)
- Exported ServiceWrapper interface for App integration (12-01)
- Error prefix changed from 'gaz:' to 'di:' for di package errors (12-01)
- Combined Task 2+3 in 12-02 due to type alias conflict (12-02)
- Exported Register(), HasService(), ResolveByName() for App access (12-02)
- gaz.Err* now alias di.Err* for errors.Is() compatibility (12-02)
- Composed interfaces: core Backend + optional Watcher/Writer/EnvBinder (13-01)
- ViperBackend in subpackage to isolate viper dependency (13-01)
- ErrConfigValidation with 'config:' prefix (not 'gaz:') (13-01)
- Backend injection via option - New() requires WithBackend to avoid import cycle (13-02)
- Internal interfaces for viper operations - avoids importing config/viper (13-02)
- ConfigManager kept as thin wrapper (not alias) to preserve Load() API (13-03)
- Mock backend for config package unit tests (13-04)
- Worker interface: Start/Stop/Name per CONTEXT.md (14-01)
- jpillora/backoff for restart delays with jitter (14-01)
- Supervisor is internal, Manager exported for App integration (14-02)
- Circuit breaker hand-rolled (counter+window) per RESEARCH.md (14-02)
- Workers discovered during Build() via Worker interface check (14-03)
- Workers start after Starter hooks, stop before Stopper hooks (14-03)
- Mock workers use channels for synchronization (14-04)
- DI error aliases kept as ergonomic conveniences (14.1-01)
- ErrConfigValidation removed - users use config.ErrConfigValidation (14.1-01)
- NewContainer kept without deprecation as permanent convenience API (14.1-01)
- Test files import config package directly for config options (14.1-02)
- doc.go shows explicit config package import pattern (14.1-02)

- ProviderValues resolved in main() AFTER Build(), not in constructor (14.3-01)
- All config keys under server namespace for consistency (14.3-01)

### Phase 14.3 Complete

Flag-Based Config Registration complete:

| Plan | Name | Status |
|------|------|--------|
| 14.3-01 | Rewrite config-loading example | ✅ Complete |

### Phase 14.1 Complete

Cleanup Re-exports complete:

| Plan | Name | Status |
|------|------|--------|
| 14.1-01 | Remove deprecated config re-exports | ✅ Complete |
| 14.1-02 | Update tests and documentation | ✅ Complete |

### Phase 14 Complete

Workers package complete with full test coverage:

| Plan | Name | Status |
|------|------|--------|
| 14-01 | Worker interface, options, backoff | ✅ Complete |
| 14-02 | WorkerManager and Supervisor | ✅ Complete |
| 14-03 | App integration | ✅ Complete |
| 14-04 | Tests and verification | ✅ Complete |

**Coverage achieved:**
- worker package: 92.1% (target: 70%)

### Roadmap Evolution

- Phase 14.1 inserted after Phase 14: Cleanup deprecated re-exports, keep only planned APIs (URGENT)
- Phase 14.2 inserted after Phase 14.1: Update all relevant documentation and examples (URGENT)
- Phase 14.3 inserted after Phase 14.2: Flag-based config registration - providers register configs via flags, fetch in constructor (URGENT)

### Phase 13 Complete

Config Package extraction complete:

| Plan | Name | Status |
|------|------|--------|
| 13-01 | Backend interfaces and ViperBackend | ✅ Complete |
| 13-02 | Manager, options, validation, accessors | ✅ Complete |
| 13-03 | App integration and backward compat | ✅ Complete |
| 13-04 | Tests and verify all tests pass | ✅ Complete |

**Coverage achieved:**
- config package: 78.5% (target: 70%)
- config/viper package: 87.5% (target: 60%)

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-01-28T22:59:26Z
Stopped at: Completed 14.3-01-PLAN.md (Rewrite config-loading example)
Resume file: None (Phase 14.3 complete)

---

*For detailed milestone history, see `.planning/MILESTONES.md`*
*For archived roadmaps, see `.planning/milestones/`*
