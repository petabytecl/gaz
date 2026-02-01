package config_test

import (
	"fmt"

	"github.com/petabytecl/gaz/config"
)

// =============================================================================
// Test types used in examples
// =============================================================================

// ServerConfig represents server configuration.
type ServerConfig struct {
	Host string `mapstructure:"host" gaz:"host"`
	Port int    `mapstructure:"port" gaz:"port"`
}

// Default implements config.Defaulter interface.
func (c *ServerConfig) Default() {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}
}

// DatabaseConfig represents database configuration.
type DatabaseConfig struct {
	Host     string `mapstructure:"host" gaz:"host" validate:"required"`
	Port     int    `mapstructure:"port" gaz:"port" validate:"min=1,max=65535"`
	Database string `mapstructure:"database" gaz:"database"`
}

// AppConfig is a complete application configuration.
type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server" gaz:"server"`
	Database DatabaseConfig `mapstructure:"database" gaz:"database"`
}

// =============================================================================
// MapBackend Examples
// =============================================================================

// ExampleNewMapBackend demonstrates creating an in-memory config backend.
// MapBackend is primarily used for testing but can also be used for
// simple applications that don't need file-based configuration.
func ExampleNewMapBackend() {
	backend := config.NewMapBackend(map[string]any{
		"server.host": "localhost",
		"server.port": 8080,
	})

	host := backend.GetString("server.host")
	port := backend.GetInt("server.port")

	fmt.Println("host:", host)
	fmt.Println("port:", port)
	// Output:
	// host: localhost
	// port: 8080
}

// ExampleMapBackend_Get demonstrates getting values from MapBackend.
// Get returns the raw value, while GetString/GetInt/etc return typed values.
func ExampleMapBackend_Get() {
	backend := config.NewMapBackend(map[string]any{
		"debug":     true,
		"log.level": "info",
		"timeout":   30,
	})

	// Get raw value
	debug := backend.Get("debug")
	fmt.Println("debug (raw):", debug)

	// Get typed values
	level := backend.GetString("log.level")
	timeout := backend.GetInt("timeout")
	isDebug := backend.GetBool("debug")

	fmt.Println("level:", level)
	fmt.Println("timeout:", timeout)
	fmt.Println("debug:", isDebug)
	// Output:
	// debug (raw): true
	// level: info
	// timeout: 30
	// debug: true
}

// ExampleMapBackend_Set demonstrates setting values at runtime.
// Values can be modified after the backend is created.
func ExampleMapBackend_Set() {
	backend := config.NewMapBackend(nil)

	// Set values
	backend.Set("server.host", "localhost")
	backend.Set("server.port", 9000)

	fmt.Println("host:", backend.GetString("server.host"))
	fmt.Println("port:", backend.GetInt("server.port"))
	// Output:
	// host: localhost
	// port: 9000
}

// ExampleMapBackend_SetDefault demonstrates default values.
// Defaults are returned only if no explicit value is set.
func ExampleMapBackend_SetDefault() {
	backend := config.NewMapBackend(nil)

	// Set defaults
	backend.SetDefault("host", "localhost")
	backend.SetDefault("port", 8080)

	// Default is returned when no value is set
	fmt.Println("host (default):", backend.GetString("host"))

	// Explicit value overrides default
	backend.Set("host", "production.server")
	fmt.Println("host (explicit):", backend.GetString("host"))
	// Output:
	// host (default): localhost
	// host (explicit): production.server
}

// ExampleMapBackend_IsSet demonstrates checking if a key is set.
// IsSet returns true for both explicit values and defaults.
func ExampleMapBackend_IsSet() {
	backend := config.NewMapBackend(map[string]any{
		"host": "localhost",
	})
	backend.SetDefault("port", 8080)

	fmt.Println("host is set:", backend.IsSet("host"))
	fmt.Println("port is set:", backend.IsSet("port"))
	fmt.Println("timeout is set:", backend.IsSet("timeout"))
	// Output:
	// host is set: true
	// port is set: true
	// timeout is set: false
}

// =============================================================================
// Manager Examples
// =============================================================================

