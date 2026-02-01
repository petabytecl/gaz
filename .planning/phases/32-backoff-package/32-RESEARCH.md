# Phase 32: Backoff Package - Research

**Researched:** 2026-02-01
**Domain:** Exponential backoff implementation in Go
**Confidence:** HIGH

## Summary

This phase replaces the external `jpillora/backoff` dependency with an internal `backoff/` package adapted from the cenkalti/backoff reference implementation in `_tmp_trust/srex/backoff/`. The research validates that this is a straightforward port with gaz conventions applied.

The reference implementation is complete and well-tested, covering all required functionality: `BackOff` interface, multiple implementations (Exponential, Constant, Zero, Stop), context-aware wrappers, retry helpers with PermanentError support, and Ticker for periodic operations.

**Primary recommendation:** Adapt the reference implementation to use `math/rand/v2` top-level functions (thread-safe) and standard library `errors` instead of `github.com/pkg/errors`.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `math/rand/v2` | stdlib | Thread-safe random jitter | Top-level functions like `Float64()` are safe for concurrent use (no mutex needed) |
| `time` | stdlib | Duration calculations | Standard for Go timing |
| `context` | stdlib | Cancellation propagation | Standard for Go cancellation patterns |
| `errors` | stdlib | Error wrapping/checking | Replaces `github.com/pkg/errors` for Go 1.13+ |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `sync` | stdlib | Once pattern for Ticker.Stop | Thread-safe stop signaling |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Global `rand.Float64()` | Per-instance `rand.Rand` with mutex | Global is simpler and already thread-safe in math/rand/v2 |
| `github.com/pkg/errors` | stdlib `errors` | pkg/errors was needed pre-Go 1.13; now stdlib is sufficient |

**Installation:** No new dependencies - this is a pure Go implementation using stdlib.

## Architecture Patterns

### Recommended Project Structure
```
backoff/
├── backoff.go       # BackOff interface, Stop constant, simple types (ZeroBackOff, StopBackOff, ConstantBackOff)
├── exponential.go   # ExponentialBackOff implementation with options
├── context.go       # WithContext() wrapper for cancellation
├── tries.go         # WithMaxRetries() wrapper
├── retry.go         # Retry(), RetryNotify(), PermanentError
├── ticker.go        # Ticker for periodic backoff-based operations
├── timer.go         # Timer interface for testing abstractions
└── doc.go           # Package documentation
```

### Pattern 1: Interface-Based BackOff
**What:** Core abstraction that all backoff strategies implement
**When to use:** Always - enables composition and testing
**Example:**
```go
// Source: Reference implementation _tmp_trust/srex/backoff/backoff.go
type BackOff interface {
    // NextBackOff returns the duration to wait before retrying.
    // Returns Stop (-1) to signal no more retries.
    NextBackOff() time.Duration
    // Reset to initial state.
    Reset()
}

// Stop sentinel constant
const Stop time.Duration = -1
```

### Pattern 2: Functional Options for ExponentialBackOff
**What:** Configurable creation following gaz conventions
**When to use:** Creating ExponentialBackOff instances
**Example:**
```go
// Gaz convention from worker/options.go
type Option func(*ExponentialBackOff)

func NewExponentialBackOff(opts ...Option) *ExponentialBackOff {
    b := &ExponentialBackOff{
        InitialInterval:     DefaultInitialInterval,
        RandomizationFactor: DefaultRandomizationFactor,
        Multiplier:          DefaultMultiplier,
        MaxInterval:         DefaultMaxInterval,
        MaxElapsedTime:      DefaultMaxElapsedTime,
        Stop:                Stop,
        Clock:               SystemClock,
    }
    for _, opt := range opts {
        opt(b)
    }
    b.Reset()
    return b
}

func WithInitialInterval(d time.Duration) Option {
    return func(b *ExponentialBackOff) {
        if d > 0 {
            b.InitialInterval = d
        }
    }
}
```

### Pattern 3: Decorator/Wrapper Pattern
**What:** Composable wrappers that add behavior
**When to use:** Adding context-awareness, max retries, etc.
**Example:**
```go
// Source: Reference implementation _tmp_trust/srex/backoff/context.go
func WithContext(ctx context.Context, backOff BackOff) Context {
    // Unwrap if already wrapped
    if b, ok := backOff.(*backOffContext); ok {
        return &backOffContext{
            BackOff: b.BackOff,
            ctx:     ctx,
        }
    }
    return &backOffContext{BackOff: backOff, ctx: ctx}
}

// Composes: context + max retries
backoff := WithContext(ctx, WithMaxRetries(NewExponentialBackOff(), 5))
```

