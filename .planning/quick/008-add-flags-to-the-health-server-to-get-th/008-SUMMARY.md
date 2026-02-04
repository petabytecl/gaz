---
type: quick-summary
task: 008
title: Add CLI flags to health server for port configuration
completed: 2026-02-04
duration: ~4 minutes
commits:
  - 3eeec92: "feat(008): add CLI flag methods to health.Config"
  - 011358a: "feat(008): create health/module subpackage with CLI flag support"
  - 3de68c9: "test(008): add tests for health/module package"
files_modified:
  - health/config.go
files_created:
  - health/module/module.go
  - health/module/module_test.go
---

# Quick Task 008: Add CLI Flags to Health Server

**One-liner:** health/module subpackage with --health-port and --health-*-path CLI flags following ConfigProvider pattern.

## What Was Done

### Task 1: Add Config Interface Methods

Added the following methods to `health.Config` matching the server/http pattern:

- `Namespace() string` - returns "health"
- `Flags(fs *pflag.FlagSet)` - registers CLI flags:
  - `--health-port` (int, default 9090)
  - `--health-liveness-path` (string, default "/live")
  - `--health-readiness-path` (string, default "/ready")
  - `--health-startup-path` (string, default "/startup")
- `SetDefaults()` - applies default values to zero-value fields
- `Validate() error` - validates port is between 1-65535

Added constants:
- `MaxPort = 65535`
- `DefaultLivenessPath = "/live"`
- `DefaultReadinessPath = "/ready"`
- `DefaultStartupPath = "/startup"`

### Task 2: Create health/module Subpackage

Created `health/module.New()` following the logger/module and config/module pattern:

```go
import healthmod "github.com/petabytecl/gaz/health/module"

app := gaz.New(gaz.WithCobra(rootCmd))
app.Use(healthmod.New())
```

The subpackage pattern was required because:
- `gaz` package imports `health` for `HealthConfigProvider`
- `health` cannot import `gaz` (import cycle)
- Solution: create `health/module` subpackage that imports both

### Task 3: Add Tests

Comprehensive tests in `health/module/module_test.go`:
- `TestNew` - verifies module creation and default config resolution
- `TestConfig_Flags` - verifies all flags are registered correctly
- `TestConfig_Validate` - tests port validation edge cases
- `TestConfig_SetDefaults` - verifies zero-value defaults
- `TestConfig_SetDefaults_PreservesExisting` - verifies custom values preserved
- `TestConfig_Namespace` - verifies namespace returns "health"

## Deviations from Plan

### [Rule 3 - Blocking] Created health/module subpackage instead of modifying health/module.go

**Found during:** Task 2
**Issue:** Plan specified adding `NewModuleWithFlags()` to `health/module.go` with `import "github.com/petabytecl/gaz"`. This causes an import cycle because `gaz` already imports `health` for `HealthConfigProvider`.
**Fix:** Created `health/module` subpackage following the same pattern used by `logger/module` and `config/module`.
**Files created:** `health/module/module.go`, `health/module/module_test.go`

## Verification

```bash
go build ./health/...        # ✓ Builds successfully
go test -race -v ./health/... # ✓ All tests pass
make lint                     # ✓ 0 issues
```

## Usage

```go
import (
    "github.com/petabytecl/gaz"
    healthmod "github.com/petabytecl/gaz/health/module"
)

func main() {
    rootCmd := &cobra.Command{Use: "myapp"}
    
    app := gaz.New(gaz.WithCobra(rootCmd))
    app.Use(healthmod.New())
    
    // Flags available:
    // --health-port=9090
    // --health-liveness-path=/live
    // --health-readiness-path=/ready
    // --health-startup-path=/startup
}
```
