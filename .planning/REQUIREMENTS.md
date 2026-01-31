# Requirements: gaz v3.0 API Harmonization

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

- [ ] **ERR-01**: Sentinel errors are consolidated in gaz/errors.go
- [ ] **ERR-02**: Error sentinels use namespaced naming (ErrDINotFound, ErrConfigNotFound, etc.)
- [ ] **ERR-03**: All packages use consistent wrapping pattern: fmt.Errorf("pkg: context: %w", err)

### Testing

- [ ] **TST-01**: gaztest package is enhanced for v3 patterns (Builder API updated, new helpers)
- [ ] **TST-02**: Per-package testing helpers exist (health/testing.go, worker/testing.go, config/testing.go)
- [ ] **TST-03**: Testing guide documentation exists

### Documentation

- [x] **DOC-01**: Style guide for contributors exists (API patterns, naming conventions, config requirements)
- [ ] **DOC-02**: User documentation exists (getting started, tutorials)
- [ ] **DOC-03**: All examples are updated to v3 patterns

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
| ERR-01 | Phase 27 | Pending |
| ERR-02 | Phase 27 | Pending |
| ERR-03 | Phase 27 | Pending |
| TST-01 | Phase 28 | Pending |
| TST-02 | Phase 28 | Pending |
| TST-03 | Phase 28 | Pending |
| DOC-01 | Phase 23 | Complete |
| DOC-02 | Phase 29 | Pending |
| DOC-03 | Phase 29 | Pending |

**Coverage:**
- v3.0 requirements: 17 total
- Mapped to phases: 17 ✓
- Unmapped: 0

---
*Requirements defined: 2026-01-29*
*Last updated: 2026-01-29 after initial definition*
