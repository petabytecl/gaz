package gaz

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// decimalBase is used for parsing decimal digits from goroutine ID.
const decimalBase = 10

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

	// resolutionChains tracks active resolution chains per goroutine.
	// This enables cycle detection when providers call Resolve[T]().
	resolutionChains map[int64][]string
	chainMu          sync.Mutex

	// dependencyGraph stores the dependency graph as an adjacency list (parent -> children).
	// This is used for lifecycle management (ordered startup/shutdown).
	dependencyGraph map[string][]string
	graphMu         sync.RWMutex
}

// New creates a new empty Container.
// Register services using For[T](), then call Build() to prepare
// the container for resolution.
func New() *Container {
	return &Container{
		services:         make(map[string]any),
		resolutionChains: make(map[int64][]string),
		dependencyGraph:  make(map[string][]string),
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

// getGoroutineID returns a unique identifier for the current goroutine.
// This is used for tracking resolution chains per-goroutine.
func getGoroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// Parse goroutine ID from stack trace: "goroutine 123 [running]:"
	var id int64
	for i := 10; i < n; i++ { // Skip "goroutine "
		if buf[i] == ' ' {
			break
		}
		id = id*decimalBase + int64(buf[i]-'0')
	}
	return id
}

// getChain returns the current resolution chain for this goroutine.
func (c *Container) getChain() []string {
	c.chainMu.Lock()
	defer c.chainMu.Unlock()
	gid := getGoroutineID()
	return c.resolutionChains[gid]
}

// pushChain adds a service name to the resolution chain for this goroutine.
func (c *Container) pushChain(name string) {
	c.chainMu.Lock()
	defer c.chainMu.Unlock()
	gid := getGoroutineID()
	c.resolutionChains[gid] = append(c.resolutionChains[gid], name)
}

// popChain removes the last service from the resolution chain for this goroutine.
func (c *Container) popChain() {
	c.chainMu.Lock()
	defer c.chainMu.Unlock()
	gid := getGoroutineID()
	chain := c.resolutionChains[gid]
	if len(chain) > 0 {
		c.resolutionChains[gid] = chain[:len(chain)-1]
	}
	if len(c.resolutionChains[gid]) == 0 {
		delete(c.resolutionChains, gid)
	}
}

// Build instantiates all eager services and validates the container.
// Call this after all registrations and before any resolves.
// Returns an error if any eager service fails to instantiate.
// Build() is idempotent - calling it multiple times is safe.
//
// Example:
//
//	c := gaz.New()
//	gaz.For[*ConnectionPool](c).Eager().Provider(NewPool)
//	if err := c.Build(); err != nil {
//	    log.Fatalf("container build failed: %v", err)
//	}
func (c *Container) Build() error {
	c.mu.Lock()
	if c.built {
		c.mu.Unlock()
		return nil // Already built, idempotent
	}
	c.mu.Unlock()

	// Collect eager services
	var eagerServices []serviceWrapper
	c.mu.RLock()
	for _, svc := range c.services {
		wrapper, ok := svc.(serviceWrapper)
		if !ok {
			c.mu.RUnlock()
			return errors.New("invalid service wrapper type for service")
		}
		if wrapper.isEager() {
			eagerServices = append(eagerServices, wrapper)
		}
	}
	c.mu.RUnlock()

	// Instantiate each eager service
	for _, svc := range eagerServices {
		// Use resolveByName to ensure dependency tracking works correctly.
		// resolveByName manages the resolution chain, which is required for
		// cycle detection and dependency graph building.
		_, err := c.resolveByName(svc.name(), nil)
		if err != nil {
			return fmt.Errorf("building eager service %s: %w", svc.name(), err)
		}
	}

	c.mu.Lock()
	c.built = true
	c.mu.Unlock()

	return nil
}

// resolveByName resolves a service by name, tracking the chain for cycle detection.
// This is the internal resolution method called by Resolve[T] and struct injection.
func (c *Container) resolveByName(name string, _ []string) (any, error) {
	// Get current chain for this goroutine
	chain := c.getChain()

	// Cycle detection - check if we're already resolving this service
	for _, seen := range chain {
		if seen == name {
			cycle := append(append([]string{}, chain...), name)
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

	// Record dependency if we are being resolved by another service
	if len(chain) > 0 {
		parent := chain[len(chain)-1]
		c.recordDependency(parent, name)
	}

	wrapper, ok := svc.(serviceWrapper)
	if !ok {
		return nil, fmt.Errorf("invalid service wrapper type for %s", name)
	}

	// Add current service to chain before getting instance
	c.pushChain(name)
	defer c.popChain()

	// Get instance (may resolve dependencies via provider)
	// The provider may call Resolve[T]() which will check the chain
	instance, err := wrapper.getInstance(c, nil)
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

// recordDependency records a dependency between a parent service and a child service.
// This is used to build the dependency graph for lifecycle management.
func (c *Container) recordDependency(parent, child string) {
	c.graphMu.Lock()
	defer c.graphMu.Unlock()
	c.dependencyGraph[parent] = append(c.dependencyGraph[parent], child)
}

// getGraph returns a copy of the dependency graph.
// The returned map keys are parent services, and values are lists of child services.
func (c *Container) getGraph() map[string][]string {
	c.graphMu.RLock()
	defer c.graphMu.RUnlock()

	// Deep copy to prevent races if caller modifies result
	clone := make(map[string][]string, len(c.dependencyGraph))
	for k, v := range c.dependencyGraph {
		// Copy slice
		deps := make([]string, len(v))
		copy(deps, v)
		clone[k] = deps
	}
	return clone
}
