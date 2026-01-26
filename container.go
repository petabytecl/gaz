package gaz

import "sync"

// Container is the dependency injection container.
// Use New() to create a new container, register services with For[T](),
// and resolve with Resolve[T]().
type Container struct {
	// services stores registered services by name.
	// The value will hold serviceWrapper instances (added in later plans).
	services map[string]any

	// mu protects concurrent access to the services map.
	mu sync.RWMutex

	// built tracks whether Build() has been called.
	// Once built, the container is ready to resolve dependencies.
	built bool
}

// New creates a new empty Container.
// Register services using For[T](), then call Build() to prepare
// the container for resolution.
func New() *Container {
	return &Container{
		services: make(map[string]any),
	}
}
