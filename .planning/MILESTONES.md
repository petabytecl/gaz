# Project Milestones: gaz

## v2.2 Test Coverage (Shipped: 2026-01-29)

**Delivered:** Pushed overall test coverage to 92.9%, exceeding 90% threshold with comprehensive tests across all packages.

**Phases completed:** 22 (4 plans total)

**Key accomplishments:**

- Overall coverage: 84.3% → 92.9% (exceeds 90% target)
- DI package: 73.3% → 96.7% with inject, types, lifecycle_engine tests
- Config/Viper: 77.1% → 95.2% with validation, accessor, manager tests
- Health package: 83.8% → 92.4% with error paths and integration tests
- Worker package: → 95.7% with supervisor and pooled worker tests

**Stats:**

- 16 files created/modified
- 1,847 lines added
- 1 phase, 4 plans
- Same day as v2.1

**Git range:** `v2.1` → `02bfe7d`

**What's next:** Next milestone TBD

---

## v2.1 API Enhancement (Shipped: 2026-01-29)

**Delivered:** Interface auto-detection for lifecycle hooks, CLI args injection, gaztest package, service builder, and module system.

**Phases completed:** 19-21 (8 plans total)

**Key accomplishments:**

- Services implementing Starter/Stopper interfaces auto-detected for lifecycle
- CLI arguments accessible via `gaz.GetArgs(container)` helper
- gaztest package with Builder API, auto-cleanup, mock injection via `Replace()`
- service.Builder fluent API for production services with health auto-registration
- ModuleBuilder with Flags() for bundling CLI flags and providers
- Module composition with `Use()` for child modules

**Stats:**

- 28 files created/modified
- 2,731 lines added
- 3 phases, 8 plans
- Same day as v2.0

**Git range:** `v2.0` → (22 requirements, all complete)

**What's next:** v2.2 Test Coverage

---

## v2.0 Cleanup & Concurrency (Shipped: 2026-01-29)

**Delivered:** Extracted DI and Config to standalone packages, added concurrency primitives (Workers, Cron, EventBus).

**Phases completed:** 11-18 (34 plans total, including 4 inserted phases)

**Key accomplishments:**

- Deprecated APIs removed (NewApp, AppOption, reflection-based Provide* methods)
- DI extracted to standalone `gaz/di` package with di.New(), For[T](), Resolve[T]()
- Config extracted to standalone `gaz/config` package with Backend interface
- Background workers with lifecycle integration, panic recovery, and circuit breaker
- Cron scheduled tasks wrapping robfig/cron with DI-aware jobs
- Type-safe EventBus with generic Publish[T]/Subscribe[T] and topic filtering
- Cobra CLI flag integration via RegisterCobraFlags()
- System Info CLI example showcasing all v2.0 features

**Stats:**

- 231 files created/modified
- 33,844 lines added (31,207 net)
- 12 phases, 34 plans
- 2 days from milestone start to ship

**Git range:** `v1.1` → `b87bc0d`

**What's next:** v2.1 - TBD (worker pools, koanf backend)

---

## v1.1 Security & Hardening (Shipped: 2026-01-27)

**Delivered:** Application robustness with config validation and shutdown hardening.

**Phases completed:** 7-10 (12 plans total)

**Key accomplishments:**

- Config validation engine with struct tags and early exit on invalid config
- Shutdown hardening with per-hook timeout, blame logging, and double-SIGINT force exit
- Provider config registration for service-level configuration
- Comprehensive documentation with README, guides, godoc examples, and 6 example apps

**Stats:**

- 22 files created/modified
- 3,441 lines added (11,319 total Go LOC)
- 4 phases, 12 plans
- 2 days from milestone start to ship

**Git range:** `aaf55bd` (v1.0) → `6099075`

**What's next:** v2.0 - Cleanup & Concurrency

---

## v1.0 MVP (Shipped: 2026-01-26)

**Delivered:** Unified Go application framework with DI, Lifecycle, Config, Health, and Logging.

**Phases completed:** 1-6 (35 plans total)

**Key accomplishments:**

- Type-safe generic DI container foundation
- Deterministic lifecycle management with signal handling
- Fluent App Builder integrated with Cobra
- Multi-source configuration system
- Production-ready health checks
- Structured logging with context propagation

**Stats:**

- 169 files created/modified
- 7926 lines of Go
- 10 phases, 35 plans
- 1 days from start to ship

**Git range:** `0c00fc6` → `aaf55bd`

**What's next:** v1.1 - Security & Hardening

---
