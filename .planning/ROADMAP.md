# Roadmap: gaz v2.0 Cleanup & Concurrency

## Overview

This milestone cleans up deprecated code, extracts DI and Config into standalone packages, and adds concurrency primitives (Workers, Cron, EventBus) that integrate with the existing lifecycle system. The cleanup phase removes technical debt before building new features. Package extractions enable standalone usage of DI and Config without the full framework. Concurrency primitives leverage gaz's existing Starter/Stopper lifecycle for seamless integration.

## Milestones

- v1.0 MVP - Phases 1-6 (shipped 2026-01-26)
- v1.1 Security & Hardening - Phases 7-10 (shipped 2026-01-27)
- **v2.0 Cleanup & Concurrency** - Phases 11-16 (in progress)

## Phases

- [x] **Phase 11: Cleanup** - Remove deprecated APIs and update examples/tests
- [x] **Phase 12: DI Package** - Extract DI into standalone gaz/di package
- [ ] **Phase 13: Config Package** - Extract Config into standalone gaz/config package
- [ ] **Phase 14: Workers** - Background workers with lifecycle integration
- [ ] **Phase 15: Cron** - Scheduled tasks wrapping robfig/cron
- [ ] **Phase 16: EventBus** - Type-safe in-process pub/sub

## Phase Details

### Phase 11: Cleanup

**Goal**: Remove all deprecated code and update all examples/tests to use generic fluent API.
**Depends on**: Nothing (first phase of v2.0)
**Requirements**: CLN-01, CLN-02, CLN-03, CLN-04, CLN-05, CLN-06, CLN-07, CLN-08, CLN-09, CLN-10, CLN-11, CLN-12
**Success Criteria** (what must be TRUE):
  1. `NewApp()` function and `AppOption` type do not exist in codebase
  2. All `App.Provide*` methods (Singleton, Transient, Eager, Instance) are removed
  3. All reflection-based helpers (`registerProvider`, `registerInstance`, service wrappers) are removed
  4. All example applications compile and use `For[T]()` registration pattern
  5. All tests pass and use `For[T]()` registration pattern
**Plans**: 2 plans

Plans:
- [x] 11-01-PLAN.md — Remove deprecated APIs and migrate tests to For[T]()
- [x] 11-02-PLAN.md — Rewrite examples and update documentation

---

### Phase 12: DI Package

**Goal**: Extract DI into `gaz/di` subpackage that works standalone without gaz App.
**Depends on**: Phase 11 (clean codebase required before extraction)
**Requirements**: DI-01, DI-02, DI-03, DI-04, DI-05, DI-06, DI-07, DI-08, DI-09, DI-10
**Success Criteria** (what must be TRUE):
  1. `gaz/di` package exists and exports `Container`, `For[T]()`, `Resolve[T]()`
  2. DI package works standalone (can create container, register, resolve without gaz App)
  3. Root `gaz` package re-exports DI types for backward compatibility
  4. All existing tests pass with updated imports
**Plans**: 4 plans

Plans:
- [x] 12-01-PLAN.md — Create di package with core DI types (Container, For, Resolve, accessor methods)
- [x] 12-02-PLAN.md — Add introspection APIs and backward compatibility wrappers
- [x] 12-03-PLAN.md — Update App to use di.Container, remove redundant root files
- [x] 12-04-PLAN.md — Create di package tests and update root package tests

---

### Phase 13: Config Package

**Goal**: Extract Config into `gaz/config` subpackage with Backend interface abstracting viper.
**Depends on**: Phase 12 (DI patterns established first)
**Requirements**: CFG-01, CFG-02, CFG-03, CFG-04, CFG-05, CFG-06, CFG-07, CFG-08, CFG-09
**Success Criteria** (what must be TRUE):
  1. `gaz/config` package exists and exports `Manager`, `Defaulter`, `Validator`
  2. `Backend` interface abstracts the viper dependency
  3. Config package works standalone (can load config without gaz App)
  4. Root `gaz` package integrates via `Backend` interface
**Plans**: 4 plans

Plans:
- [x] 13-01-PLAN.md — Create config package with Backend interfaces and ViperBackend
- [ ] 13-02-PLAN.md — Create Manager, options, validation, and generic accessors
- [ ] 13-03-PLAN.md — Integrate config package with App and backward compatibility
- [ ] 13-04-PLAN.md — Create config package tests and verify all tests pass

---

### Phase 14: Workers

**Goal**: Add background worker support with lifecycle integration, graceful shutdown, and panic recovery.
**Depends on**: Phase 13 (package extractions complete)
**Requirements**: WRK-01, WRK-02, WRK-03, WRK-04, WRK-05, WRK-06, WRK-07, WRK-08
**Success Criteria** (what must be TRUE):
  1. `Worker` interface defined with `Run(ctx context.Context) error` method
  2. Workers auto-start on `app.Run()` and auto-stop on shutdown
  3. Workers gracefully handle context cancellation (no goroutine leaks)
  4. Panics in workers are recovered and logged (don't crash app)
  5. Workers have names visible in logs for debugging
**Plans**: TBD

Plans:
- [ ] 14-01: TBD
- [ ] 14-02: TBD

---

### Phase 15: Cron

**Goal**: Add scheduled task support wrapping robfig/cron with DI-aware jobs and graceful shutdown.
**Depends on**: Phase 14 (worker patterns established)
**Requirements**: CRN-01, CRN-02, CRN-03, CRN-04, CRN-05, CRN-06, CRN-07, CRN-08, CRN-09, CRN-10
**Success Criteria** (what must be TRUE):
  1. Scheduler supports standard cron expressions and predefined schedules (@hourly, @daily)
  2. Scheduled jobs auto-start with app and wait for running jobs on shutdown
  3. Jobs that panic are recovered and logged (don't crash app)
  4. Jobs can inject dependencies from container (DI-aware)
  5. Overlapping job runs are skipped by default (SkipIfStillRunning)
**Plans**: TBD

Plans:
- [ ] 15-01: TBD
- [ ] 15-02: TBD

---

### Phase 16: EventBus

**Goal**: Add type-safe in-process pub/sub with generics API and DI integration.
**Depends on**: Phase 14 (worker patterns may be used by subscribers)
**Requirements**: EVT-01, EVT-02, EVT-03, EVT-04, EVT-05, EVT-06, EVT-07, EVT-08
**Success Criteria** (what must be TRUE):
  1. `Publish[T]()` and `Subscribe[T]()` provide type-safe event handling
  2. Events delivered synchronously by default
  3. Async mode available with bounded buffer for non-blocking publish
  4. Subscribers can unsubscribe
  5. EventBus integrates with DI container (resolvable as dependency)
**Plans**: TBD

Plans:
- [ ] 16-01: TBD
- [ ] 16-02: TBD

---

## Progress

**Execution Order:** Phases execute sequentially: 11 -> 12 -> 13 -> 14 -> 15 -> 16

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 11. Cleanup | 2/2 | Complete | 2026-01-28 |
| 12. DI Package | 4/4 | Complete | 2026-01-28 |
| 13. Config Package | 2/4 | In progress | - |
| 14. Workers | 0/2 | Not started | - |
| 15. Cron | 0/2 | Not started | - |
| 16. EventBus | 0/2 | Not started | - |

---

*Roadmap created: 2026-01-27*
*For milestone history, see .planning/MILESTONES.md*
