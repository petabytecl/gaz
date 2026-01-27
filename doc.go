// Package gaz provides a simple, type-safe dependency injection container
// with lifecycle management for Go applications.
//
// # Quick Start
//
// Create an application, register providers, build, and run:
//
//	app := gaz.New()
//	app.ProvideSingleton(func(c *gaz.Container) (*Database, error) {
//	    return &Database{DSN: "postgres://..."}, nil
//	})
//	if err := app.Build(); err != nil {
//	    log.Fatal(err)
//	}
//	app.Run(context.Background())
//
// # Service Scopes
//
// Services can be registered with different scopes:
//
//   - Singleton: One instance for container lifetime (default). Use [App.ProvideSingleton].
//   - Transient: New instance on every resolution. Use [App.ProvideTransient].
//   - Eager: Singleton instantiated at [App.Build] time. Use [App.ProvideEager].
//
// For low-level control, use [For] with the [Container]:
//
//	gaz.For[*Service](c).Transient().Provider(NewService)
//	gaz.For[*Pool](c).Eager().Provider(NewPool)
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
//	app := gaz.New()
//	app.WithConfig(&Config{}, gaz.WithEnvPrefix("APP"))
//
// The config struct is automatically registered in the container. Use
// [ConfigManager] for advanced scenarios. Config values are validated
// using struct tags with go-playground/validator.
//
// # Health Checks
//
// The health subpackage provides HTTP health check endpoints:
//
//	import "github.com/petabytecl/gaz/health"
//
//	app.Module(health.NewModule(health.Config{Port: 8081}))
//
// See the health package for [health.Manager], readiness, and liveness probes.
//
// # Resolution
//
// Resolve dependencies from the container using [Resolve]:
//
//	db, err := gaz.Resolve[*Database](c)
//	if errors.Is(err, gaz.ErrNotFound) {
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
//	type DatabaseModule struct{}
//
//	func (m *DatabaseModule) Name() string { return "database" }
//
//	func (m *DatabaseModule) Register(app *gaz.App) error {
//	    app.ProvideSingleton(NewPool)
//	    app.ProvideSingleton(NewUserRepo)
//	    return nil
//	}
//
//	app.Module(&DatabaseModule{})
//
// See [Module] interface for details.
package gaz
