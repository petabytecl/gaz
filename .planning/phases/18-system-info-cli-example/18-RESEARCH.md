# Phase 18: System Info CLI Example - Research

**Researched:** 2026-01-28
**Domain:** Go CLI example showcasing DI, ConfigProvider, Workers, and Cobra integration
**Confidence:** HIGH

## Summary

This phase creates a comprehensive CLI example that demonstrates the full gaz framework capabilities. The example will be a "system info" CLI tool that collects and displays system information (CPU, memory, disk, host) while showcasing four key gaz patterns: dependency injection via `For[T]()`/`Resolve[T]()`, ConfigProvider for flag-based configuration, Workers for background data collection, and RegisterCobraFlags for CLI visibility.

The standard approach uses `shirou/gopsutil/v4` for cross-platform system information gathering - it's the de facto Go library for this purpose (11.7k stars, pure Go, no cgo required). The example will follow existing gaz example patterns (`examples/config-loading`, `examples/cobra-cli`, `examples/lifecycle`) while combining all features into a cohesive demonstration.

**Primary recommendation:** Create `examples/system-info-cli/` with a Cobra CLI that uses ConfigProvider for config flags (refresh interval, output format), a Worker for periodic data refresh, and demonstrates `RegisterCobraFlags` for `--help` visibility.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/shirou/gopsutil/v4 | v4.25.x | System information (CPU, memory, disk, host) | De facto Go system metrics library, cross-platform, pure Go |
| github.com/petabytecl/gaz | current | DI framework | The framework being demonstrated |
| github.com/spf13/cobra | v1.10.x | CLI structure | Already used in gaz, standard Go CLI library |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| encoding/json | stdlib | JSON output formatting | When `--format json` is requested |
| text/tabwriter | stdlib | Aligned text output | For human-readable table output |
| time | stdlib | Refresh interval handling | Worker ticker implementation |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| gopsutil/v4 | mackerelio/go-osstat | gopsutil is more comprehensive and widely adopted |
| text/tabwriter | tablewriter | tabwriter is stdlib, simpler for this use case |

**Installation:**
```bash
cd examples/system-info-cli
go get github.com/shirou/gopsutil/v4@latest
```

## Architecture Patterns

### Recommended Project Structure
```
examples/system-info-cli/
├── main.go              # Entry point, Cobra setup, gaz App
├── config.go            # ConfigProvider implementation
├── collector.go         # System info collection service
├── worker.go            # Background refresh Worker
└── config.yaml          # Sample config file (optional)
```

### Pattern 1: ConfigProvider for CLI Configuration
**What:** Declare config requirements via ConfigProvider interface so flags appear in `--help`
**When to use:** When you want config values to be overridable via CLI flags
**Example:**
```go
// Source: gaz examples/config-loading/main.go pattern
type SystemInfoConfig struct {
    pv *gaz.ProviderValues
}

func (c *SystemInfoConfig) ConfigNamespace() string { return "sysinfo" }

func (c *SystemInfoConfig) ConfigFlags() []gaz.ConfigFlag {
    return []gaz.ConfigFlag{
        {Key: "refresh", Type: gaz.ConfigFlagTypeDuration, Default: 5 * time.Second, Description: "Refresh interval"},
        {Key: "format", Type: gaz.ConfigFlagTypeString, Default: "text", Description: "Output format (text, json)"},
        {Key: "once", Type: gaz.ConfigFlagTypeBool, Default: false, Description: "Run once and exit (no worker)"},
    }
}

func NewSystemInfoConfig(c *gaz.Container) (*SystemInfoConfig, error) {
    pv, err := gaz.Resolve[*gaz.ProviderValues](c)
    if err != nil {
        return nil, err
    }
    return &SystemInfoConfig{pv: pv}, nil
}
```

