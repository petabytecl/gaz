// Package main demonstrates the ConfigProvider pattern for flag-based configuration.
//
// This example shows the recommended pattern where services declare their config
// requirements via the ConfigProvider interface (ConfigNamespace + ConfigFlags).
//
// Key pattern: ProviderValues can be injected inside provider functions because
// it is registered BEFORE providers are instantiated during Build(). This allows
// providers to resolve config values at construction time.
//
// Key concepts:
//   - ConfigProvider declares config requirements (namespace + flags)
//   - ConfigFlags define typed config keys with defaults and descriptions
//   - ProviderValues provides typed access to resolved config values
//   - ProviderValues can be injected in providers (not just after Build)
//   - Values can come from config files, environment variables, or defaults
//
// By default, gaz looks for config.yaml in the current directory. No explicit
// WithConfig() call is needed for basic ConfigProvider usage.
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
)

// ServerConfig implements ConfigProvider to declare its configuration requirements.
// It stores the injected ProviderValues to provide typed accessor methods.
type ServerConfig struct {
	pv *gaz.ProviderValues
}

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

// Host returns the server host configuration value.
func (s *ServerConfig) Host() string {
	return s.pv.GetString("server.host")
}

// Port returns the server port configuration value.
func (s *ServerConfig) Port() int {
	return s.pv.GetInt("server.port")
}

// Debug returns the debug mode configuration value.
func (s *ServerConfig) Debug() bool {
	return s.pv.GetBool("server.debug")
}

// NewServerConfig injects ProviderValues during Build().
// This is now possible because ProviderValues is registered BEFORE providers run.
// The provider can resolve and store ProviderValues at construction time.
func NewServerConfig(c *gaz.Container) (*ServerConfig, error) {
	pv, err := gaz.Resolve[*gaz.ProviderValues](c)
	if err != nil {
		return nil, err
	}
	return &ServerConfig{pv: pv}, nil
}

func main() {
	// Create the application.
	// ConfigManager is auto-initialized with convention defaults:
	// - Looks for config.yaml in current directory
	// - Environment variables override config file values
	app := gaz.New()

	// Register the ConfigProvider. During Build(), the framework will:
	// 1. Load config and register ProviderValues EARLY
	// 2. Call provider functions (can now inject ProviderValues)
	// 3. Call ConfigNamespace() and ConfigFlags() to collect requirements
	// 4. Register defaults and bind environment variables
	// 5. Validate required flags are set
	if err := gaz.For[*ServerConfig](app.Container()).Provider(NewServerConfig); err != nil {
		log.Fatalf("Failed to register config provider: %v", err)
	}

	// Build triggers config loading and provider instantiation.
	// ProviderValues is registered BEFORE providers run, so NewServerConfig
	// can inject it as a dependency.
	if err := app.Build(); err != nil {
		log.Fatalf("Failed to build app: %v", err)
	}

	// Get the ServerConfig - it already has ProviderValues injected
	cfg := gaz.MustResolve[*ServerConfig](app.Container())

	// Use the accessor methods on ServerConfig
	fmt.Println("Configuration loaded via ConfigProvider pattern:")
	fmt.Printf("  Server: %s:%d\n", cfg.Host(), cfg.Port())
	fmt.Printf("  Debug:  %v\n", cfg.Debug())
	fmt.Println()
	fmt.Println("Key pattern: ProviderValues injected in provider constructor")
	fmt.Println("  - NewServerConfig receives *ProviderValues via DI")
	fmt.Println("  - Accessor methods (Host, Port, Debug) use stored ProviderValues")
	fmt.Println("  - No need to resolve ProviderValues in main()")
	fmt.Println()
	fmt.Println("Config sources (in priority order):")
	fmt.Println("  1. Environment variables (e.g., SERVER_HOST, SERVER_PORT)")
	fmt.Println("  2. Config file (config.yaml)")
	fmt.Println("  3. Defaults from ConfigFlags()")
}
