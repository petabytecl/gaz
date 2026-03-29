# Phase 25: Configuration Harmonization - Context

**Gathered:** 2026-01-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Add struct-based config resolution via `ProviderValues.Unmarshal()` methods. Enables type-safe, module-isolated configuration access. Existing `LoadInto()` pattern continues to work (no breaking change). Config namespacing enables module isolation.

</domain>

<decisions>
## Implementation Decisions

### Unmarshal API Design
- Method signature: `pv.UnmarshalKey(key string, target any) error` — matches Backend.UnmarshalKey exactly
- Offer both: `UnmarshalKey` for namespaced access AND `Unmarshal` for full config access
- Keep existing GetString/GetInt/GetBool/GetDuration/GetFloat64 methods — simple use cases don't need structs
- Return only `error`, but use specific sentinel `config.ErrKeyNotFound` for missing namespace

### Error Behavior
- Missing namespace returns `config.ErrKeyNotFound` sentinel error
- Type mismatch returns specific type conversion error (not passthrough)
- Partial fill is OK — absent struct fields stay zero-valued, only set what exists in config
- Extra config keys silently ignored — struct only takes what it declares

### Tag Integration
- Use `gaz` struct tag for key mapping (our own, not mapstructure)
- Defaults stay in ConfigFlag declarations only — no tag-based defaults
- Automatic env binding: `redis.host` automatically checks `REDIS_HOST`
- Validate tag support: integrate with go-playground/validator patterns

### Nested Config Support
- Full nesting supported: `redis.pool.size` maps to struct `Pool.Size`
- Slice support: `redis.hosts.0`, `redis.hosts.1` maps to slice fields
- No map support in this phase — slices only
- Embedded structs use struct name as key prefix (not flattened)
- Error on namespace collision when multiple modules claim same namespace

### Claude's Discretion
- Exact implementation of gaz tag parsing (can follow mapstructure patterns internally)
- How to wrap/expose viper's UnmarshalKey under the hood
- Test file organization
- Error message formatting

</decisions>

<specifics>
## Specific Ideas

- API should feel consistent with existing `LoadInto()` pattern
- The `gaz` tag gives us control for future extensions (required, default, env override, etc.) even if we don't use all features now
- Namespace collision detection happens at config registration time, not unmarshal time

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 25-configuration-harmonization*
*Context gathered: 2026-01-30*
