// Package config provides standalone configuration management for Go applications.
//
// This package defines interfaces and types for configuration loading, validation,
// and access. It is designed to work independently without requiring the full gaz
// framework, enabling standalone configuration management.
//
// # Backend Interface
//
// The [Backend] interface abstracts the underlying configuration provider (e.g., viper).
// It provides methods for getting and setting configuration values, as well as
// unmarshaling configuration into Go structs.
//
// Optional composed interfaces extend Backend capabilities:
//   - [Watcher] - for configuration file watching
//   - [Writer] - for writing configuration back to files
//   - [EnvBinder] - for environment variable binding
//
// # Viper Implementation
//
// The default viper-based Backend implementation is in the [github.com/petabytecl/gaz/config/viper]
// subpackage. This separation isolates the viper dependency from the core config package.
//
// # Config Lifecycle
//
// Configuration structs can implement [Defaulter] to provide default values,
// and [Validator] for custom validation logic. Struct tag validation using
// go-playground/validator is also supported.
//
// Example usage:
//
//	cfg := &AppConfig{}
//	mgr := config.New(
//	    config.WithName("config"),
//	    config.WithSearchPaths(".", "./config"),
//	)
//	if err := mgr.LoadInto(cfg); err != nil {
//	    log.Fatal(err)
//	}
package config
