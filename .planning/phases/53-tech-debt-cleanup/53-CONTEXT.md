# Phase 53: Tech Debt Cleanup - Context

**Gathered:** 2026-03-30
**Status:** Ready for planning
**Mode:** Auto-generated (infrastructure phase)

<domain>
## Phase Boundary

Resolve 3 tech debt items identified in v5.1 milestone audit:
1. Wire logger.NewLoggerWithCloser into App.initializeLogger and close handle on shutdown
2. Update OTEL middleware trace filter in server/vanguard/middleware.go to use health.Config paths
3. Fix server/vanguard/doc.go health path references

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — pure infrastructure phase. Use existing patterns and minimize API surface changes.

</decisions>

<code_context>
## Existing Code Insights

### Relevant Files
- `logger/provider.go` — NewLoggerWithCloser returns (logger, io.Closer)
- `app_build.go` — initializeLogger() calls logger.NewLogger (needs updating)
- `app_shutdown.go` — doStop() handles shutdown (needs closer integration)
- `server/vanguard/middleware.go:148` — OTEL filter with hardcoded health paths
- `server/vanguard/doc.go` — documentation with old health path references
- `server/vanguard/health.go` — buildHealthMux uses health.Config paths

</code_context>

<specifics>
## Specific Ideas

Items directly from v5.1-MILESTONE-AUDIT.md tech_debt section.

</specifics>

<deferred>
## Deferred Ideas

None — this phase IS the debt cleanup.

</deferred>
