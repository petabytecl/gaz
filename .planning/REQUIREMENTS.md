# Requirements: gaz

**Defined:** 2026-01-26
**Core Value:** Simple, type-safe dependency injection with sane defaults — developers register providers and resolve dependencies without fighting configuration options.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### DI Container

- [x] **DI-01**: Developer can register providers with generic type `Register[T](provider)` syntax
- [x] **DI-02**: Container lazily instantiates singletons on first resolution (default behavior)
- [x] **DI-03**: Providers can return `(T, error)` and errors propagate through dependency chain
- [x] **DI-04**: Developer can register multiple implementations of same type with names
- [x] **DI-05**: Developer can inject dependencies into struct fields via `gaz:"inject"` tag
- [x] **DI-06**: Developer can override registered services (for testing)
- [x] **DI-07**: Developer can register transient services (new instance per resolution)
- [x] **DI-08**: Developer can register eager services (instantiate at startup)
- [x] **DI-09**: Container detects circular dependencies and fails fast with clear error

### Lifecycle

- [x] **LIFE-01**: Developer can register OnStart hooks that run during app startup
- [x] **LIFE-02**: Developer can register OnStop hooks that run during shutdown
- [x] **LIFE-03**: App handles SIGTERM/SIGINT signals and initiates graceful shutdown
- [x] **LIFE-04**: Developer can configure global shutdown timeout
- [x] **LIFE-05**: All lifecycle hooks receive context.Context for cancellation
- [x] **LIFE-06**: Services start in topological order based on dependencies
- [x] **LIFE-07**: Services stop in LIFO order (reverse of start)
- [x] **LIFE-08**: Developer can configure per-hook timeouts

### App Builder

- [ ] **APP-01**: Developer can create app with `gaz.New()` entry point
- [ ] **APP-02**: Developer can add providers with fluent `.Provide()` method
- [ ] **APP-03**: Developer can compose related services into modules
- [ ] **APP-04**: Developer can start app with `.Run()` method (blocking)
- [ ] **APP-05**: Developer can integrate app with cobra.Command for CLI
- [ ] **APP-06**: Framework provides sensible defaults (shutdown timeout, logger, etc.)

### Health Checks

- [ ] **HLTH-01**: Developer can implement HealthChecker interface for custom checks
- [ ] **HLTH-02**: Framework distinguishes readiness vs liveness checks
- [ ] **HLTH-03**: Framework auto-discovers health checkers registered in DI container
- [ ] **HLTH-04**: Framework aggregates multiple health check results
- [ ] **HLTH-05**: Health checks execute concurrently with configurable timeout

### Configuration

- [ ] **CONF-01**: Developer can load config from environment variables
- [ ] **CONF-02**: Developer can load config from CLI flags (Cobra integration)
- [ ] **CONF-03**: Developer can load config from files (YAML, JSON, TOML)

### Logging

- [ ] **LOG-01**: Framework provides pre-configured slog.Logger via DI
- [ ] **LOG-02**: Logger propagates through context.Context
- [ ] **LOG-03**: Framework logs its own events (startup, shutdown, errors) via slog
- [ ] **LOG-04**: Developer can provide custom slog.Handler for formatting

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Configuration (Extended)

- **CONF-04**: Developer can bind config to typed structs
- **CONF-05**: Framework supports config validation
- **CONF-06**: Framework supports config hot-reload

### Advanced DI

- **DI-10**: Hierarchical scopes (request-scoped, session-scoped)
- **DI-11**: Value groups (collect multiple implementations)
- **DI-12**: Decorator pattern support

### Extensions

- **EXT-01**: Event bus for inter-component communication
- **EXT-02**: Worker group management for background tasks
- **EXT-03**: Metrics integration (Prometheus)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| HTTP server built-in | Many options (std, chi, echo); users choose |
| RPC/messaging built-in | Infrastructure concern; use go-micro separately |
| Service discovery | Infrastructure concern; use Consul/K8s directly |
| Database abstraction | Too many options; provide examples instead |
| Code generation (Wire-style) | Adds friction; runtime DI is more ergonomic |
| Global container singleton | Anti-pattern; defeats purpose of DI |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DI-01 | Phase 1 | Complete |
| DI-02 | Phase 1 | Complete |
| DI-03 | Phase 1 | Complete |
| DI-04 | Phase 1 | Complete |
| DI-05 | Phase 1 | Complete |
| DI-06 | Phase 1 | Complete |
| DI-07 | Phase 1 | Complete |
| DI-08 | Phase 1 | Complete |
| DI-09 | Phase 1 | Complete |
| LIFE-01 | Phase 2 | Complete |
| LIFE-02 | Phase 2 | Complete |
| LIFE-03 | Phase 2 | Complete |
| LIFE-04 | Phase 2 | Complete |
| LIFE-05 | Phase 2 | Complete |
| LIFE-06 | Phase 2 | Complete |
| LIFE-07 | Phase 2 | Complete |
| LIFE-08 | Phase 2 | Complete |
| APP-01 | Phase 3 | Pending |
| APP-02 | Phase 3 | Pending |
| APP-03 | Phase 3 | Pending |
| APP-04 | Phase 3 | Pending |
| APP-05 | Phase 3 | Pending |
| APP-06 | Phase 3 | Pending |
| CONF-01 | Phase 4 | Pending |
| CONF-02 | Phase 4 | Pending |
| CONF-03 | Phase 4 | Pending |
| HLTH-01 | Phase 5 | Pending |
| HLTH-02 | Phase 5 | Pending |
| HLTH-03 | Phase 5 | Pending |
| HLTH-04 | Phase 5 | Pending |
| HLTH-05 | Phase 5 | Pending |
| LOG-01 | Phase 6 | Pending |
| LOG-02 | Phase 6 | Pending |
| LOG-03 | Phase 6 | Pending |
| LOG-04 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 35 total
- Mapped to phases: 35
- Unmapped: 0

---
*Requirements defined: 2026-01-26*
*Last updated: 2026-01-26 after initial definition*
