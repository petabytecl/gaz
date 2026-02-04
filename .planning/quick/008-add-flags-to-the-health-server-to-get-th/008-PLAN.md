---
type: quick
task: 008
title: Add CLI flags to health server for port configuration
wave: 1
autonomous: true
files_modified:
  - health/config.go
  - health/module.go
  - health/module_test.go
---

<objective>
Add CLI flag support to the health module following the ConfigProvider pattern used by HTTP and gRPC server modules.

Purpose: Allow health server port to be configured via CLI flags (`--health-port`) instead of only programmatic options.
Output: Health module with `NewModuleWithFlags()` that integrates with gaz.NewModule builder.
</objective>

<context>
@AGENTS.md
@health/config.go
@health/module.go
@server/http/config.go (pattern reference)
@server/http/module.go (pattern reference)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add Config interface methods for CLI flag support</name>
  <files>health/config.go</files>
  <action>
Add the following methods to health.Config following the pattern from server/http/config.go:

1. `Namespace() string` - returns "health"
2. `Flags(fs *pflag.FlagSet)` - registers CLI flags:
   - `--health-port` (int, default 9090, "Health server port")
   - `--health-liveness-path` (string, default "/live", "Liveness endpoint path")
   - `--health-readiness-path` (string, default "/ready", "Readiness endpoint path")  
   - `--health-startup-path` (string, default "/startup", "Startup endpoint path")
3. `SetDefaults()` - applies default values to zero-value fields
4. `Validate() error` - validates port is between 1-65535

Add constant `MaxPort = 65535` for validation.

Import `github.com/spf13/pflag` and `errors`.
  </action>
  <verify>go build ./health/...</verify>
  <done>health.Config has Namespace(), Flags(), SetDefaults(), Validate() methods matching server/http pattern</done>
</task>

<task type="auto">
  <name>Task 2: Create NewModuleWithFlags using gaz.NewModule builder</name>
  <files>health/module.go</files>
  <action>
Add a new function `NewModuleWithFlags() gaz.Module` that:

1. Uses `gaz.NewModule("health-flags")` builder pattern (different name to avoid collision with existing module)
2. Calls `.Flags(defaultCfg.Flags)` to register CLI flags
3. Provides Config via provider that:
   - Starts with defaultCfg (which has flags bound)
   - Resolves `*gaz.ProviderValues` and unmarshals "health" namespace
   - Calls `cfg.Validate()` before returning
4. Calls the existing `Module(c)` to register ShutdownCheck, Manager, ManagementServer

Import `github.com/petabytecl/gaz` at the top.

Pattern follows server/http/module.go exactly but reuses existing Module() function for component registration.
  </action>
  <verify>go build ./health/...</verify>
  <done>NewModuleWithFlags() exists and returns gaz.Module with CLI flag support</done>
</task>

<task type="auto">
  <name>Task 3: Add tests for new flag-based module</name>
  <files>health/module_test.go</files>
  <action>
Add test cases for NewModuleWithFlags():

1. `TestNewModuleWithFlags` - verifies module creation and default values:
   - Create gaz.App with NewModuleWithFlags()
   - Verify module registers successfully
   - Verify Config resolves with default port 9090

2. `TestConfig_Flags` - verifies Flags() method:
   - Create pflag.FlagSet
   - Call cfg.Flags(fs)
   - Verify flags are registered: --health-port, --health-liveness-path, --health-readiness-path, --health-startup-path

3. `TestConfig_Validate` - verifies Validate() method:
   - Test valid config passes
   - Test port 0 fails
   - Test port > 65535 fails

4. `TestConfig_SetDefaults` - verifies SetDefaults() method:
   - Create zero Config
   - Call SetDefaults()
   - Verify all fields have expected defaults

Import `github.com/spf13/pflag` for flag tests.
  </action>
  <verify>go test -race -v ./health/... -run "TestNewModuleWithFlags|TestConfig_Flags|TestConfig_Validate|TestConfig_SetDefaults"</verify>
  <done>All new tests pass, covering Namespace, Flags, SetDefaults, Validate, and NewModuleWithFlags</done>
</task>

</tasks>

<verification>
```bash
go build ./health/...
go test -race -v ./health/...
make lint
```
</verification>

<success_criteria>
- health.Config has Namespace(), Flags(), SetDefaults(), Validate() methods
- NewModuleWithFlags() creates a gaz.Module with CLI flag support
- `--health-port` flag configures the health server port
- All existing tests continue to pass
- New tests cover the flag functionality
- Linter passes
</success_criteria>

<output>
After completion, create `.planning/quick/008-add-flags-to-the-health-server-to-get-th/008-SUMMARY.md`
</output>
