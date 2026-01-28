---
phase: 13-config
plan: 01
subsystem: config
tags: [viper, interfaces, config-management, go-generics]

# Dependency graph
requires:
  - phase: 12
    provides: DI package extraction pattern
provides:
  - Backend interface for config abstraction
  - Watcher, Writer, EnvBinder composed interfaces
  - Defaulter, Validator lifecycle interfaces
  - ErrConfigValidation sentinel error
  - ViperBackend implementation
affects: [13-02, 13-03, 13-04, app-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [interface-composition, subpackage-isolation, compile-time-assertions]

key-files:
  created:
    - config/doc.go
    - config/backend.go
    - config/types.go
    - config/errors.go
    - config/viper/doc.go
    - config/viper/backend.go
  modified: []

key-decisions:
  - "Composed interfaces: core Backend + optional Watcher/Writer/EnvBinder"
  - "ViperBackend in subpackage to isolate viper dependency"
  - "StringReplacer interface with runtime type assertion for viper compatibility"
  - "ErrConfigValidation with 'config:' prefix (not 'gaz:')"

patterns-established:
  - "Interface composition: split optional capabilities into separate interfaces"
  - "Subpackage isolation: implementation details in subpackage to avoid transitive deps"
  - "Compile-time assertions: var _ Interface = (*Impl)(nil) pattern"

# Metrics
duration: 4min
completed: 2026-01-28
---

# Phase 13 Plan 01: Backend Interfaces and ViperBackend Summary

**Backend interface hierarchy with composed optional interfaces (Watcher, Writer, EnvBinder) and ViperBackend implementing all four interfaces**

## Performance

- **Duration:** 4 min
- **Started:** 2026-01-28T17:46:02Z
- **Completed:** 2026-01-28T17:50:27Z
- **Tasks:** 2
- **Files created:** 6

## Accomplishments

- Created config package with Backend interface defining core Get/Set/Unmarshal operations
- Added composed optional interfaces: Watcher (file watching), Writer (config writing), EnvBinder (env var binding)
- Defined Defaulter and Validator interfaces for config lifecycle
- Created ErrConfigValidation sentinel error with ValidationErrors and FieldError types
- Implemented ViperBackend in config/viper subpackage isolating viper dependency
- Added compile-time interface assertions ensuring ViperBackend implements all four interfaces

## Task Commits

Each task was committed atomically:

1. **Task 1: Create config package with Backend interface and types** - `2a8ff3f` (feat)
2. **Task 2: Create ViperBackend in config/viper subpackage** - `4c6352c` (feat)

## Files Created/Modified

- `config/doc.go` - Package documentation for standalone config management
- `config/backend.go` - Backend, Watcher, Writer, EnvBinder, StringReplacer interfaces
- `config/types.go` - Defaulter, Validator interfaces for config lifecycle
- `config/errors.go` - ErrConfigValidation, ValidationErrors, FieldError types
- `config/viper/doc.go` - Package documentation for viper-based implementation
- `config/viper/backend.go` - ViperBackend struct implementing all config interfaces

## Decisions Made

1. **Composed interfaces over monolithic interface** - Split optional capabilities (watching, writing, env binding) into separate interfaces so simple backends only need to implement Backend
2. **Subpackage for viper implementation** - Placed ViperBackend in config/viper to isolate the viper dependency from the core config package
3. **StringReplacer with runtime assertion** - viper's SetEnvKeyReplacer requires *strings.Replacer specifically, so we use runtime type assertion with clear panic message for non-compatible replacers
4. **Error prefix 'config:' not 'gaz:'** - Follows pattern from DI package where extracted packages use their own prefix

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

1. **viper.SetEnvKeyReplacer type requirement** - Discovered that viper requires `*strings.Replacer` specifically (concrete type) rather than accepting any interface with Replace method. Resolved by adding runtime type assertion with fallback panic for unsupported types.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Backend interfaces established and ready for Manager implementation
- ViperBackend complete and ready for integration
- Ready for 13-02-PLAN.md (Manager, options, validation, generic accessors)

---
*Phase: 13-config*
*Completed: 2026-01-28*
