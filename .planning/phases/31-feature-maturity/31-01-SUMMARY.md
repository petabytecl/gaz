---
phase: 31-feature-maturity
plan: 01
subsystem: config
tags: [mapstructure, viper, strict-validation, config]

# Dependency graph
requires:
  - phase: 30
    provides: "Stable config infrastructure and performance improvements"
provides:
  - "WithStrictConfig() option for strict config validation"
  - "UnmarshalStrict method in viper backend with ErrorUnused"
  - "LoadIntoStrict method in config Manager"
  - "StrictUnmarshaler interface for backend abstraction"
affects: [config-validation, startup-validation, typo-detection]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ErrorUnused mapstructure option for strict unmarshaling"
    - "Optional interface pattern (StrictUnmarshaler) for backend capabilities"

key-files:
  created: []
  modified:
    - "config/viper/backend.go"
    - "config/backend.go"
    - "config/manager.go"
    - "app.go"

key-decisions:
  - "StrictUnmarshaler interface allows backends without strict support to gracefully fallback"
  - "Strict validation applied only to config target struct, not ConfigProvider pattern"

patterns-established:
  - "Optional interface pattern: use type assertion to check if backend supports strict unmarshal"
  - "Fail-fast validation: catch config typos at startup, not runtime"

# Metrics
duration: 4min
completed: 2026-02-01
---

# Phase 31 Plan 01: Strict Config Validation Summary

**WithStrictConfig() option enables fail-fast validation using mapstructure ErrorUnused, catching typos and obsolete config keys at startup**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-01T16:35:45Z
- **Completed:** 2026-02-01T16:40:20Z
- **Tasks:** 3/3
- **Files modified:** 4

## Accomplishments

- Added `UnmarshalStrict` method to viper Backend using `mapstructure.ErrorUnused = true`
- Created `StrictUnmarshaler` interface for backend abstraction
- Added `LoadIntoStrict` method to config Manager that delegates to `StrictUnmarshaler`
- Added `WithStrictConfig()` option to gaz package that enables strict validation
- Wired `loadConfig()` to use `LoadIntoStrict` when strict flag is set

## Task Commits

Each task was committed atomically:

1. **Task 1: Add UnmarshalStrict to viper backend and StrictUnmarshaler interface** - `d35c2ac` (feat)
2. **Task 2: Add LoadIntoStrict to config Manager** - `f43904d` (feat)
3. **Task 3: Add WithStrictConfig() option to App and wire to loadConfig** - `cd53fbb` (feat)

## Files Created/Modified

- `config/viper/backend.go` - Added `strictDecoderOption`, `UnmarshalStrict`, and compile-time assertion
- `config/backend.go` - Added `StrictUnmarshaler` interface
- `config/manager.go` - Added `LoadIntoStrict` method with strict unmarshal delegation
- `app.go` - Added `strictConfig` field and `WithStrictConfig()` option, wired to `loadConfig()`

## Decisions Made

1. **StrictUnmarshaler as optional interface** - Backends that don't support strict validation gracefully fallback to normal unmarshal, maintaining backward compatibility
2. **Strict validation scope** - Only applied when `configTarget` is set via `WithConfig()`, has no effect on `ConfigProvider` pattern

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## Next Phase Readiness

- FEAT-01 requirement satisfied
- Ready for 31-02-PLAN.md (Worker dead letter handling)
- All existing tests pass

---
*Phase: 31-feature-maturity*
*Completed: 2026-02-01*
