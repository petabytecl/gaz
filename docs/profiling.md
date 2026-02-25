# Profiling and Performance Monitoring

This guide covers how to integrate profiling and performance monitoring into gaz applications.

## pprof Integration

The Go standard library provides `net/http/pprof` for runtime profiling. Here's how to integrate it with a gaz application.

### Basic pprof Setup

Add pprof endpoints to your HTTP server:

```go
import (
    _ "net/http/pprof"
    "net/http"
)

func main() {
    app := gaz.New()
    
    // Register your services...
    
    if err := app.Build(); err != nil {
        log.Fatal(err)
    }
    
    // Start pprof server on a separate port
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    app.Run(context.Background())
}
```

### Using pprof with gaz HTTP Server

If you're using `server/http`, you can mount pprof endpoints:

```go
import (
    _ "net/http/pprof"
    "net/http"
    "github.com/petabytecl/gaz/server/http"
)

func main() {
    app := gaz.New()
    
    // Register HTTP server module
    app.Use(http.NewModule())
    
    // After Build(), mount pprof endpoints
    if err := app.Build(); err != nil {
        log.Fatal(err)
    }
    
    // Get HTTP server and mount pprof
    srv := gaztest.RequireResolve[*http.Server](t, app)
    srv.Handle("/debug/pprof/", http.DefaultServeMux)
    
    app.Run(context.Background())
}
```

### Signal-Based Profiling

For production environments, use signal-based profiling to avoid exposing endpoints:

```go
import (
    "os"
    "os/signal"
    "runtime/pprof"
    "syscall"
)

func setupSignalProfiling() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2)
    
    go func() {
        for sig := range sigCh {
            switch sig {
            case syscall.SIGUSR1:
                // CPU profile
                f, _ := os.Create("cpu.prof")
                pprof.StartCPUProfile(f)
                time.Sleep(30 * time.Second)
                pprof.StopCPUProfile()
                f.Close()
                
            case syscall.SIGUSR2:
                // Memory profile
                f, _ := os.Create("mem.prof")
                runtime.GC()
                pprof.WriteHeapProfile(f)
                f.Close()
            }
        }
    }()
}
```

## Memory Leak Detection

### Using runtime.MemStats

Monitor memory usage over time:

```go
import "runtime"

func checkMemory() {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    fmt.Printf("Alloc: %d KB\n", m.Alloc/1024)
    fmt.Printf("TotalAlloc: %d KB\n", m.TotalAlloc/1024)
    fmt.Printf("Sys: %d KB\n", m.Sys/1024)
    fmt.Printf("NumGC: %d\n", m.NumGC)
}
```

### Using Health Checks

The `health/checks/runtime` package provides built-in memory checks:

```go
import "github.com/petabytecl/gaz/health/checks/runtime"

manager.AddLivenessCheck("memory", runtime.MemoryUsage(1<<30)) // 1GB threshold
```

## Goroutine Leak Detection

### Monitoring Goroutine Count

Track goroutine count over time:

```go
import "runtime"

func checkGoroutines() {
    count := runtime.NumGoroutine()
    fmt.Printf("Goroutines: %d\n", count)
}
```

### Using Health Checks

The `health/checks/runtime` package provides goroutine leak detection:

```go
import "github.com/petabytecl/gaz/health/checks/runtime"

manager.AddLivenessCheck("goroutines", runtime.GoroutineCount(1000))
```

### Detecting Leaks in Tests

Use `runtime.NumGoroutine()` before and after test:

```go
func TestNoGoroutineLeak(t *testing.T) {
    before := runtime.NumGoroutine()
    
    // Run your test...
    app := gaztest.New(t).Build()
    app.RequireStart()
    app.RequireStop()
    
    after := runtime.NumGoroutine()
    if after > before {
        t.Errorf("goroutine leak: before=%d, after=%d", before, after)
    }
}
```

## Performance Benchmarks

The framework includes benchmark tests for critical paths:

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkResolveSingleton -benchmem ./di/

# Compare benchmarks
go test -bench=. -benchmem -benchcmp=old.txt new.txt
```

## Common Patterns

### Periodic Memory Checks

```go
func setupMemoryMonitoring() {
    ticker := time.NewTicker(1 * time.Minute)
    go func() {
        for range ticker.C {
            var m runtime.MemStats
            runtime.ReadMemStats(&m)
            log.Printf("Memory: Alloc=%d KB, Sys=%d KB, NumGC=%d",
                m.Alloc/1024, m.Sys/1024, m.NumGC)
        }
    }()
}
```

### Profiling Specific Operations

```go
import "runtime/pprof"

func profileOperation() {
    f, _ := os.Create("operation.prof")
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
    
    // Your operation here
    performOperation()
}
```

## Production Considerations

1. **Don't expose pprof endpoints publicly** - Use authentication or bind to localhost
2. **Use signal-based profiling** - Safer for production environments
3. **Monitor resource limits** - Use health checks to detect exhaustion
4. **Set GOMAXPROCS** - Match your CPU count for optimal performance
5. **Enable GC logging** - `GODEBUG=gctrace=1` for GC analysis

## Tools

- **go tool pprof** - Analyze profiles: `go tool pprof cpu.prof`
- **go-torch** - Flame graphs for CPU profiling
- **go tool trace** - Execution tracer for detailed analysis
- **pprof web UI** - `go tool pprof -http=:8080 cpu.prof`

## Example: Complete Profiling Setup

```go
package main

import (
    "context"
    "log"
    "net/http"
    _ "net/http/pprof"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/petabytecl/gaz"
    "github.com/petabytecl/gaz/health"
    "github.com/petabytecl/gaz/health/checks/runtime"
)

func main() {
    app := gaz.New()
    
    // Register health checks with memory/goroutine monitoring
    healthMgr := health.NewManager()
    healthMgr.AddLivenessCheck("memory", runtime.MemoryUsage(1<<30))
    healthMgr.AddLivenessCheck("goroutines", runtime.GoroutineCount(1000))
    
    // Start pprof server (development only)
    if os.Getenv("ENV") == "development" {
        go func() {
            log.Println(http.ListenAndServe("localhost:6060", nil))
        }()
    }
    
    // Setup signal-based profiling (production)
    setupSignalProfiling()
    
    if err := app.Build(); err != nil {
        log.Fatal(err)
    }
    
    app.Run(context.Background())
}

func setupSignalProfiling() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2)
    
    go func() {
        for sig := range sigCh {
            switch sig {
            case syscall.SIGUSR1:
                // Trigger CPU profile (implement as needed)
                log.Println("CPU profiling triggered")
            case syscall.SIGUSR2:
                // Trigger memory profile (implement as needed)
                log.Println("Memory profiling triggered")
            }
        }
    }()
}
```
