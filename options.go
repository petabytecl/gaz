package gaz

// ResolveOption modifies resolution behavior.
// Options are passed to Resolve[T]() to customize the resolution process.
type ResolveOption func(*resolveOptions)

// resolveOptions holds resolution configuration.
type resolveOptions struct {
	name string // Custom name to resolve (empty = use type name)
}

// Named resolves a service by its registered name instead of type.
// Use this when you have multiple registrations of the same type.
//
// Example:
//
//	primaryDB, err := gaz.Resolve[*sql.DB](c, gaz.Named("primary"))
//	replicaDB, err := gaz.Resolve[*sql.DB](c, gaz.Named("replica"))
func Named(name string) ResolveOption {
	return func(o *resolveOptions) {
		o.name = name
	}
}

// applyOptions creates resolveOptions from variadic options.
func applyOptions(opts []ResolveOption) *resolveOptions {
	o := &resolveOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// ConfigOption configures the ConfigManager.
type ConfigOption func(*ConfigManager)

// WithName sets the config file name (without extension).
// Default is "config".
func WithName(name string) ConfigOption {
	return func(c *ConfigManager) {
		c.fileName = name
	}
}

// WithType sets the config file type (yaml, json, toml, etc.).
// Default is "yaml".
func WithType(t string) ConfigOption {
	return func(c *ConfigManager) {
		c.fileType = t
	}
}

// WithEnvPrefix sets the environment variable prefix.
// If set, environment variables will be bound automatically.
func WithEnvPrefix(prefix string) ConfigOption {
	return func(c *ConfigManager) {
		c.envPrefix = prefix
	}
}

// WithSearchPaths sets the paths to search for the config file.
// Default is ["."].
func WithSearchPaths(paths ...string) ConfigOption {
	return func(c *ConfigManager) {
		c.searchPaths = paths
	}
}

// WithProfileEnv sets the environment variable name that determines the active profile.
// If set and the env var is present, a profile-specific config will be loaded and merged.
func WithProfileEnv(envVar string) ConfigOption {
	return func(c *ConfigManager) {
		c.profileEnv = envVar
	}
}

// WithDefaults sets default values for configuration keys.
func WithDefaults(defaults map[string]any) ConfigOption {
	return func(c *ConfigManager) {
		if c.defaults == nil {
			c.defaults = make(map[string]any)
		}
		for k, v := range defaults {
			c.defaults[k] = v
		}
	}
}
