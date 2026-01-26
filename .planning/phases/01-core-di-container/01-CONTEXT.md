# Phase 1: Core DI Container - Context

**Gathered:** 2026-01-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Type-safe dependency injection container with generics. Developers can register providers, resolve dependencies, and use struct field injection. Scopes are flat (Singleton, Transient only). Lifecycle hooks and App builder are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Registration API design
- Fluent builder chain: `gaz.For[T](c).Provider(fn)` — generic function entry point
- Both provider signatures allowed: `func() T` and `func() (T, error)`
- Named registrations via `.Named("name")` method in the chain
- Default lazy instantiation; `.Eager()` opts in to startup instantiation
- Default singleton scope; `.Transient()` opts in to new-instance-per-resolve
- Error on duplicate registration (no silent override)
- Explicit `.Replace()` method for intentional override (testing)
- `.Instance(val)` shorthand for pre-built values
- Always return error from registration (no panic variant)
- Provider receives `*Container` to resolve its dependencies

### Resolution behavior
- Returns `(T, error)` — no panic on missing dependency
- Generic function: `gaz.Resolve[T](c)` — mirrors registration pattern
- Named resolution via option: `gaz.Resolve[T](c, gaz.Named("name"))`
- Explicit `Build()` phase instantiates eager services before any resolves
- Circular dependency returns error (cycle is a bug)

### Error messaging
- Full dependency chain in errors: A -> B -> C with root cause
- Sentinel errors: `gaz.ErrNotFound`, `gaz.ErrCycle`, etc. for `errors.Is()`
- Full package paths in error messages: `*github.com/user/app/db.Pool`
- Provider errors wrapped with resolution context

### Struct tag syntax
- Tag format: `gaz:"inject"` — matches package name
- Named injection: `gaz:"inject,name=primary"` — comma-separated modifiers
- Optional fields: `gaz:"inject,optional"` — nil if not registered
- Auto-injection: fields checked and injected automatically on resolution

### Claude's Discretion
- Internal container data structures
- Thread-safety implementation details
- Reflection caching strategy
- Build() validation behavior

</decisions>

<specifics>
## Specific Ideas

- API should feel discoverable — fluent chain guides developers through options
- Mirrors registration and resolution patterns for consistency (`gaz.For[T]` / `gaz.Resolve[T]`)
- Error messages should make dependency problems immediately debuggable

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-core-di-container*
*Context gathered: 2026-01-26*
