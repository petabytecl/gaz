---
phase: 04-config-system
plan: 02
files_modified:
  - cobra.go
  - app.go
  - app_integration_test.go
  - config_test.go
---

# Summary: Cobra Integration & Profiles

## Completed
- Updated `WithCobra` to register a hook that binds the executing command's flags to Viper.
- Updated `loadConfig` to support configuration profiles (e.g. `config.prod.yaml`) via `MergeInConfig`.
- Added `TestCobraConfigIntegration` to verify flags override config/env.
- Added `TestProfiles` to verify profile overlay logic.

## Key Decisions
- **Flag Binding:** Flags are bound using a hook added in `PersistentPreRunE`. This ensures we bind the flags of the *executing* command (including inherited persistent flags) to the Viper instance used by `loadConfig`.
- **Profiles:** Profiles use the dot notation naming convention (`name.profile.type`). Profile config is merged *on top of* the base config.
