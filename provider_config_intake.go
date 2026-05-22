package gaz

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz/config"
)

// configProviderType is cached for efficient interface checks.
//
//nolint:gochecknoglobals // Package-level for reflect type caching.
var configProviderType = reflect.TypeOf((*ConfigProvider)(nil)).Elem()

type configManagerProvider func() *config.Manager

// providerConfigEntry stores config information from a ConfigProvider.
type providerConfigEntry struct {
	providerName string
	namespace    string
	flags        []ConfigFlag
}

// providerConfigIntake owns ConfigProvider declaration intake and projection.
type providerConfigIntake struct {
	container     *Container
	configManager configManagerProvider
	register      func(any) error

	entries                  []providerConfigEntry
	providerValuesRegistered bool
	providerConfigsCollected bool
}

func newProviderConfigIntake(
	container *Container,
	configManager configManagerProvider,
	register func(any) error,
) *providerConfigIntake {
	return &providerConfigIntake{
		container:     container,
		configManager: configManager,
		register:      register,
	}
}

// registerProviderValues registers ProviderValues before ConfigProvider services resolve.
func (intake *providerConfigIntake) registerProviderValues() error {
	if intake.providerValuesRegistered {
		return nil
	}

	manager := intake.manager()
	if manager == nil {
		return nil
	}

	values := &ProviderValues{backend: manager.Backend()}
	if err := intake.register(values); err != nil {
		return fmt.Errorf("register ProviderValues: %w", err)
	}

	intake.providerValuesRegistered = true
	return nil
}

// collect discovers ConfigProvider services, registers their defaults/env bindings,
// and validates required values.
func (intake *providerConfigIntake) collect() error {
	if intake.providerConfigsCollected {
		return nil
	}

	serviceNames := intake.container.List()
	entries, err := intake.collectEntries(serviceNames)
	if err != nil {
		return err
	}

	intake.entries = entries
	intake.providerConfigsCollected = true
	return intake.registerProviderFlags()
}

func (intake *providerConfigIntake) collectEntries(serviceNames []string) ([]providerConfigEntry, error) {
	keyOwners := make(map[string]string, len(serviceNames))
	entries := make([]providerConfigEntry, 0, len(serviceNames))
	collisionErrors := make([]error, 0, len(serviceNames))

	for _, serviceName := range serviceNames {
		entry, ok := intake.providerEntryForService(serviceName)
		if !ok {
			continue
		}

		entries = append(entries, entry)
		collisionErrors = append(collisionErrors, findProviderConfigKeyCollisions(entry, keyOwners)...)
	}

	return entries, errors.Join(collisionErrors...)
}

func (intake *providerConfigIntake) providerEntryForService(serviceName string) (providerConfigEntry, bool) {
	wrapper, exists := intake.container.GetService(serviceName)
	if !exists || wrapper.IsTransient() || !serviceTypeImplementsConfigProvider(wrapper.ServiceType()) {
		return providerConfigEntry{}, false
	}

	instance, err := intake.container.ResolveByName(serviceName, nil)
	if err != nil {
		return providerConfigEntry{}, false
	}

	provider, ok := instance.(ConfigProvider)
	if !ok {
		return providerConfigEntry{}, false
	}

	return providerConfigEntry{
		providerName: serviceName,
		namespace:    provider.ConfigNamespace(),
		flags:        provider.ConfigFlags(),
	}, true
}

func serviceTypeImplementsConfigProvider(serviceType reflect.Type) bool {
	if serviceType == nil {
		return false
	}
	if serviceType.Implements(configProviderType) {
		return true
	}
	if serviceType.Kind() == reflect.Ptr {
		return false
	}

	return reflect.PointerTo(serviceType).Implements(configProviderType)
}

