# Roadmap: gaz v2.0 Cleanup & Concurrency

## Overview

This milestone cleans up deprecated code, extracts DI and Config into standalone packages, and adds concurrency primitives (Workers, Cron, EventBus) that integrate with the existing lifecycle system. The cleanup phase removes technical debt before building new features. Package extractions enable standalone usage of DI and Config without the full framework. Concurrency primitives leverage gaz's existing Starter/Stopper lifecycle for seamless integration.

## Milestones

- v1.0 MVP - Phases 1-6 (shipped 2026-01-26)
- v1.1 Security & Hardening - Phases 7-10 (shipped 2026-01-27)
- **v2.0 Cleanup & Concurrency** - Phases 11-18 (shipped 2026-01-29)

## Phases

- [x] **Phase 11: Cleanup** - Remove deprecated APIs and update examples/tests
- [x] **Phase 12: DI Package** - Extract DI into standalone gaz/di package
- [x] **Phase 13: Config Package** - Extract Config into standalone gaz/config package
- [x] **Phase 14: Workers** - Background workers with lifecycle integration
- [x] **Phase 14.1: Cleanup Re-exports** - Remove deprecated re-exports, keep only planned APIs (INSERTED)
- [x] **Phase 14.2: Update All Relevant Documentation and Examples** (INSERTED)
- [x] **Phase 14.3: Flag-Based Config Registration** - Config via flags, fetch in constructor (INSERTED)
- [x] **Phase 14.4: Config Flag and ProviderValues** - --config flag and ProviderValues in providers (INSERTED)
- [ ] **Phase 15: Cron** - Scheduled tasks wrapping robfig/cron
- [ ] **Phase 16: EventBus** - Type-safe in-process pub/sub
- [x] **Phase 17: Cobra CLI Flags** - Expose ConfigProvider flags to Cobra CLI
- [x] **Phase 18: System Info CLI Example** - Showcase DI, ConfigProvider, Workers, and Cobra integration

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
- [x] 13-02-PLAN.md — Create Manager, options, validation, and generic accessors
- [x] 13-03-PLAN.md — Integrate config package with App and backward compatibility
- [x] 13-04-PLAN.md — Create config package tests and verify all tests pass

---

### Phase 14: Workers

