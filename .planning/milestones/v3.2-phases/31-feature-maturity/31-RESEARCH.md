# Phase 31: Feature Maturity - Research

**Researched:** 2026-02-01
**Domain:** Configuration Validation, Worker Resilience Patterns
**Confidence:** HIGH

## Summary

This phase implements two key features: strict configuration validation and enhanced worker dead letter handling. Both features build on existing infrastructure in the gaz framework.

For **strict config validation**, viper/mapstructure already supports detecting unused keys via `mapstructure.DecoderConfig.ErrorUnused = true`. The implementation requires a new `WithStrictConfig()` option that enables this during unmarshal. The strategy involves passing a decoder option to viper's `Unmarshal()` that errors on unknown keys in the config file.

For **dead letter handling**, the existing worker supervisor already has circuit breaker logic that trips when workers panic repeatedly. The enhancement adds a configurable handler that receives failed worker information when the circuit breaker trips. This follows the common Go pattern of allowing users to provide a callback for dead-letter-like behavior (log, persist, alert).

**Primary recommendation:** Implement both features as optional behaviors with sensible defaults - strict mode is opt-in (users explicitly call `WithStrictConfig()`), and dead letter handling uses a callback pattern for maximum flexibility.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| spf13/viper | v1.21.0 | Config loading | Already used in gaz, supports DecoderConfig options |
| go-viper/mapstructure/v2 | v2.4.0 | Config unmarshaling | ErrorUnused option for strict validation |
| jpillora/backoff | v1.0.0 | Exponential backoff | Already used in worker supervisor |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none needed) | - | - | All features implementable with existing deps |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| mapstructure ErrorUnused | Manual key comparison | ErrorUnused is built-in, less code, well-tested |
| Callback-based DLQ | External queue (Redis/SQS) | Callback is simpler, no external deps, flexible |

**No new dependencies required.** Both features can be implemented using existing mapstructure options and callback patterns.

## Architecture Patterns

### Pattern 1: Strict Config via DecoderConfig Option

**What:** Pass `ErrorUnused = true` to viper's Unmarshal to detect unregistered config keys.

**When to use:** User calls `WithStrictConfig()` before `Build()`.

**Example:**
```go
// Source: Context7 /spf13/viper Tips and tricks, mapstructure docs
import "github.com/go-viper/mapstructure/v2"

func strictDecoderOption(dc *mapstructure.DecoderConfig) {
    dc.ErrorUnused = true
}

// In viper backend
func (b *Backend) UnmarshalStrict(target any) error {
    return b.v.Unmarshal(target, strictDecoderOption)
}
```

### Pattern 2: Dead Letter Callback Handler

**What:** Provide a configurable callback invoked when a worker's circuit breaker trips.

**When to use:** User wants to be notified of permanently failed workers.

**Example:**
```go
// DeadLetterHandler is called when a worker exhausts restart attempts
type DeadLetterHandler func(workerName string, finalError error, panicCount int)

// In WorkerOptions
type WorkerOptions struct {
    // ... existing fields ...
    OnDeadLetter DeadLetterHandler // Called when circuit breaker trips
}

// Option function
func WithDeadLetterHandler(fn DeadLetterHandler) WorkerOption {
    return func(o *WorkerOptions) {
        o.OnDeadLetter = fn
    }
}
```

### Pattern 3: App-Level Option Propagation

**What:** App-level options that affect config/worker behavior.

**When to use:** Features that require coordination across components.

**Example:**
```go
// In AppOptions
type AppOptions struct {
    // ... existing fields ...
    StrictConfig bool
}

// Option function
func WithStrictConfig() Option {
    return func(a *App) {
        // Store flag, use during loadConfig
        a.strictConfig = true
    }
}
```

### Recommended Project Structure
```
gaz/
├── app.go              # Add WithStrictConfig() option
├── config/
│   ├── options.go      # Already exists, may need new option
│   └── viper/
│       └── backend.go  # Add UnmarshalStrict method
└── worker/
    ├── options.go      # Add OnDeadLetter field and WithDeadLetterHandler
    ├── supervisor.go   # Invoke OnDeadLetter when circuit breaks
    └── errors.go       # May add ErrDeadLettered or similar
```

