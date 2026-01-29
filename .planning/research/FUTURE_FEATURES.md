# Future Features Analysis: gaz Framework

**Based on:** go-kit, go-micro, go-zero, Kratos, GoBricks, + internal `_tmp/` reference implementations
**Date:** 2026-01-29
**Current gaz version features:** DI, Lifecycle, Config, Health, Workers, Cron, Cobra CLI, Modules

---

## Executive Summary

After analyzing go-kit, go-micro, go-zero, Kratos, and **the internal `_tmp/` reference implementations**, I've identified feature categories that would enhance gaz from an "application framework" to a "microservice-ready application framework."

The `_tmp/` directory contains **production-ready implementations** of key resilience and infrastructure patterns that can be directly integrated into gaz. These implementations follow Google SRE patterns and are already designed to work together.

---

## Reference Implementation Patterns (from `_tmp/`)

The following patterns have been extracted from the internal reference implementations and should be prioritized for integration into gaz:

### Pattern 1: SRE Circuit Breaker (`srex/circuitbreaker`)

**Key Insight:** Uses **adaptive throttling** based on the Google SRE Book, not traditional state-machine circuit breakers.

```go
// Interface
type CircuitBreaker interface {
    Allow() error      // Check if request is allowed
    MarkSuccess()      // Mark request as successful
    MarkFailed()       // Mark request as failed
}

// Formula: requests are allowed when K * accepts > total
// - Reducing K = more aggressive throttling
// - Increasing K = less aggressive throttling

// Usage
breaker := NewBreaker(
    WithSuccess(0.6),           // K = 1/0.6, 60% success threshold
    WithRequest(100),           // Minimum requests before throttling
    WithWindow(5*time.Second),  // Statistical window duration
    WithBucket(10),             // Number of buckets in window
)

if err := breaker.Allow(); err != nil {
    return ErrNotAllowed
}
// ... execute request ...
if success {
    breaker.MarkSuccess()
} else {
    breaker.MarkFailed()
}
```

**Architecture:**
- Uses `RollingCounter` (sliding window) for statistics
- Probabilistic rejection based on success/failure ratio
- Thread-safe with mutex-protected random number generator
- States: `StateClosed` (allowing) and `StateOpen` (rejecting)

---

### Pattern 2: Exponential Backoff (`srex/backoff`)

**Key Insight:** Clean interface with multiple implementations and retry integration.

```go
// Interface
type BackOff interface {
    NextBackOff() time.Duration  // Returns duration or Stop (-1)
    Reset()                       // Reset to initial state
}

// Implementations
type ZeroBackOff struct{}           // No delay
type StopBackOff struct{}           // Always stop
type ConstantBackOff struct{}       // Fixed delay
type ExponentialBackOff struct{}    // Exponential with jitter

// ExponentialBackOff defaults
DefaultInitialInterval     = 100 * time.Millisecond
DefaultRandomizationFactor = 0.5
DefaultMultiplier          = 1.5
DefaultMaxInterval         = 1 * time.Second
DefaultMaxElapsedTime      = 10 * time.Second

// Example sequence (10 retries):
// Request #  RetryInterval    Randomized Interval
//  1          0.5s            [0.25s,   0.75s]
//  2          0.75s           [0.375s,  1.125s]
//  3          1.125s          [0.562s,  1.687s]
//  ...
// 10         19.210s          backoff.Stop
```

---

### Pattern 3: Retry with Backoff (`srex/backoff/retry.go`)

**Key Insight:** Generic retry with permanent error detection and notifications.

```go
// Core functions
func Retry(o Operation, b BackOff) error
func RetryWithData[T any](o OperationWithData[T], b BackOff) (T, error)
func RetryNotify(operation Operation, b BackOff, notify Notify) error

// Permanent errors (non-retryable)
type PermanentError struct {
    Err error
}
func Permanent(err error) error

// Usage
err := backoff.Retry(func() error {
    resp, err := client.Do(req)
    if err != nil {
        return err // Will be retried
    }
    if resp.StatusCode >= 500 {
        return errors.New("server error") // Will be retried
    }
    if resp.StatusCode == 400 {
        return backoff.Permanent(errors.New("bad request")) // NOT retried
    }
    return nil
}, backoff.NewExponentialBackOff())
```

**Features:**
- Context cancellation support
- Notification callbacks for observability
- Generic version for returning data
- Timer abstraction for testing

---

