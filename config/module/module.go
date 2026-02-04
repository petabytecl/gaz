package module

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz"
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
// Returns cwd first, then XDG config directory.
func (c *Config) GetSearchPaths(appName string) []string {
	paths := []string{"."}

	// Add XDG config directory
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		if home, err := os.UserHomeDir(); err == nil {
			xdgConfig = filepath.Join(home, ".config")
		}
	}
	if xdgConfig != "" && appName != "" {
		paths = append(paths, filepath.Join(xdgConfig, appName))
	}

	return paths
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
//
//nolint:ireturn // Module is the expected return type for gaz modules
func New() gaz.Module {
	defaultCfg := DefaultConfig()

	return gaz.NewModule("config-flags").
		Flags(defaultCfg.Flags).
		Provide(func(c *gaz.Container) error {
			return gaz.For[Config](c).Provider(func(c *gaz.Container) (Config, error) {
				cfg := defaultCfg

				// Try to load from config manager if available
				pv, pvErr := gaz.Resolve[*gaz.ProviderValues](c)
				if pvErr == nil {
					if unmarshalErr := pv.UnmarshalKey(cfg.Namespace(), &cfg); unmarshalErr != nil {
						// Ignore error, use defaults (key may not exist)
						_ = unmarshalErr
					}
				}

				cfg.SetDefaults()
				if validateErr := cfg.Validate(); validateErr != nil {
					return cfg, fmt.Errorf("validate config module: %w", validateErr)
				}

				return cfg, nil
			})
		}).
		Build()
}
