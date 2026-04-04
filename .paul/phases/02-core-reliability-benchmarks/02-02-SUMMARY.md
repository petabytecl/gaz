# Summary: Plan 02-02 — Benchmarks

## Result
**Status:** Complete

## What Changed
- di/bench_test.go: Added BenchmarkResolve_NestedDependencies (3-level chain)
- Existing benchmarks already covered Singleton, Transient, Named, Parallel, ResolveAll (DI) and SingleSubscriber, TenSubscribers, Parallel (EventBus)

## Baseline Numbers (pre-optimization)

### DI Resolution
| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Singleton (cache hit) | ~165 | 128 | 4 |
| Transient (new each time) | ~200 | 136 | 5 |
| Named (cache hit) | ~95 | 32 | 2 |
| Parallel (24 goroutines) | ~685 | 128 | 4 |
| Nested 3-level (cache hit) | ~163 | 128 | 4 |
| ResolveAll (10 items) | ~1400 | 1240 | 18 |

### EventBus Publish
| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| Single subscriber | ~77 | 8 | 1 |
| 10 subscribers (fan-out) | ~980 | 80 | 1 |
| Parallel (24 goroutines) | ~155 | 8 | 1 |

### Key observations
- Singleton cache hit: 165ns with 4 allocs — the mutex + chain tracking overhead
- Parallel singleton: 685ns (4.1x single-threaded) — contention on chainMu
- Named resolution: 95ns, only 2 allocs — lighter path (no chain for cached?)
- EventBus publish: 77ns single, 155ns parallel — lock contention visible

## Acceptance Criteria
- [x] AC-1: DI benchmarks cover singleton, transient, parallel, nested (6 total)
- [x] AC-2: EventBus benchmarks cover single, fan-out, parallel (3 total)

---
*Completed: 2026-04-03*
*These baselines are the comparison target for Plans 02-03 and 02-04*
