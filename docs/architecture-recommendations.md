# Architecture Recommendations Backlog

Last updated: 2026-05-24

This document captures the remaining architecture recommendations for `gaz` after the merged lifecycle/config refactor series. It is meant as a resume point for future Codex sessions.

Use the vocabulary in `CONTEXT.md`:

- A **lifecycle plan** owns runtime policy: which services participate, which runtime participants are handed to App-managed managers, and what startup/shutdown order is used.
- **Configured module registration** owns the shared rule for module-owned config values.
- **Module identity** owns the stable name used for duplicate detection, auto-registration skip logic, and diagnostics.
- Architecture recommendations should deepen modules: put policy behind smaller interfaces, improve locality, and keep behavior testable through the public or private module seam.

## Current Baseline

Already completed:

- Lifecycle plan ownership was deepened.
- ConfigProvider declaration intake was separated from App orchestration.
- Configured module registration was centralized under `internal/configuredmodule`.
- Runtime participant classification moved into the lifecycle plan.
- Worker and cron participant registration now fails `App.Build()` when a participant cannot resolve, resolves to the wrong type, or cannot register.
- Health auto-registration moved behind a dedicated private module.
- The App build sequence is named and testable through private build phases.
- Config flag interpretation moved behind a private config flag policy.
- Runtime subsystem construction moved behind a private runtime-subsystems module.
- Lifecycle session orchestration moved behind a private lifecycle session.
- Feature module identities are exposed by the owning packages; config keeps a deliberately separate `config-flags` adapter identity.

Quality gate expectation for future PRs:

- `git diff --check`
- `make check`
- `make fmt-check`, or `go run golang.org/x/tools/cmd/goimports@latest -l .` when the local `goimports` shim is broken
- `make lint`
- `go test ./... -count=1`
- `make cover`
- Push, then monitor GitHub Actions and CodeQL for the pushed SHA until green.

## Priority 1: Extract Health Auto-Registration Policy

Recommendation strength: Strong

Status: Completed.

Files:

- `app_build.go`
- `health/config_provider.go`
- `health/module/module.go`
- `app_use.go`
- `tests/health_test.go`
- `health/module/module_test.go`

Problem:

`App.Build()` still owns health-specific feature policy. It detects `health.HealthConfigProvider`, registers `health.Config`, builds an inline health module, applies it, and mutates `a.modules`.

There is also a module identity mismatch worth fixing in the same slice: `health/module.New()` returns a gaz module named `health-flags`, while the App auto-registration path checks `a.modules["health"]`. That means explicit health-module usage and automatic `HealthConfigProvider` usage do not share one obvious ownership marker.

Solution:

Move the health auto-registration rule into a dedicated private module, for example `health_auto_registration.go`, with one small App-facing method. Keep the public behavior the same:

- If `configTarget` implements `health.HealthConfigProvider`, the health module should be registered automatically.
- If the health module was explicitly applied, auto-registration should be skipped cleanly.
- If health config or module application fails, `App.Build()` should return an actionable error.

Benefits:

- `App.Build()` becomes orchestration again, not a feature-policy host.
- Health behavior gains locality: config-provider detection, module identity, and duplicate-avoidance rules live together.
- Tests can target the health auto-registration policy directly instead of testing it only through the full App build path.

Suggested tests:

- `HealthConfigProvider` auto-registers `health.Config`, `*health.Manager`, `*health.ShutdownCheck`, and `*health.ManagementServer`.
- Explicit `health/module.New()` plus a `HealthConfigProvider` does not double-apply health providers.
- Auto-registration propagates `health.Config` registration failure with useful context.
- Auto-registration preserves current integration test behavior in `tests/health_test.go`.

Why next:

This is the smallest remaining high-leverage slice. It removes feature-specific policy from `App.Build()` and addresses a real naming mismatch without requiring a broader build pipeline redesign.

## Priority 2: Deepen the App Build Sequence

Recommendation strength: Strong, after Priority 1

Status: Completed.

Files:

- `app_build.go`
- `app_config.go`
- `provider_config_intake.go`
- `lifecycle_plan.go`
- `app_test.go`

Problem:

`App.Build()` still coordinates too many policies in one method:

- config loading
- ProviderValues registration
- logger initialization
- runtime subsystem initialization
- ConfigProvider intake
- health auto-registration
- container build
- lifecycle plan creation
- runtime participant registration

Some of these are already deeper modules, but the sequence itself is still implicit in the method body. When a future change needs to reorder one step, the caller has to understand the entire build pipeline.

Solution:

After health auto-registration is extracted, introduce a private build-sequence module that names the phases and keeps ordering constraints explicit. This does not need a public interface. The goal is a private seam that makes the build order testable and easier to audit.

Benefits:

- Better locality for build ordering rules.
- Easier tests for "phase X runs before phase Y" without full integration setup.
- `App.Build()` becomes a thin orchestration call that aggregates errors.

Suggested tests:

- ProviderValues register before ConfigProvider services are resolved.
- Logger initializes before runtime subsystems.
- Container build happens before lifecycle plan runtime participant registration.
- Build remains idempotent after success.

## Priority 3: Move Config Flag Application Policy Out Of App

Recommendation strength: Worth exploring

Status: Completed.

Files:

- `app_config.go`
- `config/module/module.go`
- `config/manager.go`
- `cobra_flags.go`
- `app_test.go`

Problem:

