# Context

## Domain Glossary

### Lifecycle plan

A lifecycle plan is the application runtime plan derived from the DI container's service graph. It owns the policy for which services participate in lifecycle hooks, which services are excluded because another runtime module owns them, and the startup and shutdown order used by the App runtime.

The lifecycle plan is distinct from lifecycle execution. Planning decides what should happen and in what order; execution starts, stops, rolls back, logs, and enforces context deadlines.

The lifecycle plan also owns runtime participant classification for App-managed workers, cron jobs, and framework participants such as the event bus and scheduler. Worker supervision remains the worker manager's implementation concern; the lifecycle plan decides which workers the App runtime hands to that manager.

Runtime participant registration is part of the lifecycle plan's policy surface. A classified worker or cron job must resolve and register successfully during `App.Build()`; otherwise the application fails to build with an actionable error instead of silently dropping the participant.

### Lifecycle session

A lifecycle session is one run of a built App. It owns the runtime orchestration policy around entering the running state, waiting for shutdown triggers, invoking lifecycle execution, enforcing process-level shutdown deadlines, and leaving the running state.

The lifecycle session is distinct from the lifecycle plan and lifecycle execution. The lifecycle plan decides what should run; lifecycle execution starts and stops planned participants; the lifecycle session coordinates when execution begins and how shutdown is triggered and finalized.

### Configured module registration

Configured module registration is the shared App-module pattern for wiring a module-owned configuration type into DI. The module owns its public adapter and flags, while the internal configured-module helper owns the repeated mechanics: start from defaults, overlay ProviderValues when available, apply the module's defaulting policy, and validate before exposing the config.

This keeps package-level modules stable as public entry points while centralizing the registration rule that makes flags, config files, environment values, and DI providers agree on one typed config instance.

### Unified server bridge

The unified server bridge is the App-module pattern for installing the transport stack on one port. It owns the policy that makes the gRPC server register services without binding its own listener, then hands that gRPC adapter to Vanguard so Vanguard serves gRPC, Connect, gRPC-Web, REST transcoding, and non-RPC HTTP routes from the single public listener.

Standalone transport modules remain valid on their own. The unified server bridge only owns the extra composition rule needed when callers choose the single-port default.

### Module identity

Module identity is the stable name a module uses for duplicate detection, auto-registration skip logic, and diagnostics. The owning package exposes the identity when callers or sibling adapter packages need to reason about whether the feature module is installed.

A flags adapter may have its own module identity when applying the adapter is not the same thing as installing the feature infrastructure. That distinction must be explicit instead of encoded as an ad hoc string in the adapter.
