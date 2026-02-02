# Phase 35: Health Package + Integration - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace `alexliesenfeld/health` with internal `health/internal/` implementation. Maintain API compatibility with existing health checks. The package provides HTTP handlers for health/liveness endpoints with configurable check execution.

Requirements: HLT-01 through HLT-13 (health/internal package), INT-01 through INT-03 (integration and cleanup).

</domain>

<decisions>
## Implementation Decisions

### Check execution model
- Checks run in parallel (concurrent) for faster response times
- Each check gets its own timeout (configurable per-check, default 5s)
- If a check panics, recover and mark that check as failed (don't crash the whole health check)

### Response output format
- Response format: Claude's discretion (IETF health+json is the baseline per requirements)
- Check details visibility: configurable, hide by default (security: don't expose internals)
- Error messages: configurable per environment (hide in prod, show in dev)

### Check naming/grouping
- Naming convention: Claude's discretion based on IETF conventions
- Grouping structure: Claude's discretion
- Check criticality: support critical vs warning checks
  - Critical checks affect overall status (default)
  - Warning checks report independently but don't fail overall health
  - Default for new checks: critical

### Claude's Discretion
- Response format choice (IETF health+json or alternatives)
- Partial failure reporting strategy
- Check naming convention (flat vs prefixed)
- Check grouping structure (flat list vs nested)

</decisions>

<specifics>
## Specific Ideas

- Security-conscious defaults: hide details and error messages by default
- Parallel execution prioritized over sequential for performance
- Critical vs warning distinction allows non-essential checks to degrade gracefully

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 35-health-package*
*Context gathered: 2026-02-01*