### Pattern 2: Worker for Background Data Collection
**What:** Implement Worker interface for periodic system info refresh
**When to use:** For continuous monitoring mode (not `--once`)
**Example:**
```go
// Source: gaz worker/worker.go interface
type RefreshWorker struct {
    name      string
    interval  time.Duration
    collector *Collector
    done      chan struct{}
    wg        sync.WaitGroup
}

func (w *RefreshWorker) Name() string { return w.name }

func (w *RefreshWorker) Start() {
    w.done = make(chan struct{})
    w.wg.Add(1)
    go func() {
        defer w.wg.Done()
        ticker := time.NewTicker(w.interval)
        defer ticker.Stop()
        for {
            select {
            case <-w.done:
                return
            case <-ticker.C:
                info, _ := w.collector.Collect()
                w.collector.Display(info)
            }
        }
    }()
}

func (w *RefreshWorker) Stop() {
    close(w.done)
    w.wg.Wait()
}
```

### Pattern 3: RegisterCobraFlags Integration
**What:** Expose ConfigProvider flags to Cobra before Execute()
**When to use:** Always when using ConfigProvider with Cobra CLI
**Example:**
```go
// Source: gaz cobra_flags.go pattern (Phase 17)
func main() {
    rootCmd := &cobra.Command{
        Use:   "sysinfo",
        Short: "Display system information",
    }
    
    app := gaz.New()
    // Register ConfigProvider
    gaz.For[*SystemInfoConfig](app.Container()).Provider(NewSystemInfoConfig)
    
    // Expose flags BEFORE Execute - they appear in --help
    app.RegisterCobraFlags(rootCmd)
    
    // Set up lifecycle
    app.WithCobra(rootCmd)
    
    // Add run command
    runCmd := &cobra.Command{Use: "run", RunE: runSystem}
    rootCmd.AddCommand(runCmd)
    
    rootCmd.Execute()
}
```

### Pattern 4: gopsutil Data Collection
**What:** Use gopsutil to gather CPU, memory, disk, and host information
**When to use:** For any system metrics collection
**Example:**
```go
// Source: Context7 /shirou/gopsutil documentation
import (
    "github.com/shirou/gopsutil/v4/cpu"
    "github.com/shirou/gopsutil/v4/mem"
    "github.com/shirou/gopsutil/v4/disk"
    "github.com/shirou/gopsutil/v4/host"
)

type SystemInfo struct {
    Hostname     string
    Platform     string
    Uptime       uint64
    CPUModel     string
    CPUCores     int
    CPUUsage     float64
    MemTotal     uint64
    MemUsed      uint64
    MemPercent   float64
    DiskTotal    uint64
    DiskUsed     uint64
    DiskPercent  float64
}

func (c *Collector) Collect() (*SystemInfo, error) {
    info := &SystemInfo{}
    
    // Host info
    hostInfo, _ := host.Info()
    info.Hostname = hostInfo.Hostname
    info.Platform = hostInfo.Platform
    info.Uptime = hostInfo.Uptime
    
    // CPU info
    cpuInfo, _ := cpu.Info()
    if len(cpuInfo) > 0 {
        info.CPUModel = cpuInfo[0].ModelName
        info.CPUCores = int(cpuInfo[0].Cores)
    }
    cpuPercent, _ := cpu.Percent(0, false)
    if len(cpuPercent) > 0 {
        info.CPUUsage = cpuPercent[0]
    }
    
    // Memory info
    memInfo, _ := mem.VirtualMemory()
    info.MemTotal = memInfo.Total
    info.MemUsed = memInfo.Used
    info.MemPercent = memInfo.UsedPercent
    
    // Disk info (root partition)
    diskInfo, _ := disk.Usage("/")
    info.DiskTotal = diskInfo.Total
    info.DiskUsed = diskInfo.Used
    info.DiskPercent = diskInfo.UsedPercent
    
    return info, nil
}
```

### Anti-Patterns to Avoid

- **Blocking in Worker.Start():** Start() must be non-blocking; spawn goroutine internally
- **Forgetting RegisterCobraFlags:** Flags won't appear in `--help` if not registered before Execute()
- **Not handling graceful shutdown:** Worker must respond to Stop() and clean up goroutines
- **Ignoring gopsutil errors:** Some platforms may not support all metrics; handle gracefully
- **Using global state:** Use DI to inject dependencies, not globals

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| System metrics gathering | Reading /proc manually | gopsutil | Cross-platform, handles edge cases |
| CLI flag parsing | flag package | Cobra + ConfigProvider | Integrated with gaz, better UX |
| Config flag to CLI bridge | Manual viper binding | RegisterCobraFlags | Automatic, handles types |
| Periodic execution | Manual sleep loops | Worker interface | Lifecycle integration, graceful shutdown |
| Table formatting | Manual spacing | text/tabwriter | Handles alignment automatically |

