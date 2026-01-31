# Phase 26: Module & Service Consolidation - Context

**Gathered:** 2026-01-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Consolidate the module system by removing `gaz/service` package and adding standardized `NewModule()` factory functions to all subsystem packages. The `service.Builder` functionality moves directly into `gaz.App` methods. All existing tests must pass with the consolidated module system.

Requirements: MOD-01, MOD-02, MOD-03, MOD-04

</domain>

<decisions>
## Implementation Decisions

### NewModule() Factory Conventions
- **Signature pattern:** Functional options — `health.NewModule(health.WithPort(8081), ...)`
- **Return type:** Returns `gaz.Module` directly (not `*ModuleBuilder`)
- **Zero-config defaults:** `NewModule()` with no arguments works with sensible defaults
- **Option naming:** `With{Property}(value)` pattern — e.g., `health.WithPort(8081)`, `worker.WithMaxRetries(3)`

### Service Builder Migration
- **Migration pattern:** Direct App methods — `gaz.New().WithConfig(cfg).WithCobra(cmd)`
- **service package:** Remove entirely in v3 — this is a major version, clean break expected
- **EnvPrefix:** Keep in `app.WithConfig(cfg, config.WithEnvPrefix("MYAPP"))` — not moved to `gaz.New()`

### Subsystem Module Scope
- **Packages getting NewModule():** worker, cron, health, eventbus, config (all five)
- **Prerequisites model:** NewModule() expects certain registrations to exist (e.g., config)
- **Prerequisite communication:** Both doc comments AND runtime error messages
- **Existing Module() functions:** Replace completely — NewModule() replaces health.Module() etc.

### Import/Package Structure
- **di↔gaz relationship:** Keep di public but document that gaz is preferred
- **Import pattern:** `import "github.com/petabytecl/gaz"` for main package, separate imports for subsystems (`gaz/health`, `gaz/worker`, etc.)
- **MOD-04 documentation:** Clear enough that new users know gaz is the entry point

### Claude's Discretion
- Health auto-registration behavior — maintain consistency with existing patterns
- Exact implementation of how NewModule() builds the gaz.Module internally
- Documentation format for package relationships

</decisions>

<specifics>
## Specific Ideas

- Pattern follows uber-go/fx and google/wire conventions for module factories
- Keep the functional options pattern consistent across all subsystem packages
- v3 is a clean break — no deprecation period for removed `service` package

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 26-module-service-consolidation*
*Context gathered: 2026-01-31*
