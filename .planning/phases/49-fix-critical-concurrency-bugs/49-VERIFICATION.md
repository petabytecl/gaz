---
phase: 49-fix-critical-concurrency-bugs
verified: 2026-03-29T21:00:00Z
status: passed
score: 7/7 must-haves verified
gaps: []
human_verification: []
notes:
  - "CONC-01 through CONC-05 are referenced in ROADMAP.md (Phase 49) but not defined in REQUIREMENTS.md — REQUIREMENTS.md covers only v5.0 requirements. These IDs originate from the v5.1 Hardening backlog (CONTEXT.md lists all 5 bugs). Traceability gap exists in REQUIREMENTS.md but all 5 bugs are fixed and verified."
---

# Phase 49: Fix Critical Concurrency Bugs Verification Report

**Phase Goal:** Fix 5 concurrency bugs found in full codebase review: goroutine closure capture race (app.go), worker OnStop cancelled context, lazySingleton Start/Stop race, Container.Build() race, startup error drain
**Verified:** 2026-03-29T21:00:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | lazySingleton.Start() and Stop() hold s.mu before reading s.built or s.instance | VERIFIED | `di/service.go` lines 174-182, 185-192: `s.mu.Lock()` / `defer s.mu.Unlock()` in both methods before `s.built` read |
| 2 | Container.Build() cannot double-instantiate eager services under concurrent calls | VERIFIED | `di/container.go` lines 28-29, 164-182: `buildOnce sync.Once` field + `c.buildOnce.Do(func() {...})` wrapping all eager instantiation |
| 3 | Startup goroutines capture name and svc by value, not by reference | VERIFIED | `app.go` line 877: `go func(n string, s di.ServiceWrapper) {` with `}(name, svc)` at line 896 |
| 4 | Worker OnStop receives a fresh context with timeout, not the cancelled supervisor context | VERIFIED | `worker/supervisor.go` lines 199-203: `context.WithTimeout(context.Background(), defaultStopTimeout)` then `s.worker.OnStop(stopCtx)` |
| 5 | Startup rollback collects all errors from the error channel, not just the first | VERIFIED | `app.go` lines 901-912: `for e := range errCh { startupErrors = append(...) }` then `errors.Join(startupErrors...)` |
| 6 | All concurrent regression tests exist and pass with race detector | VERIFIED | All 4 tests present and pass (see Behavioral Spot-Checks) |
| 7 | All existing tests continue to pass with race detector | VERIFIED | `go test -race -count=3 ./di/ ./worker/ .` all exit 0 |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `di/service.go` | lazySingleton Start/Stop with mutex | VERIFIED | Both methods lock `s.mu` before reading `s.built`/`s.instance` — matches eagerSingleton pattern |
| `di/container.go` | Container.Build with race-safe built flag | VERIFIED | `buildOnce sync.Once` field present; `Build()` body wrapped in `buildOnce.Do(...)` |
| `di/service_test.go` | Concurrent Start/Stop test | VERIFIED | `TestLazySingleton_StartStop_Concurrent` present at line 565 |
| `di/container_test.go` | Concurrent Build test | VERIFIED | `TestContainer_Build_Concurrent` present at line 431, uses `atomic.Int32` counter asserting exactly-once instantiation |
| `app.go` | Race-safe goroutine closure and full error drain | VERIFIED | Value-parameter closure at line 877; `for e := range errCh` drain at lines 902-903; `errors.Join` at line 912 |
| `worker/supervisor.go` | Fresh stop context for OnStop | VERIFIED | `defaultStopTimeout = 30 * time.Second` constant at line 19; fresh context created at lines 199-200; `OnStop(stopCtx)` at line 203 |
| `app_test.go` | Test for multi-error startup rollback | VERIFIED | `TestStartup_MultipleFailures_AllErrorsCollected` present at line 1045 |
| `worker/supervisor_test.go` | Test for OnStop receiving live context | VERIFIED | `TestSupervisor_OnStop_FreshContext` present at line 532 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `di/service.go` | `lazySingleton.mu` | Lock/Unlock in Start and Stop | VERIFIED | Pattern `s\.mu\.Lock\(\)` found at lines 175 and 186 |
| `di/container.go` | `sync.Once` | buildOnce.Do wrapping Build logic | VERIFIED | Pattern `buildOnce` found at lines 28-29 (struct field) and line 165 (Do call) |
| `app.go` | goroutine closure | value parameters in go func | VERIFIED | Pattern `go func\(n string, s di\.ServiceWrapper\)` found at line 877 |
| `worker/supervisor.go` | `context.Background()` | fresh timeout context for OnStop | VERIFIED | Pattern `context\.WithTimeout\(context\.Background\(\)` found at line 199 |
| `app.go` | `errCh` | range loop draining all errors | VERIFIED | Pattern `for e := range errCh` found at line 902 |

### Data-Flow Trace (Level 4)