### Pattern 4: Rate Limiting (`httpx/middleware/ratelimit`)

**Key Insight:** Interface-based design with in-memory and Redis implementations.

```go
// Interface
type RateLimiter interface {
    Allow(ctx context.Context, key string) (bool, time.Duration, error)
}

// Strategies
type RateLimitStrategy string
const (
    RateLimitStrategyIP     = "IP"      // Per IP address
    RateLimitStrategyUser   = "User"    // Per user ID
    RateLimitStrategyGlobal = "Global"  // Global limit
)

// In-memory implementation (Token Bucket)
limiter := NewTokenBucketLimiter(
    100,    // requests per second
    10,     // burst size
    logger,
)

// Redis implementation (Distributed)
limiter, err := NewRedisRateLimiter(
    redisClient,
    100,    // requests per second  
    10,     // burst size
    logger,
)

// Middleware usage
config := &RateLimitConfig{
    RequestsPerSecond: 10,
    BurstSize:         5,
    Strategy:          RateLimitStrategyIP,
}
mw := NewRateLimitMiddleware(limiter, config)
```

**Redis Lua Script:**
- Atomic token bucket implementation
- Returns allowed status + retry-after duration
- Automatic key expiration

---

### Pattern 5: Sliding Window Statistics (`srex/window`)

**Key Insight:** Foundation for circuit breaker and other time-series analytics.

```go
// Core types
type Bucket struct {
    Points []float64
    Count  int64
}

type Window struct {
    buckets []Bucket
    size    int
}

// Time-based policy
type RollingPolicy struct {
    window         *Window
    bucketDuration time.Duration
    // Automatically expires old buckets
}

// Counter with aggregations
type RollingCounter interface {
    Add(int64)
    Value() int64
    Min() float64
    Max() float64
    Avg() float64
    Sum() float64
    Reduce(func(Iterator) float64) float64
}

// Usage
counter := NewRollingCounter(RollingCounterOpts{
    Size:           10,                      // 10 buckets
    BucketDuration: 300 * time.Millisecond,  // 300ms per bucket = 3s window
})
counter.Add(1)  // Record event
avg := counter.Avg()
```

---

### Pattern 6: Timeout Wrapper (`srex/timeout`)

**Key Insight:** Simple but essential pattern for deadline enforcement.

```go
func Run(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    done := make(chan error, 1)
    go func() {
        done <- fn(ctx)
    }()

    select {
    case err := <-done:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}

// Usage
err := timeout.Run(ctx, 5*time.Second, func(ctx context.Context) error {
    return externalService.Call(ctx)
})
```

---

### Pattern 7: Barrier / Singleflight (`cachex/barrier`)

**Key Insight:** Per-key mutual exclusion to prevent cache stampede.

```go
type Barrier[K comparable] struct {
    mu   sync.Mutex
    keys map[K]*barrier
}

// Lock blocks until the key is available
func (b *Barrier[K]) Lock(key K)

// Unlock releases the key
func (b *Barrier[K]) Unlock(key K)

// Usage (cache stampede prevention)
var b Barrier[string]

func GetUser(id string) (*User, error) {
    b.Lock(id)
    defer b.Unlock(id)
    
    // Only one goroutine can fetch at a time per ID
    return db.GetUser(id)
}
```

**Features:**
- Reference counting for cleanup
- Generic key type
- Zero value ready for use

---

### Pattern 8: Copy-on-Write Cache (`cachex/cow`)

**Key Insight:** Optimized for many concurrent reads, rare writes.

```go
type Cow[K comparable, V any] struct {
    mu       sync.Mutex
    ptr      atomic.Pointer[map[K]V]
    capacity int
}

// Get never locks
func (c *Cow[K, V]) Get(key K) (V, bool)

// Set copies the entire map
func (c *Cow[K, V]) Set(key K, value V) bool

// Usage
cache := NewCow[string, *User](1000)  // Capacity 1000
cache.Set("user:123", user)
user, ok := cache.Get("user:123")
```

**Use cases:**
- Configuration caches
- Feature flags
- High-read, low-write scenarios

---

### Pattern 9: Worker Pool with Retry (`queuex/processor`)

**Key Insight:** Generic worker pattern for queue processing.

