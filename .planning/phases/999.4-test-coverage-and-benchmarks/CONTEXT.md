# Phase 999.4: Test Coverage and Benchmarks

## Origin
Full codebase review (2026-03-29)

## Items

### 1. `server/vanguard` lowest coverage at 74.4%
- Most complex transport code, deserves better coverage
- **Effort**: Medium

### 2. Add benchmarks for hot paths
- Only 1 benchmark file exists (`config/errors_bench_test.go`)
- Needed: `Container.Resolve`, `EventBus.Publish`, backoff calculations
- **Effort**: Medium

### 3. Add cross-package integration tests
- Combine di + worker + eventbus in realistic scenarios
- **Effort**: Medium

### 4. Investigate cron/internal test timing
- 39.9s is suspicious for parsing tests
- **Effort**: Small

### 5. Add `t.Parallel()` markers
- Many independent tests could run in parallel for faster execution
- **Effort**: Small
