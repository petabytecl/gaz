// Package module provides a gaz.Module for configuring the logger via CLI flags.
//
// # Overview
//
// This module adds CLI flags for logger configuration, allowing runtime control
// of log level, format, and output destination without modifying code.
//
// # Usage
//
// Add the module to your application:
//
//	app := gaz.New(gaz.WithCobra(cmd))
//	app.Use(loggermod.New())
//
// This registers the following CLI flags:
//
//	--log-level     Log level: debug, info, warn, error (default: info)
//	--log-format    Log format: text, json (default: json)
//	--log-output    Output destination: stdout, stderr, or file path (default: stdout)
//	--log-add-source Add source file:line to log output (default: false)
//
// # Configuration via Config File
//
// The module also reads from the config file under the "logger" namespace:
//
//	logger:
//	  level: debug
//	  format: text
//	  output: /var/log/myapp.log
//	  add_source: true
//
// CLI flags take precedence over config file values.
//
// # File Output
//
// When output is a file path, the file is created with 0644 permissions.
// If the file cannot be created, the logger falls back to stdout with a warning.
package module
