# Phase 8: Hardened Lifecycle - Research

**Researched:** 2026-01-27
**Domain:** Go signal handling, context timeouts, graceful shutdown enforcement
**Confidence:** HIGH

## Summary

This phase implements hardened shutdown guarantees for the gaz application framework. The research covers Go patterns for double-signal handling (Ctrl+C twice forces exit), context timeout enforcement for individual hooks, and blame logging for debugging production hang scenarios.

The Go standard library's `os/signal` package combined with `context.WithTimeout` provides all the primitives needed. The patterns are well-established in production Go software (Consul, Docker Compose, kubernetes tooling) and the codebase already has foundation pieces in place.

**Primary recommendation:** Implement a shutdown orchestrator that wraps each hook invocation with per-hook timeouts, tracks which hook is currently running for blame logging, and runs a parallel goroutine listening for force-exit signals. Use `os.Exit(1)` for force termination since deferred functions won't run anyway on hard exit.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/signal` | stdlib | Signal notification and handling | Go stdlib for signal handling |
| `context` | stdlib | Timeout and cancellation propagation | Standard Go context pattern |
| `time` | stdlib | Timeouts, tickers, deadlines | Standard Go timing |
| `syscall` | stdlib | SIGINT, SIGTERM constants | Standard signal constants |
| `os` | stdlib | os.Exit for force termination | Standard process control |
| `sync/atomic` | stdlib | Lock-free double-signal detection | Race-free signal counting |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `errors` | stdlib | errors.Join for multi-error collection | Collecting hook errors |
| `log/slog` | stdlib | Structured logging with fallback | Blame logging (already in codebase) |
| `fmt` | stdlib | Direct stderr fallback | When logger may be broken |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `sync/atomic` for signal counting | `sync.Mutex` | Atomic is simpler for counters, no lock contention |
| Direct `os.Exit(1)` | `panic()` + recover | Exit is cleaner for force termination, no stack traces |
| Per-hook context timeout | Single global timeout only | Per-hook gives better debugging, matches user decisions |

**Installation:**
No new dependencies required - all stdlib.

## Architecture Patterns

### Recommended Project Structure
```
app.go              # Modified: Run() method enhanced with hardened shutdown
shutdown.go         # NEW: Shutdown orchestrator with timeout enforcement
lifecycle.go        # Modified: HookConfig gets Timeout field
```

### Pattern 1: Double-Signal Force Exit (Consul Pattern)
**What:** Track signal count, force exit on second signal during graceful shutdown
**When to use:** When first signal starts graceful shutdown, second forces immediate exit
**Example:**
```go
// Source: HashiCorp Consul agent.go pattern
// Adapted for gaz

func (a *App) runWithSignals(ctx context.Context) error {
    signalCh := make(chan os.Signal, 1)
    signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
    defer signal.Stop(signalCh)

    // First signal - start graceful shutdown
    select {
    case <-ctx.Done():
        return a.gracefulShutdown()
    case sig := <-signalCh:
        a.Logger.Info("Shutting down gracefully...", "signal", sig)
        fmt.Fprintln(os.Stderr, "Shutting down gracefully... (Ctrl+C again to force)")

        // Run graceful shutdown in goroutine
        gracefulCh := make(chan error, 1)
        go func() {
            gracefulCh <- a.gracefulShutdown()
        }()

        // Wait for graceful or second signal
        select {
        case err := <-gracefulCh:
            return err
        case <-signalCh:
            a.Logger.Error("Force exit requested")
            fmt.Fprintln(os.Stderr, "force exit")
            os.Exit(1)
        }
    }
    return nil
}
```

