# Requirements: gaz v1.1

**Defined:** 2026-01-26
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v1.1 Requirements

Requirements for Security & Hardening milestone.

### Validation Engine

- [x] **VAL-01**: Config manager validates structs using `validate` tags upon load
- [x] **VAL-02**: Application fails to start (exits) if config validation fails
- [x] **VAL-03**: Config validation supports cross-field constraints (required_if, etc)

### Hardened Lifecycle

- [x] **LIFE-01**: Application enforces a hard timeout (default 30s) on shutdown
- [x] **LIFE-02**: Application forces `os.Exit(1)` if shutdown hooks exceed timeout
- [x] **LIFE-03**: Application exits immediately if SIGINT received twice
- [x] **LIFE-04**: Application logs which specific lifecycle hook caused a shutdown hang

### Provider Config Registration

- [x] **PROV-01**: Providers can implement ConfigProvider interface to declare config needs
- [x] **PROV-02**: Provider config keys are auto-prefixed with declared namespace
- [x] **PROV-03**: Duplicate config keys from different providers fail at Build() with clear error
- [x] **PROV-04**: Config values are injectable via ProviderValues type

### Documentation & Examples

- [x] **DOC-01**: All public APIs documented with usage examples
- [x] **DOC-02**: Working example applications demonstrating common patterns
- [x] **DOC-03**: API reference with complete type documentation

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
| VAL-01 | Phase 7 | Complete |
| VAL-02 | Phase 7 | Complete |
| VAL-03 | Phase 7 | Complete |
| LIFE-01 | Phase 8 | Complete |
| LIFE-02 | Phase 8 | Complete |
| LIFE-03 | Phase 8 | Complete |
| LIFE-04 | Phase 8 | Complete |
| PROV-01 | Phase 9 | Complete |
| PROV-02 | Phase 9 | Complete |
| PROV-03 | Phase 9 | Complete |
| PROV-04 | Phase 9 | Complete |
| DOC-01 | Phase 10 | Complete |
| DOC-02 | Phase 10 | Complete |
| DOC-03 | Phase 10 | Complete |

**Coverage:**
- v1.1 requirements: 14 total
- Complete: 14 (Phases 7, 8, 9, 10)
- Pending: 0

---
*Requirements defined: 2026-01-26*
