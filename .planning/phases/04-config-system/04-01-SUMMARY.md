---
phase: 04-config-system
plan: 01
files_modified:
  - go.mod
  - go.sum
  - config.go
  - app.go
  - config_test.go
---

# Summary: Core Config Infrastructure

## Completed
- Added `spf13/viper` dependency.
- Defined `ConfigOptions`, `Defaulter`, and `Validator` interfaces in `config.go`.
- Implemented `App.WithConfig` to register config target and options.
- Implemented `App.loadConfig` to load from files and environment variables.
- Added automatic environment variable binding for struct fields (reflection-based).
- Integrated `loadConfig` into `App.Build` (ensuring it runs before eager services).
- Verified with unit tests covering defaults, env vars, and validation.

## Key Decisions
- **Env Binding:** Implemented `bindStructEnv` to recursively bind struct fields to environment variables using `viper.BindEnv`. This allows `AutomaticEnv` to work even if keys are missing from the config file or defaults.
- **Loading Timing:** Config is loaded at the beginning of `Build()`. This ensures it's available for Eager services which are instantiated during `Build()`.
- **DI Registration:** The config target (pointer) is registered as a Singleton Instance immediately in `WithConfig`. Its *content* is populated later during `Build()`.
