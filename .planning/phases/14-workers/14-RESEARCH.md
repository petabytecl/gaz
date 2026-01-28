# Phase 14: Workers - Research

**Researched:** 2026-01-28
**Domain:** Go background worker lifecycle management, goroutine supervision, panic recovery
**Confidence:** HIGH

## Summary

This phase adds background worker support to gaz with lifecycle integration, graceful shutdown, and panic recovery. The research focused on Go concurrency patterns, exponential backoff algorithms, circuit breaker implementations, and structured logging for workers.

The standard approach in Go (2025/2026) uses context-driven goroutine management with `sync.WaitGroup` for coordination. Workers implement a consistent interface and are auto-discovered by a WorkerManager. Panic recovery wraps the main worker loop using `defer recover()` with structured logging via `log/slog`. Exponential backoff with jitter prevents thundering herd issues during restarts. Circuit breakers prevent cascading failures by limiting restart attempts in time windows.

The locked decisions from CONTEXT.md (Start/Stop interface, internal goroutine spawning, exponential backoff with circuit breaker) align well with Go best practices. The implementation will use jpillora/backoff for its simplicity and avoid external dependencies for circuit breaker logic (hand-roll using simple counters is appropriate for this scope).

**Primary recommendation:** Implement a WorkerManager that auto-discovers Worker implementations, wraps each in a supervisor goroutine with panic recovery, uses jpillora/backoff for restart delays, and integrates with gaz's existing Starter/Stopper lifecycle.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `context` | 1.25+ | Cancellation propagation | Standard Go pattern for goroutine lifecycle |
| Go stdlib `sync` | 1.25+ | WaitGroup for goroutine coordination | Proven, zero-allocation synchronization |
| Go stdlib `log/slog` | 1.25+ | Structured logging | Standard since Go 1.21, already used in gaz |
| Go stdlib `runtime/debug` | 1.25+ | Stack trace capture for panic recovery | Only reliable way to get goroutine stacks |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| jpillora/backoff | latest | Exponential backoff counter | For restart delay calculation with jitter |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| jpillora/backoff | cenkalti/backoff | cenkalti wraps retry logic; jpillora is simpler counter API, fits gaz's internal supervision model better |
| jpillora/backoff | hand-rolled | jpillora is 100 LOC, well-tested, jitter included; not worth hand-rolling |
| sony/gobreaker | hand-rolled | gobreaker is overkill for worker restarts; simple counter-based logic is clearer |

**Installation:**
```bash
go get github.com/jpillora/backoff
```

## Architecture Patterns

### Recommended Project Structure
```
gaz/
├── worker/
│   ├── worker.go          # Worker interface definition
│   ├── manager.go         # WorkerManager implementation
│   ├── supervisor.go      # Per-worker supervision (panic recovery, restart)
│   ├── options.go         # Registration options (poolSize, critical, etc.)
│   ├── backoff.go         # Backoff configuration wrapper
│   └── doc.go             # Package documentation
```

### Pattern 1: Worker Interface (LOCKED)
**What:** Interface with `Start()`, `Stop()`, `Name()` - worker spawns its own goroutine internally
**When to use:** All background workers that need lifecycle integration
**Example:**
```go
// Source: CONTEXT.md locked decision
type Worker interface {
    // Start begins the worker. Returns immediately; worker spawns its own goroutine.
    Start()
    
    // Stop signals the worker to shutdown. Worker decides when to return.
    Stop()
    
    // Name returns a human-readable identifier for logging.
    Name() string
}
```

### Pattern 2: Supervisor Wrapper (Panic Recovery + Restart)
**What:** WorkerManager wraps each worker in a supervisor goroutine that handles panics and restarts
**When to use:** For all workers to ensure app doesn't crash on panic
**Example:**
```go
// Source: Go best practices 2025, adapted for gaz
func (m *WorkerManager) supervise(ctx context.Context, w Worker) {
    b := &backoff.Backoff{
        Min:    1 * time.Second,
        Max:    5 * time.Minute,
        Factor: 2,
        Jitter: true,
    }
    
    failures := 0
    windowStart := time.Now()
    
    for {
        select {
        case <-ctx.Done():
            return
        default:
        }
        
        // Run with panic recovery
        panicked := m.runWithRecovery(ctx, w)
        
        if panicked {
            failures++
            
            // Circuit breaker: check failure window
            if time.Since(windowStart) > m.circuitWindow {
                failures = 1
                windowStart = time.Now()
            }
            
            if failures >= m.maxRestarts {
                m.logger.Error("worker exhausted restarts",
                    "worker", w.Name(),
                    "failures", failures,
                )
                if m.isCritical(w) {
                    m.signalAppShutdown()
                }
                return
            }
            
            delay := b.Duration()
            m.logger.Info("worker restarting",
                "worker", w.Name(),
                "delay", delay,
                "failures", failures,
            )
            
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return
            }
        } else {
            // Clean exit - don't restart
            return
        }
    }
}

func (m *WorkerManager) runWithRecovery(ctx context.Context, w Worker) (panicked bool) {
    defer func() {
        if r := recover(); r != nil {
            stack := debug.Stack()
            m.logger.Error("worker panicked",
                "worker", w.Name(),
                "panic", r,
                "stack", string(stack),
            )
            panicked = true
        }
    }()
    
    w.Start()
    // Wait for stop signal or context cancellation
    <-ctx.Done()
    w.Stop()
    return false
}
```

