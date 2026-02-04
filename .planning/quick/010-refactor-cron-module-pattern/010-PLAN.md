---
quick_task: "010"
type: execute
autonomous: true
files_modified:
  - cron/module.go
  - cron/module_test.go
  - cron/module/module.go
  - cron/module/module_test.go
---

<objective>
Refactor cron package to follow the established module pattern from worker and eventbus packages.

Purpose: Consistency across all gaz infrastructure packages. The cron module should register *Scheduler via DI, matching the pattern where `pkg/module.go` provides `Module(c *di.Container) error` and `pkg/module/module.go` provides `New() gaz.Module`.

Output: cron.Module() registers *Scheduler, cron/module.New() wraps it for gaz.App integration.
</objective>

<execution_context>
@~/.config/opencode/get-shit-done/workflows/execute-plan.md
@~/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@AGENTS.md
@worker/module.go (reference pattern)
@worker/module/module.go (reference pattern)
@cron/scheduler.go (NewScheduler signature)
@cron/wrapper.go (Resolver interface)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Refactor cron/module.go to register *Scheduler</name>
  <files>cron/module.go, cron/module_test.go</files>
  <action>
  Replace the current `NewModule(opts ...ModuleOption) di.Module` with `Module(c *di.Container) error`.
  
  Implementation:
  1. Remove `ModuleOption`, `moduleConfig`, `defaultModuleConfig()`, and `NewModule()` 
  2. Add new `Module(c *di.Container) error` function that:
     - Uses `di.For[*Scheduler](c).Provider(...)` to register the Scheduler
     - Provider resolves optional logger (`di.Resolve[*slog.Logger](c)`) with fallback to `slog.Default()`
     - Provider passes `c` as Resolver (di.Container implements cron.Resolver interface via ResolveByName)
     - Provider uses `context.Background()` for appCtx (standalone usage without gaz.App)
     - Provider returns `NewScheduler(c, context.Background(), logger), nil`
  3. Add proper error wrapping: `fmt.Errorf("register scheduler: %w", err)`
  
  Reference pattern from worker/module.go:
  ```go
  func Module(c *di.Container) error {
      if err := di.For[*Scheduler](c).Provider(func(c *di.Container) (*Scheduler, error) {
          logger := slog.Default()
          if l, err := di.Resolve[*slog.Logger](c); err == nil {
              logger = l
          }
          return NewScheduler(c, context.Background(), logger), nil
      }); err != nil {
          return fmt.Errorf("register scheduler: %w", err)
      }
      return nil
  }
  ```

  Update tests (cron/module_test.go):
  - Change test from `NewModule()` returning `di.Module` to `Module(c)` returning error
  - Test that *Scheduler resolves after calling Module(c)
  - Test logger fallback to slog.Default() when not registered
  - Keep test patterns consistent with worker/module_test.go
  </action>
  <verify>
  `go test -race ./cron/...` passes
  `go build ./cron/...` compiles
  </verify>
  <done>
  - cron.Module(c) registers *Scheduler via DI
  - *Scheduler resolves correctly from container
  - Logger fallback works when logger not registered
  - Tests pass
  </done>
</task>

<task type="auto">
  <name>Task 2: Create cron/module subpackage</name>
  <files>cron/module/module.go, cron/module/module_test.go</files>
  <action>
  Create cron/module/module.go with `New() gaz.Module`:
  
  ```go
  // Package module provides the gaz.Module for cron integration.
  package module
  
  import (
      "github.com/petabytecl/gaz"
      "github.com/petabytecl/gaz/cron"
  )
  
  // New creates a cron module that provides cron.Scheduler.
  // This module registers the cron infrastructure for scheduling jobs.
  //
  // Usage:
  //
  //     import cronmod "github.com/petabytecl/gaz/cron/module"
  //
  //     app := gaz.New(gaz.WithCobra(rootCmd))
  //     app.Use(cronmod.New())
  //
  // The module provides:
  //   - *cron.Scheduler for scheduling cron jobs
  //
  //nolint:ireturn // Module is the expected return type for gaz modules
  func New() gaz.Module {
      return gaz.NewModule("cron").
          Provide(cron.Module).
          Build()
  }
  ```
  
  Create cron/module/module_test.go with tests:
  - TestNew creates valid module
  - TestNew integrates with gaz.App and *Scheduler resolves
  
  Reference pattern from worker/module/module_test.go.
  </action>
  <verify>
  `go test -race ./cron/module/...` passes
  `go build ./cron/module/...` compiles
  </verify>
  <done>
  - cron/module/module.go exists with New() function
  - cron/module/module_test.go tests integration with gaz.App
  - *Scheduler resolves via gaz.Resolve[*cron.Scheduler](app.Container())
  </done>
</task>

<task type="auto">
  <name>Task 3: Run full test suite and lint</name>
  <files>-</files>
  <action>
  Run full verification:
  1. `make test` - all tests pass
  2. `make lint` - no lint errors
  3. `make cover` - coverage maintained
  
  Fix any issues found.
  </action>
  <verify>
  `make test && make lint` passes
  </verify>
  <done>
  All tests pass, no lint errors, refactor complete
  </done>
</task>

</tasks>

<verification>
- `make test` - all tests pass
- `make lint` - no lint errors
- cron.Module(c) registers *Scheduler
- cron/module.New() returns gaz.Module
- Pattern matches worker and eventbus packages
</verification>

<success_criteria>
1. cron/module.go exports `Module(c *di.Container) error` (not `NewModule() di.Module`)
2. Module registers *Scheduler via `di.For[*Scheduler](c).Provider(...)`
3. cron/module/module.go exports `New() gaz.Module`
4. All tests pass including new tests
5. No lint errors
</success_criteria>

<output>
After completion, create `.planning/quick/010-refactor-cron-module-pattern/010-SUMMARY.md`
</output>