**Key insight:** The example should demonstrate gaz capabilities, not reinvent them. Use existing framework features as intended.

## Common Pitfalls

### Pitfall 1: Worker Not Discovered by App
**What goes wrong:** Worker starts manually but not integrated with app lifecycle
**Why it happens:** Worker not registered via For[T]() or not implementing Worker interface
**How to avoid:** Register Worker via `gaz.For[*RefreshWorker](app.Container()).Instance(worker)` - gaz auto-discovers Workers during Build()
**Warning signs:** Worker doesn't stop on Ctrl+C, app hangs on shutdown

### Pitfall 2: RegisterCobraFlags Called After Execute
**What goes wrong:** Flags don't appear in --help, "unknown flag" errors
**Why it happens:** Execute() parses flags before hooks run
**How to avoid:** Call `app.RegisterCobraFlags(rootCmd)` BEFORE `rootCmd.Execute()`
**Warning signs:** `--help` missing expected flags, flags work but not visible

### Pitfall 3: gopsutil Import Path
**What goes wrong:** Build fails with import errors
**Why it happens:** Using old v3 import path instead of v4
**How to avoid:** Use `github.com/shirou/gopsutil/v4/...` (note the /v4/)
**Warning signs:** "module not found" or import cycle errors

### Pitfall 4: CPU Percent Returns Empty
**What goes wrong:** cpu.Percent() returns empty slice
**Why it happens:** cpu.Percent() with 0 duration returns cached value which may be empty on first call
**How to avoid:** Either pass non-zero duration (e.g., 100ms) or handle empty slice gracefully
**Warning signs:** CPU usage always shows 0%

### Pitfall 5: ConfigProvider Dependency on ProviderValues
**What goes wrong:** ProviderValues not available when provider runs
**Why it happens:** Attempting to resolve ProviderValues before config loaded
**How to avoid:** This was fixed in gaz - ProviderValues is registered BEFORE providers run during Build()
**Warning signs:** "ProviderValues not registered" error

## Code Examples

Verified patterns from official sources:

### Complete Main Setup
```go
// Source: gaz examples synthesis
func main() {
    // Root command
    rootCmd := &cobra.Command{
        Use:   "sysinfo",
        Short: "System information CLI - gaz framework demo",
        Long: `Demonstrates gaz DI framework features:
- Dependency Injection: For[T]() and Resolve[T]() patterns
- ConfigProvider: Flag-based configuration with ProviderValues
- Workers: Background data collection with lifecycle integration
- Cobra: RegisterCobraFlags for CLI flag visibility`,
    }

    // Create app
    app := gaz.New(gaz.WithShutdownTimeout(5 * time.Second))

    // Register ConfigProvider - declares config flags
    gaz.For[*SystemInfoConfig](app.Container()).Provider(NewSystemInfoConfig)

    // Register collector service
    gaz.For[*Collector](app.Container()).Provider(NewCollector)

    // IMPORTANT: Register flags BEFORE Execute() for --help visibility
    if err := app.RegisterCobraFlags(rootCmd); err != nil {
        log.Fatalf("Failed to register flags: %v", err)
    }

    // Run subcommand
    runCmd := &cobra.Command{
        Use:   "run",
        Short: "Run system info collection",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runSysInfo(cmd, app)
        },
    }
    rootCmd.AddCommand(runCmd)

    // Version subcommand (no DI needed)
    versionCmd := &cobra.Command{
        Use:   "version",
        Short: "Print version",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Println("sysinfo v1.0.0 - gaz framework demo")
        },
    }
    rootCmd.AddCommand(versionCmd)

    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func runSysInfo(cmd *cobra.Command, app *gaz.App) error {
    // Attach lifecycle to Cobra
    app.WithCobra(cmd)

    // Resolve config
    cfg := gaz.MustResolve[*SystemInfoConfig](app.Container())

    // One-shot mode
    if cfg.Once() {
        collector := gaz.MustResolve[*Collector](app.Container())
        info, err := collector.Collect()
        if err != nil {
            return err
        }
        return collector.Display(info, cfg.Format())
    }

    // Continuous mode - register worker
    collector := gaz.MustResolve[*Collector](app.Container())
    worker := NewRefreshWorker(cfg.RefreshInterval(), collector, cfg.Format())
    gaz.For[*RefreshWorker](app.Container()).Named("refresh-worker").Instance(worker)

    // Run with lifecycle management
    return app.Run(cmd.Context())
}
```