### Pattern 4: Clock Abstraction for Testing
**What:** Interface allowing time mocking in tests
**When to use:** Unit testing elapsed time behavior
**Example:**
```go
// Source: Reference implementation _tmp_trust/srex/backoff/exponential.go
type Clock interface {
    Now() time.Time
}

type systemClock struct{}
func (t systemClock) Now() time.Time { return time.Now() }

var SystemClock = systemClock{}

// In tests:
type TestClock struct {
    start time.Time
    i     time.Duration
}
func (c *TestClock) Now() time.Time {
    t := c.start.Add(c.i)
    c.i += time.Second
    return t
}
```

### Anti-Patterns to Avoid
- **Don't use per-instance rand with mutex:** `math/rand/v2.Float64()` is already thread-safe; adding mutex is unnecessary complexity
- **Don't panic on invalid options:** Silently use defaults (like gaz does in worker/options.go)
- **Don't use `github.com/pkg/errors`:** Gaz uses stdlib `errors` package exclusively

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Jitter calculation | Custom random distribution | Reference formula with `rand.Float64()` | Edge cases in range bounds |
| Overflow detection | Manual int64 checks | Reference's `float64(interval) >= float64(max)/multiplier` pattern | Subtle overflow bugs |
| Timer for retry sleeps | Direct `time.Sleep()` | Timer interface with `Start()/Stop()/C()` | Enables testing without real delays |
| Context integration | Ad-hoc context checks | `WithContext()` wrapper pattern | Proper composition, consistent behavior |

**Key insight:** The reference implementation has solved subtle edge cases in jitter distribution, overflow protection, and timer management. Adapt rather than reinvent.

## Common Pitfalls

### Pitfall 1: Using math/rand (v1) Instead of math/rand/v2
**What goes wrong:** Data race warnings or unnecessary mutex overhead
**Why it happens:** Reference implementation uses legacy `math/rand` which requires external seeding and isn't thread-safe
**How to avoid:** Use `math/rand/v2.Float64()` - top-level functions are auto-seeded and thread-safe
**Warning signs:** Import of `math/rand` instead of `math/rand/v2`, manual seed calls

### Pitfall 2: Forgetting Reset() After Construction
**What goes wrong:** First `NextBackOff()` returns unexpected value; startTime not initialized
**Why it happens:** ExponentialBackOff tracks elapsed time from `startTime` which must be set
**How to avoid:** `NewExponentialBackOff()` must call `Reset()` internally
**Warning signs:** `startTime` is zero value, elapsed time calculations wrong

### Pitfall 3: MaxInterval vs Randomized Interval Confusion
**What goes wrong:** Users expect MaxInterval to cap randomized output, but it caps base interval
**Why it happens:** Jitter is applied AFTER capping to MaxInterval
**How to avoid:** Document clearly: "MaxInterval caps RetryInterval, not the randomized result"
**Warning signs:** Occasional delays slightly exceeding MaxInterval

### Pitfall 4: PermanentError Must Use errors.As
**What goes wrong:** Wrapped PermanentErrors not detected
**Why it happens:** Using type assertion instead of `errors.As`
**How to avoid:** Reference implementation correctly uses `errors.As(err, &permanent)`
**Warning signs:** Permanent errors being retried when wrapped

### Pitfall 5: Timer Leak in Retry Loop
**What goes wrong:** Timer resources not freed after successful operation or context cancellation
**Why it happens:** Missing `defer timer.Stop()`
**How to avoid:** Reference implementation uses `defer func() { timer.Stop() }()` in doRetryNotify
**Warning signs:** Resource leaks in long-running applications

## Code Examples

Verified patterns from the reference implementation:

### Jitter Calculation (Thread-Safe)
```go
// Source: _tmp_trust/srex/backoff/exponential.go, adapted for math/rand/v2
import "math/rand/v2"

func getRandomValueFromInterval(randomizationFactor, random float64, currentInterval time.Duration) time.Duration {
    if randomizationFactor == 0 {
        return currentInterval // no randomness when factor is 0
    }
    delta := randomizationFactor * float64(currentInterval)
    minInterval := float64(currentInterval) - delta
    maxInterval := float64(currentInterval) + delta
    return time.Duration(minInterval + (random * (maxInterval - minInterval + 1)))
}

func (b *ExponentialBackOff) NextBackOff() time.Duration {
    // ...
    next := getRandomValueFromInterval(b.RandomizationFactor, rand.Float64(), b.currentInterval)
    // ...
}
```

### Overflow Protection
```go
// Source: _tmp_trust/srex/backoff/exponential.go
func (b *ExponentialBackOff) incrementCurrentInterval() {
    // Check for overflow: if current * multiplier would exceed max
    if float64(b.currentInterval) >= float64(b.MaxInterval)/b.Multiplier {
        b.currentInterval = b.MaxInterval
    } else {
        b.currentInterval = time.Duration(float64(b.currentInterval) * b.Multiplier)
    }
}
```

