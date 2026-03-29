---
phase: 17-expose-configprovider-flags-to-cobra-cli
plan: 01
subsystem: config
tags: [cobra, viper, pflag, cli, config]

# Dependency graph
requires:
  - phase: 14.4
    provides: ConfigProvider pattern and ProviderValues
provides:
  - RegisterCobraFlags method on App for CLI flag registration
  - FlagBinder interface for individual pflag binding
  - Idempotent config operations for flexible call ordering
affects: [17-02, cli-integration, config-examples]

# Tech tracking
tech-stack:
  added: []
  patterns: [explicit-flag-registration-before-execute, key-to-flag-transformation]

key-files:
  created: [cobra_flags.go]
  modified: [config/manager.go, config/viper/backend.go, app.go]

key-decisions:
  - "FlagBinder as exported interface for individual flag binding"
  - "Idempotency tracking via bool fields (configLoaded, providerValuesRegistered, providerConfigsCollected)"
  - "Key transformation: server.host -> --server-host for POSIX compliance"
  - "Bind to viper with original dot-notation key for correct precedence"

patterns-established:
  - "Call RegisterCobraFlags before Execute() for --help visibility"
  - "Config operations are idempotent - safe to call multiple times"

# Metrics
duration: 3min
completed: 2026-01-29
---

# Phase 17 Plan 01: Add FlagBinder Interface and RegisterCobraFlags Summary

**RegisterCobraFlags method to expose ConfigProvider flags to Cobra CLI with FlagBinder interface for viper binding**

## Performance

- **Duration:** 3 min
- **Started:** 2026-01-29T01:11:08Z
- **Completed:** 2026-01-29T01:14:47Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Added exported FlagBinder interface in config package for single-flag binding
- Implemented BindPFlag on ViperBackend wrapping viper's native method
- Made loadConfig, registerProviderValuesEarly, collectProviderConfigs idempotent
- Created RegisterCobraFlags method with typed flag registration and viper binding

## Task Commits

Each task was committed atomically:

1. **Task 1: Add FlagBinder interface and implement BindPFlag** - `3b512e1` (feat)
2. **Task 2: Add idempotency tracking to App config operations** - `3d23163` (feat)
3. **Task 3: Implement RegisterCobraFlags method** - `9423e04` (feat)

## Files Created/Modified
- `config/manager.go` - Added FlagBinder interface with BindPFlag(key, flag) method
- `config/viper/backend.go` - Implemented BindPFlag wrapping viper.BindPFlag, added interface assertion
- `app.go` - Added idempotency tracking fields and guards for config operations
- `cobra_flags.go` - New file with RegisterCobraFlags, registerPFlags, configKeyToFlagName, registerTypedFlag

## Decisions Made
- FlagBinder is an exported interface (uppercase) distinct from the existing internal flagBinder (lowercase for FlagSet binding)
- Idempotency uses simple bool fields rather than sync.Once for clarity and testability
- Key transformation replaces dots with hyphens: "server.host" -> "--server-host"
- Flags bind to viper with the original dot-notation key for correct precedence lookup
- PersistentFlags used for flags to work across subcommands

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- RegisterCobraFlags method ready for testing in 17-02
- FlagBinder interface available for test mocking
- Idempotent operations enable flexible call ordering in tests

---
*Phase: 17-expose-configprovider-flags-to-cobra-cli*
*Completed: 2026-01-29*
