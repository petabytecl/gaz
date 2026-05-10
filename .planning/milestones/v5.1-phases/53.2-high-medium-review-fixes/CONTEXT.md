# Phase 53.2: High and Medium Review Fixes (INSERTED)

## Origin
Full-repo code review (2026-05-09). Phase 53.1 lands the 6 critical items; this phase clears the HIGH and MEDIUM findings, adds rule-enforcement automation, and pays down the test-sleep debt.

## A. HIGH-severity findings

### A1. Force-exit race in shutdown
- **File**: `app_shutdown.go:50-58`
- **Issue**: Race between `<-timer.C` (force-exit) and `close(done)` (clean exit). Successful-but-slow shutdowns can still exit(1) when the timer fires a microsecond before `done` closes.
- **Fix**: After `<-timer.C`, do a non-blocking re-check of `done`; only `callExitFunc(1)` if shutdown didn't actually finish. Or migrate to `context.WithTimeout` and let the orderly path return `ctx.Err()`.
- **Effort**: Small

### A2. Cron OnStop holds mu and ignores ctx
- **File**: `cron/scheduler.go:93-110`
- **Issue**: `OnStop` holds `s.mu` (Mutex) for the entire `<-cronCtx.Done()` wait; no `ctx.Done()` companion. Double Rule 1 + Rule 3 violation.
- **Fix**: Set `running=false` under lock, release lock, then `select { case <-cronCtx.Done(): case <-ctx.Done(): logger.Warn("shutdown deadline exceeded"); return ctx.Err() }`.
- **Effort**: Small

### A3. EventBus OnStop discards ctx
- **File**: `eventbus/bus.go:210-213, 222-251`
- **Issue**: `OnStop(_ context.Context)` ignores ctx; `Close()` does unbounded `<-sub.done` per subscription. Rule 3 violation.
- **Fix**: Plumb ctx into Close; per subscription `select { case <-sub.done: case <-ctx.Done(): logger.Warn(...); return ctx.Err() }`.
- **Effort**: Small

### A4. Worker manager Stop has no ctx
- **File**: `worker/manager.go:143-164`
- **Issue**: `Stop()` lacks ctx parameter; `wg.Wait()` is unbounded; relies on global force-exit. Rule 3 violation by absence.
- **Fix**: Change signature to `Stop(ctx context.Context) error`; `select { case <-m.done: case <-ctx.Done(): return ctx.Err() }`. Caller in `app_shutdown.go` passes the shutdown ctx.
- **Effort**: Small (signature change ripples to gaztest mocks).

### A5. Cron Stop leaks waiter goroutine
- **File**: `cron/internal/cron.go:302-315`
- **Issue**: Returns a context cancelled when `jobWaiter.Wait()` completes, in a detached goroutine with no abort path. Any wrapped job that hangs leaks the waiter for process lifetime.
- **Fix**: Accept parent ctx: `select { case <-done: case <-parent.Done(): } cancel()`. Document that callers must respect the deadline.
- **Effort**: Small

### A6. Worker filter logic duplicated; absent from cobra.Start
- **File**: `app_run.go:46-51` ↔ `app_shutdown.go:69-74`; missing in `cobra.go`
- **Issue**: Drift hazard. Already covered by Phase 53.1#1 fix — fold the filter helper into `collectNonWorkerServices()` extracted there.
- **Effort**: Already part of 53.1#1 fix; verify here.

### A7. DI ReplaceService unguarded post-Build
- **File**: `di/container.go:91-95`; caller `gaztest/builder.go:179`
- **Issue**: No `c.built` check, no error return. Silently mutates dependency graph after Build. Rule 4 violation.
- **Fix**: Change signature to `ReplaceService(...) error`; return `ErrAlreadyBuilt` after Build. Update gaztest builder to capture and surface the error. Optional escape hatch behind explicit "test mode" flag if some tests legitimately replace post-Build.
- **Effort**: Small (signature change, ~6 callers).

