# Phase 12: DI Package - Context

**Gathered:** 2026-01-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Extract DI into `gaz/di` subpackage that works standalone without gaz App. Container, For[T](), Resolve[T]() move to new package. Root gaz package maintains backward compatibility through wrapper types.

</domain>

<decisions>
## Implementation Decisions

### Standalone container creation
- Constructor: `di.New()` returns `*Container` (no error)
- No options on constructor, configure via methods if needed later
- Validation: lazy on resolve, not upfront
- Resolution errors return error (not panic)
- Thread-safe for concurrent resolution (maintain current behavior)
- `For[T](c)` pattern: container as first argument, not method on container

### Backward compatibility approach
- Wrapper types in root gaz package (not type aliases)
- Breaking changes acceptable if they result in cleaner API
- `gaz.For[T]()` continues to work for gaz.App users
- Wrapper types maintained indefinitely (not deprecated)

### Package public API
- Export registration modes: Singleton, Transient, Eager as distinct types
- Expose `Service[T]` interface for power users/advanced use cases
- Both introspection methods: `c.List()` and `c.Has[T]()`
- Claude decides minimal core exports (Container, For, Resolve, builders)

### Usage ergonomics
- gaz.App is superset of di.Container (App has more features)
- Add `MustResolve[T]()` that panics on failure (for tests/init)
- Keep current `For[T](container)` pattern for registration
- Add test helpers (e.g., `di.NewTestContainer()`)
- Detailed errors with context ("missing dependency X needed by Y")
- Error on duplicate registration (not silent override)
- Any type allowed (not limited to interfaces)
- Documentation: both godoc examples and package README

### Claude's Discretion
- Cleanup/disposal mechanism for singletons (if needed)
- Embedding vs full wrapping for backward compat wrappers
- Exact minimal export surface for di package
- Test helper API design

</decisions>

<specifics>
## Specific Ideas

- Constructor should feel like `http.NewRequest` or `bytes.NewBuffer` — short and idiomatic
- MustResolve is specifically for test setup and main() initialization
- Error messages should help debugging: "cannot resolve *UserService: missing dependency *Database registered at inject.go:42"

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 12-di*
*Context gathered: 2026-01-28*