// ExampleNew demonstrates creating a config Manager.
// The Manager requires a Backend to be configured via WithBackend option.
func ExampleNew() {
	backend := config.NewMapBackend(map[string]any{
		"server.host": "localhost",
		"server.port": 8080,
	})

	mgr := config.New(config.WithBackend(backend))

	// Access config via backend
	host := mgr.Backend().GetString("server.host")
	port := mgr.Backend().GetInt("server.port")

	fmt.Println("host:", host)
	fmt.Println("port:", port)
	// Output:
	// host: localhost
	// port: 8080
}

// ExampleNewWithBackend demonstrates creating a Manager with a backend.
// This is the recommended constructor for most use cases.
func ExampleNewWithBackend() {
	backend := config.NewMapBackend(map[string]any{
		"app.name":    "my-service",
		"app.version": "1.0.0",
	})

	mgr := config.NewWithBackend(backend)

	name := mgr.Backend().GetString("app.name")
	version := mgr.Backend().GetString("app.version")

	fmt.Println("name:", name)
	fmt.Println("version:", version)
	// Output:
	// name: my-service
	// version: 1.0.0
}

// ExampleManager_Backend demonstrates accessing the underlying backend.
// The Backend method returns the configured backend for direct access.
func ExampleManager_Backend() {
	backend := config.NewMapBackend(map[string]any{
		"debug": true,
	})

	mgr := config.New(config.WithBackend(backend))

	// Get the backend for direct value access
	b := mgr.Backend()
	debug := b.GetBool("debug")

	fmt.Println("debug:", debug)
	// Output: debug: true
}

// ExampleTestManager demonstrates the TestManager factory.
// TestManager creates a Manager with an in-memory MapBackend,
// useful for testing without file I/O.
func ExampleTestManager() {
	mgr := config.TestManager(map[string]any{
		"database.host": "localhost",
		"database.port": 5432,
	})

	host := mgr.Backend().GetString("database.host")
	port := mgr.Backend().GetInt("database.port")

	fmt.Println("host:", host)
	fmt.Println("port:", port)
	// Output:
	// host: localhost
	// port: 5432
}

// =============================================================================
// Backend Interface Examples
// =============================================================================

// ExampleBackend_GetString demonstrates type-safe string getter.
func ExampleBackend_GetString() {
	backend := config.NewMapBackend(map[string]any{
		"app.name": "my-service",
		"app.port": 8080, // int, not string
	})

	// GetString returns string value
	name := backend.GetString("app.name")
	fmt.Println("name:", name)

	// GetString returns empty string for non-string or missing values
	port := backend.GetString("app.port")
	missing := backend.GetString("app.missing")

	fmt.Println("port (as string):", port == "")
	fmt.Println("missing:", missing == "")
	// Output:
	// name: my-service
	// port (as string): true
	// missing: true
}

// ExampleBackend_GetInt demonstrates type-safe int getter.
func ExampleBackend_GetInt() {
	backend := config.NewMapBackend(map[string]any{
		"port":    8080,
		"timeout": "not-a-number",
	})

	port := backend.GetInt("port")
	timeout := backend.GetInt("timeout") // returns 0 for non-int

	fmt.Println("port:", port)
	fmt.Println("timeout:", timeout)
	// Output:
	// port: 8080
	// timeout: 0
}

// ExampleBackend_GetBool demonstrates type-safe bool getter.
func ExampleBackend_GetBool() {
	backend := config.NewMapBackend(map[string]any{
		"debug":   true,
		"verbose": false,
	})

	debug := backend.GetBool("debug")
	verbose := backend.GetBool("verbose")
	missing := backend.GetBool("missing") // returns false for missing

	fmt.Println("debug:", debug)
	fmt.Println("verbose:", verbose)
	fmt.Println("missing:", missing)
	// Output:
	// debug: true
	// verbose: false
	// missing: false
}

// =============================================================================
// Validation Examples
// =============================================================================

