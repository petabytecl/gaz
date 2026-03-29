# Phase 16: EventBus - Context

**Gathered:** 2026-01-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Type-safe in-process pub/sub with generics API and DI integration. Publishers send events, subscribers handle them asynchronously. No cross-process messaging, no persistence, no external brokers.

</domain>

<decisions>
## Implementation Decisions

### Event typing
- Type + optional topic for flexible routing (e.g., `Subscribe[UserCreated]("admin")`)
- Explicit wildcard to subscribe to all events of a type (e.g., `Subscribe[T]()` or `Subscribe[T]("*")`)
- Events must implement an Event interface (not any arbitrary type)

### Handler errors
- Recover from panics and continue delivering to other subscribers
- No guaranteed handler ordering (unordered invocation)
- Handlers receive context for cancellation awareness

### Async behavior
- Fire-and-forget by default: Publish returns immediately, handlers run concurrently
- Block when buffer is full (no dropping)
- Per-subscription buffer size configuration
- Same Publish API regardless of subscriber sync/async mode

### Subscription lifecycle
- Manual unsubscribe via `Subscription.Unsubscribe()` method
- Drain on close: `bus.Close()` waits for all in-flight handlers to complete
- Silent no-op for publishing to closed bus (idempotent, don't panic)
- Lifecycle interface integration (Start/Stop like Workers) for DI discovery

### Claude's Discretion
- Event interface design (minimal methods needed)
- Handler function signature (whether to return error)
- Default buffer size for async subscriptions
- Exact wildcard syntax

</decisions>

<specifics>
## Specific Ideas

- Should feel consistent with existing Workers and Cron patterns (lifecycle integration)
- Correction from roadmap: async fire-and-forget is the default, not sync blocking

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope

</deferred>

---

*Phase: 16-eventbus*
*Context gathered: 2026-01-29*
