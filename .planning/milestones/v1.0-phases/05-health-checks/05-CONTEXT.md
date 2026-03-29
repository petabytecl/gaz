# Phase 5: Health Checks - Context

**Gathered:** 2026-01-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Applications expose production-ready health endpoints for monitoring and orchestration (Kubernetes). This phase covers the check interface, registry, execution logic, and HTTP exposition. It supports Liveness, Readiness, and Startup probes with distinct behaviors.

</domain>

<decisions>
## Implementation Decisions

### Check Interface
- **Return Type:** Rich Result struct containing Status (Up/Down/Degraded), Details map, and Error.
- **Signature:** Context is mandatory (`func(context.Context) ...`).
- **Dependencies:** Checks are services resolved via DI (implementing `HealthChecker` interfaces), not simple closures.
- **Panic Handling:** Let it crash the application (no recovery wrapper).
- **Configuration:** Per-check options (timeout, criticality) via Functional Options pattern during registration.
- **Naming:** Auto-derived from type/function name by default, with override option.

### Liveness vs Readiness
- **Distinction:** Interface Segregation. Separate interfaces for `LivenessChecker`, `ReadinessChecker`, and `StartupChecker`.
- **Types:** Explicitly support 3 types: Liveness, Readiness, and Startup.
- **Optional Checks:** Support "Optional" checks that report failure in details but do NOT fail the overall probe status.
- **Shutdown Behavior:** Readiness probes must immediately return 503 when the app receives a shutdown signal (before full drain).

### Output Format
- **Schema:** Follow IETF Health Check draft (Standard RFC JSON format).
- **Status Mapping:** 
  - **Readiness:** Degraded/Warn -> 503 Service Unavailable (stop traffic).
  - **Liveness:** Degraded/Warn -> 200 OK (don't restart).
- **Verbosity:** Brief output by default. Full details (metadata, per-check results) only when requested via query param (`?full=true`).
- **Error Exposure:** Internal error messages are sanitized in the output (security focus).

### Endpoint Strategy
- **Exposure:** Expose on a **Dedicated Management Port**, not the main application port.
- **Paths:** Separate paths for each type (e.g., `/live`, `/ready`, `/startup`).
- **Access:** Open access (no auth required), relying on port isolation/network security.
- **Lifecycle:** Management server shuts down **LAST**, continuing to serve probes while the main app drains.

</decisions>

<specifics>
## Specific Ideas

- "Degraded should 503 on readiness but 200 on liveness (think kubernetes health probe workflow)"
- Strict separation of probe types via Go interfaces.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 05-health-checks*
*Context gathered: 2026-01-26*