### A8. DI eager-resolve graph nondeterminism
- **File**: `di/container.go:220-231` (`resolveEager` in `Build`)
- **Issue**: Eager services iterated in map order. Inter-eager dependency edges sometimes recorded, sometimes not. Lifecycle ordering becomes nondeterministic.
- **Fix**: Sort eager services by name before resolve. Skip already-built. Don't `clearChain` between resolves; defer once at the top.
- **Effort**: Small

### A9. DI failed-Build leaves container in inconsistent state
- **File**: `di/container.go:189-217`
- **Issue**: On eager-provider failure, `buildErr` is cached but `built` stays false; `buildOnce` already fired, so re-Build returns cached error, yet `Register` still succeeds. Limbo state.
- **Fix**: On Build failure, set `built=true` (lock all further registration) — pick this semantic and document. Update tests.
- **Effort**: Small

### A10. h2c is the only transport
- **File**: `server/vanguard/server.go:152-164`
- **Issue**: No TLS path, no prod-mode warning. Direct exposure leaks tokens/JWTs. Behind a TLS-terminating proxy this is fine, but framework provides no opinion or guard.
- **Fix**: Add `TLSConfig *tls.Config` field to `Config` (or pull from DI). When `!DevMode && TLSConfig == nil`, log a WARN at startup. Document deployment expectations in `server/vanguard/doc.go`.
- **Effort**: Medium

### A11. AllowZeroWriteTimeout default contradicts its own comment
- **File**: `server/vanguard/config.go:99-111`
- **Issue**: `DefaultConfig()` sets `AllowZeroWriteTimeout: true` while doc-comment claims "WriteTimeout=0 is rejected". The Slowloris guard at `:196` only fires if a caller flips it back to false.
- **Fix**: Default `AllowZeroWriteTimeout: false`, default `WriteTimeout` to a non-zero value (e.g., 30s); streaming workloads opt in. Update doc-comment to match.
- **Effort**: Small (BREAKING — note in CHANGELOG).

### A12. Prod CORS default has empty origins + AllowCredentials true
- **File**: `server/vanguard/config.go:153-160`
- **Issue**: `cors.New` with empty origins blocks all browser traffic, but `AllowCredentials: true` is misleading and dangerous if origins later become `*`.
- **Fix**: Either ship `AllowedOrigins: nil`, `AllowCredentials: false` and require explicit opt-in, or fail validation when `len(AllowedOrigins)==0 && !DevMode`.
- **Effort**: Small

### A13. Force-exit watcher loses second SIGINT
- **File**: `app_run.go:166-181`
- **Issue**: Watcher reads from `sigCh` after `defer signal.Stop(sigCh)` (line 128). When `handleSignalShutdown` returns, `signal.Stop` runs and the watcher's "Ctrl+C again to force" feature is lost.
- **Fix**: Defer `signal.Stop` only after the watcher goroutine has exited (sync via wg or done channel), or hand ownership of `sigCh` to the watcher.
- **Effort**: Small

### A14. Backoff doc claims concurrent-safety but isn't
- **File**: `backoff/exponential.go:48-89, 157-189`
- **Issue**: Public type doc-comment says "must be safe for concurrent use" but `currentInterval`/`startTime` are written without locks. `go test -race` would flag if shared.
- **Fix**: Update doc to "not safe for concurrent use; create one per call site" (matches `backOffTries` already documented this way). Add a vet-style comment example.
- **Effort**: Trivial (docs only).

### A15. Config env-var convention split
- **File**: `config/manager.go:132` vs `:374`
- **Issue**: `AutomaticEnv` uses `"."→"__"` (double underscore); `RegisterProviderFlags` uses `"."→"_"` (single). Same key → two different env vars depending on registration path. Rule 5 doc gap.
- **Fix**: Adopt single underscore (matches CLAUDE.md/README docs). Remove the double-underscore replacer from `AutomaticEnv`. Update tests and any provider configs that relied on `__`.
- **Effort**: Small + sweep for tests.

