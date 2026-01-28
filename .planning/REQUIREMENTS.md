# Requirements: gaz v2.0

**Defined:** 2026-01-27
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v2.0 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Cleanup

- [x] **CLN-01**: Delete deprecated `NewApp()` function
- [x] **CLN-02**: Delete deprecated `AppOption` type
- [x] **CLN-03**: Delete `withShutdownTimeoutLegacy()` helper
- [x] **CLN-04**: Remove `ProvideSingleton()` method from App
- [x] **CLN-05**: Remove `ProvideTransient()` method from App
- [x] **CLN-06**: Remove `ProvideEager()` method from App
- [x] **CLN-07**: Remove `ProvideInstance()` method from App
- [x] **CLN-08**: Remove `registerProvider()` reflection-based helper
- [x] **CLN-09**: Remove `registerInstance()` reflection-based helper (public API removed; internal helper retained for WithConfig)
- [x] **CLN-10**: Remove non-generic service wrappers (`lazySingletonAny`, `transientServiceAny`, `eagerSingletonAny` removed; `instanceServiceAny` retained for internal use)
- [x] **CLN-11**: Update all examples to use generic fluent API
- [x] **CLN-12**: Update all tests to use generic fluent API

### DI Package Extraction

- [x] **DI-01**: Create `gaz/di` subpackage
- [x] **DI-02**: Move `Container` type to `gaz/di`
- [x] **DI-03**: Move `For[T]()` registration builder to `gaz/di`
- [x] **DI-04**: Move `Resolve[T]()` function to `gaz/di`
- [x] **DI-05**: Move service wrappers to `gaz/di` (internal)
- [x] **DI-06**: Move `TypeName[T]()` to `gaz/di`
- [x] **DI-07**: Move injection logic (`inject.go`) to `gaz/di`
- [x] **DI-08**: DI package works standalone without gaz App
- [x] **DI-09**: Root gaz package re-exports DI types for backward compatibility
- [x] **DI-10**: Update imports throughout codebase

### Config Package Extraction

- [ ] **CFG-01**: Create `gaz/config` subpackage
- [ ] **CFG-02**: Move `ConfigManager` to `gaz/config`
- [ ] **CFG-03**: Move `Defaulter`/`Validator` interfaces to `gaz/config`
- [ ] **CFG-04**: Move config options to `gaz/config`
- [ ] **CFG-05**: Define `Backend` interface to abstract viper
- [ ] **CFG-06**: Create `ViperBackend` implementing `Backend`
- [ ] **CFG-07**: Config package works standalone without gaz App
- [ ] **CFG-08**: Root gaz package integrates config via interface
- [ ] **CFG-09**: Update imports throughout codebase

### Workers

