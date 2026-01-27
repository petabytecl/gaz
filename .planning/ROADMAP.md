# Project Roadmap

**Milestone:** v1.1 Security & Hardening
**Focus:** Application Robustness (Validation & Lifecycle)
**Status:** Planning

## Overview

Milestone v1.1 hardens the `gaz` framework for production use by introducing two critical control gates: strict configuration validation at startup and guaranteed timeout enforcement at shutdown. This ensures applications never run in an undefined state and never hang indefinitely during rollouts.

## Phase Structure

| Phase | Goal | Requirements | Success Criteria |
|-------|------|--------------|------------------|
| **7 - Validation Engine** | Users can define struct tags that prevent application startup if configuration is invalid. | VAL-01, VAL-02, VAL-03 | 1. `validate` tags enforce constraints<br>2. App exits on config error<br>3. Cross-field rules work |
| **8 - Hardened Lifecycle** | Application guarantees process termination within a fixed timeout, preventing zombie processes. | LIFE-01, LIFE-02, LIFE-03, LIFE-04 | 1. Shutdown forces exit after 30s<br>2. Double Ctrl+C exits immediately<br>3. Logs blame hanging hooks |
| **9 - Provider Config Registration** | Services/providers can register flags/config keys on the app config manager. | PROV-01, PROV-02, PROV-03, PROV-04 | 1. ConfigProvider interface works<br>2. Keys auto-namespaced<br>3. Collisions detected<br>4. Values injectable |
| **10 - Documentation & Examples** | Comprehensive documentation and examples demonstrating all library features. | DOC-01, DOC-02, DOC-03 | 1. All features documented<br>2. Working examples provided<br>3. API reference complete |

## Detailed Phases

### Phase 7: Validation Engine

**Goal:** Users can define struct tags that prevent application startup if configuration is invalid.

**Dependencies:** None (builds on v1.0 Config)

**Requirements:**
- **VAL-01**: Config manager validates structs using `validate` tags upon load
- **VAL-02**: Application fails to start (exits) if config validation fails
- **VAL-03**: Config validation supports cross-field constraints (required_if, etc)

**Success Criteria:**
1. User can add `validate:"required"` to a config struct field and see it enforced.
2. Application exits with non-zero code immediately if validation fails.
3. User sees a human-readable error message listing specifically which fields failed validation.
4. User can use complex rules like `required_if` to validate dependencies between config fields.

**Plans:** 2 plans (2/2 complete)
Plans:
- [x] 07-01-PLAN.md — Core implementation: validator dependency, validation.go, ConfigManager integration
- [x] 07-02-PLAN.md — Comprehensive tests: basic tags, cross-field, nested structs, error formatting

---

### Phase 8: Hardened Lifecycle

**Goal:** Application guarantees process termination within a fixed timeout, preventing zombie processes.

**Dependencies:** Phase 7 (Stable startup required for stable shutdown testing)

**Requirements:**
- **LIFE-01**: Application enforces a hard timeout (default 30s) on shutdown
- **LIFE-02**: Application forces `os.Exit(1)` if shutdown hooks exceed timeout
- **LIFE-03**: Application exits immediately if SIGINT received twice
- **LIFE-04**: Application logs which specific lifecycle hook caused a shutdown hang

**Success Criteria:**
1. Application shuts down gracefully if all hooks complete within timeout.
2. Application forcefully exits (exit code 1) if a hook sleeps longer than the timeout (simulated).
3. Logs explicitly identify the component name of the hook that caused the timeout.
4. Pressing Ctrl+C twice triggers an immediate exit without waiting for the graceful timeout.

**Plans:** 3 plans (2/3 complete)
Plans:
- [x] 08-01-PLAN.md — Shutdown orchestrator with per-hook timeout and blame logging
- [x] 08-02-PLAN.md — Double-SIGINT force exit handling
- [ ] 08-03-PLAN.md — Comprehensive shutdown hardening tests

---

### Phase 9: Provider Config Registration

**Goal:** Services/providers can register flags/config keys on the app config manager.

**Dependencies:** Phase 8

**Requirements:**
- **PROV-01**: Providers can implement ConfigProvider interface to declare config needs
- **PROV-02**: Provider config keys are auto-prefixed with declared namespace
- **PROV-03**: Duplicate config keys from different providers fail at Build() with clear error
- **PROV-04**: Config values are injectable via ProviderValues type

**Success Criteria:**
1. Provider implementing ConfigProvider has config collected during Build()
2. Keys are auto-prefixed (e.g., redis + host = redis.host)
3. Two providers registering same key fails with ErrConfigKeyCollision
4. Required flags missing fails Build() with clear error message
5. ProviderValues injectable and provides typed getters (GetString, GetInt, etc)
6. Env vars work with translated names (redis.host → REDIS_HOST)

**Plans:** 2 plans (2/2 complete)
Plans:
- [x] 09-01-PLAN.md — Define ConfigProvider interface, ConfigFlag struct, ErrConfigKeyCollision
- [x] 09-02-PLAN.md — App integration, ConfigManager wiring, ProviderValues, comprehensive tests

---

### Phase 10: Documentation & Examples

**Goal:** Comprehensive documentation and examples demonstrating all library features.

**Dependencies:** Phase 9 (All features complete before documenting)

**Requirements:**
- **DOC-01**: All public APIs documented with usage examples
- **DOC-02**: Working example applications demonstrating common patterns
- **DOC-03**: API reference with complete type documentation

**Success Criteria:**
1. README covers installation, quick start, and core concepts.
2. Each major feature (DI, Config, Lifecycle, Validation, Providers) has dedicated documentation.
3. Example applications compile and run successfully.
4. API reference generated and accessible.

**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 10 to break down)

**Details:**
[To be added during planning]

---

## Progress

| Phase | Status | Completion |
|-------|--------|------------|
| 7 - Validation Engine | **✓ Complete** | 2/2 plans verified |
| 8 - Hardened Lifecycle | **In Progress** | 1/3 plans |
| 9 - Provider Config Registration | **✓ Complete** | 2/2 plans verified |
| 10 - Documentation & Examples | **Not Planned** | 0/0 plans |
