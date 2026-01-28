// Package main demonstrates gaz configuration loading from files and env vars.
package main

import (
	"fmt"
	"log"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
)

// Config holds application configuration loaded from config.yaml and env vars.
// Struct tags control how values are mapped and validated:
// - mapstructure: maps YAML/JSON keys to struct fields
// - validate: applies validation rules (required, min, max, etc.)
type Config struct {
	Server struct {
		Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
		Host string `mapstructure:"host" validate:"required"`
	} `mapstructure:"server"`
	Debug bool `mapstructure:"debug"`
}

func main() {
	// Create config struct that will receive loaded values
	cfg := &Config{}

	// Create app with configuration
	app := gaz.New()

	// WithConfig sets up configuration loading:
	// - WithName("config"): looks for config.yaml (or config.json, config.toml)
	// - WithSearchPaths("."): looks in current directory
	// - WithEnvPrefix("APP"): binds env vars like APP_SERVER_PORT
	app.WithConfig(cfg,
		config.WithName("config"),
		config.WithSearchPaths("."),
		config.WithEnvPrefix("APP"),
	)

	// Build triggers config loading and validation
	if err := app.Build(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Config is now populated from:
	// 1. config.yaml (or other config file)
	// 2. Environment variables (override file values)
	//    - APP_SERVER__PORT overrides server.port
	//    - APP_DEBUG overrides debug

	fmt.Printf("Configuration loaded:\n")
	fmt.Printf("  Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  Debug:  %v\n", cfg.Debug)
}
