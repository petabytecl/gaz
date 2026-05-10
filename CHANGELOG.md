# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking Changes

- **DI: `ReplaceService` now returns `error`** (A7) -
  Previously `ReplaceService(name, svc)` had no return value. Now returns `error` and rejects calls after `Build()` with `ErrAlreadyBuilt`.
  **Migration:** Add error handling: `if err := c.ReplaceService(name, svc); err != nil { ... }`

- **Vanguard: `WriteTimeout` defaults to 30s** (A11) -
  Previously defaulted to `0` (no timeout). Now defaults to `30s` with `AllowZeroWriteTimeout: false`.
  **Migration:** For streaming workloads, set `write_timeout: 0` and `allow_zero_write_timeout: true` explicitly.

- **Vanguard: CORS `AllowCredentials` defaults to `false`** (A12) -
  Previously defaulted to `true` in production CORS config.
  **Migration:** Set `allow_credentials: true` explicitly if your frontend sends credentials.

- **Config: Env vars use single underscore separator** (A15) -
  Previously `AutomaticEnv` used double underscore (`__`) while `RegisterProviderFlags` used single (`_`). Now both use single underscore.
  **Migration:** Change env vars from `APP_DB__HOST` to `APP_DB_HOST`.

- **Health: Management server binds `127.0.0.1`** (A16) -
  Previously bound to `0.0.0.0` (all interfaces).
  **Migration:** Set `bind_address: "0.0.0.0"` if external access to health endpoints is needed.

- **Worker: `Manager.Stop` takes `context.Context`** (A4) -
  Previously `Stop() error`. Now `Stop(ctx context.Context) error`.
  **Migration:** Internal API change -- pass shutdown context to `Stop`.

- **gRPC reflection disabled by default** (Rule 7) -
  Both `server/grpc` and `server/vanguard` modules now default `Reflection` to `false` (was `true`). Applications that relied on reflection being enabled by default must explicitly opt in via config (`reflection: true`) or code (`grpc.WithReflection(true)`). This change prevents unintended API schema exposure in production.

### Security

- **SHA-pinned GitHub Actions** - All CI workflow actions pinned to full commit SHA digests instead of mutable version tags. Dependabot configured for automated update PRs (Rule 8).

### Fixed

- Force-exit race: slow-but-successful shutdown no longer triggers exit(1) (A1)
- Cron scheduler OnStop respects shutdown context deadline (A2)
- EventBus OnStop propagates context for bounded subscription drain (A3)
- Worker manager Stop respects context deadline (A4)
- Cobra startup path divergence: `Start()` delegates to shared `startServices()` method (A5/Rule 2)
- EventBus lock-during-blocking-IO: `Publish` snapshots handlers under lock, delivers outside (A6/Rule 1)
- Cron scheduler lock-during-channel-ops: all four public methods release `runningMu` before channel ops (Rule 1)
- DI singleton init-under-lock: `lazySingleton`/`eagerSingleton` use `sync.Once` with cycle detection (Rule 1)
- DI eager services resolved in deterministic order (A8)
- DI failed Build locks container against further registration (A9)
- TLS warning logged when running Vanguard without TLS in non-dev mode (A10)
- Signal handler preserves second-SIGINT force-exit capability (A13)
- Backoff doc corrected: not concurrent-safe (A14)
- Health server adds read/write/idle timeouts (B10)
- Health ShowErrors gated behind config flag (A16)
- Duplicate Named registrations detected with ErrDuplicate (B3/B4)
- 97 time.Sleep test sites replaced with require.Eventually/channel sync (A17/Rule 6)
- DI inject field reflection memoized via sync.Map (B2)
- Worker/cron discovery uses ServiceType() before resolving (B5)
- Shutdown doStop uses cached service set (B6)
- gRPC server construction deferred to OnStart (B7)
- gRPC health adapter uses sync.Once for idempotent close (B8)
- Connect rate limit uses CodeResourceExhausted (B9)
- Empty health checks return StatusUnknown (B11)
- Logger SetDefault extracted to explicit SetGlobal function (B12)
- Logger file opens with 0o640, O_NOFOLLOW, path validation (B13)
- gaztest WithApp+WithModules returns error instead of panicking (B14)
- gaztest RequireStart/Stop use sync.Once for idempotency (B15)
- App WithConfig/WithConfigManager return error instead of panicking (B16)
- Critical-fail handler logs async-Stop outcome (B17)
- ResolveGroup returns error on type mismatch (B18)
- Cron wrapper fast-paths on cancelled appCtx (B19)
- Worker supervisor calls OnStop after failed OnStart (B20)
- Backoff ticker goroutine leak fixed (B21)

