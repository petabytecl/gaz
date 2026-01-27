package gaz_test

import (
	"context"
	"fmt"

	"github.com/petabytecl/gaz"
)

// Server demonstrates a service with lifecycle hooks.
type Server struct {
	started bool
	port    int
}

// OnStart is called when the service starts.
func (s *Server) OnStart(_ context.Context) error {
	s.started = true
	return nil
}

// OnStop is called when the service stops.
func (s *Server) OnStop(_ context.Context) error {
	s.started = false
	return nil
}

// Database demonstrates a dependency that starts first.
type Database struct {
	connected bool
}

func (d *Database) OnStart(_ context.Context) error {
	d.connected = true
	return nil
}

func (d *Database) OnStop(_ context.Context) error {
	d.connected = false
	return nil
}

// AppServer depends on Database, demonstrating startup order.
type AppServer struct {
	db *Database
}

func (s *AppServer) OnStart(_ context.Context) error {
	// By the time AppServer starts, Database is already connected
	return nil
}

func (s *AppServer) OnStop(_ context.Context) error {
	return nil
}

// Example_lifecycle demonstrates registering services with lifecycle hooks.
// Services implementing Starter and Stopper interfaces have their
// OnStart/OnStop methods called automatically during App lifecycle.
func Example_lifecycle() {
	app := gaz.New()

	// Register Server - it implements Starter and Stopper interfaces
	app.ProvideSingleton(func(c *gaz.Container) (*Server, error) {
		return &Server{port: 8080}, nil
	})

	if err := app.Build(); err != nil {
		fmt.Println("error:", err)
		return
	}

	svc, _ := gaz.Resolve[*Server](app.Container())
	fmt.Println("server registered, port:", svc.port)
	// Output: server registered, port: 8080
}

// Example_lifecycleOrder demonstrates that dependencies start before dependents.
// When Service A depends on Service B, B's OnStart is called before A's OnStart.
// During shutdown, the order is reversed: A stops before B.
func Example_lifecycleOrder() {
	c := gaz.NewContainer()

	// Register Database (no dependencies)
	err := gaz.For[*Database](c).Provider(func(_ *gaz.Container) (*Database, error) {
		return &Database{}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Register AppServer (depends on Database)
	err = gaz.For[*AppServer](c).Provider(func(c *gaz.Container) (*AppServer, error) {
		db, err := gaz.Resolve[*Database](c)
		if err != nil {
			return nil, err
		}
		return &AppServer{db: db}, nil
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// Resolve to establish dependency graph
	server, _ := gaz.Resolve[*AppServer](c)
	fmt.Println("server has database:", server.db != nil)
	// Output: server has database: true
}
