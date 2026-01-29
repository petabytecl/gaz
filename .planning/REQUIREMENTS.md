# Requirements: gaz v2.1

**Defined:** 2026-01-29
**Core Value:** Simple, type-safe dependency injection with sane defaults

## v2.1 Requirements

Requirements for API Enhancement milestone. Each maps to roadmap phases.

### Interface Auto-Detection

- [ ] **LIFE-01**: Services implementing `Starter` interface have `OnStart()` called automatically during lifecycle start
- [ ] **LIFE-02**: Services implementing `Stopper` interface have `OnStop()` called automatically during lifecycle stop
- [ ] **LIFE-03**: Explicit `.OnStart()/.OnStop()` registration takes precedence over interface detection
- [ ] **LIFE-04**: Interface detection works with both pointer and value receivers
- [ ] **LIFE-05**: `HasLifecycle()` returns true for services implementing Starter or Stopper interfaces

### CLI Integration

- [ ] **CLI-01**: CLI positional arguments accessible via `CommandArgs` type from DI container
- [ ] **CLI-02**: Helper function `gaz.GetArgs(container)` returns positional args slice
- [ ] **CLI-03**: `CommandArgs.Command` provides access to the current `*cobra.Command`

### Testing (gaztest)

- [ ] **TEST-01**: `gaztest.New(t)` creates test app with automatic cleanup via `t.Cleanup()`
- [ ] **TEST-02**: `app.RequireStart()` starts app or fails test with `t.Fatal()`
- [ ] **TEST-03**: `app.RequireStop()` stops app or fails test with `t.Fatal()`
- [ ] **TEST-04**: Test apps use shorter timeouts suitable for testing (5s default)
- [ ] **TEST-05**: `app.Replace(instance)` swaps dependency for testing (mocks)

### Service Builder

- [ ] **SVC-01**: `service.New(cmd, config)` creates pre-configured App with standard providers
- [ ] **SVC-02**: Service builder automatically registers health check module
- [ ] **SVC-03**: Service builder supports custom env prefix for configuration
- [ ] **SVC-04**: Service builder accepts additional `gaz.Option` for customization

### Unified Provider

- [ ] **PROV-01**: `Module(name)` returns fluent `ModuleBuilder` for bundling registrations
- [ ] **PROV-02**: `ModuleBuilder.Flags(fn)` registers command-line flags for the module
- [ ] **PROV-03**: `ModuleBuilder.Provide(fns...)` adds provider functions to the module
- [ ] **PROV-04**: `ModuleBuilder.Register(app)` applies all bundled registrations to an App
- [ ] **PROV-05**: Modules can be composed (module depends on another module)

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
| LIFE-01 | TBD | Pending |
| LIFE-02 | TBD | Pending |
| LIFE-03 | TBD | Pending |
| LIFE-04 | TBD | Pending |
| LIFE-05 | TBD | Pending |
| CLI-01 | TBD | Pending |
| CLI-02 | TBD | Pending |
| CLI-03 | TBD | Pending |
| TEST-01 | TBD | Pending |
| TEST-02 | TBD | Pending |
| TEST-03 | TBD | Pending |
| TEST-04 | TBD | Pending |
| TEST-05 | TBD | Pending |
| SVC-01 | TBD | Pending |
| SVC-02 | TBD | Pending |
| SVC-03 | TBD | Pending |
| SVC-04 | TBD | Pending |
| PROV-01 | TBD | Pending |
| PROV-02 | TBD | Pending |
| PROV-03 | TBD | Pending |
| PROV-04 | TBD | Pending |
| PROV-05 | TBD | Pending |

**Coverage:**
- v2.1 requirements: 22 total
- Mapped to phases: 0
- Unmapped: 22

---
*Requirements defined: 2026-01-29*
*Last updated: 2026-01-29 after initial definition*
