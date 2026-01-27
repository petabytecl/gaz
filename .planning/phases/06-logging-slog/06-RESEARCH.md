# Phase 06: Logging (slog) - Research

**Researched:** 2026-01-27
**Domain:** Structured Logging & Context Propagation
**Confidence:** HIGH

## Summary

This phase implements structured logging using Go's standard library `log/slog` (introduced in Go 1.21). The implementation focuses on dual-mode output (JSON for production, tinted text for development) and seamless context propagation.

The core architectural decision is to use a **Context Handler** pattern. Instead of passing Logger instances through `context.Context`, we pass *data* (TraceID, RequestID) in the context, and a custom `slog.Handler` extracts these values at the moment of logging. This allows developers to use `slog.InfoContext(ctx, ...)` anywhere without needing to extract a logger first.

**Primary recommendation:** Use `log/slog` with `lmittmann/tint` for development and a custom `ContextHandler` wrapper for context propagation.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `log/slog` | std lib | Structured logging | Go standard since 1.21; high performance |
| `github.com/lmittmann/tint` | v1.x | Colored/Tinted output | De-facto standard for `slog` colorization; drop-in `slog.Handler` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go.opentelemetry.io/otel/semconv` | v1.24+ | Attribute naming | Use constants for standard fields (HTTP, etc.) |

**Installation:**
```bash
go get github.com/lmittmann/tint
```

## Architecture Patterns

### Pattern 1: Context Handler (Recommended)
This pattern wraps an underlying handler (JSON or Text) and injects attributes from `context.Context` into every log record.

**What:** A middleware-style handler that intercepts `Handle()`.
**When to use:** ALWAYS. This fulfills the "Access: Custom slog.Handler automatically reads these fields" requirement.

**Example:**
```go
type ContextHandler struct {
    slog.Handler
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
    if ctx == nil {
        return h.Handler.Handle(ctx, r)
    }

    // Extract standard fields from context
    // Using string literals for keys to match OTel/JSON expectations
    if v, ok := ctx.Value("trace_id").(string); ok {
        r.AddAttrs(slog.String("trace_id", v))
    }
    if v, ok := ctx.Value("span_id").(string); ok {
        r.AddAttrs(slog.String("span_id", v))
    }
    // ... add other fields like request_id, user_id ...

    return h.Handler.Handle(ctx, r)
}
```

### Pattern 2: HTTP Middleware for Context Injection
The middleware's job is to extract headers and populate the `context.Context`.

**Example:**
```go
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Extract or Generate ID
        reqID := r.Header.Get("X-Request-ID")
        if reqID == "" {
            reqID = uuid.NewString()
        }

        // 2. Put into Context
        ctx := context.WithValue(r.Context(), "request_id", reqID)
        
        // 3. Serve
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Configuration Strategy
Use `slog.SetDefault()` to configure the global logger once at startup. This ensures libraries using `slog.Info()` (without a specific logger instance) also respect the format and level.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| **Colorized Logs** | Custom ANSI code wrapper | `lmittmann/tint` | Handles platform differences (Windows), auto-detection, and slog integration correctly. |
| **JSON Formatting** | `json.Marshal` wrapper | `slog.JSONHandler` | High-performance, safe escaping, correct attribute handling. |
| **Level Parsing** | String-to-int logic | `slog.LevelVar` | Thread-safe atomic level changing at runtime. |

**Key insight:** `slog` is designed to be extensible via Handlers. Don't wrap the *Logger* struct; wrap the *Handler* interface.

## Common Pitfalls

### Pitfall 1: Context Key Collisions
**What goes wrong:** Using raw strings like `"trace_id"` as context keys allows collisions with other libraries.
**How to avoid:** Define unexported custom types for context keys (e.g., `type ctxKey struct{}`), but *map* them to string log attributes in the Handler.
**Warning signs:** Random data appearing in your context logs, or context values being overwritten by 3rd party libs.

### Pitfall 2: Expensive Attributes in Handler
**What goes wrong:** Performing complex logic (UUID generation, DB lookups) inside `Handle()`.
**Why it happens:** The `Handle` method is called for *every* log record (even if filtered later in some chains).
**How to avoid:** Only do cheap key lookups in `ContextHandler`.

### Pitfall 3: Ignoring `ctx` in Log Calls
**What goes wrong:** Calling `slog.Info("msg")` instead of `slog.InfoContext(ctx, "msg")`.
**Result:** The `ContextHandler` receives a `context.Background()` (or nil), and trace IDs are lost.
**Prevention:** Linter checks or strict code review to enforce `*Context` methods.

## Code Examples

### Setup (DI Provider)
```go
func NewLogger(cfg *Config) *slog.Logger {
    var handler slog.Handler
    
    opts := &slog.HandlerOptions{
        Level: cfg.LevelVar, // Dynamic level
    }

    if cfg.Environment == "production" {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    } else {
        handler = tint.NewHandler(os.Stdout, &tint.Options{
            Level: opts.Level,
            TimeFormat: time.Kitchen,
        })
    }

    // Wrap with ContextHandler
    handler = &ContextHandler{Handler: handler}

    logger := slog.New(handler)
    
    // Set global default for std lib compatibility
    slog.SetDefault(logger)

    return logger
}
```

### Runtime Level Change
```go
// In Config/Manager
var logLevel = new(slog.LevelVar) // Thread-safe

func UpdateLogLevel(levelStr string) error {
    var l slog.Level
    if err := l.UnmarshalText([]byte(levelStr)); err != nil {
        return err
    }
    logLevel.Set(l) // Takes effect immediately for all loggers using this LevelVar
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `log` (std) | `log/slog` | Go 1.21 (2023) | Structured, leveled, faster, interface-based. |
| `uber/zap` | `log/slog` | 2023+ | `zap` is still faster but `slog` is standard. `zap` now provides an `slog` handler. |
| `context` Loggers | `context` Values | 2023+ | Passing `Logger` in context is now considered an anti-pattern vs passing *attributes* in context. |

## Open Questions

1.  **OTel TraceID Generation**
    - **Status:** We assume TraceIDs are generated by middleware (e.g., from upstream headers or new).
    - **Gap:** Does the framework need to *generate* them if missing?
    - **Recommendation:** Yes, the middleware should ensure a TraceID exists in the context if one wasn't received.

## Sources

### Primary (HIGH confidence)
- `pkg.go.dev/log/slog` - Standard library documentation.
- `github.com/lmittmann/tint` - Library documentation and verified popularity.

### Secondary (MEDIUM confidence)
- `go.opentelemetry.io/otel/semconv` - Key naming conventions (verified, though specific trace_id constants are usually protocol-level).

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH (Standard Lib + Verified dominant community package)
- Architecture: HIGH (Patterns are well documented in Go community)
- Pitfalls: HIGH (Known issues with context usage)

**Research date:** 2026-01-27