func findProviderConfigKeyCollisions(
	entry providerConfigEntry,
	keyOwners map[string]string,
) []error {
	collisions := make([]error, 0, len(entry.flags))
	for _, flag := range entry.flags {
		fullKey := providerConfigFullKey(entry.namespace, flag.Key)
		existingProvider, found := keyOwners[fullKey]
		if !found {
			keyOwners[fullKey] = entry.providerName
			continue
		}

		collisions = append(collisions, fmt.Errorf(
			"%w: key %q registered by both %q and %q",
			ErrConfigKeyCollision,
			fullKey,
			existingProvider,
			entry.providerName,
		))
	}

	return collisions
}

// registerProviderFlags registers collected provider flags with ConfigManager and validates.
func (intake *providerConfigIntake) registerProviderFlags() error {
	manager := intake.manager()
	if manager == nil {
		return nil
	}

	validationErrors := make([]error, 0, len(intake.entries))
	for _, entry := range intake.entries {
		cfgFlags := providerConfigFlags(entry.flags)
		if err := manager.RegisterProviderFlags(entry.namespace, cfgFlags); err != nil {
			return fmt.Errorf("registering provider flags for %s: %w", entry.namespace, err)
		}

		validationErrors = append(validationErrors, manager.ValidateProviderFlags(entry.namespace, cfgFlags)...)
	}

	return errors.Join(validationErrors...)
}

func providerConfigFlags(flags []ConfigFlag) []config.ConfigFlag {
	cfgFlags := make([]config.ConfigFlag, len(flags))
	for index, flag := range flags {
		cfgFlags[index] = config.ConfigFlag{
			Key:      flag.Key,
			Default:  flag.Default,
			Required: flag.Required,
		}
	}

	return cfgFlags
}

// registerPFlags registers typed pflags and binds them to the config backend.
func (intake *providerConfigIntake) registerPFlags(fs *pflag.FlagSet) error {
	manager := intake.manager()
	if manager == nil {
		return nil
	}

	flagBinder, ok := manager.Backend().(config.FlagBinder)
	if !ok {
		return nil
	}

	for _, entry := range intake.entries {
		if err := registerProviderEntryPFlags(fs, flagBinder, entry); err != nil {
			return err
		}
	}

	return nil
}

func registerProviderEntryPFlags(
	fs *pflag.FlagSet,
	flagBinder config.FlagBinder,
	entry providerConfigEntry,
) error {
	for _, flag := range entry.flags {
		fullKey := providerConfigFullKey(entry.namespace, flag.Key)
		flagName := configKeyToFlagName(fullKey)

		if fs.Lookup(flagName) != nil {
			continue
		}

		registerTypedFlag(fs, flag, flagName)
		if err := flagBinder.BindPFlag(fullKey, fs.Lookup(flagName)); err != nil {
			return fmt.Errorf("binding flag %s to key %s: %w", flagName, fullKey, err)
		}
	}

	return nil
}

func (intake *providerConfigIntake) manager() *config.Manager {
	if intake.configManager == nil {
		return nil
	}

	return intake.configManager()
}

func providerConfigFullKey(namespace, key string) string {
	return namespace + "." + key
}

// configKeyToFlagName transforms a config key to a POSIX flag name.
// Example: "server.host" -> "server-host".
func configKeyToFlagName(key string) string {
	return strings.ReplaceAll(key, ".", "-")
}

// registerTypedFlag registers a typed pflag based on ConfigFlag.Type.
func registerTypedFlag(fs *pflag.FlagSet, flag ConfigFlag, name string) {
	switch flag.Type {
	case ConfigFlagTypeString:
		def, _ := flag.Default.(string)
		fs.String(name, def, flag.Description)
	case ConfigFlagTypeInt:
		def, _ := flag.Default.(int)
		fs.Int(name, def, flag.Description)
	case ConfigFlagTypeBool:
		def, _ := flag.Default.(bool)
		fs.Bool(name, def, flag.Description)
	case ConfigFlagTypeDuration:
		def, _ := flag.Default.(time.Duration)
		fs.Duration(name, def, flag.Description)
	case ConfigFlagTypeFloat:
		def, _ := flag.Default.(float64)
		fs.Float64(name, def, flag.Description)
	default:
		def, _ := flag.Default.(string)
		fs.String(name, def, flag.Description)
	}
}