**Goal**: Add background worker support with lifecycle integration, graceful shutdown, and panic recovery.
**Depends on**: Phase 13 (package extractions complete)
**Requirements**: WRK-01, WRK-02, WRK-03, WRK-04, WRK-05, WRK-06, WRK-07, WRK-08
**Success Criteria** (what must be TRUE):
  1. `Worker` interface defined with `Start()`, `Stop()`, `Name() string` methods
  2. Workers auto-start on `app.Run()` and auto-stop on shutdown
  3. Workers gracefully handle context cancellation (no goroutine leaks)
  4. Panics in workers are recovered and logged (don't crash app)
  5. Workers have names visible in logs for debugging
**Plans**: 4 plans

Plans:
- [x] 14-01-PLAN.md — Worker interface, options, backoff configuration, jpillora/backoff dependency
- [x] 14-02-PLAN.md — WorkerManager and Supervisor with panic recovery and circuit breaker
- [x] 14-03-PLAN.md — App integration with auto-discovery and lifecycle
- [x] 14-04-PLAN.md — Tests and verify all tests pass

---

### Phase 14.1: Cleanup Deprecated Re-exports (INSERTED)

**Goal**: Remove backward compatibility re-exports from root gaz package, only keep methods that make sense and we plan to maintain long-term.
**Depends on**: Phase 14 (all package extractions complete)
**Plans**: 2 plans

Plans:
- [x] 14.1-01-PLAN.md — Remove deprecated config re-exports (delete options.go, config.go, update errors.go)
- [x] 14.1-02-PLAN.md — Update tests and documentation to use config.* imports

---

### Phase 14.2: Update All Relevant Documentation and Examples (INSERTED)

**Goal:** Update all documentation (README, godoc, examples) to use v2.0 APIs: For[T]() pattern, explicit config package imports, and ConfigProvider pattern.
**Depends on:** Phase 14.1
**Plans**: 4 plans

Plans:
- [x] 14.2-01-PLAN.md — Rewrite getting-started.md and concepts.md with For[T]() API
- [x] 14.2-02-PLAN.md — Rewrite configuration.md and validation.md with config package patterns
- [x] 14.2-03-PLAN.md — Update advanced.md, README.md (worker feature), and CHANGELOG.md (package additions)
- [x] 14.2-04-PLAN.md — Create package READMEs (di, config, worker) and update config-loading example README

**Details:**
All plans execute in Wave 1 (parallel) as they touch different files.

---

### Phase 14.3: Flag-Based Config Registration (INSERTED)

**Goal:** Change config pattern so providers/services register required configs via flags and fetch values in constructor before build. Remove generic struct loading pattern from examples.
**Depends on:** Phase 14.2
**Plans**: 1 plan

Plans:
- [x] 14.3-01-PLAN.md — Rewrite config-loading example to use ConfigProvider pattern

**Details:**
Current pattern (to be removed):
```go
cfg := &Config{}
app.WithConfig(cfg, config.WithName("config"), ...)
```

New pattern:
1. Providers/Services register required configs via flags
2. Providers/Services fetch values in constructor before build
3. Update examples/config-loading/main.go to demonstrate new pattern

---

### Phase 14.4: Config Flag and ProviderValues (INSERTED)

**Goal:** Add `WithConfigFile()` option for explicit config paths and enable ProviderValues injection inside provider functions.
**Depends on:** Phase 14.3
**Plans**: 1 plan

Plans:
- [x] 14.4-01-PLAN.md — Config file path support and early ProviderValues registration

**Details:**
1. `WithConfigFile(path string)` option for explicit config file path
2. `gaz.ProviderValues` available inside provider functions during Build()

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
**Plans**: 4 plans

Plans:
- [x] 15-01-PLAN.md — CronJob interface and package foundation
- [x] 15-02-PLAN.md — Scheduler and DI-aware job wrapper
- [x] 15-03-PLAN.md — App integration and lifecycle management
- [ ] 15-04-PLAN.md — Tests and verification

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

### Phase 17: Cobra CLI Flags

**Goal:** Expose ConfigProvider flags to Cobra CLI - auto-register provider config flags as cobra command flags for CLI override and --help visibility.
**Depends on:** Phase 16 (ignored - unrelated)
**Plans:** 2 plans

Plans:
- [x] 17-01-PLAN.md — Add FlagBinder interface and RegisterCobraFlags method
- [x] 17-02-PLAN.md — Comprehensive tests for flag registration and CLI override

**Details:**
- RegisterCobraFlags(cmd) method on App for explicit flag registration before Execute()
- FlagBinder interface for individual flag binding (BindPFlag wrapping viper)
- Key transformation: "server.host" -> "--server-host" for POSIX compliance
- Idempotent config operations (loadConfig, registerProviderValuesEarly, collectProviderConfigs)
- Full type support: string, int, bool, duration, float
- Viper binding with original dot-notation key for correct precedence

---

### Phase 18: System Info CLI Example

**Goal:** Create system info CLI example showcasing DI, ConfigProvider, Workers, and Cobra integration.
**Depends on:** Phase 17
**Plans:** 2 plans

Plans:
- [x] 18-01-PLAN.md — ConfigProvider and Collector with gopsutil integration
- [x] 18-02-PLAN.md — Worker, main.go with RegisterCobraFlags, and README

**Details:**
- ConfigProvider declares sysinfo.refresh, sysinfo.format, sysinfo.once flags
- Collector uses gopsutil/v4 for CPU, memory, disk, host info
- Worker refreshes data at configured interval with graceful shutdown
- RegisterCobraFlags called before Execute() for --help visibility
- One-shot mode (--sysinfo-once) for single display and exit
- Continuous mode runs worker until Ctrl+C

---

## Progress

**Execution Order:** Phases execute sequentially: 11 -> 12 -> 13 -> 14 -> 15 -> 16 -> 17 -> 18

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 11. Cleanup | 2/2 | Complete | 2026-01-28 |
| 12. DI Package | 4/4 | Complete | 2026-01-28 |
| 13. Config Package | 4/4 | Complete | 2026-01-28 |
| 14. Workers | 4/4 | Complete | 2026-01-28 |
| 14.1 Cleanup Re-exports | 2/2 | Complete | 2026-01-28 |
| 14.2 Update Docs/Examples | 4/4 | Complete | 2026-01-29 |
| 14.3 Flag-Based Config | 1/1 | Complete | 2026-01-28 |
| 14.4 Config Flag/ProviderValues | 1/1 | Complete | 2026-01-28 |
| 15. Cron | 1/4 | In progress | - |
| 16. EventBus | 0/2 | Not started | - |
| 17. Cobra CLI Flags | 2/2 | Complete | 2026-01-29 |
| 18. System Info CLI Example | 2/2 | Complete | 2026-01-29 |

---

*Roadmap created: 2026-01-27*
*For milestone history, see .planning/MILESTONES.md*
