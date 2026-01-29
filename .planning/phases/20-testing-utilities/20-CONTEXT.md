# Phase 20: Testing Utilities (gaztest) - Context

**Gathered:** 2026-01-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Testing DI apps is easy with proper utilities and automatic cleanup. Deliver a `gaztest` package that wraps GAZ for test scenarios with test-friendly defaults, automatic cleanup, and mock injection.

</domain>

<decisions>
## Implementation Decisions

### Builder API Design
- Method chaining pattern: `gaztest.New(t).WithTimeout(5s).Replace(mock).Build()`
- Explicit `Build()` call required to get the test app
- `Build()` returns `(App, error)` — caller handles errors
- Return type from Build() — Claude's discretion (wrapper type or gaz.App)

### Mock/Replacement Ergonomics
- Type inference from argument: `Replace(mockInstance)` infers type from mock
- Single `Replace()` call per mock (not variadic, not chained)
- `Replace()` must be called before `Build()` (on the builder)
- Replacing a type not in container returns error from `Build()`

### Assertion Behavior
- Only `Require*` methods (no `Assert*` variants)
- Simple error messages — no elaborate diagnostics
- Uses `t.Fatal()` to stop test on failure
- Auto cleanup via `t.Cleanup()` — stop is automatic even if test forgets

### Timeout Configuration
- Default timeout: 5 seconds for test apps
- Single timeout for all operations (start and stop share it)
- Override via builder: `WithTimeout(duration)`
- Auto-cleanup timeout failure: test fails with timeout error

### Claude's Discretion
- Whether Build() returns *gaz.App or custom TestApp wrapper
- Internal implementation of cleanup registration
- Error message formatting details

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

*Phase: 20-testing-utilities*
*Context gathered: 2026-01-29*