```go
type Processor[T any] interface {
    Process(ctx context.Context, item T) error
}

type Worker[T any] struct {
    queue      Queue[T]
    processor  Processor[T]
    maxRetries int
    retryDelay time.Duration
}

type WorkerPool[T any] struct {
    workers []*Worker[T]
    queue   Queue[T]
}

// Usage
pool := NewWorkerPool(queue, processor, 5, WorkerConfig{
    MaxRetries: 3,
    RetryDelay: time.Second,
})
pool.Start()
defer pool.Stop()
```

---

### Pattern 10: HTTP Transport with Resilience (`httpx/transport`)

**Key Insight:** Combines circuit breaker + backoff in http.RoundTripper.

```go
type Transport struct {
    tripper http.RoundTripper
    backoff backoff.BackOff
    breaker circuitbreaker.CircuitBreaker
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
    if err := t.breaker.Allow(); err != nil {
        return nil, fmt.Errorf("circuit breaker not allowing request: %w", err)
    }

    oper := func() error {
        if err := t.breaker.Allow(); err != nil {
            return backoff.Permanent(err)  // Don't retry circuit open
        }
        
        res, err := t.tripper.RoundTrip(req)
        if err != nil {
            t.breaker.MarkFailed()
            return err
        }
        if res.StatusCode >= 500 {
            t.breaker.MarkFailed()
            return errors.New("server error")  // Retry 5xx
        }
        
        t.breaker.MarkSuccess()
        return nil
    }

    return backoff.Retry(oper, t.backoff)
}

// Usage with HTTP client
client := NewClient(
    WithRetry(
        backoff.NewExponentialBackOff(),
        circuitbreaker.NewBreaker(),
    ),
)
```

---

### Pattern 11: Middleware Chain (`httpx/middleware`)

**Key Insight:** Composable middleware pattern for HTTP handlers.

```go
type Handler interface {
    ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)
}

type HandlerFunc func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)

// Wrap a single middleware
func Wrap(mw Handler, handler http.Handler) http.Handler

// Chain multiple middlewares
func Chain(middlewares ...Handler) func(http.Handler) http.Handler

// Usage
handler := middleware.Chain(
    loggingMiddleware,
    authMiddleware,
    rateLimitMiddleware,
)(finalHandler)

server.Handle("/api", handler)
```

---

## Feature Categories (Updated with Reference Implementations)

### Category 1: Middleware & Chain Pattern (HIGH PRIORITY)

**Source:** go-kit's endpoint middleware, Kratos middleware chain, **`_tmp/httpx/middleware`**

gaz already has a layered architecture (Container -> App -> Services). Adding a composable middleware system would be a natural extension.

#### Proposed: Adopt `httpx/middleware` Pattern

```go
// gaz/middleware/middleware.go
package middleware

// Handler interface (from httpx/middleware)
type Handler interface {
    ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)
}

type HandlerFunc func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)

func (f HandlerFunc) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
    f(rw, r, next)
}

func Wrap(mw Handler, handler http.Handler) http.Handler
func Chain(middlewares ...Handler) func(http.Handler) http.Handler
```

#### Built-in Middlewares (based on `_tmp/` patterns)

| Middleware | Source | Purpose |
|------------|--------|---------|
| `middleware.Recovery()` | New | Panic recovery with stack trace |
| `middleware.Logging(logger)` | New | Request/response logging |
| `middleware.RateLimit(limiter, config)` | `httpx/middleware/ratelimit` | Rate limiting |
| `middleware.Timeout(duration)` | `srex/timeout` | Per-request timeout |

**Effort:** Low (migrate from `_tmp/httpx/middleware`)
**Breaking changes:** None (additive)

---

### Category 2: Resilience Package (HIGH PRIORITY)

**Source:** go-kit hystrix/gobreaker, go-zero adaptive breaker, **`_tmp/srex`**

#### Proposed: `gaz/resilience` Package

Consolidate patterns from `_tmp/srex` into a cohesive resilience package:

```go
// gaz/resilience/resilience.go
package resilience

// Re-export from sub-packages
import (
    "github.com/petabytecl/gaz/resilience/backoff"
    "github.com/petabytecl/gaz/resilience/breaker"
    "github.com/petabytecl/gaz/resilience/ratelimit"
    "github.com/petabytecl/gaz/resilience/timeout"
    "github.com/petabytecl/gaz/resilience/window"
)
```

##### Sub-package: `resilience/backoff`

