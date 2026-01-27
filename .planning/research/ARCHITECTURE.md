# Architecture Patterns

**Domain:** Security & Hardening v1.1
**Researched:** Mon Jan 26 2026

## Recommended Architecture

The integration strategy inserts "Hardening Gates" at the extreme ends of the application lifecycle.

### 1. The Startup Gate (Config Validation)

**Current Flow:**
`Load Config` -> `Unmarshal` -> `Start DI`

**New Flow:**
`Load Config` -> `Unmarshal` -> **`Validate (Gate)`** -> `Start DI`

**Data Flow:**
1.  **Raw Config Source** (Env/File)
2.  **Koanf** unmarshals into `Config` Struct.
3.  **Validator** inspects `Config` Struct tags.
    *   *If Invalid:* Print clear field-level errors to STDERR and `os.Exit(1)`.
    *   *If Valid:* Pass `Config` struct to DI Container.

### 2. The Shutdown Guard (Timeout Enforcement)

**Current Flow:**
`Signal` -> `Lifecycle.Stop()` (Potential Hang)

**New Flow:**
`Signal` -> **`ShutdownGuard(ctx)`** -> `Lifecycle.Stop()`

**Logic:**
```go
func ShutdownGuard(lifecycle LifecycleManager) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    done := make(chan struct{})
    go func() {
        lifecycle.Stop() // Existing graceful logic
        close(done)
    }()

    select {
    case <-done:
        // Graceful exit
    case <-ctx.Done():
        // Force exit (Timeout exceeded)
        log.Error("Shutdown timed out, forcing exit")
        os.Exit(1)
    }
}
```

## Component Boundaries

| Component | Responsibility | Changes Needed |
|-----------|---------------|----------------|
| **Config Loader** | Loads data, now explicitly validates it. | Add `validator.ValidateStruct` call after unmarshal. |
| **Config Structs** | Defines the data shape and constraints. | Add `validate:"..."` tags to fields. |
| **Lifecycle Manager** | Orchestrates start/stop. | No internal changes, but needs to be wrapped or called with a context. |
| **Main Entrypoint** | Wires everything together. | Update wiring to inject Validator and wrap Shutdown. |

## Patterns to Follow

### Pattern 1: The Strict Config Object
**What:** The `Config` struct passed to the DI container is *guaranteed* to be valid.
**When:** Always.
**Why:** Consumers (Services) shouldn't need to check `if config.Port == 0`. They can assume valid state.

### Pattern 2: Context-Driven Shutdown
**What:** Propagate `context.Context` down to all `OnStop` hooks.
**Example:**
```go
// In Lifecycle Manager
func (m *Manager) Stop(ctx context.Context) error {
    // Pass ctx to services so they know how much time they have left
    return service.Shutdown(ctx)
}
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Lazy Validation
**What:** Validating config values inside the service constructors or `OnStart` methods.
**Why bad:** Scatters validation logic across the codebase; hard to audit; fails late.
**Instead:** Centralized validation at startup.

### Anti-Pattern 2: The "Just Kill It" Shutdown
**What:** Ignoring graceful shutdown and relying on the orchestrator (e.g., K8s) to SIGKILL.
**Why bad:** Drops in-flight database writes or requests; causes alerts.
**Instead:** Try graceful shutdown, but fallback to timeout.

## Scalability Considerations

*   **Validation:** Negligible overhead (runs once at startup).
*   **Shutdown:** Timeout values might need tuning as the application grows (e.g., if draining 10k connections takes >30s). Make the timeout configurable.
