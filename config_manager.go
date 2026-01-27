package gaz

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ConfigManager handles configuration loading, binding, and validation.
type ConfigManager struct {
	v           *viper.Viper
	target      any
	fileName    string
	fileType    string
	searchPaths []string
	envPrefix   string
	profileEnv  string
	defaults    map[string]any
}

// NewConfigManager creates a new ConfigManager.
func NewConfigManager(target any, opts ...ConfigOption) *ConfigManager {
	cm := &ConfigManager{
		v:           viper.New(),
		target:      target,
		fileName:    "config",
		fileType:    "yaml",
		searchPaths: []string{"."},
		defaults:    make(map[string]any),
	}

	for _, opt := range opts {
		opt(cm)
	}

	return cm
}

// Load reads configuration from files and environment variables.
func (cm *ConfigManager) Load() error {
	if cm.target == nil {
		return nil
	}

	cm.v.SetConfigName(cm.fileName)
	cm.v.SetConfigType(cm.fileType)
	for _, path := range cm.searchPaths {
		cm.v.AddConfigPath(path)
	}

	// Apply defaults
	for k, v := range cm.defaults {
		cm.v.SetDefault(k, v)
	}

	if cm.envPrefix != "" {
		cm.v.SetEnvPrefix(cm.envPrefix)
		cm.v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
		cm.v.AutomaticEnv()

		// Bind struct fields to env vars
		cm.bindStructEnv(cm.v, cm.target, "")
	}

	if err := cm.v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	// Load profile config
	if err := cm.loadProfileConfig(); err != nil {
		return err
	}

	if err := cm.v.Unmarshal(cm.target); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if d, ok := cm.target.(Defaulter); ok {
		d.Default()
	}

	// Validate struct tags (required, min, max, etc.)
	if err := validateConfigTags(cm.target); err != nil {
		return err // Already formatted with ErrConfigValidation
	}

	if val, ok := cm.target.(Validator); ok {
		if err := val.Validate(); err != nil {
			return fmt.Errorf("config validation failed: %w", err)
		}
	}

	return nil
}

func (cm *ConfigManager) loadProfileConfig() error {
	if cm.profileEnv == "" {
		return nil
	}

	profile := os.Getenv(cm.profileEnv)
	if profile == "" {
		return nil
	}

	cm.v.SetConfigName(cm.fileName + "." + profile)
	if err := cm.v.MergeInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("failed to merge profile config: %w", err)
		}
	}
	return nil
}

// BindFlags binds command line flags to the configuration.
func (cm *ConfigManager) BindFlags(fs *pflag.FlagSet) error {
	if err := cm.v.BindPFlags(fs); err != nil {
		return fmt.Errorf("failed to bind pflags: %w", err)
	}
	return nil
}

// Viper returns the underlying viper instance.
// Used internally for ProviderValues.
func (cm *ConfigManager) Viper() *viper.Viper {
	return cm.v
}

// RegisterProviderFlags registers provider config flags with defaults and env binding.
// For each flag, it:
// 1. Sets the default value if specified.
// 2. Binds the key to an environment variable (e.g., redis.host -> REDIS_HOST).
func (cm *ConfigManager) RegisterProviderFlags(namespace string, flags []ConfigFlag) error {
	for _, flag := range flags {
		fullKey := namespace + "." + flag.Key

		// Set default if provided
		if flag.Default != nil {
			cm.v.SetDefault(fullKey, flag.Default)
		}

		// Bind env var with explicit name (redis.host -> REDIS_HOST)
		envKey := strings.ToUpper(strings.ReplaceAll(fullKey, ".", "_"))
		if err := cm.v.BindEnv(fullKey, envKey); err != nil {
			return fmt.Errorf("failed to bind env var %s for key %s: %w", envKey, fullKey, err)
		}
	}
	return nil
}

// ValidateProviderFlags validates that required provider config flags are set.
// Returns a slice of errors for all missing required fields (not fail-fast).
func (cm *ConfigManager) ValidateProviderFlags(namespace string, flags []ConfigFlag) []error {
	var errs []error
	for _, flag := range flags {
		if !flag.Required {
			continue
		}

		fullKey := namespace + "." + flag.Key

		// Check if value is set and non-zero
		if !cm.v.IsSet(fullKey) {
			errs = append(errs, fmt.Errorf(
				"provider %q: required config key %q is not set",
				namespace, fullKey,
			))
		}
	}
	return errs
}

// bindStructEnv recursively binds struct fields to environment variables.
func (cm *ConfigManager) bindStructEnv(v *viper.Viper, target any, prefix string) {
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	t := val.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		name := field.Name
		// Use mapstructure tag if present
		if tag, ok := field.Tag.Lookup("mapstructure"); ok {
			parts := strings.Split(tag, ",")
			if len(parts) > 0 && parts[0] != "" {
				name = parts[0]
			}
		}

		key := name
		if prefix != "" {
			key = prefix + "." + name
		}

		if field.Type.Kind() == reflect.Struct {
			// Recursive bind for nested structs
			// We pass a zero value of the struct type for type inspection
			cm.bindStructEnv(v, reflect.New(field.Type).Interface(), key)
		} else {
			// Bind the key so AutomaticEnv can find it
			_ = v.BindEnv(key)
		}
	}
}
