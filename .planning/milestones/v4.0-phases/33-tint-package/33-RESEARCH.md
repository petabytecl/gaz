# Phase 33: Tint Package - Research

**Researched:** 2026-02-01
**Domain:** Custom slog.Handler implementation for colored console logging
**Confidence:** HIGH

## Summary

This phase replaces the `lmittmann/tint` external dependency with an internal `logger/tint/` package that implements Go's `slog.Handler` interface for colored console output. The slog.Handler interface is well-defined in Go's standard library (4 methods: `Enabled`, `Handle`, `WithAttrs`, `WithGroup`), and the complete lmittmann/tint source code (MIT licensed) provides an authoritative reference implementation.

The key technical challenges are: (1) correctly implementing `WithAttrs()` and `WithGroup()` to return NEW handler instances (not self), preserving contextual attributes, and (2) TTY detection for auto-disabling ANSI colors in non-terminal output. For TTY detection, `golang.org/x/term.IsTerminal(fd)` is the standard Go approach and avoids adding another external dependency like `mattn/go-isatty`.

**Primary recommendation:** Implement `logger/tint/Handler` by adapting lmittmann/tint's proven patterns, using `golang.org/x/term.IsTerminal()` for TTY detection, with Options matching current usage (Level, AddSource, TimeFormat, NoColor).

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `log/slog` | stdlib | Structured logging API | Go standard library, defines Handler interface |
| `golang.org/x/term` | v0.39.0 | TTY detection | Official Go sub-repository, cross-platform |
| `sync` | stdlib | Mutex for concurrent writes | Thread-safe output |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `strconv` | stdlib | Integer/float formatting | Efficient value formatting without allocations |
| `time` | stdlib | Time formatting | Timestamp output |
| `runtime` | stdlib | Source location | When AddSource is enabled |
| `path/filepath` | stdlib | Path shortening | Display basename+filename for source |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `golang.org/x/term` | `mattn/go-isatty` | x/term is official, isatty adds external dependency |
| `sync.Mutex` | `sync.Pool` for buffers | Pool reduces allocations but adds complexity; start simple |

**Installation:**
```bash
go get golang.org/x/term
```

## Architecture Patterns

### Recommended Project Structure
```
logger/tint/
├── handler.go          # Handler struct and core methods
├── handler_test.go     # Comprehensive tests including slogtest
├── options.go          # Options struct and defaults
├── buffer.go           # Buffer pool for efficient allocation
└── doc.go              # Package documentation
```

### Pattern 1: Handler Clone for WithAttrs/WithGroup
**What:** Return a shallow copy of handler with appended state
**When to use:** Always for WithAttrs and WithGroup methods
**Example:**
```go
// Source: lmittmann/tint handler.go + slog-handler-guide
func (h *Handler) clone() *Handler {
    return &Handler{
        attrsPrefix: h.attrsPrefix,
        groupPrefix: h.groupPrefix,
        groups:      h.groups,
        mu:          h.mu, // mutex shared among all clones
        w:           h.w,
        opts:        h.opts,
    }
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
    if len(attrs) == 0 {
        return h
    }
    h2 := h.clone()
    
    buf := newBuffer()
    defer buf.Free()
    
    // Pre-format attributes
    for _, attr := range attrs {
        h.appendAttr(buf, attr, h.groupPrefix, h.groups)
    }
    h2.attrsPrefix = h.attrsPrefix + string(*buf)
    return h2
}

func (h *Handler) WithGroup(name string) slog.Handler {
    if name == "" {
        return h
    }
    h2 := h.clone()
    h2.groupPrefix += name + "."
    h2.groups = append(h2.groups, name)
    return h2
}
```

### Pattern 2: Level-Based Coloring
**What:** Apply different ANSI colors based on log level
**When to use:** In Handle method when outputting level
**Example:**
```go
// Source: lmittmann/tint handler.go
const (
    ansiBrightRed    = "\u001b[91m"  // ERROR
    ansiBrightYellow = "\u001b[93m"  // WARN
    ansiBrightGreen  = "\u001b[92m"  // INFO
    ansiBrightBlue   = "\u001b[94m"  // DEBUG (per requirements)
    ansiReset        = "\u001b[0m"
    ansiFaint        = "\u001b[2m"
)

func (h *Handler) appendTintLevel(buf *buffer, level slog.Level) {
    if !h.opts.NoColor {
        switch {
        case level < slog.LevelInfo:
            buf.WriteString(ansiBrightBlue)  // DEBUG
        case level < slog.LevelWarn:
            buf.WriteString(ansiBrightGreen) // INFO
        case level < slog.LevelError:
            buf.WriteString(ansiBrightYellow) // WARN
        default:
            buf.WriteString(ansiBrightRed)    // ERROR
        }
    }
    // Write level name...
    if !h.opts.NoColor && level >= slog.LevelInfo {
        buf.WriteString(ansiReset)
    }
}
```

