# Project State

**Project:** gaz
**Core Value:** Simple, type-safe dependency injection with sane defaults

## Project Reference

See: .planning/PROJECT.md (updated 2026-01-27)

**Core value:** Simple, type-safe dependency injection with sane defaults
**Current focus:** v2.0 Cleanup & Concurrency - Phase 17 Complete, Phase 18 Added

## Current Position

- **Phase:** 16 of 18 (EventBus)
- **Plan:** 3 of 4 in current phase
- **Status:** In progress
- **Last activity:** 2026-01-29 — Completed 16-03-PLAN.md

Progress: [█████████████████████████████████░░] 97% (33/34 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 24 (v2.0)
- Average duration: 12 min
- Total execution time: 3.8 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 11. Cleanup | 2/2 | 50 min | 25 min |
| 12. DI Package | 4/4 | 100 min | 25 min |
| 13. Config Package | 4/4 | 26 min | 7 min |
| 14. Workers | 4/4 | 14 min | 3.5 min |
| 14.1 Cleanup Re-exports | 2/2 | 6 min | 3 min |
| 14.2 Update Documentation | 4/4 | 8 min | 2 min |
| 14.3 Flag-Based Config | 1/1 | 1 min | 1 min |
| 14.4 Config Flag/ProviderValues | 1/1 | 3 min | 3 min |
| 17. Cobra CLI Flags | 2/2 | 6 min | 3 min |
| 18. System Info CLI Example | 2/2 | 13 min | 6.5 min |
| 15. Cron | 4/4 | 16 min | 4 min |

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
- ProviderValues registered BEFORE collectProviderConfigs in Build() (14.4-01)
- WithConfigFile bypasses search paths when set (14.4-01)
- configFileSetter interface for backend abstraction (14.4-01)

- For[T]() is the only registration pattern shown in foundational docs (14.2-01)
- ConfigProvider pattern documented as primary config approach (14.2-02)
- Environment variable mapping updated to single underscore pattern (14.2-02)
- ConfigProvider shown as replacement for removed app.WithConfig() (14.2-03)
- Worker package added to README features (14.2-03)
- Package READMEs link back to main gaz README for pkg.go.dev (14.2-04)

- FlagBinder as exported interface for individual flag binding (17-01)
- Idempotency tracking via bool fields for config operations (17-01)
- Key transformation: server.host -> --server-host for POSIX compliance (17-01)

- testConfigProvider mock pattern for ConfigProvider testing (17-02)

- CronJob interface matches CONTEXT.md specification (Name, Schedule, Timeout, Run) (15-01)
- slog adapter adds component=cron for log correlation (15-01)
- Resolver interface abstracts container for cron package decoupling (15-02)
- Custom panic recovery (not cron.Recover) for slog + stack traces (15-02)
- Empty schedule string disables job gracefully (not an error) (15-02)
- Resolver interface uses []string opts to match Container signature (15-03)
- CronJobs discovered by checking svc.TypeName() == cron.CronJob type (15-03)
- Scheduler registered with WorkerManager only if jobs exist (15-03)

- Mock CronJob pattern with runFn callback for flexible test scenarios (15-04)
- countingResolver pattern for transient verification (15-04)

- Event interface requires EventName() for logging/debugging (16-01)
- Handler[T Event] is fire-and-forget with no error return (16-01)
- Subscription uses atomic counter ID, not UUID (16-01)
- Default event buffer size is 100 (16-01)
- unsubscriber interface pattern avoids circular dependency (16-01)
- Silent no-op for publishing to closed bus (16-02)
- Context cancellation support in Publish for graceful abort (16-02)
- Backpressure via blocking when subscriber buffer full (16-02)

- EventBus created in New() constructor with logger (16-03)
- EventBus registered as DI singleton via For[*eventbus.EventBus]().Instance() (16-03)
- EventBus registered with WorkerManager for lifecycle management (16-03)

### Phase 16 In Progress

EventBus type-safe pub/sub:

| Plan | Name | Status |
|------|------|--------|
| 16-01 | EventBus foundation (Event, Handler, Subscription, Options) | ✅ Complete |
| 16-02 | EventBus implementation | ✅ Complete |
| 16-03 | App integration | ✅ Complete |
| 16-04 | Tests and verification | Pending |

### Phase 15 Complete

Cron scheduled tasks:

| Plan | Name | Status |
|------|------|--------|
| 15-01 | CronJob interface and package foundation | ✅ Complete |
| 15-02 | Scheduler and DI-aware job wrapper | ✅ Complete |
| 15-03 | App integration and lifecycle management | ✅ Complete |
| 15-04 | Tests and verification | ✅ Complete |

**Coverage achieved:**
- cron package: 100% (target: 70%)

### Phase 18 Complete

System Info CLI Example:

| Plan | Name | Status |
|------|------|--------|
| 18-01 | ConfigProvider and Collector | ✅ Complete |
| 18-02 | Worker, main.go, README | ✅ Complete |

### Phase 17 Complete

Cobra CLI Flags:

| Plan | Name | Status |
|------|------|--------|
| 17-01 | FlagBinder interface and RegisterCobraFlags | ✅ Complete |
| 17-02 | Comprehensive tests | ✅ Complete |

### Phase 14.2 Complete

Update Documentation complete:

| Plan | Name | Status |
|------|------|--------|
| 14.2-01 | Foundational documentation rewrite | ✅ Complete |
| 14.2-02 | Configuration and validation docs | ✅ Complete |
| 14.2-03 | Advanced documentation | ✅ Complete |
| 14.2-04 | Standalone package READMEs | ✅ Complete |

### Phase 14.4 Complete

Config Flag and ProviderValues complete:

| Plan | Name | Status |
|------|------|--------|
| 14.4-01 | WithConfigFile and early ProviderValues | ✅ Complete |

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

- Phase 18 added: Create system info CLI example showcasing DI, ConfigProvider, Workers, and Cobra integration
- Phase 17 added: Expose ConfigProvider flags to Cobra CLI - auto-register provider config flags as cobra command flags for CLI override and --help visibility
- Phase 14.1 inserted after Phase 14: Cleanup deprecated re-exports, keep only planned APIs (URGENT)
- Phase 14.2 inserted after Phase 14.1: Update all relevant documentation and examples (URGENT)
- Phase 14.3 inserted after Phase 14.2: Flag-based config registration - providers register configs via flags, fetch in constructor (URGENT)
- Phase 14.4 inserted after Phase 14.3: Config flag and ProviderValues - default config via --config flag, ProviderValues available in providers (URGENT)

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

0 pending todo(s) in `.planning/todos/pending/`

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-01-29T05:19:29Z
Stopped at: Completed 16-03-PLAN.md
Resume file: None

---

*For detailed milestone history, see `.planning/MILESTONES.md`*
*For archived roadmaps, see `.planning/milestones/`*