### Added

- Custom `lockblockingio` golangci-lint analyzer for Rule 1 enforcement (C1)
- `scripts/check-rules.sh` grep harness for Rules 6/7/8 (C2)
- `scripts/check-sentinel-errors.sh` for Rule 5 sentinel error usage check (C3)
- `make check` target for rule enforcement

## [2.0.0] - 2026-01-28

### BREAKING CHANGES

- **Removed `NewApp()` function** - Use `gaz.New()` instead
- **Removed `AppOption` type** - Use `gaz.Option` instead
- **Removed `App.ProvideSingleton()`** - Use `gaz.For[T](c).Provider(fn)` instead
- **Removed `App.ProvideTransient()`** - Use `gaz.For[T](c).Transient().Provider(fn)` instead
- **Removed `App.ProvideEager()`** - Use `gaz.For[T](c).Eager().Provider(fn)` instead
- **Removed `App.ProvideInstance()`** - Use `gaz.For[T](c).Instance(value)` instead
- **Removed reflection-based service wrappers** - Internal types removed

### Migration Guide

All service registration now uses the type-safe generic fluent API:

```go
// Before (v1.x)
app.ProvideSingleton(NewDatabase)
app.ProvideTransient(NewRequest)
app.ProvideEager(NewConnectionPool)
app.ProvideInstance(config)

// After (v2.0)
gaz.For[*Database](app.Container()).Provider(NewDatabase)
gaz.For[*Request](app.Container()).Transient().Provider(NewRequest)
gaz.For[*ConnectionPool](app.Container()).Eager().Provider(NewConnectionPool)
gaz.For[*Config](app.Container()).Instance(config)
```

### Benefits

- **Type safety**: Compile-time type checking for all registrations
- **Explicit**: Clear API shows exactly what scope and options are being used
- **Fluent**: Chain methods for clean, readable registration
- **Error handling**: Registration methods return errors for proper handling

### Improved

- All examples rewritten to showcase `For[T]()` pattern
- Documentation updated with new API examples
- Codebase lint and format pass completed

### Added

- **gaz/di package** - Standalone DI container that works without gaz.App. Exports `di.New()`, `di.For[T]()`, `di.Resolve[T]()`. Use for testing or library code.
- **gaz/config package** - Standalone configuration management with Backend interface. Exports `config.New()`, `config.Manager`, `config.Backend`. Viper implementation in `gaz/config/viper`.
- **gaz/worker package** - Background workers with lifecycle integration. Exports `worker.Worker` interface, `worker.Manager`, with automatic restart, circuit breaker, and graceful shutdown.
- **ConfigProvider interface** - Services can declare config requirements via `ConfigNamespace()` and `ConfigFlags()` methods
- **ProviderValues** - Type-safe access to config values within provider functions
- **WithConfigFile option** - Explicit config file path without search path lookup

## [1.1.0] - 2026-01-27

### Added

- Config validation at startup using go-playground/validator
- Shutdown hardening with timeout enforcement and blame logging
- Provider config registration for service-level configuration
- Comprehensive documentation and examples

### Changed

- Improved shutdown sequence with per-hook timeout tracking

## [1.0.0] - 2026-01-26

### Added

- Initial release with core DI functionality
- Type-safe generic container with `For[T]()` and `Resolve[T]()`
- Singleton, transient, and eager service scopes
- Lifecycle management with `Starter` and `Stopper` interfaces
- Graceful shutdown with configurable timeout
- Configuration loading from YAML/JSON/TOML files
- Environment variable binding with prefix support
- Struct validation with validate tags
- Health check subsystem with readiness/liveness probes
- Cobra CLI integration with `WithCobra()`
- Module organization with `app.Module()`
- slog integration for structured logging
