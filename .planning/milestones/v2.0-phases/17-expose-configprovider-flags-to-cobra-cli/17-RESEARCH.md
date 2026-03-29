# Phase 17: Expose ConfigProvider Flags to Cobra CLI - Research

**Researched:** 2026-01-28
**Domain:** Cobra CLI + Viper flag integration
**Confidence:** HIGH

## Summary

This research investigates how to auto-register ConfigProvider config flags (like `server.host`, `server.port`) as Cobra CLI flags for override and `--help` visibility. The gaz framework already has a ConfigProvider pattern where services declare config via `ConfigNamespace()` and `ConfigFlags()` methods, and viper handles config loading. The missing piece is exposing these to the CLI.

The standard approach uses Cobra's `PersistentFlags()` to register typed flags on the root command and Viper's `BindPFlag()` to link them to viper keys. The critical constraint is **timing**: flags must be registered BEFORE `cmd.Execute()` is called, otherwise they won't appear in `--help` and will cause "unknown flag" errors. This means we need to collect ConfigProvider requirements early, which aligns with gaz's existing pattern of iterating providers during Build().

**Primary recommendation:** Create a `RegisterCobraFlags(cmd *cobra.Command)` method on App that collects ConfigProvider flags and registers them as persistent pflags on the command, with viper binding, BEFORE Execute() is called.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/spf13/cobra | v1.9.x | CLI framework | De facto standard for Go CLI, already used in gaz |
| github.com/spf13/viper | v1.20.x | Config management | Standard config solution, already used in gaz |
| github.com/spf13/pflag | v1.0.x | POSIX-compliant flags | Underlying flag library for both Cobra and Viper |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| strings.Replacer | stdlib | Key-to-flag name transformation | Convert "server.host" → "server-host" |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| PersistentFlags | Local Flags | Persistent works across subcommands, local is per-command only |
| BindPFlag (individual) | BindPFlags (FlagSet) | Individual gives more control, FlagSet is bulk binding |

**Installation:**
Already installed - spf13/cobra and spf13/viper are existing dependencies.

## Architecture Patterns

### Key Constraint: Flag Registration Timing

```
Cobra lifecycle:
1. Define command tree (rootCmd, subcommands)
2. Define flags on commands (PersistentFlags, Flags)
3. cmd.Execute() called
4. Cobra parses all flags  ← FLAGS MUST BE REGISTERED BEFORE THIS
5. PersistentPreRun hooks execute
6. PreRun hooks execute  
7. Run function executes

CRITICAL: Flags added in PreRun/PersistentPreRun will NOT appear in --help
```

### Recommended Integration Point

Given gaz's existing architecture where `WithCobra(cmd)` hooks into `PersistentPreRunE`, flags must be registered BEFORE this hook runs. Two options:

**Option A: Caller registers flags explicitly (Recommended)**
```go
app := gaz.New()
gaz.For[*ServerConfig](app.Container()).Provider(NewServerConfig)
// Collect and register flags BEFORE Execute
app.RegisterCobraFlags(rootCmd)  // NEW METHOD
app.WithCobra(rootCmd)
rootCmd.Execute()
```

**Option B: Auto-registration during WithCobra**
```go
func (a *App) WithCobra(cmd *cobra.Command) *App {
    // Register flags immediately, before hooks
    a.registerConfigProviderFlags(cmd)  // Happens now, not in hook
    
    // Then set up lifecycle hooks...
    cmd.PersistentPreRunE = ...
}
```

**Option A is recommended** because it makes the operation explicit and gives users control over whether they want auto-flag registration.

### Pattern: Key-to-Flag Name Transformation

```go
// Config key format: namespace.key (e.g., "server.host")
// Flag name format: namespace-key (e.g., "--server-host")
// Transform: Replace "." with "-"

func configKeyToFlagName(key string) string {
    return strings.ReplaceAll(key, ".", "-")
}

// server.host → --server-host
// server.port → --server-port
// database.pool.size → --database-pool-size
```

### Pattern: Type-Aware Flag Registration

