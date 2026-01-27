# Phase 11: Cleanup - Context

**Gathered:** 2026-01-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Remove all deprecated APIs (`NewApp()`, `AppOption`, `App.Provide*` methods, reflection-based helpers) and update all examples/tests to use the generic fluent `For[T]()` registration pattern. Full codebase cleanup including lint/format pass.

</domain>

<decisions>
## Implementation Decisions

### Example update scope
- Full rewrite of examples, not just minimal fixes
- Examples should comprehensively showcase all features
- Feature-focused organization (config-example/, di-example/, lifecycle-example/)
- Descriptive directory names, not numbered
- Purpose: demonstrate all patterns users would need

### Migration documentation
- Minimal — CHANGELOG notes only
- Simple list of what was removed and what replaces it
- No before/after code snippets needed
- README should be rewritten to align with new examples structure
- No external users to notify — internal use only

### Deprecation approach
- Hard delete immediately — no staged deprecation
- Order: remove deprecated code first, then fix examples/tests
- Tests should be improved while updating (better coverage, clearer names)
- Full codebase cleanup — lint/format everything, not just affected files

### Claude's Discretion
- Exact example structure and which features to showcase
- Test organization and naming conventions
- Lint/format tool choices (use existing project config)
- Order of operations within the "remove first" approach

</decisions>

<specifics>
## Specific Ideas

- Examples should be comprehensive enough that users see all gaz features demonstrated
- Clean break from old patterns — no backward compatibility concerns
- Take opportunity to improve overall code quality, not just remove deprecated code

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 11-cleanup*
*Context gathered: 2026-01-27*
