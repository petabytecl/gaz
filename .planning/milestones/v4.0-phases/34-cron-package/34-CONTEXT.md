# Phase 34: Cron Package - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace `robfig/cron/v3` with internal `cron/internal/` package. The internal implementation handles cron expression parsing, schedule calculation, and job scheduling. The existing `cron/scheduler.go` will switch from robfig/cron to internal.

</domain>

<decisions>
## Implementation Decisions

### Feature scope
- Minimal + future-proofing: 5-field parsing, @descriptors, SkipIfStillRunning, Start/Stop, plus Remove() and entry introspection
- Include @every support (@every 5m, @every 1h30m) in addition to standard descriptors (@daily, @hourly, @weekly, @monthly, @yearly)
- Optional seconds support: 6th field for seconds when explicitly configured
- Include introspection: Entries(), Entry(id), JobCount() for monitoring

### API surface
- Modernized API: Cleaner Go 1.21+ idioms, context-aware, slog-native
- Functional options: WithLogger, WithChain, WithLocation for configuration
- Use slog.Logger directly instead of abstract Logger interface
- Context-aware Job: Run(ctx context.Context) instead of Run()

### DST edge cases
- Support CRON_TZ= prefix for per-schedule timezones
- Spring forward: Run at next valid time (if 2:30 AM doesn't exist, run at 3:00 AM)
- Fall back: Run once at first occurrence (standard wall clock behavior)
- Document DST handling in package godoc

### Migration approach
- Direct replacement: cron/scheduler.go uses cron/internal types directly, no adapter
- Existing tests verify parity: cron/scheduler_test.go covers real usage patterns
- Remove cron/logger.go (robfig adapter) since cron/internal uses slog directly
- Start from reference: Copy _tmp_trust/cron/internal/, adapt imports and logger

### Claude's Discretion
- Internal implementation details of schedule calculation
- Test structure and coverage approach
- Error message wording

</decisions>

<specifics>
## Specific Ideas

- Reference implementation exists at `_tmp_trust/cron/internal/` - feature complete with tests
- Current usage in cron/scheduler.go is straightforward: New(), AddJob(), Start(), Stop()
- SkipIfStillRunning wrapper is already used - must be supported

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope

</deferred>

---

*Phase: 34-cron-package*
*Context gathered: 2026-02-01*
