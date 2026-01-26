package gaz

import (
	"fmt"
	"strings"
	"sync"
)

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

// register adds a service to the container. Internal use only.
func (c *Container) register(name string, svc serviceWrapper) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = svc
}

// hasService checks if a service is registered. Internal use only.
func (c *Container) hasService(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.services[name]
	return ok
}

// resolveByName resolves a service by name, tracking the chain for cycle detection.
// This is the internal resolution method called by Resolve[T] and struct injection.
func (c *Container) resolveByName(name string, chain []string) (any, error) {
	// Cycle detection - check if we're already resolving this service
	for _, seen := range chain {
		if seen == name {
			cycle := append(chain, name)
			return nil, fmt.Errorf("%w: %s", ErrCycle, strings.Join(cycle, " -> "))
		}
	}

	// Look up service
	c.mu.RLock()
	svc, ok := c.services[name]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	wrapper := svc.(serviceWrapper)

	// Add current service to chain before getting instance
	newChain := append(chain, name)

	// Get instance (may resolve dependencies via provider)
	instance, err := wrapper.getInstance(c, newChain)
	if err != nil {
		// Wrap error with resolution context
		if len(chain) > 0 {
			return nil, fmt.Errorf("resolving %s -> %s: %w",
				strings.Join(chain, " -> "), name, err)
		}
		return nil, fmt.Errorf("resolving %s: %w", name, err)
	}

	return instance, nil
}
