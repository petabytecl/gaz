---
phase: 11-cleanup
plan: 01
subsystem: core
tags: [di, generics, fluent-api, cleanup, refactoring]

# Dependency graph
requires:
  - phase: none
    provides: "This is the first plan in v2.0 milestone"
provides:
  - "Clean core library without deprecated reflection-based APIs"
  - "All tests using For[T]() fluent registration pattern"
  - "Updated documentation with new API examples"
affects: [11-02-PLAN, 12-di, 13-config]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "For[T]() fluent builder pattern is the sole registration API"
    - "Instance registration via For[T]().Instance() for pre-built instances"
    - "Internal reflection-only for WithConfig (runtime type registration)"

key-files:
  created: []
  modified:
    - app.go
    - service.go
    - doc.go
    - cobra.go
    - health/integration.go
    - health/module_test.go
    - app_test.go
    - app_integration_test.go
    - cobra_test.go
    - config_test.go
    - provider_config_test.go
    - example_test.go
    - example_lifecycle_test.go
    - service_test.go
    - examples/basic/main.go
    - examples/lifecycle/main.go
    - examples/cobra-cli/main.go
    - examples/http-server/main.go

key-decisions:
  - "Retained registerInstance() and instanceServiceAny for internal use by WithConfig and Logger registration"
  - "Renamed ExampleApp_ProvideSingleton to ExampleFor_provider to match new API"
  - "Replaced TestAnyWrappers_Lifecycle with TestInstanceServiceAny_Lifecycle (only testing retained type)"

patterns-established:
  - "For[T](): Use For[T](container).Provider()/ProviderFunc()/Instance() for all service registration"
  - "Error handling: For[T]() methods return errors; check errors after registration calls"

# Metrics
duration: 45min
completed: 2026-01-27
---

# Phase 11 Plan 01: Remove Deprecated APIs Summary

**Removed reflection-based ProvideSingleton/ProvideTransient/ProvideEager/ProvideInstance methods and migrated all 50+ usages to generic For[T]() fluent pattern**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-01-27
- **Completed:** 2026-01-27
- **Tasks:** 2
- **Files modified:** 18

## Accomplishments

- Removed all deprecated public APIs: `NewApp()`, `AppOption`, `ProvideSingleton()`, `ProvideTransient()`, `ProvideEager()`, `ProvideInstance()`, `registerProvider()`
- Removed deprecated internal types: `lazySingletonAny`, `transientServiceAny`, `eagerSingletonAny`
- Migrated 50+ API usages across test files and examples to `For[T]()` pattern
- Updated package documentation with new API examples
- All tests passing (build + test verification complete)

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove deprecated APIs from core files** - `71dd411` (refactor)
2. **Task 2: Migrate app_test.go** - `f3ef408` (refactor)
3. **Task 2: Migrate app_integration_test.go** - `0f4642d` (refactor)
4. **Task 2: Migrate cobra_test.go** - `7b142d2` (refactor)
5. **Task 2: Migrate config_test.go** - `9855601` (refactor)
6. **Task 2: Migrate provider_config_test.go** - `353d5f5` (refactor)
7. **Task 2: Migrate example_test.go** - `2944f9d` (refactor)
8. **Task 2: Migrate example_lifecycle_test.go** - `66bec63` (refactor)
9. **Task 2: Migrate health/module_test.go** - `c6dc057` (refactor)
10. **Task 2: Remove tests for deprecated service types** - `709a60a` (refactor)
11. **Task 2: Migrate example files** - `5f5dade` (refactor)

## Files Created/Modified

**Core files (deprecated code removed):**
- `app.go` - Removed deprecated methods, updated New() to use For[T]() for Logger
- `service.go` - Removed deprecated types, kept instanceServiceAny for internal use
- `doc.go` - Updated all examples to For[T]() pattern
- `cobra.go` - Updated example in doc comment

**Test files (migrated to For[T]()):**
- `app_test.go` - 20+ usages migrated
- `app_integration_test.go` - 5 usages migrated
- `cobra_test.go` - 5 usages migrated
- `config_test.go` - 1 usage migrated
- `provider_config_test.go` - 15 usages migrated
- `example_test.go` - 5 usages migrated, renamed ExampleApp_ProvideSingleton
- `example_lifecycle_test.go` - 1 usage migrated
- `service_test.go` - Removed tests for deprecated types
- `health/module_test.go` - 1 usage migrated

**Health package:**
- `health/integration.go` - ProvideInstance -> For[Config]().Instance()

**Example applications:**
- `examples/basic/main.go` - ProvideSingleton -> For[T]().Provider()
- `examples/lifecycle/main.go` - ProvideSingleton -> For[T]().Provider()
- `examples/cobra-cli/main.go` - ProvideInstance -> For[T]().Instance()
- `examples/http-server/main.go` - ProvideSingleton/ProvideInstance -> For[T]() pattern

## Decisions Made

1. **Retained registerInstance() for internal use** - Required by WithConfig() for runtime type registration via reflection. Cannot use For[T]() because type is only known at runtime.

2. **Retained instanceServiceAny for internal use** - Used by registerInstance() to wrap pre-built instances with reflection-determined types.

3. **Renamed ExampleApp_ProvideSingleton to ExampleFor_provider** - Go example naming convention follows Type_Method pattern; new API centers on For function.

4. **Replaced TestAnyWrappers_Lifecycle with focused test** - Original test covered 4 deprecated types; new test only covers retained instanceServiceAny.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Additional test files required migration**
- **Found during:** Task 2
- **Issue:** service_test.go referenced removed types newLazySingletonAny, newEagerSingletonAny, newTransientAny
- **Fix:** Replaced TestAnyWrappers_Lifecycle with TestInstanceServiceAny_Lifecycle that only tests retained type
- **Files modified:** service_test.go
- **Committed in:** 709a60a

**2. [Rule 3 - Blocking] Example applications used deprecated APIs**
- **Found during:** Task 2 verification
- **Issue:** examples/basic, examples/lifecycle, examples/cobra-cli, examples/http-server used deprecated methods
- **Fix:** Migrated all to For[T]() pattern
- **Files modified:** examples/*/main.go
- **Committed in:** 5f5dade

---

**Total deviations:** 2 auto-fixed (blocking issues)
**Impact on plan:** Both necessary for build to succeed. No scope creep.

## Issues Encountered

None - plan executed smoothly after addressing build failures from unreferenced deprecated types.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Core library is now clean of deprecated APIs
- Ready for Plan 02: Update Documentation
- All verification checks pass:
  - `go build ./...` succeeds
  - `go test ./...` succeeds
  - No deprecated API references remain

---
*Phase: 11-cleanup*
*Completed: 2026-01-27*
