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

## Progress

| Phase | Status | Completion |
|-------|--------|------------|
| 7 - Validation Engine | **Pending** | - |
| 8 - Hardened Lifecycle | Pending | - |