```go
// gaz/resilience/backoff/backoff.go
package backoff

type BackOff interface {
    NextBackOff() time.Duration
    Reset()
}

const Stop time.Duration = -1

// Implementations
type ZeroBackOff struct{}
type StopBackOff struct{}
type ConstantBackOff struct{ Delay time.Duration }
type ExponentialBackOff struct{ /* ... */ }

// Retry functions
func Retry(o Operation, b BackOff) error
func RetryWithData[T any](o OperationWithData[T], b BackOff) (T, error)
func RetryNotify(operation Operation, b BackOff, notify Notify) error

// Permanent error wrapper
func Permanent(err error) error
```

##### Sub-package: `resilience/breaker`

```go
// gaz/resilience/breaker/breaker.go
package breaker

type CircuitBreaker interface {
    Allow() error
    MarkSuccess()
    MarkFailed()
}

var ErrNotAllowed = errors.New("circuitbreaker: not allowed for circuit open")

type Option func(*options)

func WithSuccess(s float64) Option      // Success threshold (default 0.6)
func WithRequest(r int64) Option        // Min requests before throttling
func WithWindow(d time.Duration) Option // Statistical window
func WithBucket(b int) Option           // Buckets in window

func NewBreaker(opts ...Option) CircuitBreaker
```

##### Sub-package: `resilience/ratelimit`

```go
// gaz/resilience/ratelimit/limiter.go
package ratelimit

type Limiter interface {
    Allow(ctx context.Context, key string) (bool, time.Duration, error)
}

// In-memory token bucket
func NewTokenBucketLimiter(rate float64, burst int, logger *slog.Logger) *TokenBucketLimiter

// Redis distributed limiter
func NewRedisLimiter(client redis.UniversalClient, rate float64, burst int, logger *slog.Logger) (*RedisLimiter, error)

// Strategies
type Strategy string
const (
    StrategyIP     Strategy = "IP"
    StrategyUser   Strategy = "User"
    StrategyGlobal Strategy = "Global"
)
```

##### Sub-package: `resilience/timeout`

```go
// gaz/resilience/timeout/timeout.go
package timeout

func Run(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error
```

##### Sub-package: `resilience/window`

```go
// gaz/resilience/window/window.go
package window

type Window struct { /* ring buffer */ }
type RollingPolicy struct { /* time-based policy */ }
type RollingCounter interface {
    Add(int64)
    Value() int64
    Min() float64
    Max() float64
    Avg() float64
    Sum() float64
    Reduce(func(Iterator) float64) float64
}

func NewRollingCounter(opts RollingCounterOpts) RollingCounter
```

**Effort:** Medium (refactor from `_tmp/srex`)
**Breaking changes:** None (additive)

---

### Category 3: Cache Package (MEDIUM PRIORITY)

**Source:** **`_tmp/cachex`**

#### Proposed: `gaz/cache` Package

```go
// gaz/cache/barrier.go
package cache

// Barrier provides per-key mutual exclusion (singleflight)
type Barrier[K comparable] struct { /* ... */ }
func (b *Barrier[K]) Lock(key K)
func (b *Barrier[K]) Unlock(key K)

// gaz/cache/cow.go
// Cow is a copy-on-write cache optimized for reads
type Cow[K comparable, V any] struct { /* ... */ }
func NewCow[K comparable, V any](capacity int) *Cow[K, V]
func (c *Cow[K, V]) Get(key K) (V, bool)
func (c *Cow[K, V]) Set(key K, value V) bool
func (c *Cow[K, V]) Add(key K, value V) bool
func (c *Cow[K, V]) Delete(key K) bool
func (c *Cow[K, V]) Keys() []K
```

**Effort:** Low (migrate from `_tmp/cachex`)
**Breaking changes:** None (additive)

---

### Category 4: Queue Package (MEDIUM PRIORITY)

**Source:** **`_tmp/queuex`**

#### Proposed: `gaz/queue` Package

```go
// gaz/queue/queue.go
package queue

type Queue[T any] interface {
    Enqueue(item T) error
    Dequeue() (T, bool)
    DequeueBlocking() (T, bool)
    DequeueWithTimeout(timeout time.Duration) (T, bool)
    DequeueWithContext(ctx context.Context) (T, bool)
    Size() int
    IsEmpty() bool
    Close()
}

// Implementations
func NewChanQueue[T any](size int) Queue[T]
func NewSliceQueue[T any]() Queue[T]
func NewListQueue[T any]() Queue[T]

// gaz/queue/worker.go
type Processor[T any] interface {
    Process(ctx context.Context, item T) error
}

type Worker[T any] struct { /* ... */ }
func NewWorker[T any](queue Queue[T], processor Processor[T], config WorkerConfig) *Worker[T]

type WorkerPool[T any] struct { /* ... */ }
func NewWorkerPool[T any](queue Queue[T], processor Processor[T], numWorkers int, config WorkerConfig) *WorkerPool[T]
```

