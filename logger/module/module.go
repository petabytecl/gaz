// Package module provides the gaz.Module for logger configuration.
package module

import (
	"fmt"

	"github.com/petabytecl/gaz"
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
//
//nolint:ireturn // Module is the expected return type for gaz modules
func New() gaz.Module {
	defaultCfg := logger.DefaultConfig()

	return gaz.NewModule("logger").
		Flags(defaultCfg.Flags).
		Provide(func(c *gaz.Container) error {
			return gaz.For[logger.Config](c).Provider(func(c *gaz.Container) (logger.Config, error) {
				cfg := defaultCfg

				// Try to load from config manager if available
				pv, pvErr := gaz.Resolve[*gaz.ProviderValues](c)
				if pvErr == nil {
					if unmarshalErr := pv.UnmarshalKey(cfg.Namespace(), &cfg); unmarshalErr != nil {
						// Ignore error, use defaults (key may not exist)
						_ = unmarshalErr
					}
				}

				cfg.SetDefaults()
				if validateErr := cfg.Validate(); validateErr != nil {
					return cfg, fmt.Errorf("validate logger config: %w", validateErr)
				}

				return cfg, nil
			})
		}).
		Build()
}
