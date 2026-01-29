# Requirements: gaz v2.1

**Defined:** 2026-01-29
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v2.1 Requirements

Requirements for API Enhancement milestone. Each maps to roadmap phases.

### Interface Auto-Detection

- [x] **LIFE-01**: Services implementing `Starter` interface have `OnStart()` called automatically during lifecycle start
- [x] **LIFE-02**: Services implementing `Stopper` interface have `OnStop()` called automatically during lifecycle stop
- [x] **LIFE-03**: Explicit `.OnStart()/.OnStop()` registration takes precedence over interface detection
- [x] **LIFE-04**: Interface detection works with both pointer and value receivers
- [x] **LIFE-05**: `HasLifecycle()` returns true for services implementing Starter or Stopper interfaces

### CLI Integration

- [x] **CLI-01**: CLI positional arguments accessible via `CommandArgs` type from DI container
- [x] **CLI-02**: Helper function `gaz.GetArgs(container)` returns positional args slice
- [x] **CLI-03**: `CommandArgs.Command` provides access to the current `*cobra.Command`

### Testing (gaztest)

- [x] **TEST-01**: `gaztest.New(t)` creates test app with automatic cleanup via `t.Cleanup()`
- [x] **TEST-02**: `app.RequireStart()` starts app or fails test with `t.Fatal()`
- [x] **TEST-03**: `app.RequireStop()` stops app or fails test with `t.Fatal()`
- [x] **TEST-04**: Test apps use shorter timeouts suitable for testing (5s default)
- [x] **TEST-05**: `app.Replace(instance)` swaps dependency for testing (mocks)

### Service Builder

- [x] **SVC-01**: `service.New(cmd, config)` creates pre-configured App with standard providers
- [x] **SVC-02**: Service builder automatically registers health check module
- [x] **SVC-03**: Service builder supports custom env prefix for configuration
- [x] **SVC-04**: Service builder accepts additional `gaz.Option` for customization

### Unified Provider

- [x] **PROV-01**: `Module(name)` returns fluent `ModuleBuilder` for bundling registrations
- [x] **PROV-02**: `ModuleBuilder.Flags(fn)` registers command-line flags for the module
- [x] **PROV-03**: `ModuleBuilder.Provide(fns...)` adds provider functions to the module
- [x] **PROV-04**: `ModuleBuilder.Register(app)` applies all bundled registrations to an App
- [x] **PROV-05**: Modules can be composed (module depends on another module)

## Future Requirements

Deferred to later milestones.

### Build Info

- **BUILD-01**: `buildinfo.Get(name)` returns build metadata (version, commit, branch)
- **BUILD-02**: `buildinfo.RegisterVersion(cmd, name)` integrates with Cobra `--version`
- **BUILD-03**: Build info combines ldflags values with `runtime/debug.ReadBuildInfo()`

### Pre/Post Hooks

- **HOOK-01**: `App.PreRun(fn)` registers hooks that run after Build, before Start
- **HOOK-02**: `App.PostRun(fn)` registers hooks that run before Stop
- **HOOK-03**: Pre/Post hooks execute with access to DI container

### Debug Utilities

- **DEBUG-01**: `debug.CallerFunction()` returns caller function name
- **DEBUG-02**: `debug.CallerLocation()` returns file and line of caller

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Global Shutdowner | Breaks testability — keep instance-based `app.Stop()` |
| Panic on hook conflict | Bad DX — use error returns instead |
| Auto-wiring via struct tags | Hides dependencies — explicit For[T]() is cleaner |
| Config schema validation | Adds complexity — defer to v3+ |
| Automatic module discovery | Magic behavior — prefer explicit registration |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| LIFE-01 | Phase 19 | Complete |
| LIFE-02 | Phase 19 | Complete |
| LIFE-03 | Phase 19 | Complete |
| LIFE-04 | Phase 19 | Complete |
| LIFE-05 | Phase 19 | Complete |
| CLI-01 | Phase 19 | Complete |
| CLI-02 | Phase 19 | Complete |
| CLI-03 | Phase 19 | Complete |
| TEST-01 | Phase 20 | Complete |
| TEST-02 | Phase 20 | Complete |
| TEST-03 | Phase 20 | Complete |
| TEST-04 | Phase 20 | Complete |
| TEST-05 | Phase 20 | Complete |
| SVC-01 | Phase 21 | Complete |
| SVC-02 | Phase 21 | Complete |
| SVC-03 | Phase 21 | Complete |
| SVC-04 | Phase 21 | Complete |
| PROV-01 | Phase 21 | Complete |
| PROV-02 | Phase 21 | Complete |
| PROV-03 | Phase 21 | Complete |
| PROV-04 | Phase 21 | Complete |
| PROV-05 | Phase 21 | Complete |

**Coverage:**
- v2.1 requirements: 22 total
- Mapped to phases: 22 ✓
- Unmapped: 0

---
*Requirements defined: 2026-01-29*
*Last updated: 2026-01-29 after Phase 21 completion*