### A16. Health management server binds 0.0.0.0
- **File**: `health/server.go:40`; handlers at `health/handlers.go:28-32`
- **Issue**: Binds `:port` (all interfaces). Readiness handler is `WithShowDetails(true), WithShowErrors(true)` — leaks per-check error messages (DSN fragments, hostnames, panic strings) to anyone who can reach the port.
- **Fix**: Add `BindAddress` to `Config`, default `"127.0.0.1"`. Gate `WithShowErrors(true)` behind a config flag, default false. Sanitize panic output in `runChecks`.
- **Effort**: Small + tests.

### A17. Test sleep debt (98 hits)
- **File**: `**/*_test.go` (98 sites)
- **Issue**: Rule 6 explicitly requires `require.Eventually`/channel sync over `time.Sleep`. Currently flaky under CI load.
- **Fix**: Mechanical sweep — `time.Sleep(d)` → `require.Eventually(t, cond, 2*d, d/10)` where a condition exists, or channel-based readiness signals. Run `go test -count=10 -race ./...` to confirm stability.
- **Effort**: Medium (mechanical, ~98 sites).

## B. MEDIUM-severity findings (compressed)

### B1. DI singleton lifecycle holds mu across user hooks (Rule 1)
- **File**: `di/service.go:174-194, 320-341`
- **Fix**: Snapshot `instance` under lock, release, then call `runStart/StopLifecycle` outside. Pairs with 53.1#6 fix.

### B2. DI inject reflect not memoized
- **File**: `di/inject.go:46-105`
- **Fix**: Cache `[]injectField` per `reflect.Type` in `sync.Map`. Skip unexported fields silently.

### B3. DI duplicate Named() registrations not detected
- **File**: `di/container.go:235-293`, `di/registration.go`
- **Fix**: In `Register`, fail with `ErrDuplicate` when name is already taken (unless `allowReplace`).

### B4. ErrDuplicate exported but never returned (Rule 5)
- **File**: `di/errors.go:19`
- **Fix**: Either return it from `Register` (B3) or remove the sentinel.

### B5. discoverWorkers/discoverCronJobs eager-instantiate every singleton
- **File**: `app_build.go:108-132, 144-183`
- **Fix**: Use `ServiceType()` reflection check (as `collectProviderConfigs` does) before resolving; instantiate only for confirmed `worker.Worker`/`cron.CronJob` types.

### B6. doStop re-walks container and may instantiate unresolved singletons
- **File**: `app_shutdown.go:63-77`
- **Fix**: Cache the worker-name set computed during Run/Start on the App; reuse in doStop.

### B7. gRPC server constructed in NewServer not OnStart
- **File**: `server/grpc/server.go:96`
- **Fix**: Build `*grpc.Server` inside `OnStart` after registering services.

### B8. gRPC health adapter double-close panic risk
- **File**: `server/grpc/health_adapter.go:73-92`
- **Fix**: Wrap `close(s.stopCh)` in `sync.Once`. Always call `s.health.Shutdown()` in defer, even on timeout.

### B9. Connect rate-limit error code wrong
- **File**: `server/connect/interceptors.go:399, :415`
- **Fix**: Wrap as `connect.NewError(connect.CodeResourceExhausted, err)` instead of `fmt.Errorf("rate limit: %w", err)`.

### B10. Health server missing timeout suite
- **File**: `health/server.go:39-43`
- **Fix**: Add `ReadTimeout=10s`, `WriteTimeout=10s`, `IdleTimeout=60s`.

### B11. Empty health checks → StatusUp
- **File**: `health/internal/checker.go:107-111`
- **Fix**: Return `StatusUnknown` until at least one check is registered, or log a warning at startup.

### B12. Logger SetDefault is hidden global side-effect
- **File**: `logger/provider.go:64-65`
- **Fix**: Move `slog.SetDefault` out of `NewLoggerWithWriter`; expose `SetGlobal()` or call once in module wiring.

