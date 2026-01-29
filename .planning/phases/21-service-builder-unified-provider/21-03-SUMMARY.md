---
phase: 21-service-builder-unified-provider
plan: 03
subsystem: di
tags: [module-builder, cli-flags, pflag, cobra, fluent-api]

# Dependency graph
requires:
  - phase: 21-01
    provides: ModuleBuilder with Provide() and Use() methods
provides:
  - ModuleBuilder.Flags(fn) for CLI flag registration
  - ModuleBuilder.WithEnvPrefix(prefix) for config namespacing
  - App.Use() applies module flags when cobra command available
  - cobraCmd field in App struct for module flags integration
affects: [service-builder, unified-provider]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Module flags integration via interface type assertion
    - Deferred flag registration (flags applied during Use() if cobra available)

key-files:
  created: []
  modified:
    - module_builder.go
    - module_builder_test.go
    - app.go
    - app_use.go
    - cobra.go

key-decisions:
  - "Store cobraCmd in App struct during WithCobra() for later use"
  - "Use interface type assertion to check for FlagsFn() method"
  - "Flags applied to PersistentFlags on cobra command"
  - "No panic when cobra command not set - flags silently not applied"

patterns-established:
  - "Module flags pattern: Flags(fn) stores function, applied during Use()"
  - "Interface type assertion for optional module capabilities"

# Metrics
duration: 7min
completed: 2026-01-29
---

# Phase 21 Plan 03: Module Flags Integration Summary

**ModuleBuilder.Flags(fn) and WithEnvPrefix(prefix) for bundling CLI flags with modules, applied when app.Use() is called**

## Performance

- **Duration:** 7 min
- **Started:** 2026-01-29T23:20:29Z
- **Completed:** 2026-01-29T23:27:54Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments

- Added `ModuleBuilder.Flags(fn)` method for registering CLI flags with modules
- Added `ModuleBuilder.WithEnvPrefix(prefix)` method for config key namespacing
- Added `FlagsFn()` and `EnvPrefix()` accessor methods to `builtModule`
- Added `cobraCmd` field to `App` struct for storing cobra command reference
- Updated `WithCobra()` to store command reference for module flags integration
- Updated `app.Use()` to apply module flags when cobra command is available
- Comprehensive test coverage for all new functionality

## Task Commits

Each task was committed atomically:

1. **Task 1: Add Flags() and WithEnvPrefix() to ModuleBuilder** - `aed5d9a` (feat)
2. **Task 2: Update app.Use() to apply module flags** - `ac39f9c` (feat)
3. **Task 3: Tests for module flags integration** - `7abbc73` (test)

## Files Created/Modified

- `module_builder.go` - Added flagsFn, envPrefix fields; Flags(), WithEnvPrefix(), FlagsFn(), EnvPrefix() methods
- `module_builder_test.go` - Added 8 new tests for flags integration
- `app.go` - Added cobraCmd field and cobra import
- `app_use.go` - Added pflag import; flags application logic in Use()
- `cobra.go` - Store cobraCmd reference in WithCobra()

## Decisions Made

1. **Store cobraCmd in App struct** - Rather than requiring specific call order, the cobra command is stored during `WithCobra()` so modules can be added before or after cobra integration.

2. **Interface type assertion for FlagsFn()** - Use `m.(interface{ FlagsFn() func(*pflag.FlagSet) })` to check if module provides flags, maintaining backward compatibility with custom Module implementations.

3. **Apply to PersistentFlags** - Module flags are applied to `cmd.PersistentFlags()` so they're available to all subcommands.

4. **Silent no-op when no cobra** - If no cobra command is attached, flags are simply not applied (no error, no panic). This allows modules to be used in non-CLI contexts.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Module flags integration complete and tested
- Phase 21 (Service Builder + Unified Provider) is now complete
- All 3 plans completed: ModuleBuilder core, Service Builder, Module Flags
- Ready for v2.1 milestone verification

---
*Phase: 21-service-builder-unified-provider*
*Completed: 2026-01-29*