// ExampleValidateStruct demonstrates struct validation with tags.
// Fields with validate tags are checked using go-playground/validator.
func ExampleValidateStruct() {
	// Valid config
	validCfg := &DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "mydb",
	}

	err := config.ValidateStruct(validCfg)
	fmt.Println("valid config error:", err)

	// Invalid config - missing required field
	invalidCfg := &DatabaseConfig{
		Host:     "", // required but empty
		Port:     5432,
		Database: "mydb",
	}

	err = config.ValidateStruct(invalidCfg)
	fmt.Println("invalid config has error:", err != nil)
	// Output:
	// valid config error: <nil>
	// invalid config has error: true
}

// ExampleSampleConfig demonstrates the test SampleConfig type.
// SampleConfig provides a minimal structure useful for testing.
func ExampleSampleConfig() {
	cfg := &config.SampleConfig{}

	// Call Default() to set default values
	cfg.Default()

	fmt.Println("host:", cfg.Host)
	fmt.Println("port:", cfg.Port)
	// Output:
	// host: localhost
	// port: 8080
}

// =============================================================================
// Require* Test Helper Examples
// =============================================================================

// Note: Require* helpers are designed for use in tests and accept testing.TB.
// These examples show the expected behavior without actual test context.

// ExampleRequireConfigValue shows how RequireConfigValue is used.
// This function verifies a config key has the expected value.
func ExampleRequireConfigValue() {
	backend := config.NewMapBackend(map[string]any{
		"server.port": 8080,
	})

	// In tests, you would use:
	// config.RequireConfigValue(t, backend, "server.port", 8080)

	value := backend.Get("server.port")
	expected := 8080
	fmt.Println("value matches:", value == expected)
	// Output: value matches: true
}

// ExampleRequireConfigString shows how RequireConfigString is used.
// This function verifies a config key has the expected string value.
func ExampleRequireConfigString() {
	backend := config.NewMapBackend(map[string]any{
		"server.host": "localhost",
	})

	// In tests, you would use:
	// config.RequireConfigString(t, backend, "server.host", "localhost")

	value := backend.GetString("server.host")
	expected := "localhost"
	fmt.Println("value matches:", value == expected)
	// Output: value matches: true
}

// ExampleRequireConfigInt shows how RequireConfigInt is used.
// This function verifies a config key has the expected int value.
func ExampleRequireConfigInt() {
	backend := config.NewMapBackend(map[string]any{
		"server.port": 8080,
	})

	// In tests, you would use:
	// config.RequireConfigInt(t, backend, "server.port", 8080)

	value := backend.GetInt("server.port")
	expected := 8080
	fmt.Println("value matches:", value == expected)
	// Output: value matches: true
}

// ExampleRequireConfigIsSet shows how RequireConfigIsSet is used.
// This function verifies a config key is set.
func ExampleRequireConfigIsSet() {
	backend := config.NewMapBackend(map[string]any{
		"server.host": "localhost",
	})

	// In tests, you would use:
	// config.RequireConfigIsSet(t, backend, "server.host")

	isSet := backend.IsSet("server.host")
	fmt.Println("key is set:", isSet)
	// Output: key is set: true
}

// =============================================================================
// Options Examples
// =============================================================================

// ExampleWithBackend demonstrates the WithBackend option.
// This option configures the Backend for the Manager.
func ExampleWithBackend() {
	backend := config.NewMapBackend(map[string]any{
		"key": "value",
	})

	mgr := config.New(config.WithBackend(backend))

	fmt.Println("value:", mgr.Backend().GetString("key"))
	// Output: value: value
}

// ExampleWithDefaults demonstrates the WithDefaults option.
// Default values are used when no explicit value is set.
func ExampleWithDefaults() {
	backend := config.NewMapBackend(nil)

	mgr := config.New(
		config.WithBackend(backend),
		config.WithDefaults(map[string]any{
			"timeout": 30,
			"retries": 3,
		}),
	)

	// Load applies defaults
	mgr.Load()

	timeout := mgr.Backend().GetInt("timeout")
	retries := mgr.Backend().GetInt("retries")

	fmt.Println("timeout:", timeout)
	fmt.Println("retries:", retries)
	// Output:
	// timeout: 30
	// retries: 3
}
