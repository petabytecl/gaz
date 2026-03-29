---
phase: 28-testing-infrastructure
plan: 01
subsystem: testing
tags: [gaztest, di.Module, config-injection, RequireResolve, testing-helpers]

# Dependency graph
requires:
  - phase: 26
    provides: di.Module interface for subsystem modules
  - phase: 27
    provides: Standardized error handling for resolution errors
provides:
  - WithModules() method for module registration in test apps
  - WithConfigMap() method for config injection in tests
  - RequireResolve[T]() generic helper for type-safe resolution
affects: [28-02, 28-03, 28-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "gaztest Builder pattern with fluent API"
    - "di.Module registration via UseDI"
    - "config injection via MergeConfigMap"

key-files:
  created:
    - gaztest/resolve.go
    - gaztest/resolve_test.go
  modified:
    - gaztest/builder.go
    - gaztest/gaztest_test.go
    - app.go
    - config/viper/backend.go

key-decisions:
  - "WithModules uses UseDI for di.Module registration (avoids import cycle)"
  - "WithApp and WithModules are mutually exclusive (panic on both)"
  - "Config injection uses MergeConfigMap with fallback to Set() for each key"

patterns-established:
  - "RequireResolve[T] pattern for test helpers that fail on error"
  - "TB interface for testing.T/testing.B compatibility"

# Metrics
duration: 8min
completed: 2026-02-01
---

# Phase 28 Plan 01: Enhance gaztest Builder API Summary

**Added WithModules(), WithConfigMap(), and RequireResolve[T]() to gaztest for v3 module support and type-safe test resolution**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-01T02:24:23Z
- **Completed:** 2026-02-01T02:32:38Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments

- WithModules(m ...di.Module) method for registering modules in test apps
- WithConfigMap(map[string]any) method for injecting test config values
- RequireResolve[T](tb, app) generic helper that fails test on resolution error
- Comprehensive test coverage for all new functionality

## Task Commits

Each task was committed atomically:

1. **Task 1: Add WithModules and WithConfigMap to Builder** - `92a08a4` (feat)
2. **Task 2: Add RequireResolve generic helper** - `dfe3bc4` (feat)
3. **Task 3: Add tests for new gaztest API** - `356ba28` (test)

## Files Created/Modified

- `gaztest/builder.go` - Added modules and configMap fields, WithModules(), WithConfigMap() methods, updated Build()
- `gaztest/resolve.go` - New RequireResolve[T]() generic helper function
- `gaztest/resolve_test.go` - Tests for RequireResolve success and failure cases
- `gaztest/gaztest_test.go` - Tests for WithModules, WithConfigMap, and panic on conflict
- `app.go` - Added MergeConfigMap() method with configMapMerger interface
- `config/viper/backend.go` - Added MergeConfigMap() method wrapping viper.MergeConfigMap

## Decisions Made

1. **WithModules uses UseDI()** - di.Module requires UseDI() on gaz.App to avoid the Apply(app) requirement. This aligns with how subsystem modules work.

2. **Mutual exclusion of WithApp and WithModules** - Using both patterns is ambiguous (is the baseApp pre-built or should modules be registered?). Panic with clear message guides users to correct usage.

3. **Config injection via MergeConfigMap** - Added to both viper backend and gaz.App. For backends that don't support MergeConfigMap, falls back to calling Set() for each key.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added MergeConfigMap to viper backend and App**
- **Found during:** Task 1 (Build() implementation)
- **Issue:** Plan assumed gaz.App had MergeConfigMap method but it didn't exist
- **Fix:** Added MergeConfigMap() to config/viper/backend.go wrapping viper.MergeConfigMap, and added configMapMerger interface + MergeConfigMap() to app.go
- **Files modified:** config/viper/backend.go, app.go
- **Verification:** go build succeeds, tests pass
- **Committed in:** 92a08a4 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed test for RequireResolve failure case**
- **Found during:** Task 3 (test implementation)
- **Issue:** Test was capturing format string instead of formatted message
- **Fix:** Changed fatalfCatcher.Fatalf to use fmt.Sprintf for actual message
- **Files modified:** gaztest/resolve_test.go
- **Verification:** Test passes correctly
- **Committed in:** 356ba28 (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both necessary for correct operation. No scope creep.

## Issues Encountered

None - plan executed with minor adjustments for existing API structure.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- gaztest Builder API now fully supports v3 patterns
- Ready for 28-02: Per-subsystem testing.go for health, worker, cron
- WithModules() enables testing subsystem modules
- RequireResolve() enables type-safe test assertions

---
*Phase: 28-testing-infrastructure*
*Completed: 2026-02-01*
