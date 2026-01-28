package gaz

import (
	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
	"github.com/spf13/pflag"
)

// ConfigManager handles configuration loading, binding, and validation.
// It wraps config.Manager and provides backward compatibility with the original API.
// Deprecated: For new code, use config.Manager directly from github.com/petabytecl/gaz/config.
type ConfigManager struct {
	mgr    *config.Manager
	target any
}

// NewConfigManager creates a new ConfigManager.
// Deprecated: Use config.New or config.NewWithBackend directly.
func NewConfigManager(target any, opts ...config.Option) *ConfigManager {
	// Create config.Manager with viper backend
	configOpts := make([]config.Option, 0, len(opts)+1)
	configOpts = append(configOpts, config.WithBackend(cfgviper.New()))
	configOpts = append(configOpts, opts...)

	return &ConfigManager{
		mgr:    config.New(configOpts...),
		target: target,
	}
}

// Load reads configuration from files and environment variables,
// then unmarshals into the target struct.
func (cm *ConfigManager) Load() error {
	if cm.target == nil {
		return nil
	}
	return cm.mgr.LoadInto(cm.target)
}

// BindFlags binds command line flags to the configuration.
func (cm *ConfigManager) BindFlags(fs *pflag.FlagSet) error {
	return cm.mgr.BindFlags(fs)
}

// RegisterProviderFlags registers provider config flags with defaults and env binding.
func (cm *ConfigManager) RegisterProviderFlags(namespace string, flags []ConfigFlag) error {
	// Convert gaz.ConfigFlag to config.ConfigFlag
	cfgFlags := make([]config.ConfigFlag, len(flags))
	for i, f := range flags {
		cfgFlags[i] = config.ConfigFlag{
			Key:      f.Key,
			Default:  f.Default,
			Required: f.Required,
		}
	}
	return cm.mgr.RegisterProviderFlags(namespace, cfgFlags)
}

// ValidateProviderFlags validates that required provider config flags are set.
func (cm *ConfigManager) ValidateProviderFlags(namespace string, flags []ConfigFlag) []error {
	// Convert gaz.ConfigFlag to config.ConfigFlag
	cfgFlags := make([]config.ConfigFlag, len(flags))
	for i, f := range flags {
		cfgFlags[i] = config.ConfigFlag{
			Key:      f.Key,
			Default:  f.Default,
			Required: f.Required,
		}
	}
	return cm.mgr.ValidateProviderFlags(namespace, cfgFlags)
}

// Backend returns the underlying config.Backend.
// Used internally for ProviderValues.
func (cm *ConfigManager) Backend() config.Backend {
	return cm.mgr.Backend()
}
