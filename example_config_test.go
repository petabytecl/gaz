package gaz_test

import (
	"fmt"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
)

// AppConfig demonstrates configuration with validation tags.
type AppConfig struct {
	Port  int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	Host  string `mapstructure:"host"`
	Debug bool   `mapstructure:"debug"`
}

// Default sets default values when not provided in config.
func (c *AppConfig) Default() {
	if c.Port == 0 {
		c.Port = 8080
	}
	if c.Host == "" {
		c.Host = "localhost"
	}
}

// ServerConfig demonstrates nested configuration.
type ServerConfig struct {
	Name    string `mapstructure:"name"`
	Timeout int    `mapstructure:"timeout"`
}

// ExampleConfigManager demonstrates basic ConfigManager usage.
// ConfigManager handles loading configuration from files, environment
// variables, and applies defaults.
func ExampleConfigManager() {
	var cfg AppConfig

	cm := gaz.NewConfigManager(&cfg)

	// Load applies defaults via Default() method
	// In real usage, it would also load from config files
	if err := cm.Load(); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("port:", cfg.Port)
	fmt.Println("host:", cfg.Host)
	// Output:
	// port: 8080
	// host: localhost
}

// ExampleNewConfigManager demonstrates configuring the ConfigManager.
// Options include config.WithName, config.WithSearchPaths, config.WithEnvPrefix, and config.WithDefaults.
func ExampleNewConfigManager() {
	var cfg ServerConfig

	cm := gaz.NewConfigManager(&cfg,
		config.WithName("server"),             // look for server.yaml, server.json, etc.
		config.WithSearchPaths(".", "config"), // search in current dir and config/
		config.WithDefaults(map[string]any{
			"name":    "default-server",
			"timeout": 30,
		}),
	)

	// Load configuration (uses defaults since no files exist)
	if err := cm.Load(); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("name:", cfg.Name)
	fmt.Println("timeout:", cfg.Timeout)
	// Output:
	// name: default-server
	// timeout: 30
}

// ValidatedConfig demonstrates config with validation.
type ValidatedConfig struct {
	Port int    `mapstructure:"port"`
	Name string `mapstructure:"name"`
}

// Default sets sensible defaults.
func (c *ValidatedConfig) Default() {
	if c.Port == 0 {
		c.Port = 3000
	}
	if c.Name == "" {
		c.Name = "app"
	}
}

// Validate checks configuration validity.
func (c *ValidatedConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}
	return nil
}

// Example_validation demonstrates config validation with Validate() method.
// If a config struct implements Validator interface, Load() calls Validate()
// after applying defaults and unmarshaling.
func Example_validation() {
	var cfg ValidatedConfig

	cm := gaz.NewConfigManager(&cfg)

	if err := cm.Load(); err != nil {
		fmt.Println("error:", err)
		return
	}

	// Default() was called, setting port=3000 and name="app"
	// Validate() was called, confirming port is valid
	fmt.Println("config valid, port:", cfg.Port)
	// Output: config valid, port: 3000
}
