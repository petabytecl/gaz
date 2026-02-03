# Phase 37: Core Discovery - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Enable the container to resolve all registered providers of a type to support auto-discovery patterns. This involves adding list resolution capabilities to the core DI engine.

</domain>

<decisions>
## Implementation Decisions

### Registration Syntax
- **Implicit Collection:** Calling `gaz.For[*T](c)` multiple times is allowed (does not error/overwrite).
- **Tagging:** Use builder method `.InGroup("name")` to tag services.
  - Example: `gaz.For[*T](c).InGroup("routes").Provider(...)`
- No explicit "ForList" or "AsCollection" modifier required.

### Resolution Syntax
- **Explicit List:** Use `gaz.ResolveAll[*T](c)` to get `[]*T`.
  - **No Magic Slice:** `gaz.Resolve[[]*T](c)` is NOT supported (to avoid ambiguity).
- **Group Resolution:** Use `gaz.ResolveGroup[*T](c, "tag")` to get `[]*T` for a specific tag.
- **Inclusive Scope:** `ResolveAll` returns ALL registered services of that type (both tagged and untagged).

### Ambiguity Policy
- **Single Resolution:** `gaz.Resolve[*T](c)` MUST error if multiple providers are registered for `*T`.
  - It does NOT default to the first/last/primary.
  - User must use `ResolveAll` if cardinality > 1.

### Empty Collections
- `ResolveAll` and `ResolveGroup` return an **empty slice** `[]` (not error) if no matching services are found.

### Ordering
- **Unordered:** The order of services in the resolved slice is NOT guaranteed.

### Claude's Discretion
- Exact error messages.
- Internal storage structure for multi-provider records.
- How `InGroup` is stored in the binding.

</decisions>

<specifics>
## Specific Ideas

- "I like explicit, but we can use something like uber/fx value group so we can group by tag and not only by type"
- Preference for "Explicit Function" over magic or overloaded resolve methods.

</specifics>

<deferred>
## Deferred Ideas

- Ordering/Priority systems (explicitly decided as "Unordered" for now).
- Complex group filtering logic (beyond simple string tag).

</deferred>

---

*Phase: 37-core-discovery*
*Context gathered: 2026-02-03*
