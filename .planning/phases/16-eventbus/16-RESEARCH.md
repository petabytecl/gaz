# Phase 16: EventBus - Research

**Researched:** 2026-01-29
**Domain:** Go type-safe generic in-process pub/sub
**Confidence:** HIGH

## Summary

This research investigates patterns for implementing a type-safe, generic EventBus in Go that integrates with gaz's existing DI and lifecycle systems. The EventBus provides in-process pub/sub functionality with `Publish[T]()` and `Subscribe[T]()` generics-based APIs, async fire-and-forget delivery by default, and per-subscription bounded buffers.

The modern Go approach (Go 1.18+) uses generics with `reflect.Type` as the routing key, avoiding string-based topic matching that was error-prone in pre-generics implementations. The project is on Go 1.25.6, which includes `sync.WaitGroup.Go()` for cleaner goroutine spawning. The EventBus should feel consistent with existing Worker and Cron patterns, implementing the lifecycle interface (`Start()`/`Stop()`) for DI auto-discovery.

Key findings: Use atomic counters (not UUIDs) for subscription tokens, per-subscriber buffered channels for isolation, and `defer recover()` in every handler goroutine to prevent panics from crashing the bus.

**Primary recommendation:** Use type-based routing (`reflect.Type` as key), subscription handles for unsubscribe, and per-subscription buffered channels for async delivery with blocking backpressure.

## Standard Stack

This is a pure Go standard library implementation - no external dependencies required.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `reflect` | stdlib | Type-based routing keys | Get `reflect.Type` from generic `[T any]` for map keys |
| `sync` | stdlib | Concurrency primitives | `RWMutex`, `WaitGroup`, `atomic` counters |
| `context` | stdlib | Cancellation/propagation | Handlers receive context for cancellation awareness |
| `runtime/debug` | stdlib | Panic recovery | `debug.Stack()` for stack traces in panic recovery |
| `log/slog` | stdlib | Structured logging | Consistent with project's logging approach |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `sync/atomic` | stdlib | Atomic counter for subscription IDs | Use instead of UUID for token generation |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `reflect.Type` key | String topics | Type-based is compile-time safe, string is runtime discovery only |
| Atomic counter IDs | UUID generation | Atomic counter is allocation-free and faster for local use |
| Per-subscriber channels | Single channel fan-out | Per-subscriber isolates slow consumers but uses more memory |

**Installation:**
```bash
# No installation needed - pure stdlib
```

## Architecture Patterns

### Recommended Project Structure
```
eventbus/
├── bus.go           # EventBus core (Subscribe, Publish, Close)
├── event.go         # Event interface definition
├── subscription.go  # Subscription handle with Unsubscribe()
├── options.go       # Subscription options (buffer size)
├── doc.go           # Package documentation
└── bus_test.go      # Tests
```

### Pattern 1: Type-Based Registry with Generic Methods
**What:** Central bus stores handlers as `any`, but generic `Subscribe[T]()` and `Publish[T]()` methods ensure compile-time type safety.
**When to use:** Always - this is the modern Go pattern for multi-type generic collections.
**Example:**
```go
// Source: Go generics + reflect pattern (verified)
type EventBus struct {
    mu       sync.RWMutex
    handlers map[reflect.Type][]any  // Type-erased handler storage
    closed   bool
}

// Subscribe provides compile-time type safety despite internal any storage
func Subscribe[T Event](b *EventBus, handler Handler[T], opts ...SubscribeOption) *Subscription {
    eventType := reflect.TypeOf((*T)(nil)).Elem()
    // ... register handler with type key
}

// Publish dispatches only to handlers matching type T
func Publish[T Event](ctx context.Context, b *EventBus, event T) {
    eventType := reflect.TypeOf((*T)(nil)).Elem()
    // ... dispatch to registered handlers
}
```

