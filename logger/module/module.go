// Package module provides the gaz.Module for logger configuration.
package module

import (
	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/internal/configuredmodule"
	"github.com/petabytecl/gaz/logger"
)

// New creates a logger module that provides logger.Config with CLI flags.
// The App resolves this config in Build() to create the Logger.
//
// Usage:
//
//	import loggermod "github.com/petabytecl/gaz/logger/module"
//
//	app := gaz.New(gaz.WithCobra(rootCmd))
//	app.Use(loggermod.New())
//
// Flags registered:
//
//	--log-level     Log level: debug, info, warn, error (default: info)
//	--log-format    Log format: text, json (default: text)
//	--log-output    Log output: stdout, stderr, or file path (default: stdout)
//	--log-add-source  Include source file:line in logs (default: false)
func New() gaz.Module {
	defaultCfg := logger.DefaultConfig()

	return gaz.NewModule("logger").
		Flags(defaultCfg.Flags).
		Provide(configuredmodule.ProvideDefaulted[logger.Config, *logger.Config](
			&defaultCfg,
			"logger config",
			"validate logger config",
		)).
		Build()
}
