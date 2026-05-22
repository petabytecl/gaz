package gaz

import (
	"fmt"

	"github.com/spf13/cobra"
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

	// Collect ConfigProvider info (idempotent)
	if err := a.collectProviderConfigs(); err != nil {
		return fmt.Errorf("collecting provider configs: %w", err)
	}

	// Register pflags and bind to viper
	return a.registerPFlags(cmd)
}

// registerPFlags registers pflags on the command and binds to viper.
func (a *App) registerPFlags(cmd *cobra.Command) error {
	return a.providerConfigIntakeModule().registerPFlags(cmd.PersistentFlags())
}
