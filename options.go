package gaz

import "github.com/petabytecl/gaz/config"

// =============================================================================
// Config Options - Re-exported from config package for backward compatibility
// =============================================================================

// ConfigOption configures the ConfigManager.
// Deprecated: Import github.com/petabytecl/gaz/config directly.
type ConfigOption = config.Option

// WithName sets the config file name (without extension).
// Default is "config".
// Deprecated: Import github.com/petabytecl/gaz/config directly.
var WithName = config.WithName

// WithType sets the config file type (yaml, json, toml, etc.).
// Default is "yaml".
// Deprecated: Import github.com/petabytecl/gaz/config directly.
var WithType = config.WithType

// WithEnvPrefix sets the environment variable prefix.
// If set, environment variables will be bound automatically.
// Deprecated: Import github.com/petabytecl/gaz/config directly.
var WithEnvPrefix = config.WithEnvPrefix

// WithSearchPaths sets the paths to search for the config file.
// Default is ["."].
// Deprecated: Import github.com/petabytecl/gaz/config directly.
var WithSearchPaths = config.WithSearchPaths

// WithProfileEnv sets the environment variable name that determines the active profile.
// If set and the env var is present, a profile-specific config will be loaded and merged.
// Deprecated: Import github.com/petabytecl/gaz/config directly.
var WithProfileEnv = config.WithProfileEnv

// WithDefaults sets default values for configuration keys.
// Deprecated: Import github.com/petabytecl/gaz/config directly.
var WithDefaults = config.WithDefaults