### Anti-Patterns to Avoid
- **Checking keys manually:** Don't iterate viper.AllSettings() and compare with struct fields - use mapstructure's built-in ErrorUnused
- **Silent failures:** Don't swallow dead letter events - ensure they're logged even if no handler configured
- **Breaking defaults:** Don't make strict mode default - it would break existing apps with extra config keys

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detect unknown config keys | Reflect over struct, compare with viper.AllKeys() | mapstructure ErrorUnused=true | Edge cases (nested structs, squash, embedded) already handled |
| Track panic/failure history | Custom tracking in supervisor | Existing circuit breaker state | Already tracks failures and window |
| Exponential backoff | Manual delay calculation | jpillora/backoff | Already integrated, jitter support |

**Key insight:** The mapstructure library has mature support for detecting unused keys. The option passes through viper cleanly. Building custom detection would miss edge cases and duplicate existing functionality.

## Common Pitfalls

### Pitfall 1: Strict Mode Breaks Environment Variable Injection
**What goes wrong:** ErrorUnused might flag env-var-injected keys as unused
**Why it happens:** Viper merges env vars after config file reading, order matters
**How to avoid:** Apply strict check AFTER all sources merged, during final unmarshal
**Warning signs:** Strict mode works with config file but fails when env vars added

### Pitfall 2: Nested Struct Keys Appear Unused
**What goes wrong:** Config like `database.host` flagged as unused when struct has `Database struct`
**Why it happens:** mapstructure needs squash tag or proper nested unmarshal
**How to avoid:** Use existing gaz struct tag patterns consistently
**Warning signs:** Errors on valid nested config that works without strict mode

### Pitfall 3: Dead Letter Handler Called Multiple Times
**What goes wrong:** Handler invoked for each panic, not just final circuit break
**Why it happens:** Confusion between panic callback and dead letter (final failure)
**How to avoid:** Only invoke handler when circuit breaker trips, not on individual panics
**Warning signs:** Dead letter queue fills with recoverable failures

### Pitfall 4: Dead Letter Handler Panics
**What goes wrong:** Handler itself panics, crashes supervisor
**Why it happens:** User provides buggy handler, not wrapped in recover
**How to avoid:** Wrap handler invocation in defer/recover, log but don't propagate
**Warning signs:** Supervisor crashes after worker failure

## Code Examples

### Strict Config Implementation
```go
// Source: mapstructure docs, viper integration
// In config/viper/backend.go

// strictDecoderOption enables error on unused keys
func strictDecoderOption(dc *mapstructure.DecoderConfig) {
    dc.ErrorUnused = true
}

// UnmarshalStrict unmarshals config, failing on unknown keys
func (b *Backend) UnmarshalStrict(target any) error {
    return b.v.Unmarshal(target, strictDecoderOption)
}

// In config/manager.go or app.go
func (m *Manager) LoadIntoStrict(target any) error {
    // ... load from files/env ...
    
    // Use strict unmarshal
    if err := m.backend.UnmarshalStrict(target); err != nil {
        return fmt.Errorf("config: strict validation failed: %w", err)
    }
    // ... defaults, validation ...
    return nil
}
```

### App-Level WithStrictConfig
```go
// Source: Existing gaz option patterns
// In app.go

// WithStrictConfig enables strict configuration validation.
// If enabled, Build() fails if the config file contains any keys
// that are not defined in the config struct.
// This helps catch typos and obsolete configuration.
func WithStrictConfig() Option {
    return func(a *App) {
        a.strictConfig = true
    }
}

// In loadConfig(), check the flag:
func (a *App) loadConfig() error {
    if a.configTarget != nil {
        if a.strictConfig {
            if err := a.configMgr.LoadIntoStrict(a.configTarget); err != nil {
                return fmt.Errorf("loading config (strict mode): %w", err)
            }
        } else {
            if err := a.configMgr.LoadInto(a.configTarget); err != nil {
                return fmt.Errorf("loading config: %w", err)
            }
        }
    }
    // ...
}
```

