// Package service provides a fluent Builder API for creating production-ready
// gaz applications with standard wiring and health check auto-registration.
//
// The service builder reduces boilerplate when creating production services.
// It wires common components (config, CLI, health checks) based on what's provided,
// with sensible defaults.
//
// Basic usage:
//
//	app, err := service.New().
//	    WithCmd(rootCmd).
//	    WithConfig(cfg).
//	    WithEnvPrefix("MYAPP").
//	    Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	app.Run(context.Background())
//
// The builder supports:
//   - WithCmd: Sets the cobra command for CLI integration
//   - WithConfig: Sets the config struct for loading
//   - WithEnvPrefix: Sets the global environment variable prefix
//   - WithOptions: Adds gaz.Option to the underlying app
//   - Use: Adds modules to be applied at Build()
//   - Build: Creates the App with all configured components
//
// Health check auto-registration:
//
// If the config struct implements health.HealthConfigProvider, the health module
// is automatically registered:
//
//	type AppConfig struct {
//	    Port   int
//	    Health health.Config
//	}
//
//	func (c *AppConfig) HealthConfig() health.Config {
//	    return c.Health
//	}
//
//	// Health module auto-registers because AppConfig implements HealthConfigProvider
//	app, _ := service.New().
//	    WithConfig(&AppConfig{Health: health.DefaultConfig()}).
//	    Build()
package service
