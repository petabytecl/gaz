# Phase 32: Backoff Package - Context

**Gathered:** 2026-02-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace `jpillora/backoff` dependency with internal `backoff/` package. Worker supervisor uses backoff for restart delays. The reference implementation in `_tmp_trust/srex/backoff/` (from cenkalti/backoff) provides the source material.

</domain>

<decisions>
## Implementation Decisions

### Feature Scope
- Full toolkit from reference — not minimal core
- Adapt from srex reference, apply gaz conventions (not verbatim copy)
- Include Retry() and RetryNotify() for automatic retry loops
- Include Ticker for periodic backoff-based operations

### API Design
- Interface-based: BackOff interface with NextBackOff() and Reset() methods
- Functional options: NewExponentialBackOff(opts ...Option)
- Include Stop constant (-1) to signal max retries reached
- Worker updates to use internal package directly (no adapter layer)

### Package Extras
- Include all variants: ZeroBackOff, StopBackOff, ConstantBackOff, ExponentialBackOff
- Include context support: WithContext() wrappers for cancellation
- Include PermanentError type for non-retryable errors
- Include MaxElapsedTime option for total duration limits

### Claude's Discretion
- Exact jitter implementation (rand source, thread safety approach)
- Internal timer/clock abstractions for testing
- File organization within backoff/ package

</decisions>

<specifics>
## Specific Ideas

- Reference implementation: `_tmp_trust/srex/backoff/` (cenkalti/backoff port)
- Current usage: `worker/supervisor.go` and `worker/backoff.go`
- jpillora/backoff uses Duration() method; migrate to NextBackOff() interface style
- Worker's BackoffConfig can be simplified or removed after migration

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 32-backoff-package*
*Context gathered: 2026-02-01*
