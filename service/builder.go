package service

import (
	"errors"
	"fmt"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
	"github.com/petabytecl/gaz/health"
	"github.com/spf13/cobra"
)

// Builder constructs a gaz.App with fluent configuration.
// Use New() to create a Builder, then chain methods to configure,
// and finally call Build() to create the App.
type Builder struct {
	cmd       *cobra.Command
	config    any
	envPrefix string
	opts      []gaz.Option
	modules   []gaz.Module
	errs      []error
}

// New returns a new service Builder.
//
// Example:
//
//	app, err := service.New().
//	    WithCmd(rootCmd).
//	    WithConfig(cfg).
//	    Build()
func New() *Builder {
	return &Builder{}
}

// WithCmd sets the cobra command for CLI integration.
// The command's lifecycle hooks will be configured to start/stop the app.
func (b *Builder) WithCmd(cmd *cobra.Command) *Builder {
	b.cmd = cmd
	return b
}

// WithConfig sets the config struct for loading.
// The target must be a pointer to a struct.
// If the config implements health.HealthConfigProvider, the health module
// is automatically registered.
func (b *Builder) WithConfig(cfg any) *Builder {
	b.config = cfg
	return b
}

// WithEnvPrefix sets the global environment variable prefix.
// For example, if prefix is "MYAPP", then the key "database.host"
// will look for MYAPP_DATABASE__HOST environment variable.
func (b *Builder) WithEnvPrefix(prefix string) *Builder {
	b.envPrefix = prefix
	return b
}

// WithOptions adds gaz.Option to the underlying app.
// These options are applied when the App is created.
func (b *Builder) WithOptions(opts ...gaz.Option) *Builder {
	b.opts = append(b.opts, opts...)
	return b
}

// Use adds a module to be applied at Build().
// Modules are applied in the order they are added.
func (b *Builder) Use(m gaz.Module) *Builder {
	b.modules = append(b.modules, m)
	return b
}

// Build creates the App with all configured components.
// Returns error if any configuration is invalid.
//
// Build performs the following:
//  1. Creates gaz.App with provided options
//  2. Configures config loading with env prefix if set
//  3. Applies all registered modules
//  4. Auto-registers health module if config implements HealthConfigProvider
//  5. Attaches cobra command if provided
func (b *Builder) Build() (*gaz.App, error) {
	// Return early if any errors were collected
	if len(b.errs) > 0 {
		return nil, errors.Join(b.errs...)
	}

	// Create app with options
	app := gaz.New(b.opts...)

	// Configure config if provided
	if b.config != nil {
		var configOpts []config.Option
		if b.envPrefix != "" {
			configOpts = append(configOpts, config.WithEnvPrefix(b.envPrefix))
		}
		app.WithConfig(b.config, configOpts...)
	}

	// Apply modules
	for _, m := range b.modules {
		app.Use(m)
	}

	// Auto-register health module if config implements HealthConfigProvider
	if b.config != nil {
		if hp, ok := b.config.(health.HealthConfigProvider); ok {
			cfg := hp.HealthConfig()

			// Register health.Config in container so health.Module can resolve it
			if err := gaz.For[health.Config](app.Container()).Instance(cfg); err != nil {
				return nil, fmt.Errorf("register health config: %w", err)
			}

			// Create and apply health module
			healthModule := gaz.NewModule("health").
				Provide(health.Module).
				Build()
			app.Use(healthModule)
		}
	}

	// Attach cobra if provided
	if b.cmd != nil {
		app.WithCobra(b.cmd)
	}

	return app, nil
}
