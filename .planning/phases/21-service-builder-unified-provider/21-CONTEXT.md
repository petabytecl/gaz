# Phase 21: Service Builder + Unified Provider - Context

**Gathered:** 2026-01-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Convenience APIs for creating production-ready services (`service.Builder()`) and reusable modules (`Module(name)`). Service builder auto-wires standard providers based on configuration. Modules bundle providers, flags, configs, and lifecycle hooks for reuse across apps.

</domain>

<decisions>
## Implementation Decisions

### Service Builder defaults
- Adaptive approach: service builder wires what cmd/config provide, not a fixed set
- Health check auto-registers when config provides health settings
- Fluent builder API: `service.Builder().WithCmd(cmd).WithConfig(config).Build()`
- Returns `(App, error)` — caller handles configuration errors

### Module API ergonomics
- Modules can bundle: providers, CLI flags, configs, and lifecycle hooks
- Fluent builder: `Module("name")` returns builder
- Module name is documentation/debugging only, not a namespace
- Registration via `app.Use(module)` — app consumes modules

### Module composition
- Dependencies inferred from DI — no explicit `DependsOn()` declarations
- Duplicate module registration (same name) returns error
- Duplicate provider (same type from multiple modules) errors immediately at registration
- Lifecycle hook order determined by provider dependency graph (topological sort)

### Env prefix behavior
- Global prefix applies to all env vars (e.g., `MYAPP_DB_HOST`)
- Nested config structs use underscore nesting (e.g., `MYAPP_DATABASE_HOST` for `Config.Database.Host`)
- No default prefix — env vars work without prefix unless configured
- Module name becomes sub-prefix (e.g., service prefix `MYAPP_` + module `redis` = `MYAPP_REDIS_HOST`)

### Claude's Discretion
- Exact builder method names beyond documented API
- Internal implementation of dependency graph sorting
- Error message wording and format
- Health check endpoint path defaults

</decisions>

<specifics>
## Specific Ideas

- Module system should feel like Go's standard library patterns — simple, predictable
- Error messages should be actionable ("module 'redis' already registered, cannot re-register")
- The fluent builder should chain naturally without awkward intermediate types

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 21-service-builder-unified-provider*
*Context gathered: 2026-01-29*
