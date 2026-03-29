---
phase: 41-refactor-server-module-architecture-and-consistency
plan: 03
subsystem: server
tags: [grpc, http, gateway, otel, config, modules]

requires:
  - phase: 41-refactor-server-module-architecture-and-consistency
    provides: [server-logger]
provides:
  - ConfigProvider pattern for server packages
  - Simplified server module API (no options)
  - Bundled server.NewModule (gRPC + Gateway)
affects:
  - phase: 41-refactor-server-module-architecture-and-consistency
    plan: 04

tech-stack:
  added: []
  patterns: [ConfigProvider, ModuleBuilder]

key-files:
  created: []
  modified:
    - server/module.go
    - server/grpc/module.go
    - server/http/module.go
    - server/gateway/module.go
    - server/otel/module.go

key-decisions:
  - "Removed ModuleOption pattern in favor of standard ConfigProvider (flags/config)"
  - "server.NewModule bundles gRPC and Gateway modules directly"
  - "Gateway module explicitly registers server/http module to ensure HTTP server availability"

duration: 15min
completed: 2026-02-03
---

# Phase 41 Plan 03: Refactor Server Modules Summary

**Adopted ConfigProvider pattern and simplified server module API by removing legacy options.**

## Performance

- **Duration:** 15 min
- **Started:** 2026-02-03T...
- **Completed:** 2026-02-03T...
- **Tasks:** 3
- **Files modified:** 14 (config and module files + tests)

## Accomplishments
- Refactored `server/grpc`, `server/http`, `server/gateway`, `server/otel` to use `gaz.NewModule` and `ConfigProvider`.
- Removed `ModuleOption` and `moduleConfig` structs, relying on standard `Config` structs loaded via DI.
- Updated `server.NewModule` to bundle `gRPC` and `Gateway` modules cleanly.
- Ensured `Gateway` module registers `server/http` module to provide the underlying HTTP server.
- Updated all tests to reflect the new API (using `gaz.New()` for integration testing).

## Task Commits

1. **Task 1 & 2 & 3: Refactor modules and API** - `a1b6764` (feat)
2. **Task 3 (Tests): Update tests** - `c04c73d` (test)

## Files Created/Modified
- `server/grpc/module.go` - Use gaz.NewModule, remove options
- `server/http/module.go` - Use gaz.NewModule, remove options
- `server/gateway/module.go` - Use gaz.NewModule, remove options, add http module
- `server/otel/module.go` - Use gaz.NewModule, remove options
- `server/module.go` - Bundle grpc and gateway
- `server/*/config.go` - Implement Namespace/Flags methods (verified)
- `server/**/*_test.go` - Updated for new API

## Decisions Made
- **ConfigProvider Pattern:** Switched from `NewModule(opts...)` to `NewModule()` + Flags/Config. This simplifies the API and standardizes configuration handling via `gaz` framework.
- **Gateway dependencies:** `server.NewModule` no longer registers `http` explicitly (per plan), so `gateway` module now explicitly `.Use(serverhttp.NewModule())` to ensure it has a server to run on. This encapsulates the dependency correctly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added http module to gateway**
- **Found during:** Task 2/3 (Refactoring gateway module)
- **Issue:** Plan requested removing `server/http` from `server.NewModule` but `gateway` needs an HTTP server to listen.
- **Fix:** Added `.Use(serverhttp.NewModule())` to `server/gateway/module.go`.
- **Files modified:** server/gateway/module.go
- **Verification:** Tests pass, gateway module registers http server.
- **Committed in:** a1b6764

## Issues Encountered
None.

## Next Phase Readiness
- Server modules are now consistent and use the framework's configuration pattern.
- Ready for final cleanup or documentation in Plan 04.
