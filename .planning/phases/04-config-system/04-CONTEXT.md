# Phase 4: Config System - Context

**Gathered:** 2026-01-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Multi-source configuration loading (Env vars, Files, CLI flags). The goal is to provide a robust configuration system for gaz applications that unifies different sources into a coherent configuration state.

</domain>

<decisions>
## Implementation Decisions

### Configuration Interface
- **Primary Access Pattern:** Key-Value access (map-based) is the primary model, but Unmarshaling to typed structs is supported and expected.
- **Loading Responsibility:** Managed by the framework (user does not manually read files).
- **Discovery:** Smart defaults — framework searches standard paths (./, $HOME, /etc) automatically.

### Precedence Strategy
- **Hierarchy:** Standard (Flags > Env > Config File > Defaults).
- **Array Handling:** Replace strategy (new value completely replaces old list) for safety.
- **Profiles:** Support environment-specific overlays (e.g., config.prod.yaml overrides config.yaml).
- **Profile Selection:** The environment variable determining the profile is fully configurable (not hardcoded to APP_ENV).

### Environment Variables
- **Mapping:** Automatic mapping (e.g., APP_DB_HOST maps to db.host).
- **Scoping:** Require a specific prefix (e.g., GAZ_) to avoid reading all system environment variables.
- **Nesting:** Use double underscore (`__`) as the delimiter for nested keys (e.g., `APP_DB__HOST`).
- **Matching:** Case insensitive.

### Validation & Defaults
- **Validation:** Use a custom `Validate()` interface method on config structs (prefer logic over struct tags).
- **Defaults:** Use a custom `Default()` interface method on config structs (prefer logic over struct tags).
- **Strictness:** Loose unmarshaling — ignore unknown fields in config files.
- **Lifecycle:** Configuration is immutable after startup (no hot reloading).

### Claude's Discretion
- Specific interface definitions for `Validator` and `Defaulter`.
- Exact algorithm for map merging (deep merge behavior).
- Library selection (Viper vs others) — though decisions strongly point towards Viper-like capabilities.

</decisions>

<specifics>
## Specific Ideas

- The choice of "Key-Value access" as primary implies the framework holds the source of truth in a flexible map structure, with structs being a "view" into that data.
- Preference for methods (`Validate()`, `Default()`) over tags suggests a desire for cleaner struct definitions and more powerful validation logic than static tags allow.

</specifics>

<deferred>
## Deferred Ideas

- Hot reloading was explicitly rejected in favor of immutability/simplicity.

</deferred>

---

*Phase: 04-config-system*
*Context gathered: 2026-01-26*
