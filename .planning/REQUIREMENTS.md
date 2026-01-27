# Requirements: gaz v1.1

**Defined:** 2026-01-26
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v1.1 Requirements

Requirements for Security & Hardening milestone.

### Validation Engine

- [ ] **VAL-01**: Config manager validates structs using `validate` tags upon load
- [ ] **VAL-02**: Application fails to start (exits) if config validation fails
- [ ] **VAL-03**: Config validation supports cross-field constraints (required_if, etc)

### Hardened Lifecycle

- [ ] **LIFE-01**: Application enforces a hard timeout (default 30s) on shutdown
- [ ] **LIFE-02**: Application forces `os.Exit(1)` if shutdown hooks exceed timeout
- [ ] **LIFE-03**: Application exits immediately if SIGINT received twice
- [ ] **LIFE-04**: Application logs which specific lifecycle hook caused a shutdown hang

## v2 Requirements

### Advanced Validation

- **VAL-04**: Users can register custom validation functions for domain types

## Out of Scope

| Feature | Reason |
|---------|--------|
| Silent Validation Failure | Application must not run in undefined state |
| Indefinite Shutdown Wait | Zombie processes prevent automated rollouts |
| Partial Config Loading | All or nothing consistency |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| VAL-01 | — | Pending |
| VAL-02 | — | Pending |
| VAL-03 | — | Pending |
| LIFE-01 | — | Pending |
| LIFE-02 | — | Pending |
| LIFE-03 | — | Pending |
| LIFE-04 | — | Pending |

**Coverage:**
- v1 requirements: 7 total
- Mapped to phases: 0
- Unmapped: 7 ⚠️

---
*Requirements defined: 2026-01-26*
