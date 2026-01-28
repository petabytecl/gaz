package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/pflag"
)

// Manager handles configuration loading, binding, and validation.
// It provides a unified interface for loading configuration from multiple sources
// (files, environment variables, flags) and validating the result.
type Manager struct {
	backend     Backend
	fileName    string
	fileType    string
	searchPaths []string
	envPrefix   string
	profileEnv  string
	defaults    map[string]any
	configFile  string // explicit config file path (if set, ignores search paths)
}

// New creates a new Manager with the given options.
// A Backend must be provided via WithBackend option or the Manager will panic
// on first use. For convenience when using viper, import config/viper and use:
//
//	mgr := config.New(config.WithBackend(viper.New()), ...)
//
// Or use NewWithBackend directly:
//
//	mgr := config.NewWithBackend(viper.New(), ...)
//
// Example:
//
//	import (
//	    "github.com/petabytecl/gaz/config"
//	    "github.com/petabytecl/gaz/config/viper"
//	)
//
//	mgr := config.New(
//	    config.WithBackend(viper.New()),
//	    config.WithName("config"),
//	    config.WithSearchPaths(".", "./config"),
//	    config.WithEnvPrefix("APP"),
//	)
func New(opts ...Option) *Manager {
	m := &Manager{
		fileName:    "config",
		fileType:    "yaml",
		searchPaths: []string{"."},
		defaults:    make(map[string]any),
	}

	for _, opt := range opts {
		opt(m)
	}

	if m.backend == nil {
		panic("config: backend is required, use WithBackend option or NewWithBackend constructor")
	}

	return m
}

// NewWithBackend creates a Manager with a custom backend.
// This is the recommended constructor for most use cases.
//
// Example:
//
//	import (
//	    "github.com/petabytecl/gaz/config"
//	    "github.com/petabytecl/gaz/config/viper"
//	)
//
//	mgr := config.NewWithBackend(viper.New(),
//	    config.WithName("app"),
//	    config.WithSearchPaths(".", "./config"),
//	)
func NewWithBackend(backend Backend, opts ...Option) *Manager {
	if backend == nil {
		panic("config: backend cannot be nil")
	}

	m := &Manager{
		backend:     backend,
		fileName:    "config",
		fileType:    "yaml",
		searchPaths: []string{"."},
		defaults:    make(map[string]any),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Load reads configuration from files and environment variables.
// This method configures the backend and reads the config file, but does not
// unmarshal into a target struct. Use LoadInto for combined load + unmarshal.
func (m *Manager) Load() error {
	// Handle explicit config file path vs search paths
	if m.configFile != "" {
		// Use explicit config file path if backend supports it
		if cfs, ok := m.backend.(configFileSetter); ok {
			cfs.SetConfigFile(m.configFile)
		}
	} else {
		// Configure backend for file reading via viperConfigurable interface
		if vc, ok := m.backend.(viperConfigurable); ok {
			vc.SetConfigName(m.fileName)
			vc.SetConfigType(m.fileType)
			for _, path := range m.searchPaths {
				vc.AddConfigPath(path)
			}
		}
	}

	// Apply defaults
	for k, v := range m.defaults {
		m.backend.SetDefault(k, v)
	}

	// Configure environment variable binding if prefix is set
	if m.envPrefix != "" {
		if eb, ok := m.backend.(EnvBinder); ok {
			eb.SetEnvPrefix(m.envPrefix)
			eb.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
			eb.AutomaticEnv()
		}
	}

	// Read config file via configReader interface
	if cr, ok := m.backend.(configReader); ok {
		if err := cr.ReadInConfig(); err != nil {
			if !isConfigFileNotFoundError(cr, err) {
				return fmt.Errorf("config: failed to read config file: %w", err)
			}
			// Config file not found is OK - can use defaults and env vars
		}

		// Load profile config if set
		if err := m.loadProfileConfig(cr); err != nil {
			return err
		}
	}

	return nil
}

// LoadInto loads configuration from all sources and unmarshals into target.
// It performs the following steps in order:
//  1. Load config from files/environment
//  2. Unmarshal into target struct
//  3. Apply Defaulter interface if implemented
//  4. Validate using struct tags (go-playground/validator)
//  5. Validate using Validator interface if implemented
//
// Example:
//
//	cfg := &AppConfig{}
//	if err := mgr.LoadInto(cfg); err != nil {
//	    log.Fatal(err)
//	}
func (m *Manager) LoadInto(target any) error {
	if target == nil {
		return nil
	}

	// Bind struct env vars before loading (for automatic env binding)
	if m.envPrefix != "" {
		if eb, ok := m.backend.(EnvBinder); ok {
			m.bindStructEnv(eb, target, "")
		}
	}

	// Load from files/env
	if err := m.Load(); err != nil {
		return err
	}

	// Unmarshal into target
	if err := m.backend.Unmarshal(target); err != nil {
		return fmt.Errorf("config: failed to unmarshal: %w", err)
	}

	// Apply Defaulter interface
	if d, ok := target.(Defaulter); ok {
		d.Default()
	}

	// Validate using struct tags
	if err := ValidateStruct(target); err != nil {
		return err // Already wrapped with ErrConfigValidation
	}

	// Validate using Validator interface
	if v, ok := target.(Validator); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("config: custom validation failed: %w", err)
		}
	}

	return nil
}

