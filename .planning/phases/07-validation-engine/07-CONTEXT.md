# Phase 7: Validation Engine - Context

**Gathered:** 2026-01-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Configuration validation that prevents application startup if validation fails. Users define struct tags (`validate:"required"`) on config structs, and the config manager enforces these constraints at load time. The application exits with clear error messages if validation fails.

</domain>

<decisions>
## Implementation Decisions

### Error Presentation
- Show all validation errors together (not fail-fast on first error)
- Full field path + tag shown in errors: `config.database.host: required field cannot be empty (validate:"required")`
- Plain text format, one error per line
- Include source info in errors (which file or env var the value came from)

### Failure Behavior
- Validation runs at config load time (before any services start)
- Atomic behavior: all or nothing - if validation fails, no config values accessible
- Validation fail-fast at load prevents partial/undefined application state

### Validation Library
- Basic rules sufficient for v1.1: required, basic type constraints
- Custom validation rules NOT needed for v1.1
- Minimize external dependencies - keep framework lightweight

### Cross-field Validation
- Nice to have, not required for v1.1
- If/when implemented: errors should explain the relationship (`password required if auth_type is local`)
- Nested struct validation IS supported - validate recursively through config sections

### Claude's Discretion
- Termination mechanism (os.Exit vs return error)
- Opt-in vs always-on validation mode
- Library choice (go-playground/validator vs custom implementation)
- How cross-field validation is expressed (tags vs Validate() method)

</decisions>

<specifics>
## Specific Ideas

- Error messages should be actionable: show the field path, the constraint, AND where the value came from
- Want users to fix all config issues in one pass, not discover them one at a time

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope

</deferred>

---

*Phase: 07-validation-engine*
*Context gathered: 2026-01-27*
