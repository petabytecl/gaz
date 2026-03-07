---
phase: 14-ci-fixes
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - server/connect/interceptors.go
  - server/connect/interceptors_test.go
  - server/connect/doc.go
  - server/vanguard/module.go
  - server/vanguard/middleware.go
  - server/vanguard/middleware_test.go
autonomous: true
requirements: [CI-01]
must_haves:
  truths:
    - "`make test` passes with zero failures"
    - "`make lint` passes with zero issues"
    - "`gofmt -l .` returns no output (code is formatted)"
  artifacts:
    - path: "server/connect/interceptors.go"
      provides: "Renamed types without stutter + fixed named returns + wrapped interface errors"
    - path: "server/connect/interceptors_test.go"
      provides: "Updated test references + fixed perfsprint issue"
  key_links:
    - from: "server/vanguard/module.go"
      to: "server/connect/interceptors.go"
      via: "connectpkg.AuthFunc, connectpkg.Limiter, connectpkg.InterceptorBundle"
      pattern: "connectpkg\\.(AuthFunc|Limiter|InterceptorBundle)"
---

<objective>
Fix all 7 golangci-lint issues to make `make lint` pass, ensuring full CI compliance.

Purpose: CI is currently failing due to lint errors in server/connect/interceptors.go and interceptors_test.go. Tests pass but lint does not.
Output: All CI checks (`make test`, `make lint`, `gofmt`) pass cleanly.
</objective>

<execution_context>
@/home/coto/.config/opencode/get-shit-done/workflows/execute-plan.md
@/home/coto/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@AGENTS.md
@server/connect/interceptors.go
@server/connect/interceptors_test.go
@server/connect/doc.go
@server/vanguard/module.go
@server/vanguard/middleware.go
@server/vanguard/middleware_test.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Fix all lint issues in server/connect and server/vanguard</name>
  <files>server/connect/interceptors.go, server/connect/interceptors_test.go, server/connect/doc.go, server/vanguard/module.go, server/vanguard/middleware.go, server/vanguard/middleware_test.go</files>
  <action>
Fix all 7 lint issues reported by golangci-lint. These are ALL in server/connect/interceptors.go and interceptors_test.go:

**1. Revive stutter (3 issues) — Rename exported types that stutter when qualified:**
Since the package is `connect`, types prefixed with `Connect` stutter (`connect.ConnectInterceptorBundle`).

- `ConnectInterceptorBundle` → `InterceptorBundle` (interface, line 52)
- `ConnectAuthFunc` → `AuthFunc` (type, line 109)
- `ConnectLimiter` → `Limiter` (interface, line 119)

Use find-and-replace across ALL files that reference these types:
- `server/connect/interceptors.go`: Update type declarations and all internal references (including comments, doc strings, unexported struct fields like `authFunc ConnectAuthFunc` → `authFunc AuthFunc`, `limiter ConnectLimiter` → `limiter Limiter`)
- `server/connect/interceptors_test.go`: Update suite name `ConnectInterceptorBundleTestSuite` → `InterceptorBundleTestSuite`, `TestConnectInterceptorBundleTestSuite` → `TestInterceptorBundleTestSuite`, and all `ConnectInterceptorBundle`, `ConnectAuthFunc`, `ConnectLimiter` references. Also update mock types: `mockConnectLimiter` → `mockLimiter`, `mockConnectInterceptorBundle` → `mockInterceptorBundle`.
- `server/connect/doc.go`: Update all `ConnectInterceptorBundle`, `ConnectAuthFunc`, `ConnectLimiter` references in comments. Note: `[ConnectInterceptorBundle]` godoc links should become `[InterceptorBundle]`.
- `server/vanguard/module.go`: Update all qualified references `connectpkg.ConnectAuthFunc` → `connectpkg.AuthFunc`, `connectpkg.ConnectLimiter` → `connectpkg.Limiter`, and comments referencing the old names.
- `server/vanguard/middleware.go`: Update comment `connect.ConnectInterceptorBundle` → `connect.InterceptorBundle`
- `server/vanguard/middleware_test.go`: Update `connectpkg.ConnectInterceptorBundle` → `connectpkg.InterceptorBundle`

The function `CollectConnectInterceptors` should KEEP its name (it does NOT stutter since it's `connect.CollectConnectInterceptors` — the "Connect" in the function name refers to the protocol, not the package. Actually, it DOES stutter. Rename to `CollectInterceptors`).

Also update `di.ResolveAll[ConnectInterceptorBundle]` → `di.ResolveAll[InterceptorBundle]` inside `CollectInterceptors`.

**2. Named return (1 issue) — line 253:**
Change `func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error)` to remove the `resp` name. Keep `err` named because it's used by the deferred panic recovery. Change to:
`func(ctx context.Context, req connect.AnyRequest) (_ connect.AnyResponse, err error)`

Similarly check line 279 (WrapStreamingHandler in recoveryInterceptor) — it also uses named return `(err error)` which is fine since `err` is used by defer.

**3. Wrapcheck (2 issues) — lines 399 and 414:**
Wrap errors from `r.limiter.Limit()` calls in `rateLimitInterceptor`:
- Line 399 (WrapUnary): `return nil, err` → `return nil, fmt.Errorf("rate limit: %w", err)`
- Line 414 (WrapStreamingHandler): `return err` → `return fmt.Errorf("rate limit: %w", err)`

**4. Perfsprint (1 issue) — interceptors_test.go line 360:**
Change `fmt.Errorf("rate limit exceeded")` → `errors.New("rate limit exceeded")`

Ensure `"errors"` is in the import block of the test file (it may already be there).
  </action>
  <verify>
    <automated>make lint && make test</automated>
  </verify>
  <done>
- `make lint` exits 0 with no issues reported
- `make test` exits 0 with all tests passing
- `gofmt -l .` returns empty output
- All renamed types are consistent across server/connect and server/vanguard packages
  </done>
</task>

</tasks>

<verification>
```bash
# Full CI verification
make lint && make test && gofmt -l .
```
All three commands must succeed with zero errors/output.
</verification>

<success_criteria>
- `make lint` passes with 0 issues (currently 7 issues)
- `make test` passes with 0 failures
- `gofmt -l .` produces no output
- Type renames are consistent: no remaining references to `ConnectInterceptorBundle`, `ConnectAuthFunc`, or `ConnectLimiter` as type names
</success_criteria>

<output>
After completion, create `.planning/quick/14-make-sure-to-pass-all-ci-tests-like-make/14-SUMMARY.md`
</output>