### Pattern 2: Per-Hook Timeout with Blame Logging
**What:** Wrap each hook with its own timeout, log which hook exceeded timeout
**When to use:** Sequential hook execution with per-hook deadlines
**Example:**
```go
// Source: Custom pattern based on uber-go/fx and gaz requirements

type HookResult struct {
    HookName    string
    Elapsed     time.Duration
    Error       error
    TimedOut    bool
}

func (a *App) runHookWithTimeout(
    ctx context.Context,
    hookName string,
    hook func(context.Context) error,
    timeout time.Duration,
) HookResult {
    start := time.Now()
    hookCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Run hook
    errCh := make(chan error, 1)
    go func() {
        errCh <- hook(hookCtx)
    }()

    select {
    case err := <-errCh:
        return HookResult{
            HookName: hookName,
            Elapsed:  time.Since(start),
            Error:    err,
            TimedOut: false,
        }
    case <-hookCtx.Done():
        // Timed out - log blame
        elapsed := time.Since(start)
        a.logBlame(hookName, timeout, elapsed)
        return HookResult{
            HookName: hookName,
            Elapsed:  elapsed,
            Error:    hookCtx.Err(),
            TimedOut: true,
        }
    }
}

func (a *App) logBlame(hookName string, timeout, elapsed time.Duration) {
    msg := fmt.Sprintf("shutdown: %s exceeded %s timeout (elapsed: %s)",
        hookName, timeout, elapsed)
    
    // Try logger first
    if a.Logger != nil {
        a.Logger.Error(msg, "hook", hookName, "timeout", timeout, "elapsed", elapsed)
    }
    // Always write to stderr as fallback (guaranteed output)
    fmt.Fprintln(os.Stderr, msg)
}
```

### Pattern 3: Global Timeout with Force Exit
**What:** Background goroutine that forces exit when global timeout expires
**When to use:** Guarantee process termination regardless of hook behavior
**Example:**
```go
// Source: Custom pattern for LIFE-01/LIFE-02 requirements

func (a *App) gracefulShutdown() error {
    globalTimeout := a.opts.ShutdownTimeout
    
    // Start force-exit timer
    done := make(chan struct{})
    go func() {
        select {
        case <-done:
            return
        case <-time.After(globalTimeout):
            msg := fmt.Sprintf("shutdown: global timeout %s exceeded, forcing exit", globalTimeout)
            a.Logger.Error(msg)
            fmt.Fprintln(os.Stderr, msg)
            os.Exit(1)
        }
    }()
    defer close(done)

    // Run shutdown hooks sequentially
    return a.stopServicesWithBlame(context.Background())
}
```

### Pattern 4: Context Deadline Propagation to Hooks
**What:** Pass context with deadline to hooks so they can check ctx.Done()
**When to use:** Allow hooks to cooperatively check for timeout
**Example:**
```go
// Source: Go database/sql cancel-operations pattern

func (s *MyService) OnStop(ctx context.Context) error {
    // Hook can check for timeout
    select {
    case <-ctx.Done():
        return ctx.Err() // context.DeadlineExceeded or context.Canceled
    case <-s.shutdown():
        return nil
    }
}

// Or use context directly with blocking operations
func (db *Database) OnStop(ctx context.Context) error {
    return db.conn.CloseContext(ctx) // Respects context deadline
}
```

### Anti-Patterns to Avoid
- **Calling os.Exit inside defer:** Deferred functions won't run after os.Exit - only use for force termination
- **Blocking on hook completion after global timeout:** If hook is stuck, waiting more won't help
- **Using panic for force exit:** Creates stack traces, confuses error handling
- **Single shared context for all hooks:** Per-hook contexts allow finer control
- **Ignoring context in hooks:** Hooks should check ctx.Done() for cooperative cancellation

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Signal handling | Raw syscall signal handlers | `os/signal.Notify` | Handles cross-platform, buffering, cleanup |
| Timeout enforcement | Manual timer + cancel | `context.WithTimeout` | Standard pattern, composable, cancelled properly |
| Double-signal detection | Mutex + counter | `sync/atomic` counter or channel receive | Race-free, simpler |
| Multi-error collection | Custom error type | `errors.Join` (Go 1.20+) | Standard, supports errors.Is/As |

**Key insight:** The stdlib provides all primitives needed. The complexity is in orchestration, not in the building blocks.

## Common Pitfalls

