# Phase 2: Lifecycle Management - Research

**Researched:** 2026-01-26
**Domain:** Application Lifecycle (Startup, Shutdown, Signals)
**Confidence:** HIGH

## Summary

This research investigates how to implement deterministic application startup and shutdown with support for lifecycle hooks (`OnStart`, `OnStop`), signal handling (`SIGTERM`, `SIGINT`), and dependency-based execution order.

The core challenge is executing hooks in **Topological Order** (Startup) and **Reverse Topological Order** (Shutdown) while supporting **Leveled Concurrency** (running independent hooks in parallel). Since dependencies are resolved dynamically in providers, the dependency graph must be captured during the resolution/build phase to inform the lifecycle engine.

**Primary recommendation:** Implement a **Runtime Graph Recorder** within the DI container that captures dependencies (`A -> B`) during instantiation. Use this graph to drive a **Leveled Execution Engine** (using Kahn's algorithm) for startup, and a **Reverse Level** engine for shutdown.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `context` | stdlib | Cancellation & Timeouts | Standard for propagating termination signals |
| `os/signal` | stdlib | Signal Notification | `NotifyContext` is the modern pattern (Go 1.16+) |
| `golang.org/x/sync/errgroup` | v0.6.0+ | Concurrent Groups | Robust "wait for all or error on first" pattern |
| `sync` | stdlib | Mutexes & WaitGroups | Low-level synchronization primitives |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `time` | stdlib | Timeouts | `context.WithTimeout` for hooks |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `errgroup` | `sync.WaitGroup` + `chan error` | `errgroup` handles context cancellation and error propagation automatically; manual implementation is error-prone |
| `os/signal` | `syscall` | `os/signal` is portable and integrates with `context` |
| Graph-based Order | Stack-based (LIFO) | LIFO is simpler but prevents concurrent execution; Graph is required for "Leveled Concurrent" requirement |

**Installation:**
```bash
go get golang.org/x/sync
```

## Architecture Patterns

### Recommended Project Structure
```
gaz/
├── app.go                # Application entry point (Run, Start, Stop)
├── lifecycle.go          # Lifecycle engine (Graph, LeveledRunner)
├── signal.go             # Signal handling logic
└── graph.go              # Dependency graph data structure & algorithms
```

### Pattern 1: Runtime Graph Recording
**What:** Capture dependency relationships dynamically as services are resolved.
**When to use:** During Phase 1's `Resolve[T]` execution.
**Why:** Providers are opaque functions; we cannot statically analyze dependencies. We must record "Service A requested Service B" at runtime.
**Mechanism:**
1. Maintain `currentResolvingService` (via GID or context).
2. When `Resolve[T]` is called, record edge `current -> T`.
3. Store in adjacency list `map[string][]string`.

### Pattern 2: Leveled Concurrent Execution (Kahn's Algorithm)
**What:** Execute hooks in dependency levels.
**When to use:** `app.Start()` and `app.Stop()`.
**Algorithm:**
1. Calculate in-degrees for all nodes.
2. Identify nodes with 0 dependencies (Level 0).
3. Run Level 0 hooks concurrently (using `errgroup`).
4. On success, remove Level 0 nodes, update degrees, find Level 1.
5. Repeat until all visited.
**Note for Shutdown:** Use the *transpose* graph (reverse edges) or simply reverse the levels computed during startup.

### Pattern 3: Context-Driven Shutdown
**What:** Use `context.Context` to coordinate graceful shutdown.
**When to use:** Handling `SIGTERM`/`SIGINT`.
**Example:**
```go
// Source: Modern Go patterns
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

// Pass ctx to app.Run()
// If signal received, ctx.Done() closes.
// App detects ctx.Done(), initiates graceful shutdown logic.
```

### Pattern 4: Interface-Based Hooks
**What:** Define standard interfaces for lifecycle participation.
**When to use:** Service definitions.
**Example:**
```go
type Starter interface {
    OnStart(context.Context) error
}

type Stopper interface {
    OnStop(context.Context) error
}

// Fluent registration wrapper
func (c *Container) OnStart(fn func(context.Context) error) { ... }
```

### Anti-Patterns to Avoid
- **[Anti-pattern]:** **Global State for Signals:** relying on `os.Exit` or global channels. Pass `context` instead.
- **[Anti-pattern]:** **Blocking Hooks:** `OnStart` must not block indefinitely (e.g., starting a server without a goroutine). It should "start" the work and return.
- **[Anti-pattern]:** **Ignoring Errors:** Startup errors should abort the sequence and trigger rollback (shutdown of already-started services).

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Error Groups | `go func() ...` + `chan` | `errgroup.Group` | Handles first-error context cancellation correctly |
| Signal Handling | `make(chan os.Signal)` | `signal.NotifyContext` | Integrates cleanly with context propagation |
| Topological Sort | Custom sort logic | Kahn's Algorithm | Handles cycle detection and leveling naturally |

## Common Pitfalls

### Pitfall 1: Deadlocks in OnStop
**What goes wrong:** `OnStop` waits for a resource that never closes.
**Why it happens:** Resource shutdown logic is buggy or circular wait.
**How to avoid:** Always use `context.WithTimeout` for the shutdown phase. Force exit if timeout exceeded.

### Pitfall 2: Context Leaks
**What goes wrong:** `OnStart` spawns goroutines that don't respect the parent context.
**Why it happens:** Developers forget to pass `ctx` or check `ctx.Done()`.
**How to avoid:** Ensure `Starter` interface accepts `context.Context` and documentation emphasizes usage.

### Pitfall 3: Incomplete Rollback
**What goes wrong:** Startup fails at Step 5, but Steps 1-4 are left running.
**Why it happens:** Error handling catches the error but doesn't trigger the "Stop" sequence for successful predecessors.
**How to avoid:** On startup error, immediately call the Shutdown sequence for the list of *successfully started* services.

## Code Examples

### Graceful Signal Handling
```go
// Verified Standard Library Pattern
func Run() error {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    app := NewApp()
    
    // Start in background or blocking? 
    // Usually Start() is blocking or returns a "wait" channel.
    // Here we assume Start() runs hooks and returns.
    if err := app.Start(ctx); err != nil {
        return err
    }

    // Wait for signal
    <-ctx.Done()
    
    // Shutdown with timeout
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()
    
    return app.Stop(shutdownCtx)
}
```

### Leveled Execution (Conceptual)
```go
func (g *Graph) RunLevels(ctx context.Context, action func(Service) error) error {
    levels := g.ComputeLevels() // [[A, B], [C], [D]]
    
    for _, level := range levels {
        eg, lvlCtx := errgroup.WithContext(ctx)
        
        for _, svc := range level {
            s := svc
            eg.Go(func() error {
                return action(s)
            })
        }
        
        if err := eg.Wait(); err != nil {
            return err // Stop immediately on error
        }
    }
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `os.Exit` on signal | `signal.NotifyContext` | Go 1.16 | Clean cancellation propagation |
| `OnStart` blocks | `OnStart` spawns, `OnStop` cleans | Modern | Prevents startup hangs |
| Sequential Start | Leveled Concurrent | Modern (Fx, etc.) | Faster startup for large apps |

## Open Questions

Things that couldn't be fully resolved:

1.  **Phase 1 Integration:**
    *   **Question:** Does Phase 1's `Resolve` implementation support recording dependencies?
    *   **Implication:** If not, Phase 2 must modify Phase 1. `Resolve` needs to know *who* is calling it to record the edge.
    *   **Recommendation:** Use the "Invoker Chain" mechanism identified in Phase 1 research to identify the parent.

## Sources

### Primary (HIGH confidence)
- **Official Go Docs:** `os/signal`, `context` - Standard patterns.
- **Uber Fx Documentation:** Lifecycle hooks, strict timeout policies, and startup/shutdown ordering.
- **Phase 1 Research:** Confirmation of "Invoker Chain" existence for dependency tracking.

### Secondary (MEDIUM confidence)
- **Go Ecosystem:** `errgroup` usage for concurrent task management.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH (Standard Go patterns)
- Architecture: HIGH (Kahn's algorithm is mathematically sound for this)
- Pitfalls: HIGH (Common concurrency issues)

**Research date:** 2026-01-26