```go
func registerFlag(fs *pflag.FlagSet, flag ConfigFlag, fullKey string) {
    flagName := configKeyToFlagName(fullKey)
    
    switch flag.Type {
    case ConfigFlagTypeString:
        def, _ := flag.Default.(string)
        fs.String(flagName, def, flag.Description)
    case ConfigFlagTypeInt:
        def, _ := flag.Default.(int)
        fs.Int(flagName, def, flag.Description)
    case ConfigFlagTypeBool:
        def, _ := flag.Default.(bool)
        fs.Bool(flagName, def, flag.Description)
    case ConfigFlagTypeDuration:
        def, _ := flag.Default.(time.Duration)
        fs.Duration(flagName, def, flag.Description)
    case ConfigFlagTypeFloat:
        def, _ := flag.Default.(float64)
        fs.Float64(flagName, def, flag.Description)
    }
}
```

### Pattern: Viper Binding with Correct Key

```go
// After registering flag, bind it to viper with ORIGINAL key (with dots)
// Flag name: --server-host
// Viper key: server.host

viper.BindPFlag("server.host", fs.Lookup("server-host"))
```

### Anti-Patterns to Avoid

- **Registering flags in PersistentPreRunE:** Flags won't appear in `--help`
- **Using dots in flag names:** POSIX flags use hyphens, not dots (`--server-host` not `--server.host`)
- **Binding with wrong key:** Must bind flag name to viper key, keeping the original dot notation for viper
- **Registering flags after Execute() starts:** Will cause "unknown flag" errors

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Flag parsing | Custom arg parser | pflag via cobra | POSIX compliance, shorthand support, type safety |
| Config-flag bridge | Manual value copying | viper.BindPFlag | Handles precedence (flag > env > file) automatically |
| Flag type coercion | Type switch in handler | pflag typed methods | Built-in parsing, validation, default handling |
| Help text generation | Manual formatting | Cobra's built-in | Consistent formatting, wrapping, grouping |

**Key insight:** Viper's `BindPFlag` is the bridge between CLI flags and config. Once bound, reading `viper.GetString("server.host")` returns the flag value if provided, otherwise falls back through the precedence chain (env → file → default).

## Common Pitfalls

### Pitfall 1: Flag Registration Timing
**What goes wrong:** Flags registered in PreRun hooks don't appear in `--help` output
**Why it happens:** Cobra generates help from flags registered BEFORE Execute()
**How to avoid:** Register all flags before calling `cmd.Execute()`
**Warning signs:** `--help` doesn't show expected flags, but flags work when explicitly passed

### Pitfall 2: Key Name Confusion
**What goes wrong:** Flag `--server-host` doesn't update viper key `server.host`
**Why it happens:** BindPFlag called with wrong key name
**How to avoid:** Use transformation function consistently:
```go
flagName := keyToFlagName(viperKey)  // server.host → server-host
fs.String(flagName, ...)
viper.BindPFlag(viperKey, fs.Lookup(flagName))  // viperKey, not flagName
```
**Warning signs:** Flag parsing succeeds but config value unchanged

### Pitfall 3: Flag Collisions
**What goes wrong:** Two providers register the same flag name
**Why it happens:** No collision detection during registration
**How to avoid:** Check `fs.Lookup(flagName) != nil` before registering
**Warning signs:** Panic or overwritten flag definitions

### Pitfall 4: Precedence Confusion
**What goes wrong:** Users expect flag to override env, but it doesn't
**Why it happens:** Viper precedence is: explicit Set > flag > env > config > default
**How to avoid:** Always use BindPFlag so viper handles precedence correctly
**Warning signs:** Flag values ignored when env var is also set

### Pitfall 5: PersistentFlags vs Flags
**What goes wrong:** Flags only work on root command, not subcommands
**Why it happens:** Used `Flags()` instead of `PersistentFlags()`
**How to avoid:** Use `PersistentFlags()` on root command for config flags
**Warning signs:** Flag works for `app serve` but not `app migrate`

## Code Examples

Verified patterns from official sources:

### Complete Flag Registration and Binding
```go
// Source: Cobra + Viper documentation synthesis
func (a *App) RegisterCobraFlags(cmd *cobra.Command) error {
    fs := cmd.PersistentFlags()
    
    for _, entry := range a.providerConfigs {
        for _, flag := range entry.flags {
            fullKey := entry.namespace + "." + flag.Key
            flagName := strings.ReplaceAll(fullKey, ".", "-")
            
            // Skip if already registered
            if fs.Lookup(flagName) != nil {
                continue
            }
            
            // Register typed flag
            switch flag.Type {
            case ConfigFlagTypeString:
                def, _ := flag.Default.(string)
                fs.String(flagName, def, flag.Description)
            case ConfigFlagTypeInt:
                def, _ := flag.Default.(int)
                fs.Int(flagName, def, flag.Description)
            case ConfigFlagTypeBool:
                def, _ := flag.Default.(bool)
                fs.Bool(flagName, def, flag.Description)
            case ConfigFlagTypeDuration:
                def, _ := flag.Default.(time.Duration)
                fs.Duration(flagName, def, flag.Description)
            case ConfigFlagTypeFloat:
                def, _ := flag.Default.(float64)
                fs.Float64(flagName, def, flag.Description)
            }
            
            // Bind to viper with ORIGINAL key
            if err := viper.BindPFlag(fullKey, fs.Lookup(flagName)); err != nil {
                return fmt.Errorf("failed to bind flag %s: %w", flagName, err)
            }
        }
    }
    return nil
}
```

### Viper Precedence Order
```go
// Source: Viper README
// Viper uses the following precedence order (highest to lowest):
// 1. explicit call to Set
// 2. flag
// 3. env
// 4. config file
// 5. key/value store
// 6. default

// When flag is bound via BindPFlag, Viper automatically respects this
viper.GetString("server.host")  // Returns flag value if --server-host was passed
```

### Flag Registration Before Execute
```go
// Source: Cobra User Guide pattern
func main() {
    rootCmd := &cobra.Command{Use: "myapp"}
    
    // Register flags BEFORE Execute
    rootCmd.PersistentFlags().String("server-host", "localhost", "Server host")
    rootCmd.PersistentFlags().Int("server-port", 8080, "Server port")
    
    // Bind to viper
    viper.BindPFlag("server.host", rootCmd.PersistentFlags().Lookup("server-host"))
    viper.BindPFlag("server.port", rootCmd.PersistentFlags().Lookup("server-port"))
    
    rootCmd.Execute()  // Flags are registered, will appear in --help
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual flag definition per command | Auto-registration from ConfigProvider | Phase 17 | Eliminates duplicate flag definitions |
| Separate config and CLI concerns | Unified via ConfigProvider interface | Already exists | Single source of truth for config |
| Global viper instance | Scoped viper in config.Manager | Phase 14 | Better testability, no global state |

**Deprecated/outdated:**
- None in core libraries; Cobra and Viper are stable

## Open Questions

Things that couldn't be fully resolved:

1. **Should flags be grouped in help output?**
   - What we know: Cobra supports custom flag usage formatting via `FlagUsages()`
   - What's unclear: Whether grouping by namespace (e.g., "Server Flags:", "Database Flags:") is worth the complexity
   - Recommendation: Start simple without grouping; add if users request it

2. **How to handle flag name conflicts with built-in flags?**
   - What we know: Cobra has built-in flags like `--help`, `--version`
   - What's unclear: What happens if a provider tries to register `--help`?
   - Recommendation: Validate against reserved names, return error for conflicts

3. **When exactly should RegisterCobraFlags be called?**
   - What we know: Must be before Execute(), provider flags available after For[T]() calls
   - What's unclear: Best UX - should it be automatic or explicit?
   - Recommendation: Explicit call after provider registration, before Execute()

## Sources

### Primary (HIGH confidence)
- Context7 /spf13/cobra - Flag binding, PersistentPreRun hooks, flag types
- Context7 /spf13/viper - BindPFlag, precedence order, nested key access
- pkg.go.dev/github.com/spf13/pflag - Flag registration API

### Secondary (MEDIUM confidence)
- WebSearch synthesis on dynamic flag registration timing - Multiple sources agree

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Using existing gaz dependencies, well-documented APIs
- Architecture: HIGH - Pattern synthesized from official Cobra/Viper docs
- Pitfalls: HIGH - Common issues documented in official sources

**Research date:** 2026-01-28
**Valid until:** 90 days (stable libraries, stable patterns)
