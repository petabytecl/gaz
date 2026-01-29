---
phase: 19
plan: 02
subsystem: cli
tags: [cli, di, cobra, arguments]

requires:
  - phase: 18
    provides: "v2.0 cleanup"
provides:
  - "CommandArgs struct in DI"
  - "gaz.GetArgs() helper"
  - "CLI args injection in bootstrap"
affects:
  - "Phase 21 (Service Builder)"

tech-stack:
  added: []
  patterns:
    - "CLI Context Injection"

key-files:
  created:
    - "command.go"
  modified:
    - "cobra.go"
    - "cobra_test.go"

key-decisions:
  - "Inject CommandArgs as a struct pointer to allow access to both *cobra.Command and Args slice"
  - "Register CommandArgs during bootstrap to ensure availability before Build()"

metrics:
  duration: 5min
  completed: 2026-01-29
---

# Phase 19 Plan 02: CLI Arguments Integration Summary

**Enabled CLI argument injection via DI, allowing services to access positional args via `gaz.GetArgs(container)`.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-01-29T13:13:00Z
- **Completed:** 2026-01-29T13:18:36Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- Defined `CommandArgs` struct to hold `*cobra.Command` and `Args []string`
- Implemented `gaz.GetArgs(c)` helper for easy retrieval
- Updated `App.bootstrap` to inject `CommandArgs` into the container
- Verified with integration tests ensuring args are accessible in services and via helper

## Task Commits

1. **Task 1: Define CommandArgs type and helper** - `06fb5a3` (feat)
2. **Task 2: Inject CommandArgs in bootstrap** - `ff1009e` (feat)
3. **Task 3: Verify with integration test** - (will be committed with final metadata)

Note: Task 3 verification involved adding tests to `cobra_test.go`. I should commit that now before the final metadata commit.

## Files Created/Modified
- `command.go` - struct and helper definition
- `cobra.go` - bootstrap logic update
- `cobra_test.go` - integration tests

## Decisions Made
- Used `*di.Container` in `GetArgs` signature to match `gaz` package usage of DI.
- Registered `CommandArgs` in `bootstrap` so it is available to eager services during `Build()`.

## Deviations from Plan
None - plan executed exactly as written.

## Next Phase Readiness
- `CommandArgs` is ready for use in future Service Builder phase.
- No blockers.
