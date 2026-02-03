---
phase: 37-core-discovery
plan: 01
subsystem: core
tags: [di, discovery, multi-binding]

requires: []
provides:
  - "Multi-binding support in DI Container"
  - "Discovery API (ResolveAll, ResolveGroup)"
  - "Implicit Collection pattern"
affects:
  - 38-gateway-service
  - 40-observability

tech-stack:
  added: []
  patterns:
    - "Implicit Collection"
    - "Service Groups"

key-files:
  created:
    - di/discovery_test.go
  modified:
    - di/container.go
    - di/registration.go
    - di/resolution.go
    - types.go

key-decisions:
  - "Allowed Register to append duplicates (Implicit Collection) instead of returning error"
  - "Resolve returns ErrAmbiguous if multiple services registered for same key"
  - "Added ResolveAll and ResolveGroup for discovering services"

metrics:
  duration: 15m
  completed: 2026-02-02
---

# Phase 37 Plan 01: Core Discovery Summary

**Refactored Core DI to support multi-binding and Discovery API with ResolveAll/ResolveGroup**

## Performance

- **Duration:** 15 min
- **Started:** 2026-02-02T10:00:00Z (approx)
- **Completed:** 2026-02-02T10:15:00Z
- **Tasks:** 4
- **Files modified:** 7

## Accomplishments
- Refactored `Container` to store slice of services (`[]ServiceWrapper`) per key
- Updated `Register` to append new services instead of erroring on duplicates
- Implemented `ResolveAll[T]` and `ResolveGroup[T]` for service discovery
- Exposed public API in `gaz` package (`gaz.ResolveAll`, `gaz.ResolveGroup`)
- Verified behavior with new `discovery_test.go`

## Task Commits

1. **Task 1: Refactor Internal Storage** - `13d507d` (refactor)
2. **Task 2: Update Registration Logic** - `e8b7bda` (feat)
3. **Task 3: Implement Resolution Logic** - `0b2b849` (feat)
4. **Task 4: Expose Public API** - `117c3e9` (feat)

Additional commits:
- `089d790` test(37-01): update health module tests for multi-binding support
- `6c3c5e9` test(37-01): add discovery tests

## Files Created/Modified
- `di/container.go` - Storage refactoring, internal resolution logic
- `di/registration.go` - Append logic for registration
- `di/service.go` - Added Groups support to wrappers
- `di/resolution.go` - Added ResolveAll/ResolveGroup functions
- `types.go` - Public API delegation
- `di/discovery_test.go` - New tests for discovery features
- `health/module_test.go` - Updated tests to handle multi-binding behavior

## Decisions Made
- **Implicit Collection:** `Register` no longer errors on duplicates. This enables multiple modules to register the same interface (e.g., `GatewayEndpoint`) without coordination.
- **Ambiguity Handling:** `Resolve[T]` now returns `ErrAmbiguous` if multiple providers exist. To get one, use `ResolveAll` and pick, or ensure only one is registered.
- **Service Groups:** Added explicit `InGroup("name")` support for tagging services for collective resolution.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed regression in health module tests**
- **Found during:** Verification (`make test`)
- **Issue:** `health` tests expected registration error for duplicates, but multi-binding now allows it.
- **Fix:** Updated tests to assert success on registration, but `ErrAmbiguous` on resolution.
- **Files modified:** `health/module_test.go`
- **Commit:** `089d790`

## Next Phase Readiness
- Core discovery is ready.
- Next plan (if any) or Phase 38 can use `gaz.ResolveAll` to find gateway endpoints.
