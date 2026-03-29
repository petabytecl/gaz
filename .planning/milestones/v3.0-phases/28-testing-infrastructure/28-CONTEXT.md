# Phase 28: Testing Infrastructure - Context

**Gathered:** 2026-01-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Comprehensive test support for v3 patterns. This phase enhances gaztest, adds per-package test helpers to each subsystem, creates testing documentation, and ensures example tests demonstrate all v3 patterns. The goal is that users can easily test gaz applications using v3 APIs.

</domain>

<decisions>
## Implementation Decisions

### Test helper scope
- Full test utilities: mock factories, test configs, AND assertion helpers per subsystem
- Assertion helpers use `Require*` prefix (testify style): RequireHealthy, RequireWorkerStarted
- Test configs provide both patterns: `TestConfig()` for defaults, `NewTestConfig(opts...)` for customization

### Test helper location
- Claude's discretion on where helpers live (subsystem packages vs gaztest/) based on import cycles and conventions

### Testing guide content
- Layered approach: quick reference section plus detailed guide sections
- Lives in `gaztest/README.md` inside the test package
- Uses testify-specific mocking examples
- Covers both unit and integration testing with guidance on when to use each

### Example test coverage
- Godoc examples (Example_* functions) plus standalone scenario files
- Comprehensive coverage: core v3 patterns, subsystem-specific patterns, AND edge cases (error recovery, graceful shutdown)
- Godoc examples must be runnable; standalone examples can be illustrative
- Clean up old pre-v3 patterns from existing examples

### Example location
- Claude's discretion on standalone example directory location based on project structure

### gaztest API
- Add `WithModules(m ...di.Module)` for registering modules (variadic, not singular)
- Add `WithConfigMap(map[string]any)` for injecting raw config values in tests
- Keep manual lifecycle control: RequireStart()/RequireStop() as-is
- Add `RequireResolve[T](t, app)` helper that fails test on resolution error

### Claude's Discretion
- Test helper file locations (per-subsystem vs centralized)
- Standalone example directory location
- Which specific assertion helpers each subsystem needs
- Implementation details of test utilities

</decisions>

<specifics>
## Specific Ideas

- Testify is the mocking framework to use for examples and patterns
- RequireResolve should fail the test immediately on error, not return error
- Config injection via map allows setting arbitrary config keys for test scenarios

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope

</deferred>

---

*Phase: 28-testing-infrastructure*
*Context gathered: 2026-01-31*
