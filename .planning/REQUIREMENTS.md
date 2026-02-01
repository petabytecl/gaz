# Requirements: gaz v3.0 API Harmonization + v3.1 Performance & Stability

**Defined:** 2026-01-29
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v3.0 Requirements

Requirements for v3.0 milestone. Each maps to roadmap phases.

### Configuration

- [x] **CFG-01**: ProviderValues has Unmarshal(namespace, &target) method for struct-based config resolution

### Lifecycle

- [x] **LIF-01**: RegistrationBuilder does not have OnStart/OnStop fluent methods (interface-only lifecycle)
- [x] **LIF-02**: worker.Worker interface uses OnStart(ctx context.Context) error and OnStop(ctx context.Context) error
- [x] **LIF-03**: Adapt() helper exists for wrapping third-party types (sql.DB, http.Server) with lifecycle hooks (SKIPPED per user decision)

### Module Consolidation

- [x] **MOD-01**: service.Builder functionality is absorbed into gaz.App
- [x] **MOD-02**: gaz/service package is removed
- [x] **MOD-03**: Each subsystem package exports NewModule() function returning gaz.Module
- [x] **MOD-04**: di/gaz relationship is documented (which types are re-exported, when to import each)

### Error Handling

- [x] **ERR-01**: Sentinel errors are consolidated in gaz/errors.go
- [x] **ERR-02**: Error sentinels use namespaced naming (ErrDINotFound, ErrConfigNotFound, etc.)
- [x] **ERR-03**: All packages use consistent wrapping pattern: fmt.Errorf("pkg: context: %w", err)

### Testing

- [x] **TST-01**: gaztest package is enhanced for v3 patterns (Builder API updated, new helpers)
- [x] **TST-02**: Per-package testing helpers exist (health/testing.go, worker/testing.go, config/testing.go)
- [x] **TST-03**: Testing guide documentation exists

### Documentation

- [x] **DOC-01**: Style guide for contributors exists (API patterns, naming conventions, config requirements)
- [x] **DOC-02**: User documentation exists (getting started, tutorials)
- [x] **DOC-03**: All examples are updated to v3 patterns

## v3.1 Requirements

Requirements for v3.1 milestone. Address critical issues from GAZ_REVIEW.md.

### Performance

- [x] **PERF-01**: DI container uses goid.Get() instead of runtime.Stack() for goroutine ID

### Stability

- [x] **STAB-01**: collectProviderConfigs checks service type before instantiation (no side-effects)

## v3.2 Requirements

Requirements for v3.2 milestone. Feature maturity improvements from GAZ_REVIEW.md Phase 2.

### Features

- [x] **FEAT-01**: WithStrictConfig() option fails startup if config file contains unregistered keys
- [x] **FEAT-02**: Worker manager has dead letter handling for workers that panic repeatedly

## Future Requirements

Deferred to v3.1 or later.

### Configuration

- **CFG-02**: Config struct standard with mapstructure/yaml/json tags on all types
- **CFG-03**: DefaultConfig() function exists for all config types

### Developer Experience

- **DX-01**: Opt-in debug mode for DI operations visibility
- **DX-02**: Rich structured error types with hints and suggestions

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Migration tooling | Clean break, no backward compatibility |
| Backward compatibility | Breaking changes are allowed in v3 |
| Changes to _tmp packages | Internal/experimental code |
| Global container singleton | Anti-pattern, explicitly avoided |
| Spring-style field injection | Anti-pattern in Go |
| Dynamic registration after Build() | Violates build-time guarantees |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CFG-01 | Phase 25 | Complete |
| LIF-01 | Phase 24 | Complete |
| LIF-02 | Phase 24 | Complete |
| LIF-03 | Phase 24 | Skipped |
| MOD-01 | Phase 26 | Complete |
| MOD-02 | Phase 26 | Complete |
| MOD-03 | Phase 26 | Complete |
| MOD-04 | Phase 26 | Complete |
| ERR-01 | Phase 27 | Complete |
| ERR-02 | Phase 27 | Complete |
| ERR-03 | Phase 27 | Complete |
| TST-01 | Phase 28 | Complete |
| TST-02 | Phase 28 | Complete |
| TST-03 | Phase 28 | Complete |
| DOC-01 | Phase 23 | Complete |
| DOC-02 | Phase 29 | Complete |
| DOC-03 | Phase 29 | Complete |
| PERF-01 | Phase 30 | Complete |
| STAB-01 | Phase 30 | Complete |
| FEAT-01 | Phase 31 | Complete |
| FEAT-02 | Phase 31 | Complete |

**Coverage:**
- v3.0 requirements: 17 total
- v3.1 requirements: 2 total
- v3.2 requirements: 2 total
- Mapped to phases: 21 ✓
- Unmapped: 0

---
*Requirements defined: 2026-01-29*
*Last updated: 2026-02-01 after v3.2 completion*
