---
phase: 43-logger-cli-flags
plan: 02
subsystem: logging
tags: [slog, cli-flags, pflag, module]

# Dependency graph
requires:
  - phase: 43-01
    provides: Deferred logger initialization until Build()
provides:
  - Logger Config with Output field
  - DefaultConfig() function
  - Flags(), Namespace(), Validate(), SetDefaults() methods
  - NewLoggerWithWriter for testing
  - logger/module subpackage with New() function
affects: [44-config-file-cli-flag, examples]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - ConfigProvider pattern for logger module
    - Subpackage module pattern to avoid circular imports

key-files:
  created:
    - logger/module/module.go
    - logger/module/module_test.go
  modified:
    - logger/config.go
    - logger/provider.go

key-decisions:
  - "Logger module in subpackage (logger/module) to avoid circular import with gaz"
  - "Ignore UnmarshalKey errors when namespace not found, use defaults"
  - "File output uses 0644 permissions for log monitoring tools"

patterns-established:
  - "Logger module import: import loggermod \"github.com/petabytecl/gaz/logger/module\""

# Metrics
duration: 7min
completed: 2026-02-04
---

# Phase 43 Plan 02: Logger Module with CLI Flags Summary

**Extended logger.Config with Output field and CLI flag support, created logger/module subpackage providing gaz.Module with --log-level, --log-format, --log-output, --log-add-source flags**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-04T04:20:42Z
- **Completed:** 2026-02-04T04:27:54Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Extended Config with Output field for stdout/stderr/file destinations
- Added DefaultConfig(), Flags(), Namespace(), Validate(), SetDefaults(), LevelName() methods
- Added NewLoggerWithWriter() for testing with custom io.Writer
- Created logger/module subpackage with New() function providing gaz.Module
- Comprehensive tests (214 lines) covering config, flags, validation, output destinations
- File output gracefully falls back to stdout on error with warning

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend Config and Provider with CLI flag support** - `4ecf622` (feat)
2. **Task 2: Create logger module and comprehensive tests** - `a2f81b0` (feat)

## Files Created/Modified

- `logger/config.go` - Added Output field, DefaultConfig(), Flags(), Namespace(), Validate(), SetDefaults(), LevelName(), parseLevel()
- `logger/provider.go` - Added NewLoggerWithWriter(), resolveOutput() for stdout/stderr/file handling
- `logger/module/module.go` - New gaz.Module providing logger.Config with CLI flags
- `logger/module/module_test.go` - Comprehensive tests for module and config (214 lines)

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Logger module in subpackage | gaz package imports logger, so logger cannot import gaz. Created logger/module subpackage to break cycle. |
| Ignore UnmarshalKey errors | Following gateway module pattern - missing namespace key means use defaults from flags |
| File permissions 0644 | Log files need to be readable by log monitoring tools (nolint:gosec added) |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Changed module location from logger/module.go to logger/module/module.go**
- **Found during:** Task 2 (module creation)
- **Issue:** `gaz` package imports `logger` package for logger.Config and logger.NewLogger. Adding `import "github.com/petabytecl/gaz"` to logger/module.go creates a circular import.
- **Fix:** Created `logger/module` subpackage instead. Import path is `github.com/petabytecl/gaz/logger/module`
- **Files modified:** logger/module/module.go (created in subpackage instead of logger/module.go)
- **Verification:** `go build ./logger/module/...` succeeds without circular import error
- **Committed in:** a2f81b0

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Module location changed to avoid circular import. API unchanged - users import `loggermod "github.com/petabytecl/gaz/logger/module"` and call `loggermod.New()`.

## Issues Encountered

None - plan executed with one blocking issue auto-fixed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Logger module complete and tested
- Ready for Phase 44 (Config File CLI Flag)
- All CLI flags registered: --log-level, --log-format, --log-output, --log-add-source

---
*Phase: 43-logger-cli-flags*
*Completed: 2026-02-04*
