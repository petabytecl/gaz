package gaz

import (
	"fmt"

	"github.com/petabytecl/gaz/config"
)

// loadConfig loads the configuration from all sources.
// This method is idempotent - subsequent calls return nil after first load.
func (a *App) loadConfig() error {
	if a.configLoaded {
		return nil // Already loaded
	}
	if a.configMgr == nil {
		return nil
	}

	// Apply config module flags if present (--config, --env-prefix, --config-strict)
	if err := a.applyConfigFlags(); err != nil {
		return err
	}

	// If a target struct is provided, load and unmarshal into it
	if a.configTarget != nil {
		if a.strictConfig {
			if err := a.configMgr.LoadIntoStrict(a.configTarget); err != nil {
				return fmt.Errorf("loading config (strict mode): %w", err)
			}
		} else {
			if err := a.configMgr.LoadInto(a.configTarget); err != nil {
				return fmt.Errorf("loading config into target: %w", err)
			}
		}
	} else {
		// Otherwise just load the config file (for ConfigProvider pattern)
		if err := a.configMgr.Load(); err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
	}
	a.configLoaded = true
	return nil
}

// applyConfigFlags reads --config, --env-prefix, --config-strict flags and
// recreates the config manager with appropriate options.
// This is called at the start of loadConfig() and only applies if the
// config module registered these flags.
func (a *App) applyConfigFlags() error {
	if a.cobraCmd == nil {
		return nil
	}

	result, err := interpretConfigFlags(a.cobraCmd.Flags(), a.cobraCmd.Root().Name())
	if err != nil {
		return err
	}

	if len(result.options) > 0 {
		a.configMgr = config.New(result.options...)
	}
	if result.strictMode != nil {
		a.strictConfig = *result.strictMode
	}

	return nil
}

// registerProviderValuesEarly registers ProviderValues as an instance
// immediately after config loading, BEFORE providers are instantiated.
// This allows providers to inject *ProviderValues as a dependency.
// This method is idempotent - subsequent calls return nil after first registration.
func (a *App) registerProviderValuesEarly() error {
	return a.providerConfigIntakeModule().registerProviderValues()
}

// collectProviderConfigs iterates registered services, collects config from ConfigProvider
// implementers, detects key collisions, registers provider flags with ConfigManager,
// and validates required fields. ProviderValues must be registered before this runs.
// This method is idempotent - subsequent calls return nil after first collection.
func (a *App) collectProviderConfigs() error {
	return a.providerConfigIntakeModule().collect()
}

func (a *App) providerConfigIntakeModule() *providerConfigIntake {
	if a.providerConfigIntake == nil {
		a.providerConfigIntake = newProviderConfigIntake(
			a.container,
			func() *config.Manager {
				return a.configMgr
			},
			a.registerInstance,
		)
	}

	return a.providerConfigIntake
}
