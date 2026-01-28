// Package di provides a lightweight, type-safe dependency injection container.
//
// The di package can be used standalone without the gaz framework,
// or as part of gaz.App for full application lifecycle management.
//
// # Quick Start
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
// Services can implement Starter/Stopper interfaces or use OnStart/OnStop:
//
//	di.For[*Server](c).OnStart(func(ctx context.Context, s *Server) error {
//	    return s.ListenAndServe()
//	}).Provider(NewServer)
package di
