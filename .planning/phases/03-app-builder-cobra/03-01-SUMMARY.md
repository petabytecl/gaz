---
phase: 03-app-builder-cobra
plan: 01
subsystem: app
tags: [fluent-api, dependency-injection, app-builder]
dependency-graph:
  requires: [phase-02, phase-02.1]
  provides: [gaz.New(), fluent-provider-methods, App.Build()]
  affects: [phase-03-02, phase-03-03]
tech-stack:
  added: []
  patterns: [fluent-builder, reflection-based-registration]
key-files:
  created: []
  modified: [app.go, container.go, service.go, errors.go, app_test.go]
decisions:
  - id: "03-01-01"
    choice: "Use reflection for provider type extraction"
    reasoning: "Enables clean fluent API without requiring explicit type parameters"
  - id: "03-01-02"
    choice: "Non-generic *Any service wrappers for fluent API"
    reasoning: "Go doesn't support generic methods, so reflection-based registration needs non-generic wrappers"
  - id: "03-01-03"
    choice: "Panic on late registration (after Build())"
    reasoning: "Follows uber-go/fx pattern; registration after Build() is a programming error"
  - id: "03-01-04"
    choice: "Build() is idempotent"
    reasoning: "Safe to call multiple times; returns nil after first success"
metrics:
  duration: 13 min
  completed: 2026-01-26
---

# Phase 03 Plan 01: App Builder Refactor Summary

Unified fluent API centered on `gaz.New()` with scope-specific provider methods and error aggregation.

## One-liner

Refactored entry point to `gaz.New()` returning `*App` with fluent `ProvideSingleton/ProvideTransient/ProvideEager/ProvideInstance` methods and `Build()` error aggregation.

## What Was Built

### Task 1: Rename New() to NewContainer()
- Renamed `New()` to `NewContainer()` in container.go
- Updated all test files (9 files) to use the new name
- Prepares namespace for `gaz.New()` returning `*App`

### Task 2: Fluent Provider API
- Added `gaz.New(opts ...Option)` returning `*App`
- Added `Option` type and `WithShutdownTimeout(d time.Duration)` option
- Added fluent provider methods:
  - `ProvideSingleton(provider)` - singleton scope (default)
  - `ProvideTransient(provider)` - new instance per resolution
  - `ProvideEager(provider)` - instantiated at Build() time
  - `ProvideInstance(instance)` - pre-built value
- Added non-generic `*Any` service wrappers for reflection-based registration
- Added sentinel errors: `ErrAlreadyBuilt`, `ErrInvalidProvider`
- Kept `NewApp()` for backward compatibility (deprecated)

### Task 3: Build() with Error Aggregation
- Added `App.Build()` that aggregates errors with `errors.Join()`
- Build() is idempotent (returns nil after first success)
- Provider methods panic on late registration (after Build())
- Added `Container()` accessor method
- Added comprehensive tests for the new API

## Commits

| Hash | Type | Description |
|------|------|-------------|
| 25d13be | refactor | Rename New() to NewContainer() |
| 280cc85 | feat | Add fluent provider API with gaz.New() |
| 6d27b38 | feat | Add Build() with error aggregation and tests |

## Key Files

| File | Changes |
|------|---------|
| app.go | New(), Option, fluent methods, Build() |
| container.go | NewContainer() rename |
| service.go | Non-generic *Any service wrappers |
| errors.go | ErrAlreadyBuilt, ErrInvalidProvider |
| app_test.go | 15 new tests for fluent API |

## Deviations from Plan

None - plan executed exactly as written.

## Technical Decisions

1. **Reflection for type extraction**: Provider functions are analyzed via `reflect.TypeOf()` to extract return types. This enables clean API without explicit type parameters.

2. **Non-generic wrappers**: Since Go doesn't support generic methods, the fluent API uses reflection and non-generic `*Any` service wrappers that store `func(*Container) (any, error)`.

3. **typeName() reuse**: The fluent API reuses the existing `typeName()` function from types.go to ensure consistent type naming between `For[T]()` and fluent registration.

4. **Panic on late registration**: Following uber-go/fx pattern, registering after `Build()` panics rather than returning an error. This is a programming error, not a runtime condition.

## Next Phase Readiness

**Ready for 03-02**: Module system implementation
- App struct is ready for modules field
- Container() accessor enables module-level registrations
- Build() error aggregation supports module validation errors
