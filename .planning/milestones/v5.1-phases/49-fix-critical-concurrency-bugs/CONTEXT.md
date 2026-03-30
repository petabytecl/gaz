# Phase 999.1: Fix Critical Concurrency Bugs

## Origin
Full codebase review (2026-03-29)

## Items

### 1. Race condition in goroutine closure variable capture (P0)
- **File**: `app.go:874-896`
- **Issue**: `name` and `svc` captured by reference in startup goroutine closure, not by value
- **Fix**: `go func(n string, s di.ServiceWrapper) { ... }(name, svc)`
- **Effort**: Small

### 2. Worker `OnStop` receives already-cancelled context (P0)
- **File**: `worker/supervisor.go:192-196`
- **Issue**: After `<-s.ctx.Done()`, `OnStop(s.ctx)` passes a dead context. Workers can't flush/cleanup.
- **Fix**: Create fresh `context.WithTimeout(context.Background(), stopTimeout)` for OnStop
- **Effort**: Small

### 3. `lazySingleton` Start/Stop race condition (P0)
- **File**: `di/service.go:174-186`
- **Issue**: `Start()` and `Stop()` read `s.built`/`s.instance` without holding `s.mu`. Compare to `eagerSingleton` (lines 312-333) which correctly locks.
- **Fix**: Add `s.mu.Lock()/defer s.mu.Unlock()` to both methods
- **Effort**: Small

### 4. `Container.Build()` race condition (P0)
- **File**: `di/container.go:158-209`
- **Issue**: Lock released after checking `c.built`, re-acquired after eager instantiation. Two goroutines can both pass the check and double-instantiate eager services.
- **Fix**: Hold lock for entire Build, or use `sync.Once`
- **Effort**: Small

### 5. Startup rollback drops all but first error (P0)
- **File**: `app.go:901`
- **Issue**: `<-errCh` reads only one error. If multiple services in same layer fail, other errors lost.
- **Fix**: Drain channel with `for e := range errCh`
- **Effort**: Small
