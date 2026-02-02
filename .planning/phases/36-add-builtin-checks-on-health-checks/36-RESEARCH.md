# Phase 36: Add Builtin Checks on `health/checks` - Research

**Researched:** 2026-02-02
**Domain:** Go health check library builtin checks for common infrastructure dependencies
**Confidence:** HIGH

## Summary

This phase creates a `health/checks/` package with reusable, production-ready health checks for common infrastructure dependencies. Research examined established Go health check libraries (hellofresh/health-go, heptiolabs/healthcheck, alexliesenfeld/health) to identify the most valuable builtin checks and implementation patterns.

The most commonly needed checks across all libraries are: **database ping**, **Redis ping**, **HTTP upstream**, **DNS resolution**, **TCP connectivity**, and **runtime metrics** (goroutine count, memory usage, disk space). The implementation pattern is consistent: each check is a factory function that returns a `CheckFunc` (matching gaz's existing `func(context.Context) error` signature).

**Primary recommendation:** Create `health/checks/` package with database, redis, http, tcp, dns, runtime (goroutine, memory), and disk checks following the factory function pattern used by heptiolabs/healthcheck and hellofresh/health-go.

## Standard Stack

### Core Check Categories

| Check | Priority | Package | Why Standard |
|-------|----------|---------|--------------|
| Database SQL | P0 | `checks/sql` | Every app uses a database; uses `db.PingContext(ctx)` |
| Redis | P1 | `checks/redis` | Common cache/session store; uses `rdb.Ping(ctx)` |
| HTTP Upstream | P1 | `checks/http` | Verify downstream dependencies are reachable |
| TCP Dial | P1 | `checks/tcp` | Generic connectivity check for any TCP service |
| DNS Resolve | P2 | `checks/dns` | Verify DNS resolution for critical hostnames |
| Goroutine Count | P2 | `checks/runtime` | Detect goroutine leaks (resource exhaustion) |
| Memory Usage | P2 | `checks/runtime` | Detect memory leaks before OOM |
| Disk Space | P3 | `checks/disk` | Prevent disk-full failures (logging, temp files) |
| GC Pause | P3 | `checks/runtime` | Detect GC pressure affecting latency |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `database/sql` | stdlib | SQL database connectivity | SQL/database check |
| `github.com/redis/go-redis/v9` | v9.x | Redis client | Redis check with existing client |
| `github.com/shirou/gopsutil/v4` | v4.x | Cross-platform system metrics | Disk space check |
| `net` | stdlib | TCP/DNS connectivity | TCP dial, DNS resolve checks |
| `net/http` | stdlib | HTTP client | HTTP upstream check |
| `runtime` | stdlib | Go runtime metrics | Goroutine, memory, GC checks |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `gopsutil` for disk | `syscall.Statfs` | Native but Linux-only; gopsutil is cross-platform |
| Factory per check type | Single generic factory | Type-safety vs flexibility; factories are clearer |
| DSN-based config | Client-based config | DSN creates new connections; client reuses pools |

## Architecture Patterns

### Recommended Package Structure

```
health/checks/
├── doc.go            # Package documentation
├── sql/
│   └── sql.go        # SQL database check (database/sql)
├── redis/
│   └── redis.go      # Redis check (go-redis client-based)
├── http/
│   └── http.go       # HTTP upstream check
├── tcp/
│   └── tcp.go        # TCP dial check
├── dns/
│   └── dns.go        # DNS resolution check
├── runtime/
│   └── runtime.go    # Goroutine count, memory, GC pause checks
└── disk/
    └── disk.go       # Disk space check (optional gopsutil)
```

### Pattern 1: Factory Function with Config Struct

**What:** Each check type has a Config struct and New() factory returning `health.CheckFunc`
**When to use:** All builtin checks
**Example:**
```go
// Source: hellofresh/health-go checks/redis/check.go pattern
package sql

import (
    "context"
    "database/sql"
    "fmt"
)

// Config configures the SQL database health check.
type Config struct {
    // DB is the database connection pool to check. Required.
    DB *sql.DB
}

// New creates a new SQL database health check.
// Returns nil if healthy, error if unhealthy.
func New(cfg Config) func(context.Context) error {
    return func(ctx context.Context) error {
        if cfg.DB == nil {
            return fmt.Errorf("sql health check: database connection is nil")
        }
        return cfg.DB.PingContext(ctx)
    }
}
```

### Pattern 2: Client-Based vs DSN-Based Checks

**What:** Accept existing client rather than DSN
**When to use:** Databases, Redis, any pooled connection
**Why:** Reuses connection pools, avoids creating new connections per check
**Example:**
```go
// Source: heptiolabs/healthcheck DatabasePingCheck pattern
// Good: Uses existing connection pool
func New(cfg Config) func(context.Context) error {
    return func(ctx context.Context) error {
        return cfg.DB.PingContext(ctx)  // Uses pool
    }
}

// Bad: Creates new connection per check (from hellofresh pattern)
// Only use when no pooling exists
func NewWithDSN(dsn string) func(context.Context) error {
    return func(ctx context.Context) error {
        db, err := sql.Open("postgres", dsn)  // Creates new connection!
        if err != nil { return err }
        defer db.Close()
        return db.PingContext(ctx)
    }
}
```

### Pattern 3: Threshold-Based Checks

**What:** Accept threshold parameter, fail if exceeded
**When to use:** Runtime metrics (goroutines, memory, disk)
**Example:**
```go
// Source: heptiolabs/healthcheck GoroutineCountCheck
package runtime

import (
    "fmt"
    "runtime"
)

// GoroutineCountCheck returns a check that fails if goroutine count exceeds threshold.
func GoroutineCountCheck(threshold int) func(context.Context) error {
    return func(ctx context.Context) error {
        count := runtime.NumGoroutine()
        if count > threshold {
            return fmt.Errorf("too many goroutines (%d > %d)", count, threshold)
        }
        return nil
    }
}
```

### Pattern 4: Timeout Configuration

**What:** Each check respects context deadline; some have additional timeout config
**When to use:** Network-based checks (HTTP, TCP, DNS)
**Example:**
```go
// Source: heptiolabs/healthcheck TCPDialCheck
package tcp

import (
    "context"
    "net"
    "time"
)

// Config configures the TCP dial health check.
type Config struct {
    // Addr is the address to dial (host:port). Required.
    Addr string
    // Timeout is the dial timeout. Optional, defaults to 2s.
    Timeout time.Duration
}

// New creates a new TCP dial health check.
func New(cfg Config) func(context.Context) error {
    if cfg.Timeout == 0 {
        cfg.Timeout = 2 * time.Second
    }
    return func(ctx context.Context) error {
        // Respect context deadline
        deadline, ok := ctx.Deadline()
        if ok && time.Until(deadline) < cfg.Timeout {
            cfg.Timeout = time.Until(deadline)
        }
        conn, err := net.DialTimeout("tcp", cfg.Addr, cfg.Timeout)
        if err != nil {
            return err
        }
        return conn.Close()
    }
}
```

### Anti-Patterns to Avoid

- **Creating new connections per check:** Use existing client/pool, not DSN
- **Ignoring context cancellation:** Always respect `ctx.Done()`
- **Running expensive queries:** Use `Ping()` not `SELECT 1` for databases
- **Hard-coding thresholds:** Always make thresholds configurable
- **Blocking forever:** All checks should have timeout protection
- **Leaking connections:** Always close connections in checks that open them

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Database connectivity | Custom TCP check | `db.PingContext(ctx)` | Database drivers know connection health |
| Redis connectivity | TCP check to Redis port | `rdb.Ping(ctx)` | Verifies actual Redis protocol |
| Disk space (cross-platform) | Manual syscall | `github.com/shirou/gopsutil/v4/disk` | Works on Linux, macOS, Windows |
| HTTP status check | Raw TCP + parsing | `http.Client.Do()` | Handles redirects, TLS, HTTP/2 |
| DNS resolution | Manual UDP DNS | `net.Resolver.LookupHost()` | Handles system DNS config |

**Key insight:** Use the highest-level API available (Ping over TCP, library over raw syscall). Lower-level implementations miss edge cases that libraries handle.

## Common Pitfalls

### Pitfall 1: Connection Pool Exhaustion

**What goes wrong:** Health check consumes connections from limited pool
**Why it happens:** Creating new connections or holding connections too long
**How to avoid:** Use `PingContext` (quick), ensure checks are fast (< 2s)
**Warning signs:** "too many connections" errors during health checks

### Pitfall 2: Health Check DDoS

**What goes wrong:** Frequent health checks overload dependencies
**Why it happens:** Kubernetes probes every 2s, load balancers even more frequent
**How to avoid:** Use result caching at the checker level (already in health package); keep checks cheap
**Warning signs:** High load on dependencies correlated with health probe frequency

### Pitfall 3: Missing Context Cancellation

**What goes wrong:** Check continues running after timeout, leaking goroutines
**Why it happens:** Not checking `ctx.Done()` in long-running checks
**How to avoid:** Use context-aware APIs (`PingContext`, `DialContext`, etc.)
**Warning signs:** Goroutine count increasing over time

### Pitfall 4: Wrong Probe Type

**What goes wrong:** Database check on liveness causes restart loops when DB is down
**Why it happens:** Misunderstanding liveness vs readiness
**How to avoid:** Document which checks are appropriate for each probe type:
  - Liveness: Process health (goroutine count, memory, deadlock detection)
  - Readiness: Dependency health (database, redis, upstreams)
  - Startup: Initialization (cache warm, migrations complete)
**Warning signs:** Pods restarting when external dependencies have issues

### Pitfall 5: Exposing Sensitive Information

**What goes wrong:** Error messages contain connection strings, internal IPs
**Why it happens:** Passing raw errors to response
**How to avoid:** Wrap errors with generic message; IETF writer already handles this via `showErrorDetails`
**Warning signs:** Security scan findings, customer complaints about error details

### Pitfall 6: Nil Client Panic

**What goes wrong:** Check panics when passed nil client
**Why it happens:** Not validating config before using
**How to avoid:** Check for nil and return descriptive error
**Warning signs:** Panic recovery in checker logs

## Code Examples

### SQL Database Check (Primary Pattern)

```go
// Source: gaz health package pattern + heptiolabs/healthcheck DatabasePingCheck
package sql

import (
    "context"
    "database/sql"
    "fmt"
)

// Config configures the SQL database health check.
type Config struct {
    // DB is the database connection pool to check. Required.
    DB *sql.DB
}

// New creates a new SQL database health check.
// Validates connectivity using PingContext which is optimized for connection testing.
func New(cfg Config) func(context.Context) error {
    return func(ctx context.Context) error {
        if cfg.DB == nil {
            return fmt.Errorf("sql: database connection is nil")
        }
        if err := cfg.DB.PingContext(ctx); err != nil {
            return fmt.Errorf("sql: ping failed: %w", err)
        }
        return nil
    }
}
```

### Redis Check (Client-Based)

```go
// Source: go-redis ping pattern + hellofresh pattern
package redis

import (
    "context"
    "fmt"

    "github.com/redis/go-redis/v9"
)

// Config configures the Redis health check.
type Config struct {
    // Client is the Redis client to check. Required.
    Client redis.UniversalClient
}

// New creates a new Redis health check.
// Uses PING command to verify connectivity and response.
func New(cfg Config) func(context.Context) error {
    return func(ctx context.Context) error {
        if cfg.Client == nil {
            return fmt.Errorf("redis: client is nil")
        }
        pong, err := cfg.Client.Ping(ctx).Result()
        if err != nil {
            return fmt.Errorf("redis: ping failed: %w", err)
        }
        if pong != "PONG" {
            return fmt.Errorf("redis: unexpected ping response: %q", pong)
        }
        return nil
    }
}
```

### HTTP Upstream Check

```go
// Source: heptiolabs/healthcheck HTTPGetCheck + hellofresh/health-go http check
package http

import (
    "context"
    "fmt"
    "net/http"
    "time"
)

// Config configures the HTTP upstream health check.
type Config struct {
    // URL is the health check URL. Required.
    URL string
    // Timeout for the HTTP request. Optional, defaults to 5s.
    Timeout time.Duration
    // ExpectedStatusCode is the expected response status. Optional, defaults to 200.
    ExpectedStatusCode int
}

// New creates a new HTTP upstream health check.
// Performs GET request and validates response status.
func New(cfg Config) func(context.Context) error {
    if cfg.Timeout == 0 {
        cfg.Timeout = 5 * time.Second
    }
    if cfg.ExpectedStatusCode == 0 {
        cfg.ExpectedStatusCode = http.StatusOK
    }
    
    client := &http.Client{
        Timeout: cfg.Timeout,
        CheckRedirect: func(*http.Request, []*http.Request) error {
            return http.ErrUseLastResponse // Don't follow redirects
        },
    }
    
    return func(ctx context.Context) error {
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
        if err != nil {
            return fmt.Errorf("http: failed to create request: %w", err)
        }
        req.Header.Set("Connection", "close")
        
        resp, err := client.Do(req)
        if err != nil {
            return fmt.Errorf("http: request failed: %w", err)
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != cfg.ExpectedStatusCode {
            return fmt.Errorf("http: unexpected status %d (expected %d)", 
                resp.StatusCode, cfg.ExpectedStatusCode)
        }
        return nil
    }
}
```

### TCP Dial Check

```go
// Source: heptiolabs/healthcheck TCPDialCheck
package tcp

import (
    "context"
    "fmt"
    "net"
    "time"
)

// Config configures the TCP dial health check.
type Config struct {
    // Addr is the address to dial (host:port). Required.
    Addr string
    // Timeout for the dial. Optional, defaults to 2s.
    Timeout time.Duration
}

// New creates a new TCP dial health check.
func New(cfg Config) func(context.Context) error {
    if cfg.Timeout == 0 {
        cfg.Timeout = 2 * time.Second
    }
    
    return func(ctx context.Context) error {
        var d net.Dialer
        d.Timeout = cfg.Timeout
        
        conn, err := d.DialContext(ctx, "tcp", cfg.Addr)
        if err != nil {
            return fmt.Errorf("tcp: dial failed: %w", err)
        }
        return conn.Close()
    }
}
```

### DNS Resolve Check

```go
// Source: heptiolabs/healthcheck DNSResolveCheck
package dns

import (
    "context"
    "fmt"
    "net"
    "time"
)

// Config configures the DNS resolution health check.
type Config struct {
    // Host is the hostname to resolve. Required.
    Host string
    // Timeout for the resolution. Optional, defaults to 2s.
    Timeout time.Duration
}

// New creates a new DNS resolution health check.
func New(cfg Config) func(context.Context) error {
    if cfg.Timeout == 0 {
        cfg.Timeout = 2 * time.Second
    }
    
    resolver := &net.Resolver{}
    
    return func(ctx context.Context) error {
        ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
        defer cancel()
        
        addrs, err := resolver.LookupHost(ctx, cfg.Host)
        if err != nil {
            return fmt.Errorf("dns: lookup failed: %w", err)
        }
        if len(addrs) == 0 {
            return fmt.Errorf("dns: no addresses found for %s", cfg.Host)
        }
        return nil
    }
}
```

### Runtime Checks (Goroutine, Memory, GC)

```go
// Source: heptiolabs/healthcheck GoroutineCountCheck, GCMaxPauseCheck
package runtime

import (
    "context"
    "fmt"
    "runtime"
    "time"
)

// GoroutineCountCheck returns a check that fails if goroutine count exceeds threshold.
// Useful for detecting goroutine leaks.
func GoroutineCountCheck(threshold int) func(context.Context) error {
    return func(ctx context.Context) error {
        count := runtime.NumGoroutine()
        if count > threshold {
            return fmt.Errorf("runtime: too many goroutines (%d > %d)", count, threshold)
        }
        return nil
    }
}

// MemoryUsageCheck returns a check that fails if heap allocation exceeds threshold.
// threshold is in bytes.
func MemoryUsageCheck(threshold uint64) func(context.Context) error {
    return func(ctx context.Context) error {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        if m.Alloc > threshold {
            return fmt.Errorf("runtime: memory usage too high (%d bytes > %d bytes)", 
                m.Alloc, threshold)
        }
        return nil
    }
}

// GCPauseCheck returns a check that fails if recent GC pause exceeds threshold.
func GCPauseCheck(threshold time.Duration) func(context.Context) error {
    thresholdNs := uint64(threshold.Nanoseconds())
    return func(ctx context.Context) error {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        for _, pause := range m.PauseNs {
            if pause > thresholdNs {
                return fmt.Errorf("runtime: GC pause too long (%s > %s)", 
                    time.Duration(pause), threshold)
            }
        }
        return nil
    }
}
```

### Disk Space Check (Optional gopsutil)

```go
// Source: gopsutil disk.Usage pattern
package disk

import (
    "context"
    "fmt"

    "github.com/shirou/gopsutil/v4/disk"
)

// Config configures the disk space health check.
type Config struct {
    // Path is the filesystem path to check. Required.
    Path string
    // ThresholdPercent is the maximum usage percentage allowed. Required.
    ThresholdPercent float64
}

// New creates a new disk space health check.
// Fails if disk usage exceeds the threshold percentage.
func New(cfg Config) func(context.Context) error {
    return func(ctx context.Context) error {
        usage, err := disk.UsageWithContext(ctx, cfg.Path)
        if err != nil {
            return fmt.Errorf("disk: failed to get usage: %w", err)
        }
        if usage.UsedPercent > cfg.ThresholdPercent {
            return fmt.Errorf("disk: usage %.1f%% exceeds threshold %.1f%%", 
                usage.UsedPercent, cfg.ThresholdPercent)
        }
        return nil
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| DSN-based checks | Client-based checks | 2024+ | Reuses connection pools |
| `SELECT 1` for DB | `db.Ping()` / `db.PingContext()` | Always | Optimized for health checks |
| Global timeouts | Per-check context timeouts | 2024+ | Fine-grained control |
| syscall for disk | gopsutil | 2022+ | Cross-platform compatibility |

**Current best practices (2025-2026):**
- Use client/pool-based checks (not DSN)
- Always use context-aware APIs
- Separate liveness (cheap) from readiness (dependency checks)
- Cache results at checker level to prevent probe DDoS
- Threshold-based checks for runtime metrics

## Open Questions

1. **gopsutil Dependency for Disk Checks**
   - What we know: gopsutil provides cross-platform disk stats
   - What's unclear: Should we add external dependency or provide stdlib-only alternative?
   - Recommendation: Make disk check optional (separate submodule or build tag), provide stdlib fallback for Linux

2. **Redis Client Interface**
   - What we know: go-redis has `UniversalClient` interface
   - What's unclear: What about other Redis clients (redigo, rueidis)?
   - Recommendation: Accept `redis.UniversalClient` for now; users with other clients can wrap or use TCP check

3. **Check Naming Conventions**
   - What we know: Checks need unique names when registered
   - What's unclear: Should checks provide default names?
   - Recommendation: Factory returns `func(context.Context) error`; user provides name at registration time

4. **Version-Specific Redis Features**
   - What we know: go-redis v9 is current standard
   - What's unclear: Should we support older go-redis versions?
   - Recommendation: Support v9 only; v8 is deprecated

## Sources

### Primary (HIGH confidence)

- **Context7 /alexliesenfeld/health** - Factory pattern, Check struct, synchronous checks
- **heptiolabs/healthcheck (GitHub)** - TCPDialCheck, DatabasePingCheck, DNSResolveCheck, GoroutineCountCheck, GCMaxPauseCheck implementations
- **hellofresh/health-go (GitHub)** - Check organization pattern, redis/mysql/postgres/http/grpc check implementations
- **Context7 /redis/go-redis** - Ping command, client configuration
- **Context7 /shirou/gopsutil** - Disk usage, memory stats

### Secondary (MEDIUM confidence)

- **Google Search 2025-2026** - Best practices for health checks, disk/memory monitoring patterns
- **gaz health package** - Existing CheckFunc signature, Manager registration pattern

### Tertiary (LOW confidence)

None - all findings verified with primary sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Multiple established libraries use same patterns
- Architecture: HIGH - Factory function pattern verified across 3 libraries
- Pitfalls: HIGH - Common issues documented in library issue trackers
- Code examples: HIGH - Based on verified library implementations

**Research date:** 2026-02-02
**Valid until:** 2026-03-02 (30 days - stable domain)
