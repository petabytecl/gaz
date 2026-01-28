package config

// Option configures a Manager.
type Option func(*Manager)

// WithName sets the config file name (without extension).
// Default is "config".
func WithName(name string) Option {
	return func(m *Manager) {
		m.fileName = name
	}
}

// WithType sets the config file type (yaml, json, toml, etc.).
// Default is "yaml".
func WithType(t string) Option {
	return func(m *Manager) {
		m.fileType = t
	}
}

// WithEnvPrefix sets the environment variable prefix.
// If set, environment variables will be bound automatically.
// For example, if prefix is "APP", then the key "database.host" will look
// for APP_DATABASE__HOST environment variable.
func WithEnvPrefix(prefix string) Option {
	return func(m *Manager) {
		m.envPrefix = prefix
	}
}

// WithSearchPaths sets the paths to search for the config file.
// Default is ["."].
func WithSearchPaths(paths ...string) Option {
	return func(m *Manager) {
		m.searchPaths = paths
	}
}

// WithProfileEnv sets the environment variable name that determines the active profile.
// If set and the env var is present, a profile-specific config will be loaded and merged.
// For example, if the env var is "APP_PROFILE" and its value is "dev",
// then "config.dev.yaml" will be loaded and merged with "config.yaml".
func WithProfileEnv(envVar string) Option {
	return func(m *Manager) {
		m.profileEnv = envVar
	}
}

// WithDefaults sets default values for configuration keys.
// These defaults are applied before reading config files.
func WithDefaults(defaults map[string]any) Option {
	return func(m *Manager) {
		if m.defaults == nil {
			m.defaults = make(map[string]any)
		}
		for k, v := range defaults {
			m.defaults[k] = v
		}
	}
}

// WithBackend sets a custom Backend implementation.
// This is required when using New() - the Manager needs a backend to function.
func WithBackend(backend Backend) Option {
	return func(m *Manager) {
		m.backend = backend
	}
}