### Pattern 3: TTY Detection for Auto-NoColor
**What:** Automatically disable colors for non-terminal output
**When to use:** During handler construction
**Example:**
```go
// Source: golang.org/x/term
import "golang.org/x/term"

func NewHandler(w io.Writer, opts *Options) *Handler {
    h := &Handler{w: w, mu: &sync.Mutex{}}
    if opts != nil {
        h.opts = *opts
    }
    
    // Auto-detect TTY unless NoColor explicitly set
    if !h.opts.NoColor {
        if f, ok := w.(*os.File); ok {
            h.opts.NoColor = !term.IsTerminal(int(f.Fd()))
        } else {
            h.opts.NoColor = true // Not a file, assume no TTY
        }
    }
    return h
}
```

### Pattern 4: Shared Mutex for Handler Clones
**What:** All handler clones share the same mutex for atomic writes
**When to use:** Handler struct holds *sync.Mutex (pointer, not value)
**Example:**
```go
// Source: slog-handler-guide (critical for correctness)
type Handler struct {
    mu *sync.Mutex  // Pointer! Shared across all clones
    w  io.Writer
    // ...
}

// In Handle:
h.mu.Lock()
defer h.mu.Unlock()
_, err := h.w.Write(*buf)
```

### Anti-Patterns to Avoid
- **Returning self from WithAttrs/WithGroup:** Causes attribute loss across logger.With() calls
- **Value mutex instead of pointer:** Causes interleaved output from cloned handlers
- **Not resolving LogValuer:** All attribute values must call `.Resolve()` first
- **Skipping empty group check:** Empty groups should be ignored per slog spec

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TTY detection | `os.Stat` tricks | `golang.org/x/term.IsTerminal()` | Cross-platform, handles edge cases |
| ANSI escape sequences | Custom string constants | lmittmann/tint constants | Tested, complete set |
| Value resolution | Type switch on Value.Kind() | `slog.Value.Resolve()` | Handles LogValuer recursion |
| Buffer pooling | Ad-hoc buffer reuse | `sync.Pool` | Proper lifecycle, prevents leaks |
| Time formatting | `fmt.Sprintf` | `time.AppendFormat` | Zero-allocation |

**Key insight:** The slog.Handler interface looks simple but has subtle requirements around immutability, concurrent safety, and attribute resolution that lmittmann/tint has already solved correctly.

## Common Pitfalls

### Pitfall 1: WithAttrs/WithGroup Return Self
**What goes wrong:** Returning `h` (self) instead of a copy loses contextual attributes
**Why it happens:** Seems like an optimization; Handler appears to "have" the attrs
**How to avoid:** Always create and return a new handler instance:
```go
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
    h2 := h.clone()  // NEW instance
    // ... append attrs to h2
    return h2        // Return NEW, not h
}
```
**Warning signs:** `.With("key", value).Info("msg")` output missing the key

### Pitfall 2: ANSI Codes in Non-TTY Output
**What goes wrong:** Logs piped to files contain `^[[91m` garbage characters
**Why it happens:** Handler outputs colors without checking if output is terminal
**How to avoid:** Auto-detect TTY or provide NoColor option:
```go
if f, ok := w.(*os.File); ok {
    h.opts.NoColor = !term.IsTerminal(int(f.Fd()))
}
```
**Warning signs:** `./app > log.txt` produces unreadable output

### Pitfall 3: Interleaved Output from Concurrent Loggers
**What goes wrong:** Multiple goroutines using same logger produce garbled output
**Why it happens:** Mutex not shared between handler clones, or no mutex at all
**How to avoid:** Store `*sync.Mutex` (pointer), share across all clones
**Warning signs:** Log lines intermixed: "INFhelloO world"

### Pitfall 4: Not Resolving Attribute Values
**What goes wrong:** LogValuer types don't get expanded, custom types display wrong
**Why it happens:** Handler uses `attr.Value` directly without calling `.Resolve()`
**How to avoid:** Always resolve at the start of appendAttr:
```go
func (h *Handler) appendAttr(buf *buffer, a slog.Attr, ...) []byte {
    a.Value = a.Value.Resolve()  // FIRST
    if a.Equal(slog.Attr{}) {
        return buf  // Ignore empty
    }
    // ... format attr
}
```
**Warning signs:** Custom types that implement LogValuer show pointer addresses

### Pitfall 5: Group/Attribute Ordering Violated
**What goes wrong:** `logger.WithGroup("g").With("k", 1)` shows "k" outside group "g"
**Why it happens:** WithAttrs doesn't respect current group context
**How to avoid:** Track both groupPrefix (for key qualification) and groups slice (for ReplaceAttr calls)
**Warning signs:** Attributes not nested properly in groups

