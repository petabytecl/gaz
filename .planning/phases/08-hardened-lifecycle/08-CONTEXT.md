# Phase 8: Hardened Lifecycle - Context

**Gathered:** 2026-01-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Guarantee process termination within a fixed timeout, preventing zombie processes during shutdown. This phase implements hard timeout enforcement, double Ctrl+C force exit, and blame logging for hanging hooks.

</domain>

<decisions>
## Implementation Decisions

### Timeout configuration
- **Dual timeout system:** Per-hook timeout (10s default) + global timeout (30s default)
- **Configuration methods:** Builder method `app.WithShutdownTimeout()` for code, `GAZ_SHUTDOWN_TIMEOUT` env var for ops override
- Per-hook timeout if exceeded: hook is cancelled, next hook proceeds
- Global timeout if exceeded: immediate force exit regardless of hook states

### Blame logging format
- **Content:** Hook name + elapsed time + registration context (what it was registered as)
- **Output:** Configured logger first, direct stderr as fallback (guaranteed output even if logger is broken)
- Example: `shutdown: DatabasePool exceeded 10s timeout (registered as "db.pool.close", elapsed: 10.3s)`

### Shutdown orchestration
- **Order:** LIFO (last registered shuts down first) - reverse of startup order
- **Concurrency:** Sequential - one hook must complete before the next starts
- **Error handling:** Run all hooks, collect all errors, report combined errors at end
- **Timeout awareness:** Pass `context.Context` with deadline to each hook so they can check `ctx.Done()`

### Double-SIGINT behavior
- **First Ctrl+C:** Log hint: "Shutting down gracefully... (Ctrl+C again to force)"
- **Second Ctrl+C:** Brief log "force exit" then immediate `os.Exit(1)`
- **Exit code:** 1 for both timeout and force exit (abnormal termination)
- **SIGTERM:** One chance only - no double-SIGTERM behavior, SIGKILL is the force option for SIGTERM

### Claude's Discretion
- Per-hook timeout API design (option on registration vs hook declares timeout)
- Whether to log successful hook completions alongside failures
- Log level for blame messages (ERROR vs WARN)
- Exact message wording and format

</decisions>

<specifics>
## Specific Ideas

- The hint on first Ctrl+C should feel like kubectl or docker-compose - familiar to ops people
- Blame log should be immediately useful for debugging production hangs without needing to add more logging

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope

</deferred>

---

*Phase: 08-hardened-lifecycle*
*Context gathered: 2026-01-27*
