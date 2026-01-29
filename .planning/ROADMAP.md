# Roadmap: gaz v2.1 API Enhancement

## Overview

v2.1 enhances GAZ's developer experience with interface auto-detection for lifecycle hooks, CLI args injection, testing utilities, and convenience APIs for building services and modules. All features build on v2.0's stable foundation without architectural changes. The milestone closes a gap where services implementing Starter/Stopper interfaces weren't automatically detected for lifecycle ordering.

## Milestones

- ✅ **v1.0 MVP** - Phases 1-6 (shipped 2026-01-26)
- ✅ **v1.1 Security & Hardening** - Phases 7-10 (shipped 2026-01-27)
- ✅ **v2.0 Cleanup & Concurrency** - Phases 11-18 (shipped 2026-01-29)
- 🚧 **v2.1 API Enhancement** - Phases 19-21 (in progress)

## Phases

- [ ] **Phase 19: Interface Auto-Detection + CLI Args** - Foundation fixes and core additions
- [ ] **Phase 20: Testing Utilities (gaztest)** - Test builder with automatic cleanup
- [ ] **Phase 21: Service Builder + Unified Provider** - Convenience APIs for production

## Phase Details

### 🚧 v2.1 API Enhancement (In Progress)

**Milestone Goal:** Enhance developer experience with interface auto-detection, testing utilities, and convenience APIs for building services and modules.

#### Phase 19: Interface Auto-Detection + CLI Args
**Goal**: Services with lifecycle interfaces are automatically detected, and CLI args are accessible via DI
**Depends on**: v2.0 complete (Phase 18)
**Requirements**: LIFE-01, LIFE-02, LIFE-03, LIFE-04, LIFE-05, CLI-01, CLI-02, CLI-03
**Success Criteria** (what must be TRUE):
  1. Service implementing `Starter` interface has `OnStart()` called automatically without explicit registration
  2. Service implementing `Stopper` interface has `OnStop()` called automatically without explicit registration
  3. `For[T]().HasLifecycle()` returns `true` for types implementing Starter or Stopper
  4. Explicit `.OnStart()/.OnStop()` registration takes precedence over interface detection
  5. CLI positional args are accessible via `gaz.GetArgs(container)` after app startup
**Plans**: 2 plans

Plans:
- [ ] 19-01-PLAN.md — **[TDD]** Lifecycle Auto-Detection
- [ ] 19-02-PLAN.md — CLI Arguments Integration

#### Phase 20: Testing Utilities (gaztest)
**Goal**: Testing DI apps is easy with proper utilities and automatic cleanup
**Depends on**: Phase 19
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05
**Success Criteria** (what must be TRUE):
  1. `gaztest.New(t)` creates test app that automatically cleans up via `t.Cleanup()`
  2. `app.RequireStart()` starts app or fails test with `t.Fatal()`
  3. `app.RequireStop()` stops app or fails test with `t.Fatal()`
  4. Test apps use shorter default timeouts (5s) suitable for testing
  5. `app.Replace(instance)` allows swapping dependencies for mocks
**Plans**: TBD

Plans:
- [ ] 20-01: TBD

#### Phase 21: Service Builder + Unified Provider
**Goal**: Creating production-ready services and reusable modules is streamlined
**Depends on**: Phase 20
**Requirements**: SVC-01, SVC-02, SVC-03, SVC-04, PROV-01, PROV-02, PROV-03, PROV-04, PROV-05
**Success Criteria** (what must be TRUE):
  1. `service.New(cmd, config)` creates App with standard providers (health check auto-registered)
  2. Service builder supports custom env prefix for configuration
  3. `Module(name)` returns fluent `ModuleBuilder` for bundling flags and providers
  4. `ModuleBuilder.Register(app)` applies all bundled registrations to an App
  5. Modules can depend on other modules (composition works)
**Plans**: TBD

Plans:
- [ ] 21-01: TBD

## Progress

**Execution Order:** Phases execute in numeric order: 19 → 20 → 21

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 19. Interface Auto-Detection + CLI | v2.1 | 0/2 | Ready | - |
| 20. Testing Utilities (gaztest) | v2.1 | 0/TBD | Not started | - |
| 21. Service Builder + Unified Provider | v2.1 | 0/TBD | Not started | - |

<details>
<summary>✅ v2.0 Cleanup & Concurrency (Phases 11-18) - SHIPPED 2026-01-29</summary>

See `.planning/milestones/v2.0-SUMMARY.md` for details.

**Key accomplishments:**
- Deprecated APIs removed (NewApp, AppOption, Provide* methods)
- DI extracted to standalone `gaz/di` package
- Config extracted to standalone `gaz/config` package
- Workers package with lifecycle integration
- Cron package wrapping robfig/cron
- EventBus package with generics pub/sub
- RegisterCobraFlags for CLI integration
- System Info CLI example

</details>

<details>
<summary>✅ v1.1 Security & Hardening (Phases 7-10) - SHIPPED 2026-01-27</summary>

See `.planning/milestones/v1.1-SUMMARY.md` for details.

</details>

<details>
<summary>✅ v1.0 MVP (Phases 1-6) - SHIPPED 2026-01-26</summary>

See `.planning/milestones/v1.0-SUMMARY.md` for details.

</details>

---
*Roadmap created: 2026-01-29*
*Last updated: 2026-01-29*