### Display with Format Support
```go
// Source: Synthesized from Go stdlib patterns
func (c *Collector) Display(info *SystemInfo, format string) error {
    switch format {
    case "json":
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(info)
    default: // "text"
        w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
        fmt.Fprintf(w, "System Information\t\n")
        fmt.Fprintf(w, "─────────────────\t\n")
        fmt.Fprintf(w, "Hostname:\t%s\n", info.Hostname)
        fmt.Fprintf(w, "Platform:\t%s\n", info.Platform)
        fmt.Fprintf(w, "Uptime:\t%s\n", formatDuration(info.Uptime))
        fmt.Fprintf(w, "\nCPU\t\n")
        fmt.Fprintf(w, "Model:\t%s\n", info.CPUModel)
        fmt.Fprintf(w, "Cores:\t%d\n", info.CPUCores)
        fmt.Fprintf(w, "Usage:\t%.1f%%\n", info.CPUUsage)
        fmt.Fprintf(w, "\nMemory\t\n")
        fmt.Fprintf(w, "Total:\t%s\n", formatBytes(info.MemTotal))
        fmt.Fprintf(w, "Used:\t%s (%.1f%%)\n", formatBytes(info.MemUsed), info.MemPercent)
        fmt.Fprintf(w, "\nDisk (/)\t\n")
        fmt.Fprintf(w, "Total:\t%s\n", formatBytes(info.DiskTotal))
        fmt.Fprintf(w, "Used:\t%s (%.1f%%)\n", formatBytes(info.DiskUsed), info.DiskPercent)
        return w.Flush()
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gopsutil v3 | gopsutil v4 | 2024 | New import path, Go 1.23 minimum |
| Manual flag binding | RegisterCobraFlags | Phase 17 | Automatic flag registration |
| Interface-based lifecycle | Worker interface | Phase 14 | Simpler background task management |
| Global viper | ConfigProvider pattern | Phase 13/14 | Scoped config per provider |

**Deprecated/outdated:**
- gopsutil/v3: Still works but v4 is current (use `github.com/shirou/gopsutil/v4`)
- Manual viper.BindPFlag: Replaced by RegisterCobraFlags for ConfigProvider services

## Open Questions

Things that couldn't be fully resolved:

1. **Should example support Windows disk paths?**
   - What we know: gopsutil handles Windows paths, but root "/" doesn't exist
   - What's unclear: Best default path that works cross-platform
   - Recommendation: Use "/" with error handling, document Windows behavior

2. **Include network information?**
   - What we know: gopsutil/v4/net provides network stats
   - What's unclear: Whether network adds too much complexity for a demo
   - Recommendation: Keep simple with CPU/mem/disk/host; network can be added later

## Sources

### Primary (HIGH confidence)
- Context7 /shirou/gopsutil - CPU, memory, disk, host info APIs
- Context7 /spf13/cobra - Subcommands, PersistentFlags, PreRun hooks
- gaz examples/config-loading/main.go - ConfigProvider pattern
- gaz examples/cobra-cli/main.go - Cobra integration pattern
- gaz worker/worker.go - Worker interface contract

### Secondary (MEDIUM confidence)
- GitHub shirou/gopsutil releases - Version v4.25.12 is current (2026-01)

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - gopsutil is well-established, Context7 verified APIs
- Architecture: HIGH - Following existing gaz example patterns
- Pitfalls: HIGH - Based on gaz source code and documented behavior

**Research date:** 2026-01-28
**Valid until:** 60 days (gopsutil and gaz both stable)
