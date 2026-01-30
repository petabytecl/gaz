package viper

import (
	"errors"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/petabytecl/gaz/config"
)

// Compile-time interface assertions.
var (
	_ config.Backend    = (*Backend)(nil)
	_ config.Watcher    = (*Backend)(nil)
	_ config.Writer     = (*Backend)(nil)
	_ config.EnvBinder  = (*Backend)(nil)
	_ config.FlagBinder = (*Backend)(nil)
)

// Backend implements config.Backend, config.Watcher, config.Writer, and config.EnvBinder
// using spf13/viper as the underlying configuration provider.
type Backend struct {
	v *viper.Viper
}

// New creates a new ViperBackend with a fresh viper instance.
func New() *Backend {
	return &Backend{v: viper.New()}
}

// NewWithViper creates a Backend wrapping an existing viper instance.
// This is useful for integrating with existing viper configurations.
func NewWithViper(v *viper.Viper) *Backend {
	return &Backend{v: v}
}

// =============================================================================
// config.Backend implementation
// =============================================================================

// Get returns the value for a key.
func (b *Backend) Get(key string) any {
	return b.v.Get(key)
}

// GetString returns a string value for the key.
func (b *Backend) GetString(key string) string {
	return b.v.GetString(key)
}

// GetInt returns an int value for the key.
func (b *Backend) GetInt(key string) int {
	return b.v.GetInt(key)
}

// GetBool returns a bool value for the key.
func (b *Backend) GetBool(key string) bool {
	return b.v.GetBool(key)
}

// GetDuration returns a time.Duration value for the key.
func (b *Backend) GetDuration(key string) time.Duration {
	return b.v.GetDuration(key)
}

// GetFloat64 returns a float64 value for the key.
func (b *Backend) GetFloat64(key string) float64 {
	return b.v.GetFloat64(key)
}

// Set explicitly sets a value for a key.
func (b *Backend) Set(key string, value any) {
	b.v.Set(key, value)
}

// SetDefault sets a default value for a key.
func (b *Backend) SetDefault(key string, value any) {
	b.v.SetDefault(key, value)
}

// IsSet checks if a key has been set.
func (b *Backend) IsSet(key string) bool {
	return b.v.IsSet(key)
}

// Unmarshal unmarshals the entire config into a struct.
func (b *Backend) Unmarshal(target any) error {
	return b.v.Unmarshal(target)
}

// UnmarshalKey unmarshals a specific key into a struct.
func (b *Backend) UnmarshalKey(key string, target any) error {
	return b.v.UnmarshalKey(key, target)
}

// =============================================================================
// config.Watcher implementation
// =============================================================================

// WatchConfig starts watching the config file for changes.
func (b *Backend) WatchConfig() {
	b.v.WatchConfig()
}

// OnConfigChange registers a callback that is called when config changes.
// The event parameter is an fsnotify.Event.
func (b *Backend) OnConfigChange(callback func(event any)) {
	b.v.OnConfigChange(func(e fsnotify.Event) {
		callback(e)
	})
}

// =============================================================================
// config.Writer implementation
// =============================================================================

// WriteConfig writes the current config to the file from which it was read.
func (b *Backend) WriteConfig() error {
	return b.v.WriteConfig()
}

// WriteConfigAs writes the current config to the specified filename.
func (b *Backend) WriteConfigAs(filename string) error {
	return b.v.WriteConfigAs(filename)
}

// SafeWriteConfig writes config only if the file doesn't already exist.
func (b *Backend) SafeWriteConfig() error {
	return b.v.SafeWriteConfig()
}

// SafeWriteConfigAs writes config to filename only if it doesn't already exist.
func (b *Backend) SafeWriteConfigAs(filename string) error {
	return b.v.SafeWriteConfigAs(filename)
}

// =============================================================================
// config.EnvBinder implementation
// =============================================================================

// SetEnvPrefix sets a prefix that is used for environment variable names.
func (b *Backend) SetEnvPrefix(prefix string) {
	b.v.SetEnvPrefix(prefix)
}