### Pitfall 1: Signal Channel Buffer Size
**What goes wrong:** Missing signals because channel is unbuffered or too small
**Why it happens:** signal.Notify documents this: "Package signal will not block sending to c"
**How to avoid:** Use buffered channel: `make(chan os.Signal, 1)` minimum
**Warning signs:** Signals seem to be "ignored" intermittently

### Pitfall 2: Not Calling signal.Stop
**What goes wrong:** Signal handlers leak, affecting later tests or code
**Why it happens:** signal.Notify adds to set of signals, must explicitly stop
**How to avoid:** Always `defer signal.Stop(signalCh)` after Notify
**Warning signs:** Tests interfere with each other, signals handled unexpectedly

### Pitfall 3: Deferred Functions Don't Run After os.Exit
**What goes wrong:** Cleanup code in defers doesn't execute on force exit
**Why it happens:** os.Exit terminates immediately, bypassing defer stack
**How to avoid:** Accept this for force exit (that's the point), ensure graceful path uses normal returns
**Warning signs:** Resources not cleaned up after force exit (expected behavior)

### Pitfall 4: Context Cancellation vs Deadline
**What goes wrong:** Using context.WithCancel when you need WithTimeout
**Why it happens:** Both return ctx.Done(), but semantics differ
**How to avoid:** Use WithTimeout for time-based limits, WithCancel for explicit cancellation
**Warning signs:** Hooks never time out, or cancel at wrong times

### Pitfall 5: Hook Doesn't Respect Context
**What goes wrong:** Hook ignores ctx.Done(), runs forever despite timeout
**Why it happens:** Hook author didn't check context, uses blocking calls without context
**How to avoid:** Document that hooks MUST respect ctx.Done() for cooperative cancellation; per-hook timeout handles non-cooperative hooks
**Warning signs:** Blame log shows hook exceeded timeout but hook is still running

### Pitfall 6: Race Between Force Exit Timer and Hook Completion
**What goes wrong:** Force exit happens just as hook completes, unclear state
**Why it happens:** Timer and hook completion race
**How to avoid:** Close a "done" channel when shutdown completes, select on both timer and done
**Warning signs:** Intermittent force exits when hooks actually completed in time

### Pitfall 7: SIGTERM vs SIGINT Behavior Differences
**What goes wrong:** Treating SIGTERM same as SIGINT for double-signal behavior
**Why it happens:** Both trigger shutdown, but double-SIGTERM isn't a thing
**How to avoid:** Per CONTEXT.md decision: double-SIGINT forces exit, SIGTERM only gets one chance (SIGKILL is the force option)
**Warning signs:** User confusion when Ctrl+C twice works but kill + kill doesn't

### Pitfall 8: Testing Signal Handling
**What goes wrong:** Tests that send signals affect other tests or the test runner
**Why it happens:** Signals are process-wide
**How to avoid:** Use `syscall.Kill(syscall.Getpid(), signal)` for controlled testing; reset handlers after tests
**Warning signs:** Tests pass individually but fail when run together

## Code Examples

Verified patterns from official sources and production code:

### Signal Notification with Buffered Channel
```go
// Source: https://pkg.go.dev/os/signal#Notify
c := make(chan os.Signal, 1)
signal.Notify(c, os.Interrupt, syscall.SIGTERM)
defer signal.Stop(c)

s := <-c
fmt.Println("Got signal:", s)
```

### Context with Timeout
```go
// Source: https://go.dev/doc/database/cancel-operations
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel() // Always call cancel to release resources

err := operation(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}
```

### NotifyContext for Signal-Based Cancellation
```go
// Source: https://pkg.go.dev/os/signal#NotifyContext (Go 1.16+)
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

select {
case <-ctx.Done():
    fmt.Println("Interrupted:", ctx.Err())
case <-workDone:
    fmt.Println("Work completed")
}
```

### os.Exit for Force Termination
```go
// Source: https://go.dev/doc/comment (os.Exit documentation)
// os.Exit causes the current program to exit with the given status code.
// Conventionally, code zero indicates success, non-zero an error.
// The program terminates immediately; deferred functions are not run.
os.Exit(1) // Abnormal termination
```

### Atomic Signal Counter for Double-Signal Detection
```go
// Source: Pattern from production Go applications
import "sync/atomic"

var signalCount atomic.Int32

func handleSignal(sig os.Signal) {
    count := signalCount.Add(1)
    if count == 1 {
        // First signal - graceful shutdown
        go gracefulShutdown()
    } else {
        // Second signal - force exit
        os.Exit(1)
    }
}
```

### Fallback stderr Logging (Logger May Be Broken)
```go
// Source: Production pattern for guaranteed output
func logCritical(logger *slog.Logger, msg string) {
    // Try structured logger first
    if logger != nil {
        logger.Error(msg)
    }
    // Always write to stderr as fallback
    fmt.Fprintln(os.Stderr, msg)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual signal loop | signal.NotifyContext | Go 1.16 | Cleaner context integration |
| Custom multi-error | errors.Join | Go 1.20 | Standard error aggregation |
| Multiple error returns | errors.Join | Go 1.20 | Single error with multiple causes |
| log.Fatal for force exit | os.Exit(1) | N/A | log.Fatal runs defers, os.Exit doesn't |

**Deprecated/outdated:**
- `signal.Incoming()` was replaced with `signal.Notify` in Go 1
- Direct `runtime.Goexit()` - use `os.Exit()` for process termination

## Open Questions

Things that couldn't be fully resolved:

1. **Per-hook timeout API design**
   - What we know: HookConfig struct already has comment about Timeout field
   - What's unclear: Whether timeout is set at registration or declared by hook
   - Recommendation: Add `Timeout time.Duration` to HookConfig, defaulting to per-hook timeout (10s per CONTEXT.md)

2. **Log level for blame messages**
   - What we know: Should be visible in production logs
   - What's unclear: ERROR vs WARN - both are reasonable
   - Recommendation: Use ERROR level since timeout is a failure condition

3. **Logging successful hook completions**
   - What we know: Currently logs service stopped with duration
   - What's unclear: Whether to log in blame-tracking mode too
   - Recommendation: Keep existing INFO logs for success, add ERROR for timeout/blame

## Sources

### Primary (HIGH confidence)
- `/websites/go_dev_doc` Context7 - signal handling, context timeout, os.Exit documentation
- `/uber-go/fx` Context7 - lifecycle hooks, timeout enforcement patterns
- https://pkg.go.dev/os/signal - Official signal package documentation
- https://go.dev/doc/database/cancel-operations - Official context timeout patterns

### Secondary (MEDIUM confidence)
- HashiCorp Consul agent.go - Double-signal handling pattern (verified in source)
- Docker Compose cmd/compose/compose.go - Signal handling with context cancellation
- Kubernetes util/runtime/runtime.go - Panic and error handling patterns

### Tertiary (LOW confidence)
- General Go community patterns for graceful shutdown (WebSearch, multiple sources agree)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All stdlib, well-documented
- Architecture patterns: HIGH - Verified in production Go software (Consul, Docker Compose)
- Signal handling: HIGH - Official Go documentation and examples
- Per-hook timeout: MEDIUM - Pattern is sound but specific API is discretionary
- Blame logging: MEDIUM - Format is discretionary per CONTEXT.md

**Research date:** 2026-01-27
**Valid until:** 60 days (stable Go stdlib patterns, unlikely to change)

## Implementation Notes for Planner

Key files to modify:
1. **app.go**: Modify `Run()` method for double-signal handling, modify `stopServices()` for per-hook timeout and blame logging
2. **lifecycle.go**: Add `Timeout` field to `HookConfig` struct
3. Potentially new **shutdown.go**: Shutdown orchestrator if complexity warrants separate file

Existing foundation already in place:
- `signal.Notify` already used in `Run()` for SIGINT/SIGTERM
- `WithShutdownTimeout()` option already exists (global timeout)
- `stopServices()` already iterates hooks in LIFO order
- `HookConfig` struct exists with comment about future Timeout field
- Logger infrastructure already in place

The implementation is mostly about enhancing existing code, not greenfield development.
