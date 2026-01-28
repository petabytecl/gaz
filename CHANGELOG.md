# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
