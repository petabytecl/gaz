package gaz

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

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
	if a.providerValuesRegistered {
		return nil // Already registered
	}
	if a.configMgr == nil {
		return nil
	}
	pv := &ProviderValues{backend: a.configMgr.Backend()}
	if err := a.registerInstance(pv); err != nil {
		return err
	}
	a.providerValuesRegistered = true
	return nil
}

// getSortedServiceNames returns service names in sorted order for deterministic iteration.
func (a *App) getSortedServiceNames() []string {
	return a.container.List()
}

// collectProviderConfigs iterates registered services, collects config from ConfigProvider
// implementers, detects key collisions, registers provider flags with ConfigManager,
// validates required fields, and registers ProviderValues.
// This method is idempotent - subsequent calls return nil after first collection.
func (a *App) collectProviderConfigs() error {
	if a.providerConfigsCollected {
		return nil // Already collected
	}
	keyOwners := make(map[string]string)
	var collisionErrors []error

	// Iterate in sorted order for deterministic dependency graph recording
	for _, typeName := range a.getSortedServiceNames() {
		wrapper, exists := a.container.GetService(typeName)
		if !exists {
			continue
		}

		if wrapper.IsTransient() {
			continue
		}

		// Check if service type implements ConfigProvider BEFORE instantiation
		// This avoids side effects of instantiating non-ConfigProvider services
		serviceType := wrapper.ServiceType()
		if serviceType == nil {
			continue
		}

		// For pointer types, check both pointer and element type
		if !serviceType.Implements(configProviderType) {
			// Also check pointer-to-type in case methods are on *T
			if serviceType.Kind() != reflect.Ptr {
				ptrType := reflect.PointerTo(serviceType)
				if !ptrType.Implements(configProviderType) {
					continue
				}
			} else {
				continue
			}
		}

		// Only now instantiate - we know it implements ConfigProvider
		instance, err := a.container.ResolveByName(typeName, nil)
		if err != nil {
			continue // Skip services that fail to resolve
		}

		cp, ok := instance.(ConfigProvider)
		if !ok {
			// This shouldn't happen if type check above is correct, but be defensive
			continue
		}

		namespace := cp.ConfigNamespace()
		flags := cp.ConfigFlags()

		a.providerConfigs = append(a.providerConfigs, providerConfigEntry{
			providerName: typeName,
			namespace:    namespace,
			flags:        flags,
		})

		// Check for collisions
		for _, flag := range flags {
			fullKey := namespace + "." + flag.Key
			if existingProvider, found := keyOwners[fullKey]; found {
				collisionErrors = append(collisionErrors, fmt.Errorf(
					"%w: key %q registered by both %q and %q",
					ErrConfigKeyCollision, fullKey, existingProvider, typeName,
				))
			} else {
				keyOwners[fullKey] = typeName
			}
		}
	}

	if len(collisionErrors) > 0 {
		return errors.Join(collisionErrors...)
	}

	// Set flag BEFORE registerProviderFlags to avoid re-entry issues
	a.providerConfigsCollected = true
	return a.registerProviderFlags()
}

// registerProviderFlags registers collected provider flags with ConfigManager and validates.
// Note: ProviderValues is already registered by registerProviderValuesEarly().
func (a *App) registerProviderFlags() error {
	if a.configMgr == nil {
		return nil
	}

	var validationErrors []error
	for _, entry := range a.providerConfigs {
		// Convert gaz.ConfigFlag to config.ConfigFlag
		cfgFlags := make([]config.ConfigFlag, len(entry.flags))
		for i, f := range entry.flags {
			cfgFlags[i] = config.ConfigFlag{
				Key:      f.Key,
				Default:  f.Default,
				Required: f.Required,
			}
		}

		if err := a.configMgr.RegisterProviderFlags(entry.namespace, cfgFlags); err != nil {
			return fmt.Errorf("registering provider flags for %s: %w", entry.namespace, err)
		}
		errs := a.configMgr.ValidateProviderFlags(entry.namespace, cfgFlags)
		validationErrors = append(validationErrors, errs...)
	}

	if len(validationErrors) > 0 {
		return errors.Join(validationErrors...)
	}

	return nil
}
