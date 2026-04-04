# Summary: Plan 01-02 — DI Container Safety Guards

## Result
**Status:** Complete
**Duration:** ~15 minutes

## What Changed
- `di/container.go`: Register() now returns error with ErrAlreadyBuilt guard after Build(). Added MustRegister() helper. ReplaceService kept as void (supports test mocking).
- `di/registration.go`: Provider() and Instance() propagate Register() errors.
- `di/resolution.go`: ResolveAll[T] uses checked type assertion (no panic).
- `di/service.go`: instanceServiceAny.ServiceType() handles nil values.
- `di/container.go`: ResolveAllByType guards against nil ServiceType.
- `app_build.go`: registerInstance wraps Register error.
- `gaztest/builder.go`: Handles ReplaceService correctly.
- `cobra_test.go`: Updated to use Replace() for post-Build registration pattern.
- `di/container_test.go`: 5 new tests (RegisterAfterBuild, ReplaceAfterBuild, RegisterBeforeBuild, NilServiceType, plus Replace succeeds).
- `di/resolution_test.go`: 1 new test (ResolveAll checked assertion).

## Acceptance Criteria
- [x] AC-1: Register after Build returns ErrAlreadyBuilt
- [x] AC-2: ResolveAll uses checked type assertion
- [x] AC-3: Nil ServiceType does not panic
- [x] AC-4: All callers compile, make test + make lint pass

## Deviations
- ReplaceService remains void (not error-returning) to preserve gaztest Replace() pattern for test mocking. This is intentional — Replace() is explicitly opt-in.
- Cobra test updated to use Replace() for post-Build registration. This surfaces the Cobra/Run divergence (Playbook 02) — the test was exercising a pattern that should be handled by unified startup.

## Decisions
- ReplaceService is exempt from built guard (supports test mocking use case)

---
*Completed: 2026-04-03*