### B13. Logger file 0o644, no O_NOFOLLOW, no path validation
- **File**: `logger/provider.go:81`
- **Fix**: Use `0o640`, add `O_NOFOLLOW`, `filepath.Clean`, reject `..` segments and non-absolute paths in production.

### B14. gaztest WithApp + WithModules panics instead of erroring
- **File**: `gaztest/builder.go:145`
- **Fix**: Append to `b.errs`, return joined error from `Build()`.

### B15. gaztest TOCTOU in RequireStart/Stop
- **File**: `gaztest/app.go:35-83`
- **Fix**: Hold lock across Start/Stop, or use `sync.Once`.

### B16. App.WithConfig/WithConfigManager panic on misuse
- **File**: `app.go:252-253, 286-289`
- **Fix**: Append to `a.buildErrors` instead of panicking. Inconsistent with rest of API.

### B17. App critical-fail handler async-Stop is invisible
- **File**: `app_build.go:65-69`
- **Fix**: Log when handler observes `stopOnce` already fired; document async behavior.

### B18. ResolveGroup silently filters mismatched types
- **File**: `di/resolution.go:99-112`
- **Fix**: Return error for first non-assignable instance, matching `ResolveAll[T]`.

### B19. Cron wrapper doesn't fast-path on cancelled appCtx
- **File**: `cron/wrapper.go:79-92`
- **Fix**: Return immediately if `w.appCtx.Err() != nil` after unlock.

### B20. Worker supervisor doesn't OnStop on failed OnStart
- **File**: `worker/supervisor.go:195-211`
- **Fix**: Defensive `worker.OnStop(stopCtx)` after OnStart error.

### B21. Backoff ticker goroutine leak under misuse
- **File**: `backoff/ticker.go:83-98`
- **Fix**: Add `case <-t.ctx.Done(): return nil` to send select.

## C. Rule-enforcement automation

### C1. Custom golangci-lint analysispass: lock-during-blocking-IO
- **Why**: Without enforcement, Rule 1 will keep regressing — three subsystems already proved that.
- **Fix**: Write a small `analysispass` that flags `defer m.Unlock()` (or `defer m.RUnlock()`) followed by any of: channel `<-`/send, `wg.Wait`, `ctx.Done`, network I/O calls. Wire into `.golangci.yml` `custom` plugins. Allowlist exceptions explicitly via `//nolint:lockblockingio` with rationale.

### C2. Pre-commit grep harness for documentary rules
- **Why**: Rules 6, 7, 8 are easy to grep; no point writing analysispasses.
- **Fix**: Shell script at `scripts/check-rules.sh`:
  - `grep -RE 'time\.Sleep\(' --include='*_test.go'` → fail (Rule 6)
  - `grep -RE 'Reflection:\s*true' server/` → fail (Rule 7)
  - `grep -RE 'uses:\s+[^@]+@v[0-9]' .github/workflows/` → fail (Rule 8)
  - Wire into `Makefile` `check` target and CI.

### C3. Doc-reality CI check (Rule 5)
- **Why**: `ErrDuplicate` was exported and documented for months without ever being returned.
- **Fix**: For each exported sentinel error in `errors.go`, assert it appears at least once as a non-definition reference in production code. Simple shell script; runs in CI.

## Acceptance criteria

- All A1–A17 fixes landed with regression tests where applicable.
- All B1–B21 cleanups landed.
- All C1–C3 automation in place and CI-gated.
- `go test -race -count=10 ./...` clean (Rule 6 stability).
- `make check` (new target) passes locally and in CI.
- CHANGELOG.md notes the breaking defaults: A11 (zero-write-timeout), A12 (CORS), A15 (env var convention), A16 (mgmt bind address).

## Sequencing
Land 53.1 first. Then for 53.2: A-series first (correctness/security), B-series second (cleanup), C-series last (so the new linters don't preempt the in-flight fixes).
