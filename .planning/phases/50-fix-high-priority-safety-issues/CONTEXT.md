# Phase 999.2: Fix High-Priority Safety Issues

## Origin
Full codebase review (2026-03-29)

## Items

### 1. EventBus `Close()` / `Publish()` race — send on closed channel (P1)
- **File**: `eventbus/bus.go:225-235`
- **Issue**: `Close()` releases lock before closing channels. Concurrent `Publish()` panics.
- **Fix**: Keep lock held while closing channels
- **Effort**: Small

### 2. Resolution chain memory leak potential (P1)
- **File**: `di/container.go:118-144`
- **Issue**: `resolutionChains` keyed by goroutine ID. Panic during resolution leaks entry. Recycled goroutine IDs can contaminate future resolutions.
- **Fix**: Add cleanup on panic paths, consider context-based tracking
- **Effort**: Medium

### 3. X-Request-ID header injection (P1)
- **File**: `logger/middleware.go:28-34`
- **Issue**: User-supplied request IDs echoed back and logged without validation. Allows log injection.
- **Fix**: Validate length (max 64 chars) and character set (alphanumeric + dashes)
- **Effort**: Small

### 4. Vanguard health endpoints ignore configured paths (P1)
- **File**: `server/vanguard/health.go:15-19`
- **Issue**: Hardcodes `/healthz`, `/readyz`, `/livez` instead of using `health.Config` paths.
- **Fix**: Read paths from health.Config
- **Effort**: Small

### 5. Logger `ContextHandler` breaks `WithAttrs`/`WithGroup` chain (P2)
- **File**: `logger/handler.go:19-31`
- **Issue**: Doesn't delegate `WithAttrs()` and `WithGroup()`. Attributes via `logger.With()` silently dropped.
- **Fix**: Implement proper delegation methods
- **Effort**: Small

### 6. Logger file handle never closed (P2)
- **File**: `logger/provider.go:67`
- **Issue**: `os.OpenFile` called but handle never stored for cleanup. File descriptor leak.
- **Fix**: Track handle, close on shutdown
- **Effort**: Small

### 7. Vanguard `WriteTimeout=0` allows Slowloris (P2)
- **File**: `server/vanguard/config.go:156-190`
- **Issue**: Zero timeouts disable all protection. Slow clients can tie up resources indefinitely.
- **Fix**: Enforce minimum WriteTimeout or require explicit opt-in for zero
- **Effort**: Small
