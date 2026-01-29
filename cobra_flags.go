package gaz

import (
	"fmt"
	"strings"
	"time"

	"github.com/petabytecl/gaz/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// RegisterCobraFlags registers ConfigProvider flags as persistent pflags on the command.
// This must be called BEFORE cmd.Execute() for flags to appear in --help output.
//
// The method:
// 1. Loads configuration (so defaults are available)
// 2. Registers ProviderValues for provider dependency injection
// 3. Collects ConfigProvider flags from registered services
// 4. Registers typed pflags on cmd.PersistentFlags()
// 5. Binds each flag to viper with the original dot-notation key
//
// Example:
//
//	app := gaz.New()
//	gaz.For[*ServerConfig](app.Container()).Provider(NewServerConfig)
//	app.RegisterCobraFlags(rootCmd)  // Register before Execute
//	app.WithCobra(rootCmd)
//	rootCmd.Execute()
func (a *App) RegisterCobraFlags(cmd *cobra.Command) error {
	// Load config (idempotent) - needed for defaults
	if err := a.loadConfig(); err != nil {
		return fmt.Errorf("loading config for flag registration: %w", err)
	}

	// Register ProviderValues early (idempotent) - needed if providers depend on it
	if err := a.registerProviderValuesEarly(); err != nil {
		return fmt.Errorf("registering provider values: %w", err)
	}

	// Collect ConfigProvider info (idempotent) - populates a.providerConfigs
	if err := a.collectProviderConfigs(); err != nil {
		return fmt.Errorf("collecting provider configs: %w", err)
	}

	// Register pflags and bind to viper
	return a.registerPFlags(cmd)
}

// registerPFlags registers pflags on the command and binds to viper.
func (a *App) registerPFlags(cmd *cobra.Command) error {
	fs := cmd.PersistentFlags()

	// Get FlagBinder from backend
	fb, ok := a.configMgr.Backend().(config.FlagBinder)
	if !ok {
		// Backend doesn't support individual flag binding, skip
		return nil
	}

	for _, entry := range a.providerConfigs {
		for _, flag := range entry.flags {
			fullKey := entry.namespace + "." + flag.Key
			flagName := configKeyToFlagName(fullKey)

			// Skip if already registered (collision prevention)
			if fs.Lookup(flagName) != nil {
				continue
			}

			// Register typed flag with default and description
			if err := registerTypedFlag(fs, flag, flagName); err != nil {
				return fmt.Errorf("registering flag %s: %w", flagName, err)
			}

			// Bind to viper with ORIGINAL dot-notation key
			if err := fb.BindPFlag(fullKey, fs.Lookup(flagName)); err != nil {
				return fmt.Errorf("binding flag %s to key %s: %w", flagName, fullKey, err)
			}
		}
	}
	return nil
}

// configKeyToFlagName transforms a config key to a POSIX flag name.
// Example: "server.host" -> "server-host"
func configKeyToFlagName(key string) string {
	return strings.ReplaceAll(key, ".", "-")
}

// registerTypedFlag registers a typed pflag based on ConfigFlag.Type.
func registerTypedFlag(fs *pflag.FlagSet, flag ConfigFlag, name string) error {
	switch flag.Type {
	case ConfigFlagTypeString:
		def, _ := flag.Default.(string)
		fs.String(name, def, flag.Description)
	case ConfigFlagTypeInt:
		def, _ := flag.Default.(int)
		fs.Int(name, def, flag.Description)
	case ConfigFlagTypeBool:
		def, _ := flag.Default.(bool)
		fs.Bool(name, def, flag.Description)
	case ConfigFlagTypeDuration:
		def, _ := flag.Default.(time.Duration)
		fs.Duration(name, def, flag.Description)
	case ConfigFlagTypeFloat:
		def, _ := flag.Default.(float64)
		fs.Float64(name, def, flag.Description)
	default:
		// Unknown type, treat as string
		def, _ := flag.Default.(string)
		fs.String(name, def, flag.Description)
	}
	return nil
}
