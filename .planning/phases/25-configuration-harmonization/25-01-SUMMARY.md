---
phase: 25-configuration-harmonization
plan: 01
subsystem: config
tags: [viper, mapstructure, struct-unmarshal, gaz-tag]

requires:
  - phase: 24-lifecycle-interface-alignment
    provides: stable lifecycle interfaces for module integration

provides:
  - config.ErrKeyNotFound sentinel for missing namespace detection
  - viper.Backend.UnmarshalWithGazTag and UnmarshalKeyWithGazTag methods
  - viper.Backend.HasKey for namespace existence checking
  - ProviderValues.Unmarshal and UnmarshalKey with gaz tag support

affects:
  - 25-02 (comprehensive tests for Unmarshal functionality)
  - 26 (module consolidation may use config unmarshaling)

tech-stack:
  added: []
  patterns:
    - "Type assertion for optional backend capabilities (gazUnmarshaler interface)"
    - "gaz struct tag for config field mapping"
    - "Sentinel error wrapping with fmt.Errorf(%w)"

key-files:
  created: []
  modified:
    - config/errors.go
    - config/viper/backend.go
    - provider_config.go

key-decisions:
  - "Local gazUnmarshaler interface in provider_config.go (not exported)"
  - "Fallback to standard Backend methods if gaz tag support unavailable"

duration: 2min
completed: 2026-01-30
---

# Phase 25 Plan 01: Add ErrKeyNotFound sentinel and Unmarshal methods Summary

**Struct-based config unmarshaling via ProviderValues.Unmarshal/UnmarshalKey with custom gaz tag support and ErrKeyNotFound sentinel for missing namespace detection**

## Performance

- **Duration:** 2 min
- **Started:** 2026-01-30T19:18:34Z
- **Completed:** 2026-01-30T19:20:09Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Added `config.ErrKeyNotFound` sentinel error for missing config keys/namespaces
- Added viper Backend methods for gaz tag unmarshaling (UnmarshalWithGazTag, UnmarshalKeyWithGazTag, HasKey)
- Added `ProviderValues.Unmarshal` and `UnmarshalKey` methods with gaz tag support
- Implemented namespace existence checking before unmarshal to return typed errors

## Task Commits

Each task was committed atomically:

1. **Task 1: Add ErrKeyNotFound sentinel and viper gaz tag methods** - `59abcd5` (feat)
2. **Task 2: Add Unmarshal methods to ProviderValues** - `39de858` (feat)

## Files Created/Modified

- `config/errors.go` - Added ErrKeyNotFound sentinel error
- `config/viper/backend.go` - Added gazDecoderOption, UnmarshalWithGazTag, UnmarshalKeyWithGazTag, HasKey
- `provider_config.go` - Added gazUnmarshaler interface, Unmarshal, UnmarshalKey methods

## Decisions Made

- **Local gazUnmarshaler interface**: Defined in provider_config.go as unexported, since it's an implementation detail for type-asserting backends that support gaz tags
- **Fallback behavior**: If backend doesn't implement gazUnmarshaler, falls back to standard Backend.Unmarshal (uses mapstructure tag) for compatibility

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ErrKeyNotFound sentinel ready for use
- viper Backend has full gaz tag support
- ProviderValues ready for struct-based config access
- Ready for 25-02-PLAN.md (comprehensive tests)

---
*Phase: 25-configuration-harmonization*
*Completed: 2026-01-30*
