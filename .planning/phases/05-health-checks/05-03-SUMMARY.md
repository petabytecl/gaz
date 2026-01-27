---
phase: 05-health-checks
plan: 03
subsystem: observability
tags: [health, management-server, lifecycle, di, integration]

requires:
  - phase: 05-health-checks
    provides: "IETF-compliant Health HTTP Handlers"
provides:
  - "Management Server with dedicated port"
  - "Health Module integration"
  - "App configuration option for health checks"
affects:
  - 06-lifecycle-hooks
  - 04-config-system

tech-stack:
  added: []
  patterns: [lifecycle-hooks, module-pattern]

key-files:
  created: [health/server.go, health/config.go, health/module.go, health/integration.go]
  modified: []

key-decisions:
  - "WithHealthChecks option is defined in health package to avoid circular dependency with gaz root package"
  - "ManagementServer uses Start/Stop methods (not OnStart/OnStop) to avoid double invocation by gaz lifecycle engine which auto-detects Starter/Stopper interfaces"
  - "Shutdown sequence explicitly marks readiness as failed BEFORE stopping the server to allow load balancers to drain traffic"

patterns-established:
  - "Module function pattern for grouping related providers and hooks"

duration: 20 min
completed: 2026-01-27
---

# Phase 05 Plan 03: Management Server Integration Summary

**Integrated dedicated Management Server with App lifecycle, exposing IETF health endpoints on port 9090.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-01-27T00:59:57Z
- **Completed:** 2026-01-27T01:20:00Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- Implemented `ManagementServer` running on a separate, configurable port (default 9090).
- Created `health.Module` that wires up `Manager`, `ManagementServer`, and `ShutdownCheck`.
- Integrated with `gaz.App` via `health.WithHealthChecks(config)` option.
- Verified end-to-end functionality including lifecycle hooks and port binding.

## Task Commits

1. **Task 1: Create Management Server** - `63472bf` (feat)
2. **Task 2: Implement Health Module** - `450e07a` (feat)
3. **Task 3: Integrate with App** - `e2ab984` (feat)

## Files Created/Modified
- `health/server.go` - Management Server implementation with lifecycle control.
- `health/config.go` - Configuration struct with JSON/YAML tags.
- `health/module.go` - DI wiring for the health subsystem.
- `health/integration.go` - Public API for enabling health checks in the App.
- `tests/health_test.go` - E2E integration verification.

## Decisions Made
- **Circular Dependency Avoidance:** Placed `WithHealthChecks` in the `health` package instead of `gaz` root, as `health` depends on `gaz` for container types.
- **Lifecycle Hook Management:** Renamed `ManagementServer` methods to `Start`/`Stop` to avoid `gaz`'s auto-detection of `Starter`/`Stopper` interfaces, ensuring hooks are only registered once via the Module.
- **Graceful Shutdown:** Configured `OnStop` hook to mark the application as shutting down (failing readiness probes) *before* stopping the HTTP server, enabling graceful traffic draining.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed double lifecycle hook invocation**
- **Found during:** Task 3 (Integration testing)
- **Issue:** `gaz` auto-detected `OnStart` method on `ManagementServer` AND ran the manually registered `OnStart` hook, causing the server to try binding the port twice ("address already in use").
- **Fix:** Renamed methods to `Start`/`Stop` to opt-out of auto-detection, relying solely on explicit Module registration.
- **Files modified:** `health/server.go`, `health/module.go`
- **Verification:** Integration test passed without port conflict.
- **Committed in:** `e2ab984` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (Blocking issue)
**Impact on plan:** Corrected implementation details while maintaining functional requirements.

## Next Phase Readiness
- Health subsystem is complete and integrated.
- Ready for Phase 06: Lifecycle Hooks (Standardizing the lifecycle engine further if needed, or using the one verified here).
- Ready for deployment (Ops) related tasks as observability is now in place.
