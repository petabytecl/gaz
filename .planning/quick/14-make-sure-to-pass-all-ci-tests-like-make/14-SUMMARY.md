---
phase: 14-ci-fixes
plan: 01
subsystem: server/connect, server/vanguard
tags: [lint, ci, refactor, naming]
dependency_graph:
  requires: []
  provides: [clean-lint, ci-compliance]
  affects: [server/connect, server/vanguard]
tech_stack:
  added: []
  patterns: [stutter-free-naming, error-wrapping]
key_files:
  created: []
  modified:
    - server/connect/interceptors.go
    - server/connect/interceptors_test.go
    - server/connect/doc.go
    - server/vanguard/module.go
    - server/vanguard/middleware.go
    - server/vanguard/middleware_test.go
    - server/vanguard/server.go
decisions:
  - Renamed ConnectInterceptorBundle → InterceptorBundle to fix revive stutter
  - Renamed ConnectAuthFunc → AuthFunc to fix revive stutter
  - Renamed ConnectLimiter → Limiter to fix revive stutter
  - Renamed CollectConnectInterceptors → CollectInterceptors to fix stutter
metrics:
  duration: 176s
  completed: 2026-03-07
---

# Quick Task 14: Fix CI Lint Issues Summary

Renamed stuttering exported types and fixed all 7 golangci-lint issues across server/connect and server/vanguard packages to achieve full CI compliance.

## Changes Made

### Task 1: Fix all lint issues in server/connect and server/vanguard

**Commit:** `0f4ec16`

**Revive stutter fixes (3 issues):**
- `ConnectInterceptorBundle` → `InterceptorBundle` (interface)
- `ConnectAuthFunc` → `AuthFunc` (type alias)
- `ConnectLimiter` → `Limiter` (interface)
- `CollectConnectInterceptors` → `CollectInterceptors` (function)

**Named return fix (1 issue):**
- `recoveryInterceptor.WrapUnary`: changed `(resp connect.AnyResponse, err error)` → `(_ connect.AnyResponse, err error)`

**Wrapcheck fixes (2 issues):**
- `rateLimitInterceptor.WrapUnary`: wrapped `return nil, err` → `return nil, fmt.Errorf("rate limit: %w", err)`
- `rateLimitInterceptor.WrapStreamingHandler`: wrapped `return err` → `return fmt.Errorf("rate limit: %w", err)`

**Perfsprint fix (1 issue):**
- Test file: `fmt.Errorf("rate limit exceeded")` → `errors.New("rate limit exceeded")`
- Removed unused `fmt` import from test file

**Cross-package reference updates:**
- `server/connect/doc.go`: Updated all godoc references
- `server/vanguard/module.go`: Updated `connectpkg.ConnectAuthFunc` → `connectpkg.AuthFunc`, `connectpkg.ConnectLimiter` → `connectpkg.Limiter`, and comments
- `server/vanguard/middleware.go`: Updated doc comment
- `server/vanguard/middleware_test.go`: Updated interface compliance checks
- `server/vanguard/server.go`: Updated `CollectConnectInterceptors` → `CollectInterceptors` call

## Verification Results

| Check | Result |
|-------|--------|
| `make lint` | 0 issues |
| `make test` | All packages pass |
| `gofmt -l .` | No output (formatted) |

## Deviations from Plan

None — plan executed exactly as written.
