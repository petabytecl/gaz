# Domain Pitfalls

**Domain:** Configuration & Lifecycle
**Researched:** Mon Jan 26 2026

## Critical Pitfalls

### Pitfall 1: Zero-Value Ambiguity
**What goes wrong:** A required integer field (e.g., `MaxRetries`) is missing in config, so it defaults to `0`. `0` might be a valid value logically, but "missing" was the intent.
**Why it happens:** Go's zero values make "missing" and "default" indistinguishable.
**Prevention:** Use pointer types (`*int`) for optional fields where `0` is meaningful, or use `validate:"required"` to ban the zero value if `0` is invalid.

### Pitfall 2: The Blocked Shutdown
**What goes wrong:** A service's `OnStop` method hangs indefinitely (e.g., waiting for a channel that never closes).
**Consequences:** The application never exits, requiring manual SIGKILL or orchestrator intervention.
**Prevention:** The "Shutdown Guard" pattern (wrapping stop in a timeout) is the direct mitigation for this.

## Moderate Pitfalls

### Pitfall 3: Unfriendly Validation Errors
**What goes wrong:** The app crashes with `Field 'Port' failed on tag 'required'`.
**Why it happens:** Raw validator output is technical.
**Prevention:** Translate validator errors into human-readable messages (e.g., "Configuration Error: 'Port' is required but was not found in config.yaml or ENV").

### Pitfall 4: Context Cancellation Propagation
**What goes wrong:** The root context is cancelled for shutdown, but downstream DB drivers or HTTP clients aren't using that context.
**Consequences:** They keep trying to work while the app is dying.
**Prevention:** Ensure all blocking calls (DB queries, HTTP requests) accept and respect `ctx`.

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| **Validation** | Existing config files might fail validation immediately. | Audit existing deployments/configs before deploying the hardened version. |
| **Shutdown** | Setting the timeout too short (e.g., 1s). | Default to a safe value (30s) and make it configurable. |