**Effort:** Low (migrate from `_tmp/queuex`)
**Breaking changes:** None (additive)

---

### Category 5: Transport Abstraction (HIGH PRIORITY)

**Source:** go-kit transport layer, go-micro pluggable transport, Kratos transport, **`_tmp/httpx`**

#### Proposed: HTTP Transport (adopt from `_tmp/httpx`)

```go
// gaz/transport/http/server.go
package http

type Server struct {
    server          *http.Server
    mux             *http.ServeMux
    middleware      []middleware.Handler
    shutdownTimeout time.Duration
}

func NewServer(config *ServerConfig) (*Server, error)

// Lifecycle integration
func (s *Server) Start(ctx context.Context) error
func (s *Server) Stop(ctx context.Context) error

// Handler registration with automatic middleware
func (s *Server) Handle(pattern string, handler http.Handler)
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
func (s *Server) Use(mw ...middleware.Handler)

// gaz/transport/http/client.go
func NewClient(opts ...ClientOption) *http.Client
func NewTransport(opts ...ClientOption) http.RoundTripper

// Client options
func WithTimeout(timeout time.Duration) ClientOption
func WithTLSConfig(cfg *tls.Config) ClientOption
func WithRetry(backoff BackOff, breaker CircuitBreaker) ClientOption
```

**Effort:** Medium (refactor from `_tmp/httpx`)
**Breaking changes:** None (additive)

---

### Category 6: Observability Integration (HIGH PRIORITY)

**Source:** go-zero built-in observability, Kratos OpenTelemetry integration, **`_tmp/otelx`**

#### Proposed: OpenTelemetry Integration

```go
// gaz/otel/provider.go
package otel

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/metric"
)

type Provider struct {
    TracerProvider trace.TracerProvider
    MeterProvider  metric.MeterProvider
}

func New(opts ...Option) (*Provider, error)
func WithOTel(provider *Provider) gaz.Option
```

**Effort:** Medium
**Breaking changes:** None (additive)

---

## Updated Priority Matrix

| Feature | Priority | Effort | Source | Recommended Version |
|---------|----------|--------|--------|---------------------|
| Middleware Chain | HIGH | Low | `_tmp/httpx/middleware` | v0.2 |
| Resilience Package | HIGH | Medium | `_tmp/srex` | v0.2 |
| HTTP Transport | HIGH | Medium | `_tmp/httpx` | v0.3 |
| Observability (OTel) | HIGH | Medium | `_tmp/otelx` | v0.2 |
| Cache Package | MEDIUM | Low | `_tmp/cachex` | v0.3 |
| Queue Package | MEDIUM | Low | `_tmp/queuex` | v0.3 |
| Service Discovery | MEDIUM | High | New | v0.4 |
| gRPC Transport | MEDIUM | High | `_tmp/grpcx` | v0.4 |
| Message Broker | LOW | High | New | v0.5+ |
| Code Generation | LOW | Very High | Evaluate | Later |

---

## Recommended Roadmap (Updated)

### v0.2: Resilience & Middleware Foundation

1. **Middleware Package** (`gaz/middleware`)
   - Migrate from `_tmp/httpx/middleware`
   - Handler interface + Chain/Wrap functions

2. **Resilience Package** (`gaz/resilience`)
   - Circuit breaker from `_tmp/srex/circuitbreaker`
   - Backoff + Retry from `_tmp/srex/backoff`
   - Rate limiting from `_tmp/httpx/middleware/ratelimit`
   - Sliding window from `_tmp/srex/window`
   - Timeout from `_tmp/srex/timeout`

3. **OpenTelemetry Integration** (`gaz/otel`)
   - Adopt patterns from `_tmp/otelx`

### v0.3: Transport & Cache

1. **HTTP Transport** (`gaz/transport/http`)
   - Server with lifecycle integration from `_tmp/httpx`
   - Client with resilience from `_tmp/httpx`
   - Built-in middleware support

2. **Cache Package** (`gaz/cache`)
   - Barrier (singleflight) from `_tmp/cachex`
   - Copy-on-Write cache from `_tmp/cachex`

