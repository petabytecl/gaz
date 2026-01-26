# Phase 3: App Builder + Cobra - Context

**Gathered:** 2026-01-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Fluent API for building applications with DI and Cobra CLI integration. Developers can create apps with `gaz.New()`, register services via fluent methods, compose modules, and integrate with existing Cobra commands. Flag binding to config is Phase 4.

</domain>

<decisions>
## Implementation Decisions

### Fluent API design
- Separate Build() and Run() steps — Build() validates and wires, Run() executes lifecycle
- Separate methods per scope: `ProvideSingleton()`, `ProvideTransient()`, etc.
- `gaz.New(opts ...Option)` with functional options for logger, timeout, etc.
- Builder pattern style: Claude's discretion (mutable vs immutable)

### Module composition
- Modules are functions returning `[]Provider` — simple, composable
- Named modules required: `app.Module("database", providers...)` for debugging
- Error only on named duplicate — multiple registrations of same type allowed unless same name
- Module nesting: Claude's discretion

### Cobra integration
- `app.WithCobra(rootCmd)` attaches app to existing Cobra command
- Full DI access in subcommands — each command can resolve from container
- Automatic lifecycle for RunE, with manual override option for commands that don't need full lifecycle
- Flag binding to config is Phase 4 (not this phase)

### Error handling
- Validate all registrations on Build() — fail early, not on first resolution
- Collect all errors on Build() — return combined error listing all problems
- Typed sentinel errors (ErrMissingDependency, ErrCyclicDependency, etc.)
- Strict dependency resolution — missing dependency is always an error

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

- Flag binding to DI config — Phase 4 (Config System)

</deferred>

---

*Phase: 03-app-builder-cobra*
*Context gathered: 2026-01-26*