### Dead Letter Handler in Worker Options
```go
// Source: Existing gaz worker patterns
// In worker/options.go

// DeadLetterInfo contains information about a worker that has permanently failed.
type DeadLetterInfo struct {
    WorkerName   string
    FinalError   error     // Last error/panic before circuit tripped
    PanicCount   int       // Number of panics in window
    CircuitWindow time.Duration
    Timestamp    time.Time
}

// DeadLetterHandler is called when a worker exhausts restart attempts.
// This allows applications to log, alert, persist to external queue, etc.
type DeadLetterHandler func(info DeadLetterInfo)

// WorkerOptions - add field:
type WorkerOptions struct {
    // ... existing fields ...
    
    // OnDeadLetter is called when the circuit breaker trips.
    // Use this to log, alert, or persist failed worker info.
    OnDeadLetter DeadLetterHandler
}

// WithDeadLetterHandler sets a callback for dead letter handling.
func WithDeadLetterHandler(fn DeadLetterHandler) WorkerOption {
    return func(o *WorkerOptions) {
        o.OnDeadLetter = fn
    }
}
```

### Supervisor Dead Letter Invocation
```go
// Source: Existing supervisor.go patterns
// In worker/supervisor.go, inside supervise() when circuit trips:

// Check if circuit breaker should trip
if s.failures >= s.opts.MaxRestarts {
    s.logger.Error("circuit breaker tripped",
        slog.Int("failures", s.failures),
        slog.Duration("window", s.opts.CircuitWindow),
    )

    // Invoke dead letter handler if configured
    if s.opts.OnDeadLetter != nil {
        s.invokeDeadLetterHandler(lastError)
    }

    if s.opts.Critical && s.onCriticalFail != nil {
        s.logger.Error("critical worker failed, triggering shutdown")
        s.onCriticalFail()
    }
    return
}

// Helper method with panic protection:
func (s *supervisor) invokeDeadLetterHandler(lastErr error) {
    defer func() {
        if r := recover(); r != nil {
            s.logger.Error("dead letter handler panicked",
                slog.Any("panic", r),
            )
        }
    }()
    
    info := DeadLetterInfo{
        WorkerName:    s.worker.Name(),
        FinalError:    lastErr,
        PanicCount:    s.failures,
        CircuitWindow: s.opts.CircuitWindow,
        Timestamp:     time.Now(),
    }
    s.opts.OnDeadLetter(info)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| mitchellh/mapstructure | go-viper/mapstructure/v2 | 2024-2025 | Same API, actively maintained fork |
| Manual key validation | ErrorUnused option | Long-standing | Built-in, reliable |

**Deprecated/outdated:**
- mitchellh/mapstructure: Use go-viper/mapstructure/v2 (already in go.mod)

## Open Questions

1. **ProviderConfig interaction with strict mode**
   - What we know: ConfigProvider keys are registered dynamically
   - What's unclear: Should strict mode apply to provider-registered keys too?
   - Recommendation: Initially apply strict mode only to target struct, document behavior

2. **Manager-level dead letter handler**
   - What we know: Options are per-worker currently
   - What's unclear: Should there be a default handler at Manager level?
   - Recommendation: Keep per-worker for flexibility, can add Manager-level later

3. **Dead letter persistence**
   - What we know: Callback pattern provides flexibility
   - What's unclear: Should gaz provide built-in persistence options?
   - Recommendation: Keep simple callback, persistence is user's responsibility

## Sources

### Primary (HIGH confidence)
- Context7 /spf13/viper - strict config via mapstructure ErrorUnused
- pkg.go.dev/github.com/go-viper/mapstructure/v2 - DecoderConfig documentation
- Existing gaz codebase - worker/supervisor.go, config/viper/backend.go patterns

### Secondary (MEDIUM confidence)
- Google Search - circuit breaker + dead letter patterns in Go
- mapstructure v2.5.0 pkg.go.dev - Remainder Values, ErrorUnused sections

### Tertiary (LOW confidence)
- (None - all key findings verified with official sources)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Using existing deps, no new libraries
- Architecture: HIGH - Patterns verified in existing codebase and official docs
- Pitfalls: HIGH - Based on mapstructure behavior and existing supervisor code

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (stable, 30 days - mature libraries, unlikely to change)
