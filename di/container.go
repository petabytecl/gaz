package di

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
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
	// The value will hold ServiceWrapper instances.
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
//
// Example:
//
//	c := di.New()
//	di.For[*Database](c).Provider(NewDatabase)
//	if err := c.Build(); err != nil {
//	    log.Fatal(err)
//	}
//	db, _ := di.Resolve[*Database](c)
func New() *Container {
	return &Container{
		services:         make(map[string]any),
		resolutionChains: make(map[int64][]string),
		dependencyGraph:  make(map[string][]string),
	}
}

// Register adds a service to the container.
// Exported for use by gaz.App for reflection-based registration.
func (c *Container) Register(name string, svc ServiceWrapper) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = svc
}

// HasService checks if a service is registered by name.
// Exported for use by gaz.App for duplicate detection.
func (c *Container) HasService(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.services[name]
	return ok
}

// ForEachService iterates over all registered services.
// The callback receives the service name and the service wrapper.
// This is used by gaz.App for lifecycle management.
func (c *Container) ForEachService(fn func(name string, svc ServiceWrapper)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for name, svc := range c.services {
		if wrapper, ok := svc.(ServiceWrapper); ok {
			fn(name, wrapper)
		}
	}
}

// GetService returns a service wrapper by name.
// Returns nil, false if the service is not found.
// This is used by gaz.App for lifecycle management.
func (c *Container) GetService(name string) (ServiceWrapper, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	svc, ok := c.services[name]
	if !ok {
		return nil, false
	}
	wrapper, ok := svc.(ServiceWrapper)
	return wrapper, ok
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
//	c := di.New()
//	di.For[*ConnectionPool](c).Eager().Provider(NewPool)
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
	var eagerServices []ServiceWrapper
	c.mu.RLock()
	for _, svc := range c.services {
		wrapper, ok := svc.(ServiceWrapper)
		if !ok {
			c.mu.RUnlock()
			return errors.New("invalid service wrapper type for service")
		}
		if wrapper.IsEager() {
			eagerServices = append(eagerServices, wrapper)
		}
	}
	c.mu.RUnlock()

	// Instantiate each eager service
	for _, svc := range eagerServices {
		// Use resolveByName to ensure dependency tracking works correctly.
		// resolveByName manages the resolution chain, which is required for
		// cycle detection and dependency graph building.
		_, err := c.ResolveByName(svc.Name(), nil)
		if err != nil {
			return fmt.Errorf("building eager service %s: %w", svc.Name(), err)
		}
	}

	c.mu.Lock()
	c.built = true
	c.mu.Unlock()

	return nil
}

// ResolveByName resolves a service by name, tracking the chain for cycle detection.
// Exported for use by gaz.App for config provider collection.
func (c *Container) ResolveByName(name string, _ []string) (any, error) {
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

	wrapper, ok := svc.(ServiceWrapper)
	if !ok {
		return nil, fmt.Errorf("invalid service wrapper type for %s", name)
	}

	// Add current service to chain before getting instance
	c.pushChain(name)
	defer c.popChain()

	// Get instance (may resolve dependencies via provider)
	// The provider may call Resolve[T]() which will check the chain
	instance, err := wrapper.GetInstance(c, nil)
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

// GetGraph returns a copy of the dependency graph.
// The returned map keys are parent services, and values are lists of child services.
func (c *Container) GetGraph() map[string][]string {
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

// List returns the names of all registered services.
// Names are returned in sorted order for deterministic output.
//
// Example:
//
//	for _, name := range c.List() {
//	    fmt.Println("Registered:", name)
//	}
func (c *Container) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	names := make([]string, 0, len(c.services))
	for name := range c.services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Has returns true if a service of type T is registered in the container.
//
// Example:
//
//	if di.Has[*Database](c) {
//	    db, _ := di.Resolve[*Database](c)
//	}
func Has[T any](c *Container) bool {
	return c.HasService(TypeName[T]())
}
