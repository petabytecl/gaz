---
phase: 12-di
plan: 01
subsystem: di
tags: [dependency-injection, container, generics, go]

# Dependency graph
requires:
  - phase: 11
    provides: Clean codebase with For[T]() as sole registration API
provides:
  - Standalone di package with Container, For[T](), Resolve[T]()
  - MustResolve[T]() for test setup
  - NewTestContainer() for clarity in tests
  - ServiceWrapper interface for App integration
  - Lifecycle engine for startup/shutdown ordering
affects: [12-02, 12-03, 12-04, 13-config]

# Tech tracking
tech-stack:
  added: []
  patterns: [standalone-subpackage, exported-interface-for-integration]

key-files:
  created:
    - di/container.go
    - di/registration.go
    - di/resolution.go
    - di/service.go
    - di/types.go
    - di/inject.go
    - di/errors.go
    - di/options.go
    - di/lifecycle.go
    - di/lifecycle_engine.go
    - di/doc.go
    - di/testing.go
  modified: []

key-decisions:
  - "Renamed NewContainer() to New() for idiomatic Go constructor"
  - "Exported ServiceWrapper interface for gaz.App integration"
  - "Added ForEachService() and GetService() accessors for lifecycle management"
  - "Error prefix changed from 'gaz:' to 'di:' for di package errors"

patterns-established:
  - "di.New() pattern for standalone container creation"
  - "MustResolve[T]() for panic-on-failure in tests/init"

# Metrics
duration: 5min
completed: 2026-01-28
---

# Phase 12 Plan 01: Create di Package Core Summary

**Standalone di package with Container, For[T](), Resolve[T](), MustResolve[T](), and 12 supporting files**

## Performance

- **Duration:** 5 min
- **Started:** 2026-01-28T04:09:44Z
- **Completed:** 2026-01-28T04:14:59Z
- **Tasks:** 3
- **Files created:** 12

## Accomplishments

- Created complete `gaz/di` subpackage that works standalone without gaz App
- Exported ServiceWrapper interface enabling App lifecycle integration
- Added MustResolve[T]() and NewTestContainer() as new ergonomic APIs
- Zero import cycles with parent gaz package

## Task Commits

Each task was committed atomically:

1. **Task 1: Create di package core files** - `c364f6e` (feat)
2. **Task 2: Create di package support files** - `6208e3c` (feat)
3. **Task 3: Add MustResolve and test helpers** - `c7562c2` (feat)

## Files Created

- `di/container.go` - Container type with New(), Build(), ForEachService(), GetService(), resolution chain management
- `di/registration.go` - For[T]() fluent registration builder
- `di/resolution.go` - Resolve[T]() and MustResolve[T]() functions
- `di/service.go` - ServiceWrapper interface and all wrapper implementations
- `di/types.go` - TypeName[T]() for consistent type naming
- `di/inject.go` - Struct field injection with gaz:"inject" tags
- `di/errors.go` - DI-specific errors with di: prefix
- `di/options.go` - ResolveOption and Named() option
- `di/lifecycle.go` - Starter, Stopper interfaces and HookConfig
- `di/lifecycle_engine.go` - ComputeStartupOrder/ComputeShutdownOrder
- `di/doc.go` - Package documentation with examples
- `di/testing.go` - NewTestContainer() helper

## Decisions Made

1. **Renamed NewContainer() → New()** - Follows idiomatic Go constructor naming (bytes.NewBuffer, http.NewRequest)
2. **Exported ServiceWrapper interface** - Required for gaz.App to iterate services during lifecycle management
3. **Added ForEachService() and GetService() accessors** - Container.services field is private; these methods provide controlled access for App
4. **Changed error prefix to di:** - Package-specific errors should reflect their origin package

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- di package compiles as standalone package ✓
- ServiceWrapper interface ready for App integration ✓
- Ready for 12-02-PLAN.md: Add introspection APIs and backward compatibility wrappers

---
*Phase: 12-di*
*Completed: 2026-01-28*
