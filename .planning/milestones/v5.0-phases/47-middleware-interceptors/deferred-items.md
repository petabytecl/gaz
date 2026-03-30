# Deferred Items — Phase 47: Middleware & Interceptors

## Pre-existing Lint Issues in `server/connect/`

These 7 lint warnings exist in `server/connect/interceptors.go` from Plan 01 and are **out of scope** for Plan 02.

| # | Linter | File | Description |
|---|--------|------|-------------|
| 1 | revive | interceptors.go:52 | `ConnectInterceptorBundle` stutters — should be `InterceptorBundle` |
| 2 | revive | interceptors.go:109 | `ConnectAuthFunc` stutters — should be `AuthFunc` |
| 3 | revive | interceptors.go:119 | `ConnectLimiter` stutters — should be `Limiter` |
| 4 | nonamedreturns | interceptors.go:253 | Named return `resp` in recovery closure |
| 5 | perfsprint | interceptors_test.go:360 | `fmt.Errorf` can be `errors.New` |
| 6 | wrapcheck | interceptors.go:399 | Interface method error not wrapped |
| 7 | wrapcheck | interceptors.go:414 | Interface method error not wrapped |

**Recommendation:** Address in a follow-up cleanup pass. Renaming the stuttering types is a breaking change that should be coordinated.