Not applicable — this phase fixes concurrency logic and error collection, not data rendering pipelines. No components render dynamic data from external sources.

### Behavioral Spot-Checks

All regression tests run with `-race -count=3` to exercise concurrent paths:

| Behavior | Result | Status |
|----------|--------|--------|
| `TestLazySingleton_StartStop_Concurrent` (10+10 goroutines on Start/Stop) | PASS (3/3 runs, no race) | PASS |
| `TestContainer_Build_Concurrent` (10 goroutines, atomic counter = 1) | PASS (3/3 runs, no race) | PASS |
| `TestStartup_MultipleFailures_AllErrorsCollected` | PASS (3/3 runs) | PASS |
| `TestSupervisor_OnStop_FreshContext` | PASS (3/3 runs, no race) | PASS |
| Full `./di/` suite with `-race -count=3` | `ok github.com/petabytecl/gaz/di 1.073s` | PASS |
| Full `./worker/` suite with `-race` | `ok github.com/petabytecl/gaz/worker 1.031s` | PASS |

### Requirements Coverage

The CONC-01 through CONC-05 IDs are listed in ROADMAP.md Phase 49 and in both plan files, but are NOT defined in REQUIREMENTS.md. REQUIREMENTS.md currently covers only v5.0 requirements. The IDs are traceable to CONTEXT.md which defines all 5 bugs explicitly.

| Requirement | Source Plan | Description (from CONTEXT.md) | Status | Evidence |
|-------------|-------------|-------------------------------|--------|---------|
| CONC-01 | 49-02-PLAN.md | Goroutine closure variable capture race in app.go startup | SATISFIED | `go func(n string, s di.ServiceWrapper)` at app.go:877 |
| CONC-02 | 49-02-PLAN.md | Worker OnStop receives already-cancelled context | SATISFIED | `context.WithTimeout(context.Background(), defaultStopTimeout)` at supervisor.go:199 |
| CONC-03 | 49-01-PLAN.md | lazySingleton Start/Stop race condition | SATISFIED | `s.mu.Lock()` in both Start (line 175) and Stop (line 186) of service.go |
| CONC-04 | 49-01-PLAN.md | Container.Build() TOCTOU race condition | SATISFIED | `buildOnce sync.Once` + `buildOnce.Do(...)` in container.go |
| CONC-05 | 49-02-PLAN.md | Startup rollback drops all but first error | SATISFIED | `for e := range errCh` drain + `errors.Join` at app.go:902-912 |

**Orphaned requirements:** CONC-01 through CONC-05 are referenced in ROADMAP.md Phase 49 but not defined in REQUIREMENTS.md. This is a documentation gap — REQUIREMENTS.md has not been updated to include v5.1 Hardening requirements. The bugs themselves are fully fixed and verified. Recommend updating REQUIREMENTS.md to add CONC-01 through CONC-05 definitions.

### Anti-Patterns Found

No problematic anti-patterns found in modified files. Reviewed: `di/service.go`, `di/container.go`, `di/service_test.go`, `di/container_test.go`, `app.go`, `app_test.go`, `worker/supervisor.go`, `worker/supervisor_test.go`.

| File | Pattern | Severity | Assessment |
|------|---------|----------|------------|
| All files | No TODOs, placeholders, or empty implementations found | — | Clean |
| `worker/supervisor.go` | `defaultStopTimeout = 30 * time.Second` constant | INFO | Correct extraction from magic number; plan noted this as a deviation from the original hardcoded value |

### Human Verification Required

None. All fixes are code-level concurrency patches verifiable via static analysis and race-detector tests.

### Gaps Summary

No gaps. All 5 concurrency bugs are fixed:

1. **CONC-03** — `lazySingleton.Start()` and `Stop()` now hold `s.mu.Lock()` before accessing `s.built` or `s.instance`, matching the `eagerSingleton` pattern.
2. **CONC-04** — `Container.Build()` uses `sync.Once` (`buildOnce.Do(...)`) to guarantee exactly-once eager service instantiation under concurrent calls, eliminating the TOCTOU window.
3. **CONC-01** — Startup goroutines pass `name` and `svc` as value parameters `go func(n string, s di.ServiceWrapper)`, eliminating the loop-variable capture race.
4. **CONC-05** — Startup error drain uses `for e := range errCh` collecting all errors, then `errors.Join(startupErrors...)`, so multiple same-layer failures are all reported.
5. **CONC-02** — Worker `OnStop` receives `context.WithTimeout(context.Background(), defaultStopTimeout)` — a fresh live context — not the already-cancelled supervisor context.

All regression tests pass with `-race -count=3`.

**Documentation note:** REQUIREMENTS.md should be updated to define CONC-01 through CONC-05 as v5.1 Hardening requirements (analogous to the v5.0 USRV/CONN/MDDL/SMOD entries), as they are currently referenced in ROADMAP.md but not formally defined.

---

_Verified: 2026-03-29T21:00:00Z_
_Verifier: Claude (gsd-verifier)_