### PermanentError Pattern (Using stdlib errors)
```go
// Adapted from _tmp_trust/srex/backoff/retry.go for stdlib errors
type PermanentError struct {
    Err error
}

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }
func (e *PermanentError) Is(target error) bool {
    _, ok := target.(*PermanentError)
    return ok
}

func Permanent(err error) error {
    if err == nil {
        return nil
    }
    return &PermanentError{Err: err}
}

// In retry loop - use stdlib errors.As
var permanent *PermanentError
if errors.As(err, &permanent) {
    return res, permanent.Err
}
```

### Worker Integration Pattern
```go
// Replacing current worker/backoff.go usage
// Before: jpillora/backoff with Duration() method
// After: internal backoff with NextBackOff() method

// In worker/supervisor.go
import "github.com/petabytecl/gaz/backoff"

type supervisor struct {
    backoff *backoff.ExponentialBackOff  // Changed type
    // ...
}

func newSupervisor(...) *supervisor {
    return &supervisor{
        backoff: backoff.NewExponentialBackOff(
            backoff.WithInitialInterval(1 * time.Second),
            backoff.WithMaxInterval(5 * time.Minute),
            backoff.WithMultiplier(2),
        ),
        // ...
    }
}

// In supervision loop
delay := s.backoff.NextBackOff()  // Changed from Duration()
// ...
s.backoff.Reset()
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `math/rand` with manual seeding | `math/rand/v2` with auto-seeding | Go 1.22 | Simpler, safer code |
| `github.com/pkg/errors` | stdlib `errors` | Go 1.13 | No external dependency needed |
| jpillora/backoff `Duration()` | cenkalti/backoff `NextBackOff()` | N/A (API choice) | Interface-based composition |

**Deprecated/outdated:**
- `rand.Seed()`: Not needed with `math/rand/v2` - auto-seeded from crypto/rand
- `github.com/pkg/errors`: Gaz uses stdlib; reference must be adapted

## Integration Details

### Current jpillora/backoff Usage

The current `worker/backoff.go` wraps jpillora/backoff:
- `BackoffConfig` struct with Min, Max, Factor, Jitter
- `NewBackoff()` creates `*backoff.Backoff` from jpillora
- Supervisor calls `Duration()` to get next delay and `Reset()` after stable runs

### Migration Path

1. Create `backoff/` package with adapted reference implementation
2. Update `worker/supervisor.go` to use internal `backoff.ExponentialBackOff`
3. Simplify or remove `worker/backoff.go` (BackoffConfig becomes redundant)
4. Remove `jpillora/backoff` from go.mod
5. Run `go mod tidy`

### API Mapping
| jpillora/backoff | internal backoff |
|------------------|------------------|
| `backoff.Backoff{}` struct literal | `NewExponentialBackOff(opts...)` |
| `b.Min` | `WithInitialInterval(d)` |
| `b.Max` | `WithMaxInterval(d)` |
| `b.Factor` | `WithMultiplier(f)` |
| `b.Jitter` | `WithRandomizationFactor(f)` (0.0 = no jitter, 0.5 = default) |
| `b.Duration()` | `b.NextBackOff()` |
| `b.Reset()` | `b.Reset()` |

## Open Questions

Things that couldn't be fully resolved:

1. **Worker's BackoffConfig Future**
   - What we know: Current BackoffConfig wraps jpillora/backoff
   - What's unclear: Keep as convenience wrapper or remove entirely?
   - Recommendation: Remove - functional options on ExponentialBackOff are sufficient

2. **Default Values Alignment**
   - What we know: Reference uses 100ms initial; worker uses 1s initial
   - What's unclear: Should internal package match reference or worker defaults?
   - Recommendation: Keep worker's 1s default (more appropriate for service restarts)

## Sources

### Primary (HIGH confidence)
- `_tmp_trust/srex/backoff/*.go` - Reference implementation (cenkalti/backoff port)
- `worker/supervisor.go`, `worker/backoff.go` - Current usage in gaz
- https://pkg.go.dev/math/rand/v2 - Thread safety of top-level functions confirmed
- Context7: `/jpillora/backoff` - API documentation

### Secondary (MEDIUM confidence)
- Context7: `/cenkalti/backoff` - Retry pattern documentation
- https://github.com/jpillora/backoff - Current dependency README

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - stdlib only, verified with pkg.go.dev
- Architecture: HIGH - directly based on working reference implementation
- Pitfalls: HIGH - identified from reference code and test patterns

**Research date:** 2026-02-01
**Valid until:** Indefinite (stdlib-only, stable patterns)