### Pattern 2: Subscription Handle for Unsubscribe
**What:** `Subscribe()` returns a handle object with `Unsubscribe()` method, using internal unique ID.
**When to use:** Always - enables unsubscribing closures (can't compare function pointers).
**Example:**
```go
// Source: Modern Go EventBus pattern (verified via WebSearch 2025)
type Subscription struct {
    id      uint64       // Atomic counter, not UUID
    topic   reflect.Type
    bus     *EventBus
}

func (s *Subscription) Unsubscribe() {
    s.bus.unsubscribe(s.topic, s.id)
}

// Use atomic counter for ID generation (fast, no allocation)
var nextSubscriptionID uint64

func nextID() uint64 {
    return atomic.AddUint64(&nextSubscriptionID, 1)
}
```

### Pattern 3: Per-Subscriber Buffered Channel
**What:** Each async subscription gets its own buffered channel with configurable size.
**When to use:** For async subscriptions - provides isolation between slow/fast consumers.
**Example:**
```go
// Source: Per-subscriber async pattern (verified)
type asyncHandler[T Event] struct {
    ch      chan T
    handler Handler[T]
    wg      *sync.WaitGroup
    done    chan struct{}
}

func newAsyncHandler[T Event](h Handler[T], bufferSize int) *asyncHandler[T] {
    ah := &asyncHandler[T]{
        ch:      make(chan T, bufferSize),
        handler: h,
        done:    make(chan struct{}),
    }
    go ah.run()
    return ah
}

func (ah *asyncHandler[T]) run() {
    defer close(ah.done)
    for event := range ah.ch {
        ah.safeInvoke(event)  // Includes panic recovery
    }
}

func (ah *asyncHandler[T]) deliver(event T) {
    ah.ch <- event  // Blocks if buffer full (per CONTEXT.md decision)
}
```

### Pattern 4: Safe Handler Invocation with Panic Recovery
**What:** Wrap every handler call in `defer recover()` to prevent one panic from crashing the bus.
**When to use:** Always - critical for resilience.
**Example:**
```go
// Source: Go best practices (official docs + WebSearch verified)
func (ah *asyncHandler[T]) safeInvoke(ctx context.Context, event T) {
    defer func() {
        if r := recover(); r != nil {
            slog.Error("handler panic recovered",
                "error", r,
                "stack", string(debug.Stack()),
            )
        }
    }()
    ah.handler(ctx, event)
}
```

### Pattern 5: Drain on Close
**What:** `bus.Close()` closes all handler channels and waits for in-flight handlers to complete.
**When to use:** For graceful shutdown - ensures no events are lost mid-processing.
**Example:**
```go
// Source: Go graceful shutdown pattern (verified)
func (b *EventBus) Close() error {
    b.mu.Lock()
    if b.closed {
        b.mu.Unlock()
        return nil  // Idempotent
    }
    b.closed = true
    
    // Collect all async handlers
    handlers := b.collectAsyncHandlers()
    b.mu.Unlock()
    
    // Close all channels (signals handlers to drain and exit)
    for _, h := range handlers {
        close(h.ch)
    }
    
    // Wait for all handlers to finish processing
    for _, h := range handlers {
        <-h.done
    }
    return nil
}
```

### Pattern 6: Topic Filtering with Optional String
**What:** Subscribe with optional topic string for sub-routing within a type.
**When to use:** When same event type needs routing to different handlers (e.g., `Subscribe[UserEvent]("admin")`)
**Example:**
```go
// Source: Project CONTEXT.md decision
// Type + optional topic for flexible routing
type subscriptionKey struct {
    eventType reflect.Type
    topic     string  // Empty string = wildcard (all topics)
}

func Subscribe[T Event](b *EventBus, handler Handler[T], opts ...SubscribeOption) *Subscription {
    options := applyOptions(opts)
    key := subscriptionKey{
        eventType: reflect.TypeOf((*T)(nil)).Elem(),
        topic:     options.topic,  // Empty = subscribe to all
    }
    // ...
}

// On Publish, match both exact topic and wildcard subscribers
func Publish[T Event](ctx context.Context, b *EventBus, event T, topic string) {
    eventType := reflect.TypeOf((*T)(nil)).Elem()
    
    // Get handlers for exact topic
    exactKey := subscriptionKey{eventType, topic}
    wildcardKey := subscriptionKey{eventType, ""}  // Wildcard
    
    // Deliver to both exact and wildcard subscribers
}
```

### Anti-Patterns to Avoid
- **String-based topics only:** Type-based routing prevents typo bugs and provides IDE autocomplete
- **Single mutex for all operations:** Use RWMutex to allow concurrent reads (Publish) vs writes (Subscribe/Unsubscribe)
- **Shared channel for all handlers:** Slow handler blocks all others - use per-handler channels
- **Ignoring panics:** Uncaught panic in handler kills the whole bus
- **Dropping messages on buffer full:** Per CONTEXT.md, block instead (backpressure)
- **Using UUID for subscription IDs:** Atomic counter is faster and allocation-free for local use

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Subscription ID generation | UUID library | `sync/atomic` counter | Local-only, needs to be fast/allocation-free |
| Type-safe map key | Custom type registry | `reflect.Type` | Already unique per Go type, works with map |
| Graceful shutdown coordination | Custom signaling | `sync.WaitGroup` + channel close | Go's channel close-on-drain is the standard |
| Safe goroutine spawning | Manual Add/Done | `sync.WaitGroup.Go()` (Go 1.25+) | Cleaner API, available in project's Go version |

**Key insight:** This is pure Go stdlib work. The patterns are well-established; complexity is in the integration (DI, lifecycle) and edge cases (panic recovery, drain on close).

## Common Pitfalls

### Pitfall 1: Type Erasure Confusion
**What goes wrong:** Storing `Handler[T]` in `map[reflect.Type][]any` then failing to retrieve correctly
**Why it happens:** Go generics don't allow heterogeneous collections directly
**How to avoid:** Use `reflect.TypeOf((*T)(nil)).Elem()` consistently for both storage and retrieval
**Warning signs:** Type assertion failures at runtime

### Pitfall 2: Blocking on Full Buffer in Async Mode
**What goes wrong:** `Publish()` blocks indefinitely when async subscriber's buffer is full
**Why it happens:** Per CONTEXT.md decision, buffer full = block (not drop)
**How to avoid:** Document clearly, provide per-subscription buffer size configuration
**Warning signs:** Publisher goroutine never returns, deadlock symptoms

### Pitfall 3: Panic in Handler Kills Bus
**What goes wrong:** One panicking handler takes down all event delivery
**Why it happens:** Go panics propagate up unless recovered in same goroutine
**How to avoid:** Every handler goroutine must have `defer recover()` at top
**Warning signs:** Bus stops processing after single handler failure

### Pitfall 4: Race on Close
**What goes wrong:** Publishing after Close() causes panic (send on closed channel)
**Why it happens:** Not checking closed flag before channel operations
**How to avoid:** Check `closed` flag under lock, return early/no-op if closed
**Warning signs:** "panic: send on closed channel" during shutdown

### Pitfall 5: Memory Leak from Orphaned Subscriptions
**What goes wrong:** Subscribers that never unsubscribe keep handlers alive forever
**Why it happens:** No automatic cleanup, relies on manual `Unsubscribe()` calls
**How to avoid:** Document that consumers must unsubscribe; consider context-based auto-unsubscribe option
**Warning signs:** Growing handler count over time, memory growth

### Pitfall 6: Lock Contention on High-Volume Publish
**What goes wrong:** RWMutex read lock in Publish path becomes bottleneck
**Why it happens:** Every Publish acquires read lock to copy handler slice
**How to avoid:** Copy-on-write pattern - Publish works with immutable snapshot of handlers
**Warning signs:** High lock contention in profiling, latency spikes

## Code Examples

### Event Interface (Minimal)
```go
// Source: Project design decision (CONTEXT.md)
// Events must implement this interface (not any arbitrary type)
type Event interface {
    // EventName returns a string identifier for logging/debugging
    EventName() string
}

// Example concrete event
type UserCreated struct {
    ID    string
    Email string
}

func (e UserCreated) EventName() string { return "UserCreated" }
```

### Handler Signature
```go
// Source: Project design decision (CONTEXT.md)
// Handlers receive context for cancellation awareness
// No error return per CONTEXT.md discussion (fire-and-forget)
type Handler[T Event] func(ctx context.Context, event T)
```

### Subscription Options
```go
// Source: Project design decision (CONTEXT.md)
type SubscribeOption func(*subscribeOptions)

type subscribeOptions struct {
    topic      string  // Optional topic filter
    bufferSize int     // For async delivery (default: 100)
}

func WithTopic(topic string) SubscribeOption {
    return func(o *subscribeOptions) {
        o.topic = topic
    }
}

func WithBufferSize(size int) SubscribeOption {
    return func(o *subscribeOptions) {
        o.bufferSize = size
    }
}
```

### EventBus Lifecycle Integration
```go
// Source: Consistent with existing Worker interface pattern
// EventBus implements worker.Worker for DI auto-discovery
func (b *EventBus) Name() string { return "eventbus.EventBus" }

func (b *EventBus) Start() {
    // No-op for EventBus - always ready to accept events
    // Could initialize internal state if needed
}

func (b *EventBus) Stop() {
    b.Close()  // Drain and wait for in-flight handlers
}
```

### Full Subscribe/Publish Example
```go
// Source: Synthesized from patterns above
func Subscribe[T Event](b *EventBus, handler Handler[T], opts ...SubscribeOption) *Subscription {
    options := defaultSubscribeOptions()
    for _, opt := range opts {
        opt(&options)
    }
    
    b.mu.Lock()
    defer b.mu.Unlock()
    
    if b.closed {
        return nil  // Or return subscription that's already cancelled
    }
    
    eventType := reflect.TypeOf((*T)(nil)).Elem()
    key := subscriptionKey{eventType: eventType, topic: options.topic}
    
    id := atomic.AddUint64(&b.nextID, 1)
    
    // Create async handler with per-subscription buffer
    ah := &asyncSubscription[T]{
        id:         id,
        key:        key,
        ch:         make(chan T, options.bufferSize),
        handler:    handler,
        done:       make(chan struct{}),
    }
    
    // Start worker goroutine
    go ah.run(b.ctx, b.logger)
    
    b.handlers[key] = append(b.handlers[key], ah)
    
    return &Subscription{id: id, key: key, bus: b}
}

func Publish[T Event](ctx context.Context, b *EventBus, event T) {
    b.mu.RLock()
    if b.closed {
        b.mu.RUnlock()
        return  // Silent no-op for closed bus (per CONTEXT.md)
    }
    
    eventType := reflect.TypeOf(event)
    
    // Find all matching handlers (exact topic + wildcard)
    var handlers []any
    for key, hs := range b.handlers {
        if key.eventType == eventType {
            handlers = append(handlers, hs...)
        }
    }
    b.mu.RUnlock()
    
    // Deliver to each handler (blocks if buffer full)
    for _, h := range handlers {
        ah := h.(*asyncSubscription[T])
        select {
        case ah.ch <- event:
            // Delivered
        case <-ctx.Done():
            return  // Context cancelled, stop publishing
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `interface{}` handlers | Generic `[T any]` handlers | Go 1.18 (2022) | Compile-time type safety |
| String topic keys | `reflect.Type` + optional topic | Go 1.18+ patterns | Typo prevention, IDE support |
| `wg.Add(1); go func()` | `wg.Go(func())` | Go 1.25 (2025) | Cleaner, less error-prone |
| Manual function comparison for unsubscribe | Subscription handle with ID | Modern pattern | Works with closures |

**Deprecated/outdated:**
- **Pre-generics EventBus libraries:** Most use `interface{}` and runtime assertions - project should build custom

## Open Questions

1. **Handler error return semantics**
   - What we know: CONTEXT.md says fire-and-forget, handlers receive context
   - What's unclear: Should Handler return error for logging purposes even if not acted upon?
   - Recommendation: No error return - handlers log internally. Keeps API simple.

2. **Default buffer size**
   - What we know: Per-subscription configurable
   - What's unclear: What's a sensible default?
   - Recommendation: Default 100, configurable via `WithBufferSize(n)`

3. **Wildcard syntax for topic subscription**
   - What we know: CONTEXT.md says "explicit wildcard to subscribe to all events of a type"
   - What's unclear: Exact API - `Subscribe[T]()` vs `Subscribe[T]("")` vs `Subscribe[T]("*")`
   - Recommendation: `Subscribe[T](handler)` without topic = subscribe to all; `Subscribe[T](handler, WithTopic("x"))` = specific topic

## Sources

### Primary (HIGH confidence)
- Go official documentation (pkg.go.dev) - `sync`, `reflect`, `context` packages
- Project source code - existing Worker, Cron, DI patterns
- Project CONTEXT.md - user decisions on behavior

### Secondary (MEDIUM confidence)
- WebSearch 2025: "Go generics EventBus type-safe pub/sub implementation pattern" - multiple sources agree on reflect.Type routing
- WebSearch 2025: "Go graceful shutdown drain pending goroutines" - confirmed channel close + WaitGroup pattern
- WebSearch 2025: "Go panic recovery in goroutine handler" - confirmed defer/recover per-goroutine requirement
- WebSearch 2025: "Go EventBus unsubscribe pattern subscription handle" - confirmed token/ID approach

### Tertiary (LOW confidence)
- None - all findings verified with official or multiple sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Pure stdlib, well-documented
- Architecture: HIGH - Patterns verified with official Go docs and multiple sources
- Pitfalls: HIGH - Common Go concurrency pitfalls, well-documented

**Research date:** 2026-01-29
**Valid until:** 2026-03-01 (30 days - patterns are stable)