- [ ] **WRK-01**: Define `Worker` interface with `Run(ctx context.Context) error`
- [ ] **WRK-02**: Create `WorkerManager` for tracking multiple workers
- [ ] **WRK-03**: Workers integrate with lifecycle (auto-start/stop)
- [ ] **WRK-04**: Workers respect context cancellation for graceful shutdown
- [ ] **WRK-05**: Workers have panic recovery (don't crash app)
- [ ] **WRK-06**: Workers expose Done() channel for shutdown verification
- [ ] **WRK-07**: Workers integrate with slog for logging
- [ ] **WRK-08**: Workers have names for debugging

### Cron

- [ ] **CRN-01**: Create `Scheduler` type wrapping robfig/cron v3
- [ ] **CRN-02**: Support standard 5-field cron expressions
- [ ] **CRN-03**: Support predefined schedules (@hourly, @daily, etc.)
- [ ] **CRN-04**: Scheduler integrates with lifecycle (OnStart/OnStop)
- [ ] **CRN-05**: Scheduler waits for running jobs on shutdown
- [ ] **CRN-06**: Jobs have panic recovery by default
- [ ] **CRN-07**: Jobs are DI-aware (can inject dependencies from container)
- [ ] **CRN-08**: Jobs use SkipIfStillRunning by default
- [ ] **CRN-09**: Scheduler exposes health check (job status)
- [ ] **CRN-10**: Jobs have names for logging

### EventBus

- [ ] **EVT-01**: Create type-safe generics EventBus with `Publish[T]`/`Subscribe[T]`
- [ ] **EVT-02**: Synchronous delivery by default
- [ ] **EVT-03**: Async mode option for non-blocking publish
- [ ] **EVT-04**: Unsubscribe capability
- [ ] **EVT-05**: Bounded buffer with explicit size configuration
- [ ] **EVT-06**: Topic filtering (route events to matched handlers)
- [ ] **EVT-07**: Context propagation through events
- [ ] **EVT-08**: EventBus integrates with DI container

## v2.1 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Workers

- **WRK-P1**: Worker pools with fixed size
- **WRK-P2**: Submit tasks to queue
- **WRK-P3**: Bounded queue with configurable size
- **WRK-P4**: Graceful drain on shutdown
- **WRK-P5**: Restart policies with backoff for crashed workers

### Config

- **CFG-K1**: Replace viper with koanf backend
- **CFG-K2**: Better struct-based config with koanf

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Distributed workers | Use asynq for distributed jobs |
| Persistent job history | Adds storage dependency, not core functionality |
| Priority queues | Complex, defer to v2.1+ |
| External message queues (Kafka, RabbitMQ) | EventBus is in-process only |
| Metrics/Prometheus integration | Complex, phase-specific, defer to v2.1 |
| Wildcard event subscriptions | Complexity vs value tradeoff |
| Distributed cron | Requires leader election, out of scope |
| HTTP server integration | Keep framework transport-agnostic |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CLN-01 | Phase 11 | Complete |
| CLN-02 | Phase 11 | Complete |
| CLN-03 | Phase 11 | Complete |
| CLN-04 | Phase 11 | Complete |
| CLN-05 | Phase 11 | Complete |
| CLN-06 | Phase 11 | Complete |
| CLN-07 | Phase 11 | Complete |
| CLN-08 | Phase 11 | Complete |
| CLN-09 | Phase 11 | Complete |
| CLN-10 | Phase 11 | Complete |
| CLN-11 | Phase 11 | Complete |
| CLN-12 | Phase 11 | Complete |
| DI-01 | Phase 12 | Complete |
| DI-02 | Phase 12 | Complete |
| DI-03 | Phase 12 | Complete |
| DI-04 | Phase 12 | Complete |
| DI-05 | Phase 12 | Complete |
| DI-06 | Phase 12 | Complete |
| DI-07 | Phase 12 | Complete |
| DI-08 | Phase 12 | Complete |
| DI-09 | Phase 12 | Complete |
| DI-10 | Phase 12 | Complete |
| CFG-01 | Phase 13 | Pending |
| CFG-02 | Phase 13 | Pending |
| CFG-03 | Phase 13 | Pending |
| CFG-04 | Phase 13 | Pending |
| CFG-05 | Phase 13 | Pending |
| CFG-06 | Phase 13 | Pending |
| CFG-07 | Phase 13 | Pending |
| CFG-08 | Phase 13 | Pending |
| CFG-09 | Phase 13 | Pending |
| WRK-01 | Phase 14 | Pending |
| WRK-02 | Phase 14 | Pending |
| WRK-03 | Phase 14 | Pending |
| WRK-04 | Phase 14 | Pending |
| WRK-05 | Phase 14 | Pending |
| WRK-06 | Phase 14 | Pending |
| WRK-07 | Phase 14 | Pending |
| WRK-08 | Phase 14 | Pending |
| CRN-01 | Phase 15 | Pending |
| CRN-02 | Phase 15 | Pending |
| CRN-03 | Phase 15 | Pending |
| CRN-04 | Phase 15 | Pending |
| CRN-05 | Phase 15 | Pending |
| CRN-06 | Phase 15 | Pending |
| CRN-07 | Phase 15 | Pending |
| CRN-08 | Phase 15 | Pending |
| CRN-09 | Phase 15 | Pending |
| CRN-10 | Phase 15 | Pending |
| EVT-01 | Phase 16 | Pending |
| EVT-02 | Phase 16 | Pending |
| EVT-03 | Phase 16 | Pending |
| EVT-04 | Phase 16 | Pending |
| EVT-05 | Phase 16 | Pending |
| EVT-06 | Phase 16 | Pending |
| EVT-07 | Phase 16 | Pending |
| EVT-08 | Phase 16 | Pending |

**Coverage:**
- v2.0 requirements: 57 total
- Mapped to phases: 57 ✓
- Unmapped: 0 ✓

---
*Requirements defined: 2026-01-27*
*Last updated: 2026-01-28 after Phase 11 completion*
