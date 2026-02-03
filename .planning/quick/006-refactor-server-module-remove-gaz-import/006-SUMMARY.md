---
quick_task: 006
subsystem: server
tags: [di, pflag, module, refactor]

key-files:
  modified:
    - server/module.go
    - server/module_test.go

key-decisions:
  - "Deferred pointer evaluation: Flag values are read via pointer dereferencing at module registration time, not creation time"
  - "Gateway pattern: NewModuleWithFlags takes *pflag.FlagSet as first parameter, matching server/gateway/module.go"
  - "Return di.Module: Instead of gaz.Module, eliminating potential import cycle"

patterns-established:
  - "Flag-first modules: Pass *pflag.FlagSet to module constructors for CLI integration"

duration: 4min
completed: 2026-02-03
---

# Quick Task 006: Refactor server/module.go Remove gaz Import

**Eliminates potential import cycle by refactoring NewModuleWithFlags to gateway pattern with deferred flag evaluation.**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-02-03T22:57:42Z
- **Completed:** 2026-02-03T23:01:00Z
- **Tasks:** 2/2
- **Files modified:** 2

## Accomplishments

- Removed `gaz` package import from `server/module.go`
- Changed `NewModuleWithFlags` signature to take `*pflag.FlagSet` as first parameter
- Updated all tests to work with new signature (7/7 pass)
- Maintained 91.3% coverage (above 90% threshold)

## Task Commits

1. **Task 1: Refactor server/module.go** - `943b106` (refactor)
   - Remove gaz import
   - Change signature: `func NewModuleWithFlags(fs *pflag.FlagSet, opts ...ModuleOption) di.Module`
   - Use deferred pointer evaluation for flag values

2. **Task 2: Update server/module_test.go** - `c06f475` (test)
   - Remove gaz import from tests
   - Update all test cases to pass fs as first parameter
   - Test di.Container directly instead of gaz.App

## Files Modified

- `server/module.go` - Refactored NewModuleWithFlags to take *pflag.FlagSet, return di.Module
- `server/module_test.go` - Updated tests for new signature

## Decisions Made

1. **Deferred pointer evaluation:** Flag values are captured via pointers (`*grpcPortFlag`) and dereferenced inside the di.NewModuleFunc closure. This ensures values are read after flag parsing, not at module creation time.

2. **Gateway pattern alignment:** The new signature matches `server/gateway/module.go`, establishing a consistent pattern for flag-enabled modules.

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

```
✓ go build ./server/...           # Compiles without gaz import
✓ go test -race ./server/...      # All 14 tests pass
✓ make test                       # Full test suite passes
✓ make lint                       # 0 issues
✓ make cover                      # 91.3% coverage (>90%)
```

## API Change Summary

**Before:**
```go
func NewModuleWithFlags(opts ...ModuleOption) gaz.Module
```

**After:**
```go
func NewModuleWithFlags(fs *pflag.FlagSet, opts ...ModuleOption) di.Module
```

**Migration:** Callers must now pass a FlagSet as the first argument:
```go
// Before
app.Use(server.NewModuleWithFlags())

// After
app.Use(server.NewModuleWithFlags(cmd.Flags()))
```