`applyConfigFlags()` lives in App and knows the config module flag names, config-file existence policy, XDG search-path behavior, env-prefix behavior, and strict-mode parsing. That makes the App core responsible for details that feel closer to configured module registration and config infrastructure.

Solution:

Create a private config flag policy module that interprets the Cobra flags and returns config manager options plus strict-mode settings. App should still decide when to call it, but not know the details of every config flag.

Benefits:

- Better locality for config/Cobra behavior.
- Easier focused tests for flag interpretation without building a full App.
- Less risk that future config module changes require editing App internals.

Suggested tests:

- Explicit `--config` path validates file existence and feeds `config.WithConfigFile`.
- Empty config path builds the current search path list.
- `--env-prefix` maps into config options.
- `--config-strict=false` disables strict config.

## Priority 4: Deepen Runtime Subsystem Initialization

Recommendation strength: Worth exploring

Status: Completed.

Files:

- `app_build.go`
- `app_shutdown.go`
- `lifecycle_plan.go`
- `worker/manager.go`
- `cron/scheduler.go`
- `eventbus/eventbus.go`

Problem:

`initializeSubsystems()` creates the worker manager, cron scheduler context, scheduler, event bus, critical-worker failure hook, and EventBus DI registration. This is runtime subsystem policy, not just App construction.

Solution:

Introduce a private runtime-subsystems module that owns the App-managed runtime participants:

- worker manager
- cron scheduler and cancellation context
- framework event bus
- critical worker failure callback

Do not change public APIs in the first slice. Keep App as the owner of execution, but move construction and registration policy behind one private module.

Benefits:

- Runtime subsystem lifecycle becomes easier to test without invoking the whole App.
- The lifecycle plan keeps owning participant policy, while runtime subsystem construction gains its own locality.
- Future worker/cron/eventbus changes have a clear place to land.

Suggested tests:

- EventBus is registered exactly once in DI.
- Scheduler context is cancelled on `Stop()`.
- Critical worker failure triggers App shutdown with the configured shutdown timeout.
- Nil logger fallback remains safe.

## Priority 5: Separate Lifecycle Execution From Signal Handling

Recommendation strength: Worth exploring

Status: Completed.

Files:

- `app_run.go`
- `app_shutdown.go`
- `cobra.go`
- `lifecycle_plan.go`
- `shutdown_test.go`
- `cobra_test.go`

Problem:

Startup, rollback, shutdown, signal waiting, double-SIGINT behavior, and Cobra bootstrap all live close together in App methods. Prior work already made startup paths share `startServices()`, so this is less urgent than the build and health work.

Solution:

After the build/runtime subsystem slices, consider a private lifecycle executor module. It would execute a lifecycle plan with logger, worker manager, shutdown timeout, and per-hook timeout supplied by App. Signal handling can remain in App unless it also becomes a source of friction.

Benefits:

- Startup and shutdown execution rules become a smaller test surface.
- Cobra and non-Cobra paths can share more behavior without duplicating run-state details.
- Rollback behavior becomes easier to reason about.

Suggested tests:

- Startup layer failure rolls back already-started services.
- Worker manager start failure rolls back services.
- Per-hook timeout records blame and continues.
- Global shutdown timeout still force-exits through `exitFunc`.

## Priority 6: Rationalize Feature Module Identity

Recommendation strength: Worth exploring

Status: Completed.

Files:

- `app_use.go`
- `module_builder.go`
- `health/module/module.go`
- `logger/module/module.go`
- `config/module.go`

Problem:

Feature modules did not have a clear naming convention for "feature is installed" versus "flags adapter is installed". Health originally exposed this most clearly: `health/module.New()` used the module name `health-flags`, while App auto-registration checked `health`.

Solution:

Define a convention for module identity. A feature module exposes one stable identity for duplicate detection and auto-registration skip logic. If flags are a separate adapter, encode that deliberately instead of relying on ad hoc string checks.

Benefits:

- Auto-registration rules can reliably detect explicit feature registration.
- Duplicate module behavior becomes less surprising.
- Future modules can follow one convention.

Suggested tests:

- Applying a module twice still reports duplicate module errors.
- Feature auto-registration skips when the equivalent explicit module is already present.
- Child module identity remains visible for duplicate detection.

## Priority 7: Decide Whether `config.NewModule()` Should Exist

Recommendation strength: Speculative

Status: Completed.

Files:

- `config/module.go`
- `config/module_test.go`
- `docs/configuration.md`
- `docs/concepts.md`

Problem:

`config.NewModule()` was a placeholder. The deletion test showed it was shallow: deleting it would remove little behavior, but the public API already implied that it had meaning.

Solution:

Document it as deprecated/no-op compatibility and stop presenting it as an advanced module.

Benefits:

- Removes a misleading public module.
- Prevents future architecture reviews from treating a placeholder as a real seam.
- Improves documentation honesty.

Suggested tests:

- Assert it remains harmless and update docs to be explicit.

## Current Architecture Review Status

The architecture backlog created from the earlier review is now exhausted. Priorities 1 through 7 are complete. Before selecting another implementation slice, run a fresh architecture review against current `main` and create a new short backlog from live friction instead of continuing from this historical queue.

## Suggested Resume Order

1. Create branch from updated `main`.
2. Run a fresh architecture review against current `main`.
3. Run local gates.
4. Push and monitor remote CI and CodeQL.
5. Address review comments and resolve conversations.
6. Rebase/pull `main` after merge.
7. Refresh this backlog before selecting the next slice.

Avoid inventing new work from this exhausted backlog. New work should come from current code evidence.
