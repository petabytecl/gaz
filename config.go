package gaz

// ConfigOptions defines how configuration should be loaded.
type ConfigOptions struct {
	// Name is the name of the config file (without extension).
	// Default: "config"
	Name string

	// Type is the config file type (yaml, json, toml, etc.).
	// Default: "yaml"
	Type string

	// Paths is a list of paths to search for the config file.
	// Default: ["."]
	Paths []string

	// EnvPrefix is the prefix for environment variables.
	// AutomaticEnv will be enabled if this is set.
	// Default: "" (no env loading)
	EnvPrefix string

	// ProfileEnv is the name of the environment variable that determines the active profile.
	// If set and the env var is present (e.g. APP_ENV=prod), gaz will try to load
	// a profile-specific config (e.g. config.prod.yaml) and merge it.
	ProfileEnv string
}

// Defaulter allows a config struct to set its own default values.
// The Default() method is called after unmarshaling but before validation.
type Defaulter interface {
	Default()
}

// Validator allows a config struct to validate its own state.
// The Validate() method is called after defaults are applied.
// If it returns an error, the application startup will fail.
type Validator interface {
	Validate() error
}
