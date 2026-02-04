---
phase: quick
plan: 009
type: execute
wave: 1
depends_on: []
files_modified:
  - worker/module.go
  - worker/module_test.go
  - worker/module/module.go
  - worker/module/module_test.go
  - eventbus/module.go
  - eventbus/module_test.go
  - eventbus/module/module.go
  - eventbus/module/module_test.go
autonomous: true
---

<objective>
Refactor worker and eventbus modules to follow the health package module pattern.

Purpose: Align worker and eventbus with the established DI pattern where:
- `pkg/module.go` provides `Module(c *di.Container) error` for direct DI usage
- `pkg/module/module.go` provides `New() gaz.Module` for CLI/App integration

Output: Both packages follow the same module architecture as health package
</objective>

<context>
@.planning/STATE.md
@health/module.go (reference pattern for Module(c *di.Container) error)
@health/module/module.go (reference pattern for New() gaz.Module)
</context>

<tasks>

<task type="auto">
  <name>Task 1: Refactor worker module to follow health pattern</name>
  <files>
    - worker/module.go
    - worker/module_test.go
    - worker/module/module.go
    - worker/module/module_test.go
  </files>
  <action>
    1. **worker/module.go** - Replace current `NewModule() di.Module` with `Module(c *di.Container) error`:
       - Remove ModuleOption, moduleConfig, defaultModuleConfig
       - Create `Module(c *di.Container) error` function that:
         - Uses `di.For[*Manager](c).Provider(...)` to register *Manager
         - Provider resolves *slog.Logger (optional, fallback to slog.Default())
         - Creates Manager with `NewManager(logger)`
         - Returns nil on success
       - Add doc comment explaining this is for direct DI usage
       - Reference health/module.go for pattern

    2. **worker/module_test.go** - Update tests:
       - Remove tests for NewModule(), module.Name(), module.Register()
       - Add TestModule() that creates di.Container, calls Module(c), verifies *Manager resolves
       - Add test for optional logger (fallback to slog.Default())
       - Reference health/module_test.go for pattern

    3. **worker/module/module.go** - Create new file:
       - Package `module`
       - Import gaz, worker packages
       - Create `New() gaz.Module` function that:
         - Returns `gaz.NewModule("worker").Provide(worker.Module).Build()`
         - No flags needed (worker has no CLI-configurable options)
       - Add doc comment with usage example
       - Reference health/module/module.go for pattern

    4. **worker/module/module_test.go** - Create new file:
       - Test New() creates valid module
       - Test integration with gaz.App (Build succeeds, Manager resolves)
       - Reference health/module/module_test.go for pattern
  </action>
  <verify>
    - `go test -race ./worker/...` passes
    - `go test -race ./worker/module/...` passes
    - `go build ./worker/...` succeeds
    - `go build ./worker/module/...` succeeds
  </verify>
  <done>
    - worker/module.go exports `Module(c *di.Container) error`
    - worker/module/module.go exports `New() gaz.Module`
    - *Manager is registered and resolvable via both patterns
    - All tests pass
  </done>
</task>

<task type="auto">
  <name>Task 2: Refactor eventbus module to follow health pattern</name>
  <files>
    - eventbus/module.go
    - eventbus/module_test.go
    - eventbus/module/module.go
    - eventbus/module/module_test.go
  </files>
  <action>
    1. **eventbus/module.go** - Replace current `NewModule() di.Module` with `Module(c *di.Container) error`:
       - Remove ModuleOption, moduleConfig, defaultModuleConfig
       - Create `Module(c *di.Container) error` function that:
         - Uses `di.For[*EventBus](c).Provider(...)` to register *EventBus
         - Provider resolves *slog.Logger (optional, fallback to slog.Default())
         - Creates EventBus with `New(logger)`
         - Returns nil on success
       - Add doc comment explaining this is for direct DI usage
       - Reference health/module.go for pattern

    2. **eventbus/module_test.go** - Update tests:
       - Remove tests for NewModule(), module.Name(), module.Register()
       - Add TestModule() that creates di.Container, calls Module(c), verifies *EventBus resolves
       - Add test for optional logger (fallback to slog.Default())
       - Reference health/module_test.go for pattern

    3. **eventbus/module/module.go** - Create new file:
       - Package `module`
       - Import gaz, eventbus packages
       - Create `New() gaz.Module` function that:
         - Returns `gaz.NewModule("eventbus").Provide(eventbus.Module).Build()`
         - No flags needed (eventbus has no CLI-configurable options)
       - Add doc comment with usage example
       - Reference health/module/module.go for pattern

    4. **eventbus/module/module_test.go** - Create new file:
       - Test New() creates valid module
       - Test integration with gaz.App (Build succeeds, EventBus resolves)
       - Reference health/module/module_test.go for pattern
  </action>
  <verify>
    - `go test -race ./eventbus/...` passes
    - `go test -race ./eventbus/module/...` passes
    - `go build ./eventbus/...` succeeds
    - `go build ./eventbus/module/...` succeeds
  </verify>
  <done>
    - eventbus/module.go exports `Module(c *di.Container) error`
    - eventbus/module/module.go exports `New() gaz.Module`
    - *EventBus is registered and resolvable via both patterns
    - All tests pass
  </done>
</task>

<task type="auto">
  <name>Task 3: Run linter and fix any issues</name>
  <files>
    - worker/module.go
    - worker/module/module.go
    - eventbus/module.go
    - eventbus/module/module.go
  </files>
  <action>
    1. Run `make lint` to check for linter issues
    2. Fix any issues found:
       - ireturn: Add `//nolint:ireturn` to New() functions if needed (gaz.Module interface return)
       - goimports: Ensure import groups are correct (stdlib, external, local)
       - godot: Ensure doc comments end with periods
    3. Run `make fmt` to auto-fix formatting
    4. Run full test suite: `make test`
  </action>
  <verify>
    - `make lint` passes with no errors
    - `make test` passes
    - `make cover` shows >90% coverage still maintained
  </verify>
  <done>
    - All linter checks pass
    - All tests pass
    - Coverage threshold maintained
  </done>
</task>

</tasks>

<verification>
- `make lint` passes
- `make test` passes
- `make cover` shows >90% coverage
- Both worker and eventbus follow the same pattern as health
</verification>

<success_criteria>
- worker/module.go exports `Module(c *di.Container) error`
- worker/module/module.go exports `New() gaz.Module`
- eventbus/module.go exports `Module(c *di.Container) error`
- eventbus/module/module.go exports `New() gaz.Module`
- All tests pass, linter passes, coverage >90%
</success_criteria>

<output>
After completion, create `.planning/quick/009-refactor-worker-eventbus-module-pattern/009-SUMMARY.md`
</output>
