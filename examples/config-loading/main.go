// Package main demonstrates the ConfigProvider pattern for flag-based configuration.
//
// This example shows the recommended pattern where services declare their config
// requirements via the ConfigProvider interface (ConfigNamespace + ConfigFlags).
// Config values are accessed via ProviderValues AFTER app.Build() completes.
//
// Key concepts:
//   - ConfigProvider declares config requirements (namespace + flags)
//   - ConfigFlags define typed config keys with defaults and descriptions
//   - ProviderValues provides typed access to resolved config values
//   - Values can come from config files, environment variables, or defaults
//
// Environment variable mapping:
//   - server.host -> SERVER_HOST
//   - server.port -> SERVER_PORT
//   - server.debug -> SERVER_DEBUG
package main

import (
	"fmt"
	"log"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
)

// ServerConfig implements ConfigProvider to declare its configuration requirements.
// The struct itself is simple - it just satisfies the interface.
// Config values are accessed via ProviderValues, not stored in the struct.
type ServerConfig struct{}

// ConfigNamespace returns the namespace prefix for all config keys.
// Keys returned by ConfigFlags() are prefixed with this namespace.
// For example: namespace "server" + key "host" = "server.host"
func (s *ServerConfig) ConfigNamespace() string {
	return "server"
}

// ConfigFlags declares the configuration flags this provider needs.
// Each flag specifies a key, type, default value, and description.
func (s *ServerConfig) ConfigFlags() []gaz.ConfigFlag {
	return []gaz.ConfigFlag{
		{Key: "host", Type: gaz.ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
		{Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 8080, Description: "Server port"},
		{Key: "debug", Type: gaz.ConfigFlagTypeBool, Default: false, Description: "Debug mode"},
	}
}

// NewServerConfig is a simple constructor that returns the ConfigProvider.
// IMPORTANT: Do NOT resolve ProviderValues here - it's not registered yet.
// ProviderValues is only available AFTER app.Build() completes.
func NewServerConfig(c *gaz.Container) (*ServerConfig, error) {
	return &ServerConfig{}, nil
}

func main() {
	// Create the application
	app := gaz.New()

	// Enable config manager with search paths and env prefix.
	// The empty struct just enables the ConfigManager - we use ProviderValues for values.
	app.WithConfig(&struct{}{},
		config.WithName("config"),
		config.WithSearchPaths("."),
	)

	// Register the ConfigProvider. During Build(), the framework will:
	// 1. Call ConfigNamespace() and ConfigFlags() to collect requirements
	// 2. Register defaults and bind environment variables
	// 3. Validate required flags are set
	// 4. Register ProviderValues for resolving config values
	if err := gaz.For[*ServerConfig](app.Container()).Provider(NewServerConfig); err != nil {
		log.Fatalf("Failed to register config provider: %v", err)
	}

	// Build triggers config collection, loading, and validation.
	// After this, ProviderValues is registered and config values are accessible.
	if err := app.Build(); err != nil {
		log.Fatalf("Failed to build app: %v", err)
	}

	// NOW we can resolve ProviderValues (after Build completed).
	// Access config values using the full key: "namespace.key"
	pv := gaz.MustResolve[*gaz.ProviderValues](app.Container())

	// Get typed config values
	host := pv.GetString("server.host")
	port := pv.GetInt("server.port")
	debug := pv.GetBool("server.debug")

	// Display loaded configuration
	fmt.Println("Configuration loaded via ConfigProvider pattern:")
	fmt.Printf("  Server: %s:%d\n", host, port)
	fmt.Printf("  Debug:  %v\n", debug)
	fmt.Println()
	fmt.Println("Config sources (in priority order):")
	fmt.Println("  1. Environment variables (e.g., SERVER_HOST, SERVER_PORT)")
	fmt.Println("  2. Config file (config.yaml)")
	fmt.Println("  3. Defaults from ConfigFlags()")
}
