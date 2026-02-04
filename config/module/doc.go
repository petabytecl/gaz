// Package module provides a gaz.Module for configuring config loading via CLI flags.
//
// # Overview
//
// This module adds CLI flags for configuration file loading, enabling runtime
// specification of config file path, environment variable prefix, and strict mode.
//
// # Usage
//
// Add the module to your application:
//
//	app := gaz.New(gaz.WithCobra(cmd))
//	app.Use(configmod.New())
//
// This registers the following CLI flags:
//
//	--config        Path to configuration file (optional)
//	--env-prefix    Environment variable prefix (default: app name)
//	--config-strict Fail on unknown config keys (default: true)
//
// # Auto-Search Behavior
//
// When --config is not provided, the module searches for config files in:
//
//  1. Current working directory: ./config.{yaml,json,toml}
//  2. XDG config directory: $XDG_CONFIG_HOME/{appname}/config.{yaml,json,toml}
//     Falls back to ~/.config/{appname}/ if XDG_CONFIG_HOME is not set
//
// The first file found is used. If no config file is found, the application
// runs with default values and environment variables only.
//
// # Strict Mode
//
// When strict mode is enabled (default), the application fails at startup if
// the config file contains unknown keys. This helps catch typos in config files
// early. Set --config-strict=false to allow unknown keys.
//
// # Environment Variables
//
// Environment variables are bound automatically using the specified prefix.
// For example, with prefix "MYAPP":
//
//	MYAPP_DATABASE_HOST -> database.host in config
//
// Nested keys use underscore as separator.
package module
