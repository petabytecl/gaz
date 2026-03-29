# Phase 9: Provider Config Registration - Context

**Gathered:** 2026-01-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Services/providers can register flags/config keys on the app config manager. Providers declare their configuration needs through an interface, and the framework handles namespacing, validation, and injection. This enables providers to be self-contained modules with their own configurable behavior.

</domain>

<decisions>
## Implementation Decisions

### Registration API
- Provider implements an interface that returns a slice of flag/key definitions
- Interface name is Claude's discretion
- Only DI-registered providers can implement the interface (not arbitrary structs)
- Framework calls the interface method at provider construction time
- Provider defines config only — does not receive values back directly
- Config values are injected as a generic container dependency (not typed struct)

### Config key naming
- All provider keys auto-prefixed with provider's declared namespace (e.g., `redis.host`)
- Provider explicitly declares its namespace (not derived from type name)
- Key format: unified dot notation everywhere, framework translates for env vars (`redis.host` → `REDIS_HOST`)
- Collision handling: fail at startup with clear error listing both providers if two register the same key

### Validation timing
- Provider config validated during app.Build(), after all providers registered
- Basic validation only: required + type checking (not full Phase 7 validation engine)
- Collect all validation errors before failing (not fail-fast)
- Error messages include: provider name + key + reason

### Default values & required flags
- Default value specified in the flag definition struct
- Required marked with bool field on definition (`Required: true`)
- Description field is required (for --help, documentation)
- Explicit type field for config values (int, bool, string, etc.)

### Claude's Discretion
- Interface naming
- Exact definition struct shape
- Generic container implementation details
- Error message formatting

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 09-provider-config-registration*
*Context gathered: 2026-01-27*
