---
phase: quick
plan: 009
subsystem: di-modules
tags:
  - worker
  - eventbus
  - refactor
  - module-pattern
dependency-graph:
  requires:
    - health module pattern reference
  provides:
    - worker/module subpackage
    - eventbus/module subpackage
    - consistent Module(c *di.Container) error pattern
  affects:
    - examples using worker/eventbus modules
tech-stack:
  patterns:
    - Module(c *di.Container) error for direct DI
    - New() gaz.Module for CLI/App integration
key-files:
  created:
    - worker/module/module.go
    - worker/module/module_test.go
    - eventbus/module/module.go
    - eventbus/module/module_test.go
  modified:
    - worker/module.go
    - worker/module_test.go
    - eventbus/module.go
    - eventbus/module_test.go
    - worker/example_test.go
    - eventbus/example_test.go
    - examples/background-workers/main.go
    - examples/microservice/main.go
decisions:
  - id: worker-eventbus-module-pattern
    date: 2026-02-04
    summary: Worker and eventbus follow same module pattern as health
    reasoning: Consistency across all DI modules improves discoverability
  - id: eventbus-skip-if-registered
    date: 2026-02-04
    summary: eventbus.Module skips registration if EventBus already exists
    reasoning: gaz.App auto-registers EventBus, module is for di.Container direct usage
metrics:
  duration: 13m
  completed: 2026-02-04
---

# Quick Task 009: Refactor Worker/EventBus Module Pattern Summary

**One-liner:** Aligned worker and eventbus modules with health package pattern (Module(c) error + module/New()).

## What Changed

### Worker Package

**worker/module.go** - Simplified to `Module(c *di.Container) error`:
- Removed `NewModule() di.Module`, `ModuleOption`, `moduleConfig`
- Added `Module(c *di.Container) error` that registers `*Manager`
- Logger is optional - uses `slog.Default()` fallback

**worker/module/module.go** - New subpackage for gaz.Module:
- Package `module` provides `New() gaz.Module`
- Returns `gaz.NewModule("worker").Provide(worker.Module).Build()`
- For CLI/App integration

### EventBus Package

**eventbus/module.go** - Simplified to `Module(c *di.Container) error`:
- Removed `NewModule() di.Module`, `ModuleOption`, `moduleConfig`
- Added `Module(c *di.Container) error` that registers `*EventBus`
- Logger is optional - uses `slog.Default()` fallback
- Skips registration if EventBus already exists (for gaz.App compatibility)

**eventbus/module/module.go** - New subpackage for gaz.Module:
- Package `module` provides `New() gaz.Module`
- Returns `gaz.NewModule("eventbus").Provide(eventbus.Module).Build()`
- For CLI/App integration

### Examples Updated

- **examples/background-workers**: Uses `workermod.New()`
- **examples/microservice**: Uses `workermod.New()`, removed redundant eventbus module (auto-registered by gaz.App)

## Commits

| Hash | Type | Description |
|------|------|-------------|
| f1c6792 | refactor | worker module to follow health pattern |
| 964975e | refactor | eventbus module to follow health pattern |
| b27297c | fix | wrap errors per wrapcheck linter |

## Pattern Reference

**For DI Container direct usage:**
```go
c := di.New()
worker.Module(c)    // registers *worker.Manager
eventbus.Module(c)  // registers *eventbus.EventBus
c.Build()
```

**For gaz.App CLI integration:**
```go
import workermod "github.com/petabytecl/gaz/worker/module"

app := gaz.New(gaz.WithCobra(rootCmd))
app.Use(workermod.New())
```

## Important Note

EventBus is auto-registered by `gaz.App.initializeSubsystems()`. Using `eventbusmod.New()` with gaz.App is redundant and not recommended. The eventbus/module subpackage exists for consistency but is primarily for di.Container direct usage scenarios.

## Verification

- `make lint` passes (0 issues)
- `make test` passes (all packages)
- Worker and eventbus follow same pattern as health

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] EventBus duplicate registration handling**
- **Found during:** Task 2
- **Issue:** Using `eventbusmod.New()` with gaz.App caused duplicate EventBus registration
- **Fix:** Added `di.Has[*EventBus](c)` check to skip if already registered
- **Files modified:** eventbus/module.go
- **Commit:** 964975e

**2. [Rule 1 - Bug] wrapcheck linter errors**
- **Found during:** Task 3
- **Issue:** `di.For[T](c).Provider()` errors not wrapped
- **Fix:** Wrapped with `fmt.Errorf("register X: %w", err)`
- **Files modified:** worker/module.go, eventbus/module.go
- **Commit:** b27297c
