package gaz

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
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

	flags := a.cobraCmd.Flags()

	// Only apply if config module registered --config flag
	configFlag := flags.Lookup("config")
	if configFlag == nil {
		return nil
	}

	var opts []config.Option
	opts = append(opts, config.WithBackend(cfgviper.New()))

	// --config flag: explicit config file path
	configPath := configFlag.Value.String()
	if configPath != "" {
		// Explicit config file - validate exists
		if _, statErr := os.Stat(configPath); statErr != nil {
			return fmt.Errorf("config: file not found: %s", configPath)
		}
		opts = append(opts, config.WithConfigFile(configPath))
	} else {
		// Auto-search: cwd first, then XDG config dir
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			if home, homeErr := os.UserHomeDir(); homeErr == nil {
				xdgConfig = filepath.Join(home, ".config")
			}
		}
		searchPaths := []string{"."}
		if xdgConfig != "" {
			appName := a.cobraCmd.Root().Name()
			if appName != "" {
				searchPaths = append(searchPaths, filepath.Join(xdgConfig, appName))
			}
		}
		opts = append(opts, config.WithSearchPaths(searchPaths...))
	}

	// --env-prefix flag
	if envPrefixFlag := flags.Lookup("env-prefix"); envPrefixFlag != nil {
		envPrefix := envPrefixFlag.Value.String()
		if envPrefix != "" {
			opts = append(opts, config.WithEnvPrefix(envPrefix))
		}
	}

	// --config-strict flag
	if strictFlag := flags.Lookup("config-strict"); strictFlag != nil {
		if strictFlag.Value.String() == "true" {
			a.strictConfig = true
		} else if strictFlag.Value.String() == "false" {
			a.strictConfig = false
		}
	}

	// Recreate config manager with collected options
	a.configMgr = config.New(opts...)

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
