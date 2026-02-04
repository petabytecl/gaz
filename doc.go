// Package gaz provides a simple, type-safe dependency injection container
// with lifecycle management for Go applications.
//
// # Quick Start
//
// Create an application, register providers, build, and run:
//
//	app := gaz.New()
//	gaz.For[*Database](app.Container()).Provider(func(c *gaz.Container) (*Database, error) {
//	    return &Database{DSN: "postgres://..."}, nil
//	})
//	if err := app.Build(); err != nil {
//	    log.Fatal(err)
//	}
//	app.Run(context.Background())
//
// # Service Scopes
//
// Services can be registered with different scopes using the [For] fluent API:
//
//   - Singleton: One instance for container lifetime (default). Use [For][T].Provider().
//   - Transient: New instance on every resolution. Use [For][T].Transient().Provider().
//   - Eager: Singleton instantiated at [Container.Build] time. Use [For][T].Eager().Provider().
//
// Examples:
//
//	gaz.For[*Service](c).Provider(NewService)           // Lazy singleton (default)
//	gaz.For[*Service](c).Transient().Provider(NewService) // New instance each time
//	gaz.For[*Pool](c).Eager().Provider(NewPool)         // Created at Build() time
//
// # Lifecycle Management
//
// Services implementing [Starter] or [Stopper] get automatic lifecycle hooks.
// [Starter.OnStart] is called after [App.Build], and [Stopper.OnStop] is called
// during graceful shutdown.
//
//	type Server struct{}
//
//	func (s *Server) OnStart(ctx context.Context) error {
//	    return s.listen()
//	}
//
//	func (s *Server) OnStop(ctx context.Context) error {
//	    return s.shutdown()
//	}
//
// Hooks are called in dependency order: dependencies start first and stop last.
// Shutdown timeout is configurable via [WithShutdownTimeout], with per-hook
// limits via [WithPerHookTimeout].
//
// # Configuration
//
// Load configuration from files, environment variables, and CLI flags:
//
//	type Config struct {
//	    Port int    `mapstructure:"port" validate:"required,min=1"`
//	    Host string `mapstructure:"host" validate:"required"`
//	}
//
//	import "github.com/petabytecl/gaz/config"
//
//	app := gaz.New()
//	app.WithConfig(&Config{}, config.WithEnvPrefix("APP"))
//
// The config struct is automatically registered in the container. Use
// [ConfigManager] for advanced scenarios. Config values are validated
// using struct tags with go-playground/validator.
//
// # Health Checks
//
// The health subpackage provides HTTP health check endpoints:
//
//	import healthmod "github.com/petabytecl/gaz/health/module"
//
//	app.Use(healthmod.New())
//
// See the health package for [health.Manager], readiness, and liveness probes.
//
// # Resolution
//
// Resolve dependencies from the container using [Resolve]:
//
//	db, err := gaz.Resolve[*Database](c)
//	if errors.Is(err, gaz.ErrDINotFound) {
//	    // Handle missing dependency
//	}
//
// For named registrations, use the [Named] option:
//
//	primary, _ := gaz.Resolve[*sql.DB](c, gaz.Named("primary"))
//	replica, _ := gaz.Resolve[*sql.DB](c, gaz.Named("replica"))
//
// # Modules
//
// Group related providers into reusable modules:
//
//	app.Module("database",
//	    func(c *gaz.Container) error {
//	        return gaz.For[*Pool](c).Provider(NewPool)
//	    },
//	    func(c *gaz.Container) error {
//	        return gaz.For[*UserRepo](c).Provider(NewUserRepo)
//	    },
//	)
//
// See [App.Module] for details.
package gaz