// AutomaticEnv enables automatic environment variable binding.
func (b *Backend) AutomaticEnv() {
	b.v.AutomaticEnv()
}

// BindEnv binds one or more environment variable names to a config key.
func (b *Backend) BindEnv(keys ...string) error {
	return b.v.BindEnv(keys...)
}

// SetEnvKeyReplacer sets a replacer used to transform keys to env var names.
// The replacer must be a *strings.Replacer because viper requires this concrete type.
// If a different StringReplacer implementation is passed, this method panics.
func (b *Backend) SetEnvKeyReplacer(replacer config.StringReplacer) {
	if sr, ok := replacer.(*strings.Replacer); ok {
		b.v.SetEnvKeyReplacer(sr)
		return
	}
	// This shouldn't happen in practice since strings.Replacer satisfies StringReplacer
	// and is the primary use case. If someone passes a custom implementation,
	// viper can't handle it, so we panic with a clear message.
	panic("config/viper: SetEnvKeyReplacer requires a *strings.Replacer (viper limitation)")
}

// SetStringsReplacer is a convenience method that takes a *strings.Replacer directly.
// This avoids the interface type assertion for the common case.
func (b *Backend) SetStringsReplacer(replacer *strings.Replacer) {
	b.v.SetEnvKeyReplacer(replacer)
}

// =============================================================================
// Viper-specific methods for Manager integration
// =============================================================================

// SetConfigName sets the name of the config file (without extension).
func (b *Backend) SetConfigName(name string) {
	b.v.SetConfigName(name)
}

// SetConfigType sets the type of the config file (e.g., "yaml", "json").
func (b *Backend) SetConfigType(t string) {
	b.v.SetConfigType(t)
}

// AddConfigPath adds a path for viper to search for the config file.
func (b *Backend) AddConfigPath(path string) {
	b.v.AddConfigPath(path)
}

// SetConfigFile sets an explicit config file path.
// Unlike SetConfigName + AddConfigPath, this uses the exact file path.
// The file type is inferred from the extension.
func (b *Backend) SetConfigFile(path string) {
	b.v.SetConfigFile(path)
}

// ReadInConfig reads the config file from disk.
func (b *Backend) ReadInConfig() error {
	return b.v.ReadInConfig()
}

// MergeInConfig merges a new config file into the existing config.
func (b *Backend) MergeInConfig() error {
	return b.v.MergeInConfig()
}

// BindPFlags binds pflags to configuration keys.
func (b *Backend) BindPFlags(fs *pflag.FlagSet) error {
	return b.v.BindPFlags(fs)
}

// BindPFlag binds a single pflag to a configuration key.
// This allows CLI flags to override config values via viper precedence.
// The key uses dot notation (e.g., "server.host") while the flag uses
// hyphen notation (e.g., "server-host").
func (b *Backend) BindPFlag(key string, flag *pflag.Flag) error {
	return b.v.BindPFlag(key, flag)
}

// ConfigFileUsed returns the file used to populate the config.
func (b *Backend) ConfigFileUsed() string {
	return b.v.ConfigFileUsed()
}

// AllSettings returns all settings from viper as a map.
func (b *Backend) AllSettings() map[string]any {
	return b.v.AllSettings()
}

// Viper returns the underlying viper instance.
// This is useful for advanced configuration or direct viper access.
func (b *Backend) Viper() *viper.Viper {
	return b.v
}

// IsConfigFileNotFoundError returns true if the error is a config file not found error.
// This is a helper for error handling.
func IsConfigFileNotFoundError(err error) bool {
	var configFileNotFoundError viper.ConfigFileNotFoundError
	ok := errors.As(err, &configFileNotFoundError)
	return ok
}

// IsConfigFileNotFoundError returns true if the given error indicates a missing config file.
// This implements the configFileNotFoundChecker interface for the Manager.
func (b *Backend) IsConfigFileNotFoundError(err error) bool {
	return IsConfigFileNotFoundError(err)
}
