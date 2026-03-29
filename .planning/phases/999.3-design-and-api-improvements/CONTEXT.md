# Phase 999.3: Design and API Improvements

## Origin
Full codebase review (2026-03-29)

## Items

### 1. Split `app.go` (1172 lines) into focused files
- **File**: `app.go`
- **Suggestion**: `app_build.go`, `app_run.go`, `app_shutdown.go`
- **Effort**: Medium

### 2. EventBus handlers always get `context.Background()`
- **File**: `eventbus/bus.go:29`
- **Issue**: Loses trace/request context from publisher. Limits observability.
- **Effort**: Medium

### 3. Cron scheduler receives `context.Background()`
- **File**: `app.go:279`
- **Issue**: App lifecycle context doesn't propagate to cron jobs.
- **Effort**: Small

### 4. Shutdown rollback ignores `Stop()` errors
- **File**: `app.go:908`
- **Issue**: `_ = a.Stop(shutdownCtx)` silently discards shutdown errors.
- **Fix**: `errors.Join(startupErr, stopErr)`
- **Effort**: Small

### 5. No pool size upper bound validation
- **File**: `worker/options.go:100-106`
- **Issue**: `WithPoolSize(1000000)` happily accepted.
- **Effort**: Small

### 6. Duplicate comment on `Option` type
- **File**: `app.go:64-66`
- **Effort**: Trivial

### 7. Config `SetEnvKeyReplacer` panics instead of returning error
- **File**: `config/viper/backend.go:201-210`
- **Effort**: Small

### 8. Dead letter handler lacks stack trace
- **File**: `worker/supervisor.go:218-226`
- **Issue**: `DeadLetterInfo` doesn't include panic stack.
- **Effort**: Small

### 9. HTTP server `ListenAndServe` error is async-only
- **File**: `server/http/server.go:67-71`
- **Issue**: Returns success even if port bind fails.
- **Effort**: Medium

### 10. `time.After` leak in supervisor and shutdown
- **Files**: `worker/supervisor.go:158`, `app.go:1020`
- **Fix**: Use `time.NewTimer` with `Stop()`
- **Effort**: Small

### 11. Backoff jitter off-by-one
- **File**: `backoff/exponential.go:202`
- **Issue**: `+1` causes result to slightly exceed MaxInterval
- **Effort**: Trivial
