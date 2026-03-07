---
phase: quick-15
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - server/connect/interceptors_test.go
  - server/vanguard/module_test.go
  - health/config_test.go
autonomous: true
requirements: ["COVER-90"]
must_haves:
  truths:
    - "`make cover` passes with ≥90% total coverage"
  artifacts:
    - path: "server/connect/interceptors_test.go"
      provides: "Tests for WrapUnary, WrapStreamingClient, WrapStreamingHandler on logging, auth, ratelimit interceptors"
    - path: "server/vanguard/module_test.go"
      provides: "Tests for module provider functions (provideConfig, provideCORS, provideLogging, provideRecovery, provideValidation, provideAuth, provideRateLimit)"
    - path: "health/config_test.go"
      provides: "Tests for Config methods (Namespace, Flags, SetDefaults, Validate)"
  key_links: []
---

<objective>
Raise test coverage from 87.7% to ≥90% to pass the `make cover` threshold.

Purpose: CI enforces 90% statement coverage. Current coverage is 87.7%, primarily due to untested interceptor Wrap* methods in `server/connect`, untested module provider functions in `server/vanguard`, and untested Config methods in `health`.
Output: Updated test files achieving ≥90% total coverage.
</objective>

<execution_context>
@/home/coto/.config/opencode/get-shit-done/workflows/execute-plan.md
@/home/coto/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@AGENTS.md
@server/connect/interceptors.go
@server/connect/interceptors_test.go
@server/vanguard/module.go
@server/vanguard/module_test.go
@health/config.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add tests for untested Connect interceptor Wrap methods and Vanguard module providers</name>
  <files>server/connect/interceptors_test.go, server/vanguard/module_test.go, health/config_test.go</files>
  <action>
The biggest coverage gaps are:

**1. `server/connect/interceptors.go` (59.8% → target ~90%)**

The following Wrap* methods have 0% coverage. Add tests to `interceptors_test.go`:

- `loggingInterceptor.WrapUnary` (line 169): Test that it calls the next handler and logs. Create a `connect.NewRequest[any](nil)`, wrap a handler that returns nil,nil, call it, verify no error. Also test with an error-returning handler.
- `loggingInterceptor.WrapStreamingClient` (line 189): Test it's a pass-through (returns `next` unchanged). Call `WrapStreamingClient` and verify returned func equals `next`.
- `loggingInterceptor.WrapStreamingHandler` (line 194): Test it calls next and returns. Use a mock `connect.StreamingHandlerConn` or pass nil with a handler that accepts nil.
- `authInterceptor.WrapUnary` (line 332): Test with an AuthFunc that succeeds (returns enriched context) and one that fails (returns error). Verify the error propagates and success calls next.
- `authInterceptor.WrapStreamingClient` (line 343): Test it's a pass-through.
- `authInterceptor.WrapStreamingHandler` (line 348): Test with succeeding and failing AuthFunc. For the streaming case, create a minimal mock StreamingHandlerConn that implements RequestHeader() and Spec().
- `rateLimitInterceptor.WrapStreamingClient` (line 406): Test it's a pass-through.
- `rateLimitInterceptor.WrapStreamingHandler` (line 411): Test with AlwaysPassLimiter (should pass) and with rejecting limiter (should return error). Need a mock StreamingHandlerConn.
- `recoveryInterceptor.WrapStreamingClient` (line 273): Test it's a pass-through.
- `recoveryInterceptor.WrapStreamingHandler` with dev mode (line 278): Already tested in production mode. Add a test for dev mode panic in streaming handler.

For `WrapStreamingHandler` tests, create a `mockStreamingHandlerConn` struct in the test file that implements `connect.StreamingHandlerConn` interface with minimal stubs (Spec() returns connect.Spec{}, RequestHeader() returns http.Header{}, all other methods are no-ops or return nil). This is the standard approach since Connect doesn't expose a test constructor.

**2. `server/vanguard/module.go` (57.6% → target ~80%+)**

Add tests to `module_test.go` using the DI container directly. The module provider functions are unexported but can be tested by registering prerequisites and calling the function:

- `resolveLogger`: Test with container having *slog.Logger registered → returns it. Test with empty container → returns slog.Default().
- `provideConfig`: Test by calling `provideConfig(DefaultConfig())(container)` → verify Config is registered. Test with invalid config (port=0) → verify validation error when resolving.
- `provideCORSMiddleware`: Register a Config first, then call provideCORSMiddleware(container) → verify *CORSMiddleware resolves.
- `provideConnectLoggingBundle`: Call it → verify *connect.LoggingBundle resolves.
- `provideConnectRecoveryBundle`: Register Config first → call it → verify *connect.RecoveryBundle resolves.
- `provideConnectValidationBundle`: Call it → verify *connect.ValidationBundle resolves.
- `provideConnectAuthBundle`: Test without AuthFunc (should skip silently, no error). Test with AuthFunc registered → verify *connect.AuthBundle resolves.
- `provideConnectRateLimitBundle`: Test without Limiter (should register with AlwaysPassLimiter). Test with Limiter → verify *connect.RateLimitBundle resolves.

Since these are package-internal tests (package vanguard), they have direct access to unexported functions.

Import `gaz` and `gaz/di` packages as needed. Register prerequisite types using `gaz.For[T](c).Instance(val)` or `gaz.For[T](c).Provider(fn)`.

**3. `health/config.go` (0% on Namespace, Flags, SetDefaults, Validate)**

Create `health/config_test.go` with a `ConfigTestSuite`:

- `TestNamespace`: `cfg := DefaultConfig(); s.Equal("health", cfg.Namespace())`
- `TestFlags`: Create a `pflag.FlagSet`, call `cfg.Flags(fs)`, verify flags exist: `fs.Lookup("health-port")` is not nil, etc.
- `TestSetDefaults_ZeroValues`: Create zero Config{}, call SetDefaults(), verify all fields get defaults.
- `TestSetDefaults_PreservesExisting`: Create Config with custom Port=8080, call SetDefaults(), verify Port stays 8080.
- `TestValidate_Valid`: DefaultConfig().Validate() returns nil.
- `TestValidate_InvalidPortZero`: Config{Port: 0}.Validate() returns error.
- `TestValidate_InvalidPortNegative`: Config{Port: -1}.Validate() returns error.
- `TestValidate_InvalidPortTooHigh`: Config{Port: 70000}.Validate() returns error.

Follow project conventions: testify suite, three import groups, godot comments.
  </action>
  <verify>
    <automated>make cover</automated>
  </verify>
  <done>
    - `make cover` passes (exits 0) with total coverage ≥ 90%
    - All new tests pass with `go test -race ./server/connect/... ./server/vanguard/... ./health/...`
    - No lint errors from `make lint`
  </done>
</task>

</tasks>

<verification>
```bash
make cover
```
Must exit 0 with "Coverage: XX.X%" where XX.X ≥ 90.0.
</verification>

<success_criteria>
- `make cover` passes with ≥90% coverage
- `make test` passes (all tests green, no race conditions)
- `make lint` passes (no new lint warnings)
</success_criteria>

<output>
After completion, create `.planning/quick/15-make-sure-the-test-coverage-is-over-the-/15-SUMMARY.md`
</output>
