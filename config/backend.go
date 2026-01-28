package config

import "time"

// Backend is the core interface for configuration access.
// All backends must implement at minimum Get/Set/Unmarshal operations.
// Keys use dot notation for nested values (e.g., "database.host").
type Backend interface {
	// Get returns the value for a key. Keys use dot notation (e.g., "database.host").
	Get(key string) any

	// GetString returns a string value for the key.
	GetString(key string) string

	// GetInt returns an int value for the key.
	GetInt(key string) int

	// GetBool returns a bool value for the key.
	GetBool(key string) bool

	// GetDuration returns a time.Duration value for the key.
	GetDuration(key string) time.Duration

	// GetFloat64 returns a float64 value for the key.
	GetFloat64(key string) float64

	// Set explicitly sets a value for a key.
	Set(key string, value any)

	// SetDefault sets a default value for a key.
	SetDefault(key string, value any)

	// IsSet checks if a key has been set.
	IsSet(key string) bool

	// Unmarshal unmarshals the entire config into a struct.
	Unmarshal(target any) error

	// UnmarshalKey unmarshals a specific key into a struct.
	UnmarshalKey(key string, target any) error
}

// Watcher is implemented by backends that support configuration file watching.
// This is an optional interface that extends Backend with file watching capabilities.
type Watcher interface {
	// WatchConfig starts watching the config file for changes.
	WatchConfig()

	// OnConfigChange registers a callback that is called when config changes.
	// The event parameter is typically an fsnotify.Event but is typed as any
	// to avoid leaking file watching implementation details.
	OnConfigChange(callback func(event any))
}

// Writer is implemented by backends that can write configuration to files.
// This is an optional interface that extends Backend with write capabilities.
type Writer interface {
	// WriteConfig writes the current config to the file from which it was read.
	WriteConfig() error

	// WriteConfigAs writes the current config to the specified filename.
	WriteConfigAs(filename string) error

	// SafeWriteConfig writes config only if the file doesn't already exist.
	SafeWriteConfig() error

	// SafeWriteConfigAs writes config to filename only if it doesn't already exist.
	SafeWriteConfigAs(filename string) error
}

// EnvBinder is implemented by backends that support environment variable binding.
// This is an optional interface that extends Backend with env binding capabilities.
type EnvBinder interface {
	// SetEnvPrefix sets a prefix that is used for environment variable names.
	// For example, if prefix is "APP", then the key "database.host" will look
	// for APP_DATABASE_HOST environment variable.
	SetEnvPrefix(prefix string)

	// AutomaticEnv enables automatic environment variable binding.
	// When enabled, keys will automatically be bound to corresponding env vars.
	AutomaticEnv()

	// BindEnv binds one or more environment variable names to a config key.
	// If only a key is provided, the env var name is generated from the key.
	// If additional names are provided, they are used as-is.
	BindEnv(keys ...string) error

	// SetEnvKeyReplacer sets a replacer used to transform keys to env var names.
	// Commonly used to replace dots with underscores.
	SetEnvKeyReplacer(replacer StringReplacer)
}

// StringReplacer is used for transforming config key names to environment variable names.
// This interface is satisfied by strings.Replacer.
type StringReplacer interface {
	Replace(s string) string
}
