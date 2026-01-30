# Phase 24: Lifecycle Interface Alignment - Context

**Gathered:** 2026-01-30
**Status:** Ready for planning

<domain>
## Phase Boundary

Unify interface-based lifecycle management across all service types. Remove fluent OnStart/OnStop hooks from RegistrationBuilder, align worker.Worker interface with di.Starter/Stopper patterns, and ensure automatic lifecycle wiring for any type implementing these interfaces. This is a clean-break v3.0 change.
</domain>

<decisions>
## Implementation Decisions

### Worker interface migration
- Clean break: v3.0 changes the interface in one release, no gradual migration
- Worker interface changes from Start()/Stop() to OnStart(ctx)/OnStop(ctx) error
- Minimal interface: Worker only has Name() + OnStart/OnStop (no Run method)
- Workers continue spawning their own goroutines in OnStart

### Fluent hook removal
- Hard remove: OnStart/OnStop methods deleted from RegistrationBuilder entirely
- Services requiring lifecycle must implement Starter/Stopper interfaces on the type itself
- Auto-detection remains implicit: any Starter/Stopper automatically participates in lifecycle
- No opt-out mechanism: implementing the interface always means lifecycle participation

### Third-party type adaptation
- No Adapt() helper: users create their own wrapper structs for third-party types
- Recommended pattern: composition (type DBWrapper struct { db *sql.DB }) not embedding
- Documentation example deferred to Phase 29 (Documentation & Examples)

### Interface naming consistency
- Keep Starter and Stopper names (-er suffix per Go idiom)
- Keep OnStart(ctx) error and OnStop(ctx) error method signatures
- Claude's discretion: package location (di vs gaz vs new lifecycle package)
- Claude's discretion: whether Worker embeds Starter+Stopper or has matching signatures

### Claude's Discretion
- Package location for Starter/Stopper interfaces
- Whether Worker interface embeds Starter + Stopper or just has matching signatures
- Implementation details for lifecycle auto-detection
- Test migration strategy
</decisions>