### Pattern 3: Worker Pool Registration
**What:** Single worker type can be registered with pool size, creating N instances
**When to use:** Queue processors, parallel work consumers
**Example:**
```go
// Source: Gaz registration pattern + pool extension
type WorkerRegistration struct {
    worker   Worker
    poolSize int
    critical bool
}

// Registration creates pool workers with indexed names
func (m *WorkerManager) registerPool(w Worker, poolSize int) {
    for i := 1; i <= poolSize; i++ {
        pooledWorker := &pooledWorker{
            delegate: w,
            index:    i,
            name:     fmt.Sprintf("%s-%d", w.Name(), i),
        }
        m.workers = append(m.workers, pooledWorker)
    }
}
```

### Pattern 4: Scoped Logger per Worker
**What:** Each worker gets a logger with pre-attached worker name field
**When to use:** All worker logging to ensure consistent traceability
**Example:**
```go
// Source: Go slog best practices 2025
func (m *WorkerManager) startWorker(w Worker) {
    workerLogger := m.logger.With(
        slog.String("worker", w.Name()),
    )
    
    workerLogger.Info("worker starting")
    // Pass logger to supervisor
}
```

### Anti-Patterns to Avoid
- **Panic in recovery block:** Keep recover logic simple; avoid operations that can fail (like database writes)
- **Global state in workers:** Pass context and logger explicitly for testability
- **Ignoring context cancellation:** Always check `ctx.Done()` in long-running loops
- **Unbounded restart loops:** Always use circuit breaker to prevent infinite restart cycles
- **Synchronous Stop():** Worker's Stop() should signal shutdown, not block indefinitely

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Exponential backoff with jitter | Manual time.Sleep multiplication | jpillora/backoff | Jitter algorithm is tricky; AWS-style jitter prevents thundering herd |
| Stack trace capture | fmt.Sprintf with runtime.Stack | runtime/debug.Stack() | debug.Stack returns formatted, human-readable trace |
| Context cancellation | Manual done channels | context.Context | Standard, composable, works with timeouts |

**Key insight:** The circuit breaker for worker restarts is simple enough to hand-roll (just a counter + time window). Full circuit breaker libraries like sony/gobreaker are designed for request-level failures, not worker supervision.

## Common Pitfalls

### Pitfall 1: Goroutine Leaks on Shutdown
**What goes wrong:** Worker's internal goroutine doesn't exit when Stop() is called
**Why it happens:** Worker doesn't check for stop signal in its main loop
**How to avoid:** Workers must implement internal stop channel or use context cancellation
**Warning signs:** Tests timeout on shutdown; memory grows over time

### Pitfall 2: Panic in Nested Goroutines
**What goes wrong:** Worker spawns sub-goroutines that panic; parent's recover doesn't catch them
**Why it happens:** Each goroutine needs its own defer/recover; panics don't propagate
**How to avoid:** Document that workers spawning goroutines must handle their own panics OR provide SafeGo utility
**Warning signs:** App crashes despite recovery wrapper

### Pitfall 3: Thundering Herd on Restart
**What goes wrong:** All workers restart simultaneously after a shared failure (e.g., database outage)
**Why it happens:** Fixed backoff without jitter
**How to avoid:** Use jpillora/backoff with Jitter: true
**Warning signs:** Spiky resource usage patterns after recovery

### Pitfall 4: Race Between Start and Stop
**What goes wrong:** Stop() called before Start() has fully initialized
**Why it happens:** Worker's internal state not protected
**How to avoid:** Workers should be safe to Stop() immediately after Start(); use sync primitives internally
**Warning signs:** Nil pointer panics during shutdown

### Pitfall 5: Critical Worker Not Crashing App
**What goes wrong:** Critical worker marked but app continues running after it dies
**Why it happens:** Circuit breaker logic didn't propagate to app shutdown
**How to avoid:** WorkerManager has callback or channel to signal app shutdown; App.Run() listens
**Warning signs:** App in degraded state without health checks detecting it

## Code Examples

Verified patterns from official sources:

### Backoff Configuration (Recommended Defaults)
```go
// Source: jpillora/backoff documentation + research
b := &backoff.Backoff{
    Min:    1 * time.Second,   // First retry after 1s
    Max:    5 * time.Minute,   // Cap at 5 minutes
    Factor: 2,                 // Double each time: 1s, 2s, 4s, 8s...
    Jitter: true,              // Randomize to prevent thundering herd
}

// Usage in restart loop
delay := b.Duration()  // Get next delay (increments internal counter)
// ... wait ...

// After stable run period, reset
b.Reset()
```