3. **Queue Package** (`gaz/queue`)
   - Queue interface + implementations from `_tmp/queuex`
   - Worker/WorkerPool with retry from `_tmp/queuex`

### v0.4: Service Communication

1. **gRPC Transport** (`gaz/transport/grpc`)
   - Adopt patterns from `_tmp/grpcx`

2. **Service Discovery** (`gaz/registry`)
   - New implementation (Consul, etcd)

### v0.5+: Advanced Features

1. Message Broker integration
2. API code generation (evaluate need)

---

## Migration Strategy

### Step 1: Direct Migration (v0.2)

Files that can be moved with minimal changes:
- `_tmp/srex/backoff/*.go` -> `gaz/resilience/backoff/`
- `_tmp/srex/circuitbreaker/*.go` -> `gaz/resilience/breaker/`
- `_tmp/srex/window/*.go` -> `gaz/resilience/window/`
- `_tmp/srex/timeout/*.go` -> `gaz/resilience/timeout/`
- `_tmp/httpx/middleware/*.go` -> `gaz/middleware/`
- `_tmp/cachex/*.go` -> `gaz/cache/`
- `_tmp/queuex/*.go` -> `gaz/queue/`

### Step 2: Integration (v0.3)

Packages requiring gaz-specific integration:
- `_tmp/httpx/server.go` - Needs gazx.Lifecycle compatibility
- `_tmp/httpx/client.go` - Works standalone, add DI providers
- `_tmp/httpx/transport.go` - Integrate with resilience package

### Step 3: Import Path Updates

Update internal imports from:
```go
"github.com/petabytecl/gir/pkg/x/srex/backoff"
"github.com/petabytecl/gir/pkg/x/srex/circuitbreaker"
```

To:
```go
"github.com/petabytecl/gaz/resilience/backoff"
"github.com/petabytecl/gaz/resilience/breaker"
```

---

## Comparison: What gaz Already Has vs. Competition vs. `_tmp/`

| Feature | gaz (Current) | go-zero | Kratos | `_tmp/` |
|---------|---------------|---------|--------|---------|
| DI Container | **Yes (core)** | No | Wire | dibx |
| Lifecycle | **Yes (core)** | No | Yes | gazx |
| Config | **Yes** | Yes | Yes | configx |
| Health | **Yes** | Yes | No | gazx |
| Workers | **Yes** | No | No | queuex |
| Cron | **Yes** | Yes | No | cronx |
| Middleware | No | Yes | **Yes** | **httpx/middleware** |
| HTTP Server | No | **Yes** | Yes | **httpx** |
| Circuit Breaker | No | **Yes** | Yes | **srex/circuitbreaker** |
| Rate Limiting | No | **Yes** | Yes | **httpx/middleware/ratelimit** |
| Backoff/Retry | No | Yes | Yes | **srex/backoff** |
| Sliding Window | No | Yes | Yes | **srex/window** |
| Cache | No | Yes | No | **cachex** |
| Queue | No | No | No | **queuex** |
| Tracing | No | **Yes** | **Yes** | otelx |

**Key Insight:** The `_tmp/` directory already contains most of the features needed to make gaz competitive with go-zero and Kratos. The main work is migration, integration, and documentation.

---

## Sources

| Source | Type | Confidence |
|--------|------|------------|
| `_tmp/srex/circuitbreaker` | Internal implementation | HIGH |
| `_tmp/srex/backoff` | Internal implementation | HIGH |
| `_tmp/srex/window` | Internal implementation | HIGH |
| `_tmp/srex/timeout` | Internal implementation | HIGH |
| `_tmp/httpx/middleware` | Internal implementation | HIGH |
| `_tmp/httpx/server.go` | Internal implementation | HIGH |
| `_tmp/httpx/client.go` | Internal implementation | HIGH |
| `_tmp/httpx/transport.go` | Internal implementation | HIGH |
| `_tmp/cachex/barrier.go` | Internal implementation | HIGH |
| `_tmp/cachex/cow.go` | Internal implementation | HIGH |
| `_tmp/queuex/*.go` | Internal implementation | HIGH |
| go-kit/kit GitHub | Context7 (Official) | HIGH |
| micro/go-micro GitHub | Context7 (Official) | HIGH |
| zeromicro/go-zero GitHub | Context7 (Official) | HIGH |
| go-kratos/kratos GitHub | Context7 (Official) | HIGH |
