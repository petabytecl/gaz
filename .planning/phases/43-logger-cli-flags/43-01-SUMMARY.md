# Phase 43 Plan 01: Defer Logger and Subsystem Initialization Summary

**One-liner:** Restructured App initialization to defer Logger/subsystems until Build(), enabling CLI flags to be available before Logger creation.

## What Was Done

### Task 1: Refactor App to defer Logger and subsystem initialization

**Changes to app.go:**
- Added `loggerInitialized` field to App struct for tracking initialization state
- Removed Logger/WorkerManager/Scheduler/EventBus initialization from `New()`
- Added `initializeLogger()` method - creates logger from resolved `logger.Config` or defaults
- Added `initializeSubsystems()` method - creates WorkerManager, Scheduler, EventBus after logger
- Added `getLogger()` helper for safe logging before Build()
- Updated `Build()` to call `initializeLogger()` and `initializeSubsystems()`
- Set default LoggerConfig in New() for consistent defaults

**Changes to cobra.go:**
- Converted `WithCobra()` from method to Option function
- Extracted `makePreRunE()` and `makePostRunE()` helper methods to reduce complexity
- Flags applied immediately via `AddFlagsFn()` when cobra cmd is attached
- Flags registered before WithCobra() are applied when WithCobra() is processed

**Updates to AddFlagsFn():**
- Now applies flags immediately if cobra command already attached
- Enables order-independent flag registration

**Safety improvements:**
- `doStop()` returns early if app was never built
- `getLogger()` provides fallback to slog.Default() for nil Logger
- Nil checks for workerMgr in shutdown

### Task 2: Add tests for new initialization pattern

**New tests in app_test.go:**
- `TestLoggerInitializedInBuild` - Verifies Logger is nil before Build(), initialized after
- `TestLoggerConfigResolution` - Verifies logger.Config resolution from container

**Updated tests:**
- `TestEventBus` - Updated to expect nil EventBus before Build()
- All Cobra-related tests updated to use Option pattern
- All shutdown tests updated to set `loggerInitialized = true` when setting custom Logger

## Commits

| Hash | Message |
|------|---------|
| 79635f5 | refactor(43-01): defer Logger and subsystem init until Build() |
| af3b6ec | test(43-01): add tests for deferred Logger initialization |

## Files Changed

### Created
- None

### Modified
- `app.go` - Core App initialization changes
- `cobra.go` - WithCobra as Option, helper methods
- `app_test.go` - Updated EventBus test, added Logger tests
- `app_integration_test.go` - Updated 4 tests to use Option pattern
- `cobra_test.go` - Updated tests for Option pattern
- `cobra_flags_test.go` - Updated 6 tests for Option pattern
- `module_builder_test.go` - Updated 3 tests for Option pattern
- `shutdown_test.go` - Fixed logger initialization in tests
- `examples/cobra-cli/main.go` - Updated to use Option pattern, added RunE
- `examples/grpc-gateway/main.go` - Updated to use Option pattern
- `server/gateway/module.go` - Fixed shadowed variable warning

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] cobra-cli example missing RunE**
- **Found during:** Test execution for cobra-cli
- **Issue:** serveCmd had no RunE, so server.Start() was never called
- **Fix:** Added RunE that resolves and starts the server
- **Files modified:** examples/cobra-cli/main.go
- **Commit:** 79635f5

**2. [Rule 3 - Blocking] Shutdown tests failing with nil Logger**
- **Found during:** Full test run
- **Issue:** Tests set app.Logger directly but Build() overwrote it
- **Fix:** Set app.loggerInitialized = true after setting custom Logger
- **Files modified:** shutdown_test.go
- **Commit:** 79635f5

**3. [Rule 3 - Blocking] Lint errors after refactoring**
- **Found during:** make lint
- **Issue:** Cognitive complexity, shadowed variables, unused parameters, unwrapped errors
- **Fix:** Extracted helper methods, renamed variables, used _ for unused params, wrapped errors
- **Files modified:** cobra.go, app.go, server/gateway/module.go
- **Commit:** 79635f5

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| WithCobra as Option | Enables flags to be registered and parsed before Logger creation |
| Immediate flag application | AddFlagsFn applies immediately if cobra cmd attached for order-independence |
| Default LoggerConfig in New() | Ensures consistent defaults even before Build() |
| Early return in doStop() | Prevents nil panics when Stop() called on non-built app |

## Verification

All verification criteria met:
- [x] All tests pass with `go test -race ./...`
- [x] Lint passes with `make lint`
- [x] WithCobra works as Option in gaz.New()
- [x] Logger is nil before Build(), initialized after
- [x] EventBus is nil before Build(), available after
- [x] Flags registered before/after WithCobra() all work correctly
