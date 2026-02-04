# Phase 42: Refactor Framework Ergonomics - Research

**Researched:** 2026-02-03
**Domain:** Go Framework Design / CLI Integration
**Confidence:** HIGH

## Summary

This phase aims to close the gap between the current verbose boilerplate required to use `gaz` with Cobra/Viper and the desired "zero-config" developer experience. The primary issues are:
1.  **Flag Registration:** Modules define flags, but they are not automatically exposed to the CLI command.
2.  **Order of Operations:** `App.Use()` is typically called before `App.WithCobra()`, causing current flag registration logic to fail (as it depends on `cobraCmd` being present).
3.  **Nested Modules:** The current `App.Use` logic does not recursively register flags for child modules (e.g., `server` module containing `grpc` and `gateway`).
4.  **Boilerplate:** Users must currently manually bind flags to Viper and write their own `RunE` loop with signal handling.

The research confirms that `gaz` already has the building blocks (`gaz.Module` supports `Flags()`), but the wiring in `App` and `WithCobra` is incomplete. The recommended solution involves deferring flag registration until `WithCobra` is called and enhancing `WithCobra` to provide a default blocking run loop.

**Primary recommendation:** Refactor `App` to store module flag functions and apply them lazily in `WithCobra`, and update `WithCobra` to provide a default `RunE` that blocks on the application lifecycle.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `spf13/cobra` | v1.8+ | CLI Framework | Standard Go CLI library |
| `spf13/viper` | v1.18+ | Configuration | Standard Go config library |
| `gaz` | Local | Framework | The framework being improved |

## Architecture Patterns

### 1. Deferred Flag Registration
Since `App.Use()` is called before `App.WithCobra()`, we cannot register flags on the Cobra command immediately.
*   **Pattern:** Store flag registration functions in the `App` struct.
*   **Execution:** Iterate and execute these functions inside `WithCobra()` (before `Execute` is called by the user).

### 2. Recursive Module Application
`gaz.Module` supports composition (`Use(child)`). Flag registration must handle this recursion.
*   **Pattern:** Move flag registration logic from `App.Use()` into `Module.Apply()`.
*   **Mechanism:** `Apply(app *App)` should register its own flags with `app` and then call `child.Apply(app)`, ensuring all nested flags are captured.

### 3. Auto-Blocking Run Loop
To remove the need for manual `RunE` with signal channels:
*   **Pattern:** `WithCobra(cmd)` should inspect `cmd.Run` and `cmd.RunE`.
*   **Logic:** If both are nil, assign a default `RunE` that calls `app.waitForShutdownSignal(ctx)`. This allows `cmd.Execute()` to start the app, wait, and stop it gracefully.

## Proposed Changes

### 1. Update `App` Struct
Add a field to store flag functions:
```go
type App struct {
    // ...
    flagFns []func(*pflag.FlagSet)
    // ...
}

func (a *App) AddFlagsFn(fn func(*pflag.FlagSet)) {
    if fn != nil {
        a.flagFns = append(a.flagFns, fn)
    }
}
```

### 2. Update `Module.Apply`
Modify `builtModule.Apply` (and `module_builder.go` logic) to register flags:
```go
func (m *builtModule) Apply(app *App) error {
    // Register flags
    if m.flagsFn != nil {
        app.AddFlagsFn(m.flagsFn)
    }
    
    // Apply children (which will register their own flags)
    for _, child := range m.childModules {
        if err := child.Apply(app); err != nil {
            return err
        }
    }
    // ... providers ...
}
```

### 3. Update `WithCobra`
Enhance `WithCobra` to apply flags and set default run loop:
```go
func (a *App) WithCobra(cmd *cobra.Command) *App {
    a.cobraCmd = cmd
    
    // 1. Apply stored flags
    for _, fn := range a.flagFns {
        fn(cmd.PersistentFlags())
    }
    
    // ... existing hooks ...

    // 2. Set default RunE if missing
    if cmd.Run == nil && cmd.RunE == nil {
        cmd.RunE = func(c *cobra.Command, args []string) error {
            // Context is already set by PreRun hook
            // App is already started by PreRun hook
            
            // Just wait for shutdown
            return a.waitForShutdownSignal(c.Context())
        }
    }
    
    return a
}
```

## Common Pitfalls

### Pitfall 1: Flag Visibility in Help
**Issue:** If flags are registered inside `PersistentPreRun`, they won't show up in `help` output because `help` runs before `PreRun`.
**Solution:** `WithCobra` is called at setup time (before `Execute`). By applying flags immediately inside `WithCobra`, they are registered before parsing/help generation.

### Pitfall 2: Nested Flags Ignoring
**Issue:** `App.Use(parent)` currently only checks `parent` for flags.
**Solution:** By moving registration to `Apply`, recursion handles nested flags automatically.

### Pitfall 3: Viper Binding Timing
**Issue:** `BindPFlags` must be called *after* flags are registered but *before* config loading.
**Solution:** 
1. `WithCobra` registers flags (Setup time).
2. `PreRun` calls `bootstrap` (Runtime).
3. `bootstrap` calls `configMgr.BindFlags` (Runtime).
This order is correct.

## Code Examples

### Expected User Code
```go
func execute() error {
    serveCmd := &cobra.Command{
        Use:   "serve",
        Short: "Start the server",
    }

    app := gaz.New()
    // Flags from nested grpc/gateway modules are auto-collected
    app.Use(server.NewModule()) 
    
    // Flags are applied to serveCmd here
    // Default RunE is set here
    app.WithCobra(serveCmd) 

    rootCmd.AddCommand(serveCmd)
    return rootCmd.Execute()
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual `BindPFlags` | Auto-binding in `bootstrap` | Phase 42 | No boilerplate |
| Manual `RunE` & `signal` | Auto-blocking in `WithCobra` | Phase 42 | No boilerplate |
| Flags lost in `Use` | Deferred flag registration | Phase 42 | Nested modules work |

## Open Questions
None. The path forward is clear.

## Metadata
**Confidence breakdown:**
- Standard stack: HIGH - Core Go ecosystem.
- Architecture: HIGH - Validated against existing `gaz` codebase.
- Pitfalls: HIGH - Known Cobra behavior.

**Research date:** 2026-02-03