### Circuit Breaker Logic (Simple Counter)
```go
// Source: Research synthesis - simple counter is sufficient for worker supervision
type circuitBreaker struct {
    maxFailures int           // e.g., 5
    window      time.Duration // e.g., 10 * time.Minute
    
    failures    int
    windowStart time.Time
}

func (cb *circuitBreaker) recordFailure() (tripped bool) {
    now := time.Now()
    
    // Reset window if expired
    if now.Sub(cb.windowStart) > cb.window {
        cb.failures = 0
        cb.windowStart = now
    }
    
    cb.failures++
    return cb.failures >= cb.maxFailures
}

func (cb *circuitBreaker) reset() {
    cb.failures = 0
    cb.windowStart = time.Now()
}
```

### Worker Integration with Gaz Lifecycle
```go
// Source: Existing gaz patterns (lifecycle.go, app.go)

// Worker auto-discovery via interface check during Build()
func (a *App) discoverWorkers() {
    a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
        instance, _ := svc.GetInstance(a.container.inner, nil)
        if w, ok := instance.(Worker); ok {
            a.workerManager.Register(w)
        }
    })
}

// Workers start after all OnStart hooks complete
func (a *App) Run(ctx context.Context) error {
    // ... existing startup ...
    
    // Start worker manager (spawns all supervisors)
    if err := a.workerManager.Start(ctx); err != nil {
        return err
    }
    
    return a.waitForShutdownSignal(ctx)
}
```

### Panic Recovery with Stack Trace
```go
// Source: Go 2025 best practices, runtime/debug documentation
func runWithRecovery(logger *slog.Logger, name string, fn func()) (panicked bool) {
    defer func() {
        if r := recover(); r != nil {
            stack := debug.Stack()
            logger.Error("recovered from panic",
                "worker", name,
                "panic", r,
                "stack", string(stack),
            )
            panicked = true
        }
    }()
    
    fn()
    return false
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `fmt.Printf` panic logging | `log/slog` structured logging | Go 1.21 (2023) | Parseable logs, consistent fields |
| Manual recover() everywhere | Centralized wrapper function | 2024+ | Reduced boilerplate, consistent handling |
| Fixed sleep delays | Exponential backoff with jitter | Standard practice | Prevents thundering herd |
| Run(ctx) blocking pattern | Start()/Stop() non-blocking | Per CONTEXT.md decision | More control, cleaner shutdown |

**Deprecated/outdated:**
- `log` package for production: Use `log/slog` instead (standard since Go 1.21)
- rand.Seed(): Deprecated in Go 1.20+, random is auto-seeded

## Open Questions

Things that couldn't be fully resolved:

1. **Stable run period for backoff reset**
   - What we know: After a worker runs without crashing for some period, backoff should reset
   - What's unclear: Exact duration (30 seconds? 1 minute? 5 minutes?)
   - Recommendation: Use 30 seconds as default, make configurable via option

2. **Critical worker mechanism**
   - What we know: Critical workers should crash app when circuit breaker trips
   - What's unclear: Interface method (`IsCritical() bool`) vs registration option (`WithCritical()`)
   - Recommendation: Use registration option `WithCritical()` - keeps Worker interface clean, matches gaz's Option pattern

3. **Pool worker cloning**
   - What we know: Pool creates N workers from one registration
   - What's unclear: Should workers be cloned or should each call a factory?
   - Recommendation: Require factory function for pools (`WithPoolSize(n, factory)`) to ensure separate state per instance

## Sources

### Primary (HIGH confidence)
- Go stdlib documentation (context, sync, runtime/debug, log/slog)
- jpillora/backoff Context7 documentation - backoff counter API, jitter implementation
- sony/gobreaker Context7 documentation - circuit breaker patterns (for reference, not direct use)
- Existing gaz codebase patterns (di/lifecycle.go, app.go, di/registration.go)

### Secondary (MEDIUM confidence)
- WebSearch: "Go worker pool goroutine graceful shutdown context cancellation 2026" - modern context.WithCancelCause patterns
- WebSearch: "Go circuit breaker pattern goroutine restart max retries 2025" - supervisor patterns
- WebSearch: "Go slog structured logging worker goroutine fields 2025" - scoped logger patterns
- WebSearch: "Go panic recover goroutine background worker 2025" - SafeGo pattern, stack trace capture

### Tertiary (LOW confidence)
- None - all findings verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Go stdlib is definitive; jpillora/backoff verified via Context7
- Architecture: HIGH - Patterns match existing gaz code and Go best practices
- Pitfalls: HIGH - Based on well-documented Go concurrency issues

**Research date:** 2026-01-28
**Valid until:** 2026-02-28 (30 days - stable domain)
