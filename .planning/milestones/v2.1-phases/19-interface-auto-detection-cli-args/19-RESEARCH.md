# Phase 19: Interface Auto-Detection + CLI Args - Research

**Researched:** Thu Jan 29 2026
**Domain:** Dependency Injection & CLI Integration
**Confidence:** HIGH

## Summary

This phase implements automatic detection of `Starter` and `Stopper` interfaces on services, ensuring their lifecycle methods are called without explicit registration. It also exposes CLI arguments to the DI container.

The research confirms that `gaz` already has the core `Starter`/`Stopper` interfaces and execution logic in place (`di/service.go`). The missing piece is the **detection** logic in `HasLifecycle()`, which currently only checks for explicit hooks. Additionally, the execution logic needs to be updated to respect the "explicit hooks replace interface" precedence rule.

For CLI integration, `gaz/cobra.go` provides the `bootstrap` entry point where `cobra` commands and arguments can be captured and registered into the container before the app builds.

**Primary recommendation:** Implement `HasLifecycle` checks using generic type assertions (`any(new(T)).(Starter)`) and update `gaz/cobra.go` to inject `CommandArgs`.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 | CLI Framework | Existing project dependency, industry standard |
| `reflect` (stdlib) | - | Type Inspection | Used for interface checks if generics fall short (though generics likely suffice) |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `context` | - | Lifecycle Context | Passing context to `OnStart`/`OnStop` |

## Architecture Patterns

### Interface Auto-Detection
The `di` package already defines:
```go
type Starter interface { OnStart(context.Context) error }
type Stopper interface { OnStop(context.Context) error }
```

**Current State:**
- `baseService` handles execution logic: `if starter, ok := instance.(Starter); ok { ... }`
- **Gap:** `HasLifecycle()` only returns `true` if `len(hooks) > 0`. If no hooks are registered, the `App` ignores the service for lifecycle purposes.

**Proposed Pattern:**
Modify `lazySingleton[T]` and `eagerSingleton[T]` to override `HasLifecycle()`:
```go
func (s *lazySingleton[T]) HasLifecycle() bool {
    // 1. Check explicit hooks
    if s.baseService.HasLifecycle() {
        return true
    }
    // 2. Check T (Value receiver)
    var z T
    if _, ok := any(z).(Starter); ok { return true }
    if _, ok := any(z).(Stopper); ok { return true }
    
    // 3. Check *T (Pointer receiver) - crucial for "struct value" services with pointer hooks
    if _, ok := any(new(T)).(Starter); ok { return true }
    if _, ok := any(new(T)).(Stopper); ok { return true }
    
    return false
}
```

### CLI Arguments Injection
**Integration Point:** `gaz/cobra.go` -> `bootstrap` function.

**Pattern:**
1.  **Define Type:** `CommandArgs` struct in `gaz` package.
2.  **Register:** In `bootstrap`, create `CommandArgs` from `cmd` and `args`, and register as an instance.
3.  **Access:** Provide `GetArgs(c *Container) []string` helper.

```go
// gaz/cobra.go
func (a *App) bootstrap(ctx context.Context, cmd *cobra.Command, args []string) error {
    // Register CommandArgs BEFORE Build()
    cmdArgs := &CommandArgs{Command: cmd, Args: args}
    gaz.For[*CommandArgs](a.container).Instance(cmdArgs)
    
    // ... Build() ...
}
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI Framework | Custom flag parsing | `cobra` | Already integrated, robust, standard |
| Interface Checks | Complex reflection | `any(new(T)).(Interface)` | Go generics provide typesafe runtime checks |

## Common Pitfalls

### Pitfall 1: Pointer Receiver on Value Type
**What goes wrong:** User registers `gaz.For[MyStruct]`. `MyStruct` has `func (s *MyStruct) OnStart(...)`.
**Why it happens:** `di` stores `MyStruct`. Calling `instance.(Starter)` fails because `MyStruct` doesn't implement it; `*MyStruct` does.
**How to avoid:**
1.  `HasLifecycle` must check `new(T)` (which is `*T`).
2.  `Start` execution must pass `&instance` (address) if `T` is a value type and `instance` fails the interface check but `&instance` passes.
    - *Note:* `lazySingleton[T]` stores `instance T`, so it can take `&s.instance`.

### Pitfall 2: Precedence Logic
**What goes wrong:** Both explicit hooks and interface methods run, or interface runs when explicit hook was intended to replace it.
**Requirement:** Explicit `.OnStart()` **replaces** `Starter` interface.
**Fix:**
```go
func (s *baseService) runStartLifecycle(...) {
    if len(s.startHooks) > 0 {
        return s.runStartHooks(...) // Explicit ONLY
    }
    // Interface ONLY if no explicit hooks
    if starter, ok := instance.(Starter); ok { ... }
}
```

### Pitfall 3: Validating CommandArgs Availability
**What goes wrong:** `GetArgs` called when not using Cobra (e.g. tests).
**How to avoid:** `Resolve` might fail. `GetArgs` should handle error or return nil/empty. Tests using `gaztest` might need to provide mock args if the service depends on them.

## Code Examples

### 1. Correct HasLifecycle Implementation (Generic)
```go
func (s *lazySingleton[T]) HasLifecycle() bool {
    if s.baseService.HasLifecycle() { return true }
    
    // Check T and *T
    var z T
    _, isStarter := any(z).(Starter)
    _, isStopper := any(z).(Stopper)
    _, isPtrStarter := any(new(T)).(Starter)
    _, isPtrStopper := any(new(T)).(Stopper)
    
    return isStarter || isStopper || isPtrStarter || isPtrStopper
}
```

### 2. Passing Address for Lifecycle
```go
func (s *lazySingleton[T]) Start(ctx context.Context) error {
    if !s.built { return nil }
    
    // If T is value type but needs pointer for interface
    // Note: This logic might need to be inside runStartLifecycle or handled by passing &s.instance
    // Simpler: Always pass &s.instance if T is not a pointer?
    // Actually, runStartLifecycle takes 'any'.
    
    // Best approach:
    // If T implements, pass s.instance.
    // If *T implements and T doesn't, pass &s.instance.
    
    var inst any = s.instance
    if _, ok := inst.(Starter); !ok {
        // Try pointer
        ptr := &s.instance
        if _, ok := any(ptr).(Starter); ok {
            inst = ptr
        }
    }
    return s.runStartLifecycle(ctx, inst)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Explicit `.OnStart` required | Interface Auto-Detection | v2.1 | Less boilerplate, clearer lifecycle intent |
| No CLI args in DI | `CommandArgs` injected | v2.1 | Services can access CLI context cleanly |

## Open Questions

1. **Transient Services:** Should they support lifecycle interfaces?
   - **Recommendation:** No. Transient services are created on demand and not managed by the app lifecycle loop. `HasLifecycle` should remain `false`.

## Sources

### Primary (HIGH confidence)
- `gaz/di/service.go` - Existing lifecycle implementation
- `gaz/cobra.go` - Existing Cobra integration
- `spf13/cobra` - Official docs/code

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Core Go + Cobra
- Architecture: HIGH - Extending existing patterns
- Pitfalls: HIGH - Identified specific pointer/value edge cases

**Research date:** Thu Jan 29 2026
**Valid until:** 30 days
