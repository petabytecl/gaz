package module

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
	"github.com/petabytecl/gaz/internal/configuredmodule"
)

// Config holds configuration for the config module.
type Config struct {
	// ConfigFile is the explicit config file path.
	// If empty, auto-search is enabled.
	ConfigFile string

	// EnvPrefix is the environment variable prefix.
	// Defaults to "GAZ".
	EnvPrefix string

	// Strict enables strict mode where unknown config keys cause errors.
	// Defaults to true per CONTEXT.md.
	Strict bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ConfigFile: "",    // Auto-search if empty
		EnvPrefix:  "GAZ", // Default prefix
		Strict:     true,  // Exit on unknown keys per CONTEXT.md
	}
}

// Namespace returns the configuration namespace for config binding.
func (c *Config) Namespace() string {
	return "config"
}

// Flags registers CLI flags for configuration.
func (c *Config) Flags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ConfigFile, "config", c.ConfigFile,
		"Config file path (auto-searches if not set)")
	fs.StringVar(&c.EnvPrefix, "env-prefix", c.EnvPrefix,
		"Environment variable prefix")
	fs.BoolVar(&c.Strict, "config-strict", c.Strict,
		"Exit on unknown config keys")
}

// Validate validates the configuration.
// If ConfigFile is set, validates the file exists.
func (c *Config) Validate() error {
	if c.ConfigFile != "" {
		if _, err := os.Stat(c.ConfigFile); err != nil {
			return fmt.Errorf("config file not found: %s", c.ConfigFile)
		}
	}
	return nil
}

// SetDefaults applies default values to zero-value fields.
func (c *Config) SetDefaults() {
	if c.EnvPrefix == "" {
		c.EnvPrefix = "GAZ"
	}
}

// GetSearchPaths returns the search paths for auto-discovery mode.
// Delegates to config.DefaultSearchPaths for the shared convention.
func (c *Config) GetSearchPaths(appName string) []string {
	return config.DefaultSearchPaths(appName)
}

// New creates a config module that provides Config with CLI flags.
// The App applies this config to recreate the config manager with
// the correct options in Build().
//
// Usage:
//
//	import configmod "github.com/petabytecl/gaz/config/module"
//
//	app := gaz.New(gaz.WithCobra(rootCmd))
//	app.Use(configmod.New())
//
// Flags registered:
//
//	--config         Config file path (auto-searches if not set)
//	--env-prefix     Environment variable prefix (default: GAZ)
//	--config-strict  Exit on unknown config keys (default: true)
func New() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule("config-flags").
		Flags(defaultCfg.Flags).
		Provide(configuredmodule.ProvideDefaulted[Config, *Config](
			&defaultCfg,
			"config module",
			"validate config module",
		)).
		Build()
}
