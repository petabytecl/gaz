# Deferred Items — Phase 48

## Pre-existing Lint Issues (out of scope)

The following lint errors exist in `server/connect/interceptors.go` and `server/connect/interceptors_test.go`. They are pre-existing and unrelated to the gateway removal work:

1. **nonamedreturns** — `interceptors.go:253` named return in recovery interceptor
2. **perfsprint** — `interceptors_test.go:360` `fmt.Errorf` → `errors.New`
3. **revive (stutter)** — `interceptors.go:52,109,119` types `ConnectInterceptorBundle`, `ConnectAuthFunc`, `ConnectLimiter`
4. **wrapcheck** — `interceptors.go:399,414` unwrapped interface method errors

These should be addressed in a separate cleanup PR.
