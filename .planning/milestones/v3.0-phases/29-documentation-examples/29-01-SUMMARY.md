---
phase: 29-documentation-examples
plan: 01
subsystem: documentation
tags: [godoc, examples, testable-examples, di, config]

# Dependency graph
requires:
  - phase: 28-testing-infrastructure
    provides: Testing infrastructure and per-package helpers
provides:
  - Godoc examples for di package (15 testable examples)
  - Godoc examples for config package (20 testable examples)
affects: [29-02, 29-03, 29-04, 29-05, documentation, v3-release]

# Tech tracking
tech-stack:
  added: []
  patterns: [testable-examples-with-output, example-naming-convention]

key-files:
  created:
    - di/example_test.go
    - config/example_test.go
  modified: []

key-decisions:
  - "Used strings.Contains for type name verification to avoid full package path dependency"
  - "Removed Server type from di examples as lifecycle tests belong in gaz package"
  - "20 examples for config package (exceeds 10 requirement) covering all major APIs"

patterns-established:
  - "Example function naming: ExampleTypeName_method for methods"
  - "Error handling pattern: if err != nil { fmt.Println(\"error:\", err); return }"
  - "Use _test package suffix for external test packages"

# Metrics
duration: 5min
completed: 2026-02-01
---

# Phase 29 Plan 01: Core Package Examples Summary

**Comprehensive godoc examples for di and config packages with testable Example functions covering all major public APIs**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-01T03:57:00Z
- **Completed:** 2026-02-01T04:02:10Z
- **Tasks:** 2
- **Files created:** 2

## Accomplishments

- Created 15 testable Example functions for di package (465 lines)
- Created 20 testable Example functions for config package (448 lines)
- All examples have `// Output:` comments and pass `go test`
- Examples cover Container, For, Resolve, Module (di) and Manager, Backend, MapBackend (config)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create DI package examples** - `cfd21c0` (docs)
2. **Task 2: Create Config package examples** - `7fd8409` (docs)

## Files Created

- `di/example_test.go` - 15 testable examples covering Container, For, Resolve, Module, Named, Has, TypeName APIs
- `config/example_test.go` - 20 testable examples covering Manager, MapBackend, Backend getters, validation, options

## API Coverage

### DI Package Examples

| Example | API Covered |
|---------|-------------|
| ExampleNew | di.New() |
| ExampleContainer_Build | Container.Build() |
| ExampleContainer_List | Container.List() |
| ExampleFor_singleton | For[T]().Instance() |
| ExampleFor_transient | For[T]().Transient() |
| ExampleFor_eager | For[T]().Eager() |
| ExampleFor_instance | For[T]().Instance() |
| ExampleFor_named | For[T]().Named(), Named() option |
| ExampleResolve | Resolve[T]() |
| ExampleResolve_withDependencies | Resolve[T]() with nested deps |
| ExampleMustResolve | MustResolve[T]() |
| ExampleNewModuleFunc | NewModuleFunc() |
| ExampleModule | Module interface implementation |
| ExampleTypeName | TypeName[T]() |
| ExampleHas | Has[T]() |

### Config Package Examples

| Example | API Covered |
|---------|-------------|
| ExampleNewMapBackend | NewMapBackend() |
| ExampleMapBackend_Get | MapBackend.Get() |
| ExampleMapBackend_Set | MapBackend.Set() |
| ExampleMapBackend_SetDefault | MapBackend.SetDefault() |
| ExampleMapBackend_IsSet | MapBackend.IsSet() |
| ExampleNew | config.New() with options |
| ExampleNewWithBackend | NewWithBackend() |
| ExampleManager_Backend | Manager.Backend() |
| ExampleTestManager | TestManager() factory |
| ExampleBackend_GetString | Backend.GetString() |
| ExampleBackend_GetInt | Backend.GetInt() |
| ExampleBackend_GetBool | Backend.GetBool() |
| ExampleValidateStruct | ValidateStruct() |
| ExampleSampleConfig | SampleConfig with Defaulter |
| ExampleRequireConfigValue | RequireConfigValue() |
| ExampleRequireConfigString | RequireConfigString() |
| ExampleRequireConfigInt | RequireConfigInt() |
| ExampleRequireConfigIsSet | RequireConfigIsSet() |
| ExampleWithBackend | WithBackend() option |
| ExampleWithDefaults | WithDefaults() option |

## Decisions Made

1. **Type name verification approach:** Used `strings.Contains()` instead of exact match for TypeName examples because type names include full package path which varies by test environment.

2. **Removed Server lifecycle type from di examples:** Lifecycle examples with OnStart/OnStop belong in the gaz package examples (already exist in example_lifecycle_test.go). DI package focuses on container mechanics.

3. **Config examples exceed minimum:** Created 20 examples instead of required 10 to cover all MapBackend methods, Manager APIs, validation helpers, and options comprehensively.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all examples passed on first or second iteration after minor output format adjustments.

## Next Phase Readiness

- Core di and config packages now have comprehensive godoc examples
- Ready for 29-02-PLAN.md (Health & EventBus Examples)
- Same patterns established can be applied to remaining subsystem packages

---
*Phase: 29-documentation-examples*
*Completed: 2026-02-01*
