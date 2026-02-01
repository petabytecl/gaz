// Package di provides a lightweight, type-safe dependency injection container.
//
// # When to Use di vs gaz
//
// Most applications should import "github.com/petabytecl/gaz" directly:
//
//	import "github.com/petabytecl/gaz"
//
//	app := gaz.New()
//	gaz.For[*MyService](app.Container()).Provider(NewMyService)
//	app.Build()
//	app.Run(ctx)
//
// The gaz package re-exports all di types (Container, For, Resolve, Has, etc.)
// and adds application lifecycle, configuration, workers, cron, health, and eventbus.
//
// Import di directly only when:
//   - You need standalone DI without gaz.App lifecycle
//   - You're building a library that depends only on the container
//   - You want to minimize import surface in tests
//
// # Re-exported Types
//
// The following types are re-exported by the gaz package:
//   - Container → gaz.Container
//   - For[T]() → gaz.For[T]()
//   - Resolve[T]() → gaz.Resolve[T]()
//   - Has[T]() → gaz.Has[T]()
//   - Named() → gaz.Named()
//   - RegistrationBuilder → gaz.RegistrationBuilder
//   - ServiceWrapper → gaz.ServiceWrapper
//   - TypeName[T]() → gaz.TypeName[T]()
//
// For full application development, prefer the gaz package.
//
// # Error Handling
//
// DI errors use the "di: action" format and are defined in this package.
// The gaz package re-exports them with ErrDI* naming:
//
//	di.ErrNotFound     → gaz.ErrDINotFound
//	di.ErrCycle        → gaz.ErrDICycle
//	di.ErrDuplicate    → gaz.ErrDIDuplicate
//	di.ErrNotSettable  → gaz.ErrDINotSettable
//	di.ErrTypeMismatch → gaz.ErrDITypeMismatch
//	di.ErrAlreadyBuilt → gaz.ErrDIAlreadyBuilt
//	di.ErrInvalidProvider → gaz.ErrDIInvalidProvider
//
// Both forms work with errors.Is:
//
//	if errors.Is(err, di.ErrNotFound) { ... }
//	if errors.Is(err, gaz.ErrDINotFound) { ... } // same error
//
// # Container Usage
//
// Create a container, register services, and resolve them:
//
//	c := di.New()
//
//	di.For[*Database](c).Provider(func(c *di.Container) (*Database, error) {
//	    return NewDatabase("postgres://...")
//	})
//
//	if err := c.Build(); err != nil {
//	    log.Fatal(err)
//	}
//
//	db, err := di.Resolve[*Database](c)
//
// # Registration Patterns
//
// Services can be registered as singletons (default), transient, or eager:
//
//	di.For[*Config](c).Instance(cfg)                // Pre-built instance
//	di.For[*Service](c).Provider(NewService)        // Lazy singleton (default)
//	di.For[*Pool](c).Eager().Provider(NewPool)      // Eager singleton
//	di.For[*Request](c).Transient().Provider(fn)    // New instance each time
//
// # Named Services
//
// Multiple services of the same type can be registered with different names:
//
//	di.For[*sql.DB](c).Named("primary").Provider(NewPrimaryDB)
//	di.For[*sql.DB](c).Named("replica").Provider(NewReplicaDB)
//	primary, _ := di.Resolve[*sql.DB](c, di.Named("primary"))
//
// # Lifecycle Hooks
//
// Services implementing Starter or Stopper interfaces automatically participate
// in the container's lifecycle. No fluent API needed - interface implementation
// is the sole mechanism.
//
//	type Server struct { addr string }
//
//	func (s *Server) OnStart(ctx context.Context) error {
//	    // Called after container Build() when service is instantiated
//	    return s.ListenAndServe()
//	}
//
//	func (s *Server) OnStop(ctx context.Context) error {
//	    // Called during graceful shutdown
//	    return s.Shutdown(ctx)
//	}
//
//	// Registration is simple - no lifecycle methods needed
//	di.For[*Server](c).Provider(NewServer)
//
// See the gaz package for full application examples with lifecycle management.
package di
