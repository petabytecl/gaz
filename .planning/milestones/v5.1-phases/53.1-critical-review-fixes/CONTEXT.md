# Phase 53.1: Critical Review Fixes (INSERTED)

## Origin
Full-repo code review (2026-05-09) ran after v5.1 close. The review was a 6-agent parallel audit against the 8 codified rules in CLAUDE.md plus standard security/correctness/performance lenses. Every one of the 8 rules has at least one active violation; the items below are the CRITICAL subset that block any "hardened framework" claim.

## Items

### 1. Cobra startup path diverges from Run (P0, Rule 2)
- **File**: `cobra.go:179-214` (`Start`) vs `app_run.go:39-121` (`Run`)
- **Issue**: `Start()` reimplements lifecycle inline — no worker filtering/start, no parallel layer startup, no panic recovery, no rollback, ignores `a.opts.ShutdownTimeout`, uses non-context logging. Direct violation of Rule 2 ("Single Startup Path"). Every Cobra-based gaz app silently loses worker supervision and rollback.
- **Fix**: Extract a private `(a *App) startServices(ctx context.Context) error` from `app_run.go`. Move worker filter helper to a shared `collectNonWorkerServices()`. Have both `Run` and `cobra.bootstrap` call `startServices`. Verify with a regression test that asserts both entry points start the worker manager and recover from a panicking `OnStart`.
- **Effort**: Medium

### 2. EventBus.Publish holds RLock across blocking channel sends (P0, Rule 1)
- **File**: `eventbus/bus.go:155-191` (delivery loop), `eventbus/bus.go:210-213, 222-251` (Close), `eventbus/bus.go:258-281` (unsubscribe)
- **Issue**: `Publish` holds `b.mu.RLock()` for the entire delivery loop, including blocking `select { case h.ch <- env: case <-ctx.Done(): }` per subscriber. The comment at L176 codifies the bug as intentional. A slow handler with a full buffer + a concurrent `Close()` waiting for the write lock = guaranteed shutdown deadlock — the exact failure Rule 1 was added to prevent. `unsubscribe` repeats the pattern by holding `mu.Lock()` while waiting on `<-sub.done`, serialising the entire bus on any one slow handler.
- **Fix**: Snapshot the matching `[]*asyncSubscription` slice under RLock, release lock, iterate sends outside. Give each subscription a `tombstone` chan; Close sets the tombstone before closing `ch` so post-snapshot senders can short-circuit via `select { case h.ch<-env: case <-h.tombstone: case <-ctx.Done(): }`. In `unsubscribe`: remove from map and close `sub.ch` under lock, release, then `<-sub.done` outside.
- **Effort**: Medium

### 3. Cron internal scheduler holds runningMu across channel ops (P0, Rule 1)
- **File**: `cron/internal/cron.go:136-148` (Schedule), `:155-161` (Entries), `:181-188` (Remove), `:302-308` (Stop)
- **Issue**: All four public methods hold `runningMu` then send/recv on `c.add` / `c.snapshot` / `c.remove` / `c.stop`. If the scheduler goroutine stalls (long-running `e.Next.After`, GC, test fakes), every concurrent caller deadlocks holding the same mutex.
- **Fix**: Read `c.running` under lock, capture the decision, drop lock, then channel-send. Use `select { case c.add<-entry: case <-c.stop: }` to avoid hang if scheduler exited. Apply the pattern uniformly across all four methods.
- **Effort**: Small per call site, Medium total (need careful re-review of the original robfig vendored code).

### 4. gRPC reflection defaults to true (P0, Rule 7) — site 1
- **File**: `server/grpc/config.go:62, :79` (`DefaultConfig`, flag default), `server/grpc/doc.go:47-64`
- **Issue**: `DefaultConfig()` returns `Reflection: true`, flag default also true, doc comment claims "Defaults to true" — every gaz application built on the gRPC module exposes its full RPC schema unless explicitly opted out. Direct Rule 7 violation.
- **Fix**: Flip default to `false` in `DefaultConfig()`, change flag default to `false`, update doc comment ("Defaults to false. Enable only in dev/staging via WithReflection(true)."). Update `vanguard/module_test.go:43` and `vanguard/config_test.go:28` expectations.
- **Effort**: Small (~10 lines + tests + CHANGELOG entry — this is a behavioral break for downstream consumers).

### 5. Vanguard reflection defaults to true (P0, Rule 7) — site 2
- **File**: `server/vanguard/config.go:52, :106, :128` (`Reflection` field, `DefaultConfig`, flag default), `server/vanguard/doc.go:37-49`
- **Issue**: Same as #4 but on the Vanguard path. Connect reflection v1 + v1alpha exposed by default.
- **Fix**: Same as #4 applied to Vanguard config.
- **Effort**: Small (paired with #4 in the same PR + CHANGELOG note).

### 6. DI singleton GetInstance runs provider while holding s.mu (P0, Rule 1)
- **File**: `di/service.go:151-172` (`lazySingleton.GetInstance`), `di/service.go:297-318` (`eagerSingleton.GetInstance`)
- **Issue**: Provider is invoked while `s.mu` is held. (a) If the provider does network I/O (typical), the lock spans blocking I/O — Rule 1 violation. (b) If the provider's resolution path leads back to the same singleton from a *different* goroutine, the second goroutine blocks on `s.mu`; cycle detection is per-goid so cross-goroutine cycles produce a hang instead of `ErrCycle`. (c) If the provider panics, `built` stays false but `mu` is released, so a retry silently re-runs the provider with side effects.
- **Fix**: Replace `mu` + `built` with `sync.Once` + `atomic.Pointer[T]` for the instance, plus a separate `errOnce` cache for failed providers. Record the initializing goid; a re-entrant call from the same goid fails fast with `ErrCycle`. Run provider entirely outside the lock.
- **Effort**: Medium (touches both singleton variants and their lifecycle Start/Stop methods which also hold `mu` across user hooks — fix together).

### 7. GitHub Actions not SHA-pinned (P0, Rule 8)
- **File**: `.github/workflows/ci.yml`
- **Issue**: All three actions use mutable tags: `actions/checkout@v4`, `actions/setup-go@v5`, `golangci/golangci-lint-action@v9.2.0`. A maintainer (or attacker who compromises the action repo) can force-push the tag and pivot CI.
- **Fix**: Replace each tag with the full commit SHA + version comment, e.g. `uses: actions/checkout@<sha> # v4.x.x`. Add a `github-actions` ecosystem entry to Dependabot so updates arrive as PRs. If Dependabot is not yet present, this phase introduces it.
- **Effort**: Small (~5 lines + dependabot.yml).

## Acceptance criteria

- All 7 items resolved with code changes in the listed files.
- Regression test added for #1 asserting both `Run` and Cobra entry points start the worker manager.
- Regression test added for #2 asserting `Close()` returns within deadline when a subscriber is hung.
- `go test -race ./...` passes.
- CHANGELOG.md notes the breaking default change for #4 and #5.
- README.md note added explaining reflection is now opt-in.

## Out of scope
HIGH and MEDIUM findings live in Phase 53.2.
