# System Info CLI Example

A comprehensive example demonstrating gaz DI framework features with a practical system information CLI tool.

## Features Demonstrated

This example showcases the full gaz pattern:

- **Dependency Injection**: `For[T]()` and `Resolve[T]()` patterns for type-safe service registration and resolution
- **ConfigProvider**: Flag-based configuration with `ProviderValues` - services declare config requirements via `ConfigNamespace()` and `ConfigFlags()`
- **Workers**: Background data collection with lifecycle integration - automatic start/stop with the application
- **Cobra Integration**: `RegisterCobraFlags` for CLI flag visibility in `--help`

## Usage

```bash
# Build the CLI
go build -o sysinfo .

# One-shot mode (display and exit)
./sysinfo run --sysinfo-once

# JSON output
./sysinfo run --sysinfo-once --sysinfo-format json

# Continuous monitoring (default 5s refresh)
./sysinfo run

# Custom refresh interval
./sysinfo run --sysinfo-refresh 10s

# View available flags
./sysinfo run --help

# View version
./sysinfo version
```

## Configuration Flags

Flags are exposed via the `ConfigProvider` pattern and visible in `--help`:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--sysinfo-refresh` | duration | 5s | Refresh interval for continuous mode |
| `--sysinfo-format` | string | text | Output format (text, json) |
| `--sysinfo-once` | bool | false | Run once and exit (no continuous monitoring) |

## Architecture

```
main.go       - CLI entry point, Cobra setup, gaz App lifecycle
config.go     - ConfigProvider implementing ConfigNamespace() and ConfigFlags()
collector.go  - System info collection using gopsutil/v4
worker.go     - Background refresh Worker implementing Worker interface
```

## Key Patterns

### 1. ConfigProvider before Execute()

The ConfigProvider pattern allows services to declare their configuration requirements. Flags must be registered before `Execute()` to appear in `--help`:

```go
// Register ConfigProvider
gaz.For[*SystemInfoConfig](app.Container()).Provider(NewSystemInfoConfig)

// CRITICAL: Register flags BEFORE Execute()
app.RegisterCobraFlags(rootCmd)

// Now execute - flags appear in --help
rootCmd.Execute()
```

### 2. Worker Lifecycle Integration

Workers implement the `Worker` interface and are auto-discovered during `Build()`:

```go
// Create worker
worker := NewRefreshWorker("sysinfo-worker", interval, format, collector)

// Register via Instance() - enables auto-discovery
gaz.For[*RefreshWorker](app.Container()).Named("sysinfo-worker").Instance(worker)

// Run handles worker lifecycle - starts after services, stops on shutdown
app.Run(cmd.Context())
```

### 3. Non-blocking Worker Start

Workers must follow the contract where `Start()` is non-blocking:

```go
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
                w.collectAndDisplay()
            }
        }
    }()
}

func (w *RefreshWorker) Stop() {
    close(w.done)
    w.wg.Wait()
}
```

### 4. Graceful Shutdown

Ctrl+C triggers graceful shutdown:
- Worker receives `Stop()` signal via done channel
- Worker waits for current cycle to complete
- Clean exit without goroutine leaks

## Output Examples

### Text Format (default)

```
System Information
─────────────────
Hostname:  myhost
Platform:  linux
Uptime:    5d 2h 15m

CPU
Model:     AMD Ryzen 9 5900X 12-Core Processor
Cores:     12
Usage:     15.3%

Memory
Total:     32.00 GB
Used:      18.45 GB (57.7%)

Disk (/)
Total:     500.00 GB
Used:      245.32 GB (49.1%)
```

### JSON Format

```json
{
  "hostname": "myhost",
  "platform": "linux",
  "uptime": 447300,
  "cpu_model": "AMD Ryzen 9 5900X 12-Core Processor",
  "cpu_cores": 12,
  "cpu_usage": 15.3,
  "mem_total": 34359738368,
  "mem_used": 19807436800,
  "mem_percent": 57.7,
  "disk_total": 536870912000,
  "disk_used": 263485128704,
  "disk_percent": 49.1
}
```

## Dependencies

- [gaz](https://github.com/petabytecl/gaz) - DI framework
- [gopsutil/v4](https://github.com/shirou/gopsutil) - System information gathering
- [cobra](https://github.com/spf13/cobra) - CLI framework
