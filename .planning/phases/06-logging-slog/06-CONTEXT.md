# Phase 6: Logging (slog) - Context

**Gathered:** 2026-01-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Applications have structured logging with context propagation. Framework provides pre-configured slog.Logger via DI, context propagation, framework events logging, and custom handlers.

</domain>

<decisions>
## Implementation Decisions

### Output Format
- **Environment-based:** JSON in production, Text (tinted/colored) in development.
- **Override:** `LOG_FORMAT` environment variable overrides the default.
- **Output Destination:** Configurable, defaulting to Stdout.

### Context Propagation
- **Standard Fields:** TraceID, RequestID, UserID, TenantID.
- **Extraction:** Middleware extracts these from headers and puts them into context.
- **Access:** Custom `slog.Handler` automatically reads these fields from context and adds them to log records.
- **Key Naming:** Use OpenTelemetry standard keys (e.g., `trace_id`, `span_id`).
- **Helper:** Provide a `LogWith(ctx, keys...)` helper to add arbitrary scoped fields to context.

### Level Configuration
- **Configuration Source:** Use the existing ConfigManager (Phase 4).
- **Runtime Updates:** Log level can be updated dynamically via the Management API (Phase 5 integration).
- **Granularity:** Global log level only (no per-module levels).
- **Default:** INFO.

### Default Attributes
- **Fields:** App/Service Name, Version, Environment, Hostname, PID.
- **Source:** Values come from App Config (passed to `gaz.New`).

### Claude's Discretion
- Exact implementation of the tinted handler (or selection of a library).
- Middleware implementation details (integrating with HTTP stack).
- Exact Management API endpoint design for log level updates.

</decisions>

<specifics>
## Specific Ideas

- "I want the dev experience to be nice (colored logs), but prod to be machine-parseable (JSON)."
- "Follow OpenTelemetry standards for field names to ensure compatibility with observability tools."

</specifics>

<deferred>
## Deferred Ideas

- Per-module log levels (future enhancement).
- Log sampling for high-volume environments (future enhancement).

</deferred>

---

*Phase: 06-logging-slog*
*Context gathered: 2026-01-27*