// Backend returns the underlying Backend for direct access.
// This is useful for advanced operations not covered by the Manager API.
func (m *Manager) Backend() Backend {
	return m.backend
}

// BindFlags binds command line flags to the configuration.
// This allows flag values to override config file and environment values.
func (m *Manager) BindFlags(fs *pflag.FlagSet) error {
	if fb, ok := m.backend.(flagBinder); ok {
		if err := fb.BindPFlags(fs); err != nil {
			return fmt.Errorf("config: failed to bind flags: %w", err)
		}
	}
	return nil
}

// loadProfileConfig loads and merges profile-specific configuration.
// Profile is determined by the profileEnv environment variable.
func (m *Manager) loadProfileConfig(cr configReader) error {
	if m.profileEnv == "" {
		return nil
	}

	profile := os.Getenv(m.profileEnv)
	if profile == "" {
		return nil
	}

	if vc, ok := m.backend.(viperConfigurable); ok {
		vc.SetConfigName(m.fileName + "." + profile)
	}

	if mc, ok := cr.(configMerger); ok {
		if err := mc.MergeInConfig(); err != nil {
			if !isConfigFileNotFoundError(cr, err) {
				return fmt.Errorf("config: failed to merge profile config: %w", err)
			}
		}
	}
	return nil
}

// bindStructEnv recursively binds struct fields to environment variables.
// This ensures that AutomaticEnv can find the keys.
func (m *Manager) bindStructEnv(eb EnvBinder, target any, prefix string) {
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
			m.bindStructEnv(eb, reflect.New(field.Type).Interface(), key)
		} else {
			// Bind the key so AutomaticEnv can find it
			_ = eb.BindEnv(key)
		}
	}
}

// RegisterProviderFlags registers provider config flags with defaults and env binding.
// For each flag, it sets the default value and binds to an environment variable.
func (m *Manager) RegisterProviderFlags(namespace string, flags []ConfigFlag) error {
	for _, flag := range flags {
		fullKey := namespace + "." + flag.Key

		// Set default if provided
		if flag.Default != nil {
			m.backend.SetDefault(fullKey, flag.Default)
		}

		// Bind env var with explicit name
		if eb, ok := m.backend.(EnvBinder); ok {
			envKey := strings.ToUpper(strings.ReplaceAll(fullKey, ".", "_"))
			if err := eb.BindEnv(fullKey, envKey); err != nil {
				return fmt.Errorf("config: failed to bind env var %s for key %s: %w", envKey, fullKey, err)
			}
		}
	}
	return nil
}

// ValidateProviderFlags validates that required provider config flags are set.
// Returns a slice of errors for all missing required fields (not fail-fast).
func (m *Manager) ValidateProviderFlags(namespace string, flags []ConfigFlag) []error {
	var errs []error
	for _, flag := range flags {
		if !flag.Required {
			continue
		}

		fullKey := namespace + "." + flag.Key

		if !m.backend.IsSet(fullKey) {
			errs = append(errs, fmt.Errorf(
				"provider %q: required config key %q is not set",
				namespace, fullKey,
			))
		}
	}
	return errs
}

// ConfigFlag represents a configuration flag for provider registration.
type ConfigFlag struct {
	Key      string
	Default  any
	Required bool
}

// =============================================================================
// Internal interfaces for viper-specific operations
// =============================================================================

// viperConfigurable is implemented by backends that support viper-like configuration.
type viperConfigurable interface {
	SetConfigName(name string)
	SetConfigType(t string)
	AddConfigPath(path string)
}

// configFileSetter is implemented by backends that support explicit config file paths.
type configFileSetter interface {
	SetConfigFile(path string)
}

// configReader is implemented by backends that can read config files.
type configReader interface {
	ReadInConfig() error
}

// configMerger is implemented by backends that can merge config files.
type configMerger interface {
	MergeInConfig() error
}

// flagBinder is implemented by backends that can bind pflags.
type flagBinder interface {
	BindPFlags(fs *pflag.FlagSet) error
}

// configFileNotFoundChecker is implemented by backends that can check for file not found errors.
type configFileNotFoundChecker interface {
	IsConfigFileNotFoundError(err error) bool
}

// isConfigFileNotFoundError checks if the error indicates a missing config file.
func isConfigFileNotFoundError(backend any, err error) bool {
	if checker, ok := backend.(configFileNotFoundChecker); ok {
		return checker.IsConfigFileNotFoundError(err)
	}
	// Fallback: check error message (case-insensitive for viper compatibility)
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}
