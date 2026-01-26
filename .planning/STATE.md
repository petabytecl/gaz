# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-01-26)

**Core value:** Simple, type-safe dependency injection with sane defaults
**Current focus:** Phase 3 - App Builder + Cobra (next)

## Current Position

Phase: 3 of 6 (App Builder + Cobra)
Plan: 3 of 4 in current phase
Status: In progress
Last activity: 2026-01-26 — Completed 03-03-PLAN.md

Progress: [█████████████████████░░░] 90% (of defined plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 20
- Average duration: 5 min
- Total execution time: 2.1 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1 | 6 | 28 min | 5 min |
| 1.1 | 3 | 11 min | 4 min |
| 1.2 | 1 | 5 min | 5 min |
| 2 | 4 | 39 min | 10 min |
| 2.1 | 5 | 36 min | 7 min |
| 3 | 3 | 23 min | 8 min |

**Recent Trend:**
- Last 5 plans: 02.1-05 (11 min), 03-01 (13 min), 03-02 (3 min), 03-03 (7 min)
- Trend: Variable (larger plans take longer)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Init]: Drop hierarchical scopes — flat scope model only (Singleton, Transient)
- [Init]: Clean break from dibx/gazx API — enables ideal design without legacy constraints
- [Init]: slog over third-party loggers — stdlib, sufficient for structured logging
- [01-01]: Package-level var for sentinel errors — enables errors.Is() compatibility
- [01-01]: TypeName uses reflect.TypeOf(&zero).Elem() — handles interface types correctly
- [01-01]: Container storage as map[string]any — flexibility for serviceWrapper in later plans
- [01-02]: Four service wrapper types — lazy, transient, eager, instance cover all DI lifecycle patterns
- [01-02]: getInstance() receives chain parameter — prepared for cycle detection in resolution
- [01-03]: For[T]() returns builder, terminal methods return error — clean separation between configuration and execution
- [01-03]: ProviderFunc for simple providers — convenience method for providers that cannot fail
- [01-04]: Per-goroutine chain tracking for cycle detection — providers calling Resolve[T]() participate in detection
- [01-04]: goroutine ID extracted from runtime.Stack() — enables per-goroutine resolution chain tracking
- [01-05]: Injection after provider returns — keeps provider code simple, injection is automatic
- [01-05]: instanceService skips injection — pre-built values already have dependencies
- [01-05]: Silent skip for non-struct pointers — allows injection to work seamlessly with any type
- [01-06]: Build() is idempotent — calling multiple times is safe
- [01-06]: Build() error includes service name — enables debugging of which eager service failed
- [01.1-01]: Use suite.Suite for all test files — consistent pattern across codebase
- [01.1-01]: require for critical assertions, assert for value checks — follows testify best practices
- [01.1-03]: Use assert.Same for pointer equality, assert.ErrorIs for sentinel errors
- [01.2-01]: Enforced 90% code coverage threshold in Makefile
- [01.2-01]: Used golangci-lint-action in CI for caching and speed
- [01.2-01]: Included goimports in fmt target for import management
- [02-01]: Used separate graphMu RWMutex for granular locking
- [02-01]: Return deep copy from getGraph() for thread safety
- [02-02]: RegistrationBuilder stores hooks as generic wrappers — type safety at API boundary, flexibility internally
- [02-02]: Lazy singletons only execute hooks if instantiated — avoids unnecessary startup cost
- [02-02]: Transient services ignore hooks — avoids resource leaks for untracked instances
- [02-03]: Used Kahn's algorithm for topological sorting to support parallel startup layers
- [02-03]: Filtered out services without hooks to optimize startup/shutdown process
- [02-03]: Sorted layers alphabetically for deterministic behavior
- [02-04]: App.Run blocks until Stop() called or Signal received
- [02-04]: Stop() can be called externally to initiate shutdown
- [02-04]: Fixed startup order bug by ensuring all services are initialized in counts
- [02-04]: Fixed Build dependency tracking by using resolveByName
- [02.1-01]: Disabled testpackage linter — tests need internal package access
- [02.1-01]: Path-based exclusions over nolint comments — cleaner code
- [02.1-03]: Wrap Starter/Stopper interface errors with service name for debugging
- [02.1-03]: Configure exhaustive linter with default-signifies-exhaustive: true
- [02.1-02]: Extract magic numbers to named constants (defaultShutdownTimeout, decimalBase)
- [02.1-02]: Use errors.Join for multi-error aggregation instead of fmt.Errorf
- [02.1-02]: Check type assertions with ok pattern for defensive programming
- [02.1-04]: Use s.Require().Error/NoError/ErrorIs for error assertions to stop test on failure
- [02.1-04]: Rename unused closure parameters to _ for clarity
- [02.1-04]: Convert package-level assert.X(s.T()) to suite methods s.X()
- [02.1-05]: Use resolveErr naming in provider closures to avoid shadowing
- [02.1-05]: Use nolint comments for intentionally unused test fields
- [02.1-05]: Use Go 1.22+ integer range loops (for i := range n)
- [03-01]: Use reflection for provider type extraction in fluent API
- [03-01]: Non-generic *Any service wrappers for reflection-based registration
- [03-01]: Panic on late registration (after Build()) - programming error not runtime
- [03-01]: Build() is idempotent (safe to call multiple times)
- [03-02]: Module accepts func(*Container) error registration functions — enables For[T]() API in modules
- [03-02]: Empty modules are valid — allows declaring module names before adding providers
- [03-02]: Panic on late module registration (after Build()) — consistent with fluent API pattern

- [03-03]: Preserve existing Cobra hooks via chaining — don't replace, chain with original
- [03-03]: Stop() works without Run() for Cobra integration — Cobra uses Start/Stop directly
- [03-03]: Start() auto-builds if not already built — convenience for users

### Pending Todos

None.

### Blockers/Concerns

None yet.

### Roadmap Evolution

- Phase 1.1 inserted after Phase 1: update test framework for testify (URGENT)
- Phase 1.2 inserted after Phase 1: create makefile for testing, coverage, formatting, linting (URGENT)
- Phase 2.1 inserted after Phase 2: improve code quality validating the linter and the new config (URGENT)

## Session Continuity

Last session: 2026-01-26T21:46:30Z
Stopped at: Completed 03-03-PLAN.md
Resume file: None
