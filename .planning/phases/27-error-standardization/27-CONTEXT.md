# Phase 27: Error Standardization - Context

**Gathered:** 2026-01-31
**Status:** Ready for planning

<domain>
## Phase Boundary

Consolidate and namespace all sentinel errors with consistent formatting. All errors move to a single source of truth (`gaz/errors.go`), follow `ErrSubsystemAction` naming, use `"pkg: context: %w"` wrapping format, and properly support `errors.Is/As`.

</domain>

<decisions>
## Implementation Decisions

### Error naming convention
- Subsystem prefix style: `ErrDI*`, `ErrConfig*`, `ErrWorker*`, `ErrCron*`, `ErrHealth*`, `ErrEventBus*`
- Subsystem prefixes match package names exactly (no abbreviations)
- Action part uses descriptive names: `NotFound`, `Missing`, `Stopped`, `Invalid`
- Error message format: structured `"pkg: action"` (e.g., `errors.New("di: not found")`)

### Error organization
- Single `gaz/errors.go` as source of truth for all sentinel errors
- Clean break (v3 approach) — remove existing errors from subsystem packages, no deprecation period
- Existing errors in subsystem packages are deleted and replaced

### Wrapping format
- All wrapping uses `"pkg: context: %w"` format (e.g., `fmt.Errorf("di: resolve %s: %w", name, err)`)
- Wrap at package boundaries — always add context when crossing boundaries
- Always use `%w` verb so `errors.Is` works through wrapping chain

### Error types vs sentinels
- Hybrid approach: sentinels for common/simple errors, typed errors when caller needs context to recover
- Typed errors for: DI resolution failures, config parsing errors, lifecycle errors
- Typed error names: `ActionError` style (e.g., `ResolutionError`, `ParseError`) — subsystem context comes from package location
- Typed errors implement `Is`, `As`, and `Unwrap` for proper error chaining

### Claude's Discretion
- Whether to re-export errors from subsystem packages for convenience
- Organization within `gaz/errors.go` (grouped by subsystem vs flat list)
- Specific identifiers to include in wrap context (service names, types)

</decisions>

<specifics>
## Specific Ideas

- Follows v3 clean-break philosophy — no deprecation period, just migrate
- Error messages should be debuggable: when wrapping adds context, developers can trace the error path
- Typed errors are for recovery scenarios where the caller needs metadata (e.g., "which service wasn't found?")

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 27-error-standardization*
*Context gathered: 2026-01-31*