## Code Examples

Verified patterns from official sources:

### Complete Handler Structure
```go
// Source: lmittmann/tint handler.go (MIT licensed)
type Handler struct {
    attrsPrefix string      // pre-formatted attributes
    groupPrefix string      // "group1.group2." prefix for keys
    groups      []string    // group names for ReplaceAttr

    mu   *sync.Mutex        // shared across clones
    w    io.Writer
    opts Options
}

type Options struct {
    Level      slog.Leveler  // Minimum level (default: LevelInfo)
    AddSource  bool          // Include file:line
    TimeFormat string        // Time format (default: time.StampMilli)
    NoColor    bool          // Disable colors
}
```

### Handle Method Structure
```go
// Source: slog-handler-guide + lmittmann/tint
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
    buf := newBuffer()
    defer buf.Free()

    // 1. Time (if not zero)
    if !r.Time.IsZero() {
        h.appendTime(buf, r.Time)
        buf.WriteByte(' ')
    }

    // 2. Level
    h.appendLevel(buf, r.Level)
    buf.WriteByte(' ')

    // 3. Source (if AddSource and PC != 0)
    if h.opts.AddSource && r.PC != 0 {
        h.appendSource(buf, r.PC)
        buf.WriteByte(' ')
    }

    // 4. Message
    buf.WriteString(r.Message)
    buf.WriteByte(' ')

    // 5. Pre-formatted attrs from WithAttrs
    if len(h.attrsPrefix) > 0 {
        buf.WriteString(h.attrsPrefix)
    }

    // 6. Record attrs
    r.Attrs(func(a slog.Attr) bool {
        h.appendAttr(buf, a, h.groupPrefix, h.groups)
        return true
    })

    (*buf)[len(*buf)-1] = '\n'  // Replace last space with newline

    h.mu.Lock()
    defer h.mu.Unlock()
    _, err := h.w.Write(*buf)
    return err
}
```

### Enabled Method
```go
// Source: pkg.go.dev/log/slog#Handler
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
    minLevel := slog.LevelInfo
    if h.opts.Level != nil {
        minLevel = h.opts.Level.Level()
    }
    return level >= minLevel
}
```

### Testing with slogtest
```go
// Source: slog-handler-guide
import "testing/slogtest"

func TestHandler(t *testing.T) {
    var buf bytes.Buffer
    h := tint.NewHandler(&buf, &tint.Options{NoColor: true})
    
    results := func() []map[string]any {
        // Parse buf into []map[string]any
        // Each log entry should be parseable
    }
    
    if err := slogtest.TestHandler(h, results); err != nil {
        t.Error(err)
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `mattn/go-isatty` | `golang.org/x/term` | Go 1.13+ | Official library, no external dep |
| `fmt.Sprintf` in Handle | `strconv.Append*` | Best practice | Zero-allocation formatting |
| Per-handler mutex | Shared mutex pointer | slog design | Correct concurrent behavior |

**Deprecated/outdated:**
- lmittmann/tint doesn't color DEBUG level (requirements want blue)
- Pre-1.21 buffer management; sync.Pool is now standard

## Open Questions

Things that couldn't be fully resolved:

1. **Should TimeFormat default match current usage?**
   - What we know: Current usage is `TimeFormat: "15:04:05.000"`
   - What's unclear: lmittmann/tint default is `time.StampMilli` ("Jan _2 15:04:05.000")
   - Recommendation: Match current usage `"15:04:05.000"` for zero-delta replacement

2. **Should we support ReplaceAttr callback?**
   - What we know: lmittmann/tint has it, but current usage doesn't use it
   - What's unclear: Future use cases
   - Recommendation: Defer; not in requirements, adds complexity

## Sources

### Primary (HIGH confidence)
- lmittmann/tint handler.go (MIT licensed) - Complete implementation reference
- pkg.go.dev/log/slog#Handler - Interface specification
- golang/example slog-handler-guide - Official implementation guide
- golang.org/x/term - TTY detection API

### Secondary (MEDIUM confidence)
- v4.0-DEPENDENCY-REPLACEMENT-PITFALLS.md - Project-specific pitfall documentation
- v4.0-SUMMARY.md - Phase planning context

### Tertiary (LOW confidence)
- None - All findings verified with authoritative sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Well-defined stdlib interface, official x/term
- Architecture: HIGH - lmittmann/tint provides complete reference
- Pitfalls: HIGH - Documented in pitfalls research + slog-handler-guide

**Research date:** 2026-02-01
**Valid until:** 2026-03-01 (stable interface, 30 days)
