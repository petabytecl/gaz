// Package viper provides a viper-based Backend implementation for the config package.
//
// This package isolates the viper dependency from the core config package, allowing
// users who need a different configuration backend to avoid importing viper.
//
// The [Backend] type implements all four config interfaces:
//   - [config.Backend] - core configuration operations
//   - [config.Watcher] - configuration file watching
//   - [config.Writer] - configuration file writing
//   - [config.EnvBinder] - environment variable binding
//
// # Basic Usage
//
//	import (
//	    "github.com/petabytecl/gaz/config"
//	    configviper "github.com/petabytecl/gaz/config/viper"
//	)
//
//	backend := configviper.New()
//	mgr := config.NewWithBackend(backend)
//
// The Backend wraps a viper.Viper instance and delegates all operations to it.
// Additional viper-specific methods are exposed for configuration loading
// (SetConfigName, AddConfigPath, ReadInConfig, etc.).
package viper
