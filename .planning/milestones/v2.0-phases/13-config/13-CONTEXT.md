# Phase 13: Config Package - Context

**Gathered:** 2026-01-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Extract Config into standalone `gaz/config` subpackage with Backend interface abstracting viper. Config package works standalone (can load config without gaz App). Root `gaz` package integrates via Backend interface but does NOT re-export config types.

</domain>

<decisions>
## Implementation Decisions

### Backend interface design
- Full feature set: Get, Set, Unmarshal + watching, env binding, config writing
- Composed interfaces: core Backend interface + optional Watcher, Writer, EnvBinder interfaces
- ViperBackend lives in separate subpackage: `gaz/config/viper`
- Keys use dot notation for nested values ("database.host")

### Standalone usage API
- `config.New()` returns ConfigManager
- Combined load+unmarshal pattern: `Manager.LoadInto(&cfg)` does both in one call
- Generic accessor for individual values: `config.Get[T](mgr, "key")` returns typed value

### Integration with App
- Keep `WithConfig[T]()` option pattern (current behavior)
- App creates ViperBackend internally (simpler for users, no explicit backend needed)
- Users import config package directly (no re-export of config types in gaz root)
- ConfigManager is registered in DI container and resolvable as dependency

### Config validation & defaults
- Validation runs automatically during Load/Unmarshal
- Return ValidationErrors collection (multiple errors, not just first)
- Support struct tag validation (e.g., `validate:"required"`)

### Claude's Discretion
- Whether Backend requires initialization or works with nil-safe default
- Exact Defaulter/Validator interface design (keep separate or combine)
- Which struct tags to support for validation
- Internal error types and error wrapping patterns

</decisions>

<specifics>
## Specific Ideas

- Pattern mirrors di package extraction: standalone package, App integrates but doesn't re-export
- ViperBackend in subpackage keeps viper dependency isolated from core config package
- Composed interfaces allow simple backends to implement just Backend, full-featured ones to add Watcher/Writer

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 13-config*
*Context gathered: 2026-01-28*
