# Phase 19: Interface Auto-Detection + CLI Args - Context

**Gathered:** 2026-01-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Services with lifecycle interfaces (Starter/Stopper) are automatically detected and their hooks executed. CLI arguments are exposed via DI. Explicit registration takes precedence.

</domain>

<decisions>
## Implementation Decisions

### Lifecycle Error Policy
- **Fail fast + Cleanup**: If `OnStart` fails, stop everything immediately and attempt cleanup of started services.
- **Best Effort on Stop**: If `OnStop` fails, log error and continue stopping other services.
- **Default Timeout**: Auto-detected hooks are subject to a default timeout (e.g. 30s).
- **Verbose Logging**: Log all auto-detected start/stop events (Info level).

### Interface Signatures
- **Strict Signature**:
  - `OnStart(context.Context) error`
  - `OnStop(context.Context) error`
- **Independent**: Service can implement just `Starter` or just `Stopper`.
- **Smart Check**: Inspect both `T` and `*T` for implementation.
- **Signature Mismatch**: Log a warning if a method (e.g., `OnStart`) exists but signature doesn't match.

### CLI Args Access
- **Access Pattern**: `gaz.GetArgs(container)` helper only (no direct injection into constructors).
- **Data Structure**: Raw `[]string`.
- **Content**: Strip index 0 (program name), start at index 1.
- **Immutability**: Args are immutable (read-only copy).

### Conflict Handling
- **Precedence**: Explicit `.OnStart()` replaces `Starter` interface (Explicit Wins).
- **Warning**: Log a warning when an explicit hook overrides an interface implementation.
- **No Opt-out**: No mechanism to disable auto-detection for a specific type (except by not implementing the interface).
- **Execution Order**: Reverse dependency order (standard DI behavior).

### Claude's Discretion
- Specific timeout duration (e.g., 30s vs 10s).
- Exact warning message formats.

</decisions>

<specifics>
## Specific Ideas

- Adhere strictly to standard Go context patterns.
- Ensure `gaz.GetArgs` is thread-safe if needed (though immutable implies safety).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 19-interface-auto-detection-cli-args*
*Context gathered: 2026-01-29*
