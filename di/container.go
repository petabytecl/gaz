package di

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/petermattis/goid"
)

// Container is the dependency injection container.
// Use New() to create a new container, register services with For[T](),
// and resolve with Resolve[T]().
type Container struct {
	// services stores registered services by name.
	// The value holds a list of ServiceWrapper instances to support multi-binding.
	services map[string][]ServiceWrapper

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
		services:         make(map[string][]ServiceWrapper),
		resolutionChains: make(map[int64][]string),
		dependencyGraph:  make(map[string][]string),
	}
}

// Register adds a service to the container.
// Exported for use by gaz.App for reflection-based registration.
func (c *Container) Register(name string, svc ServiceWrapper) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = append(c.services[name], svc)
}

// ReplaceService replaces all services registered under the given name with the new service.
// This is used when RegistrationBuilder.Replace() is called.
func (c *Container) ReplaceService(name string, svc ServiceWrapper) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = []ServiceWrapper{svc}
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
	for name, wrappers := range c.services {
		for _, wrapper := range wrappers {
			fn(name, wrapper)
		}
	}
}

// GetService returns the first service wrapper by name.
// Returns nil, false if the service is not found.
// This is used by gaz.App for lifecycle management.
func (c *Container) GetService(name string) (ServiceWrapper, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	wrappers, ok := c.services[name]
	if !ok || len(wrappers) == 0 {
		return nil, false
	}
	return wrappers[0], true
}

// getGoroutineID returns a unique identifier for the current goroutine.
// This is used for tracking resolution chains per-goroutine.
func getGoroutineID() int64 {
	return goid.Get()
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
	for _, wrappers := range c.services {
		for _, wrapper := range wrappers {
			if wrapper.IsEager() {
				eagerServices = append(eagerServices, wrapper)
			}
		}
	}
	c.mu.RUnlock()

	// Instantiate each eager service
	for _, svc := range eagerServices {
		// We cannot use ResolveByName here because it might be ambiguous if there are multiple services with the same name.
		// Instead, we resolve the specific instance directly.
		// However, we still need to respect the resolution chain for cycle detection.
		//
		// For now, if we assume Eager services are usually singletons and uniquely named or we don't care about the name resolution conflict in ResolveByName (which now errors),
		// we must call GetInstance directly on the wrapper.

		// Manually manage chain for this specific service
		name := svc.Name()
		c.pushChain(name)

		// Check for cycle (though eager init usually starts chain)
		// But wait, ResolveByName managed the chain.
		// If we skip ResolveByName, we must manage it.

		// Actually, let's use a helper or just do it inline.
		_, err := svc.GetInstance(c, nil)
		c.popChain()

		if err != nil {
			return fmt.Errorf("di: building eager service %s: %w", svc.Name(), err)
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
	wrappers, ok := c.services[name]
	c.mu.RUnlock()

	if !ok || len(wrappers) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}

	if len(wrappers) > 1 {
		return nil, fmt.Errorf("%w: %s (found %d)", ErrAmbiguous, name, len(wrappers))
	}

	wrapper := wrappers[0]

	// Record dependency if we are being resolved by another service
	if len(chain) > 0 {
		parent := chain[len(chain)-1]
		c.recordDependency(parent, name)
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
			return nil, fmt.Errorf("di: resolving %s -> %s: %w",
				strings.Join(chain, " -> "), name, err)
		}
		return nil, fmt.Errorf("di: resolving %s: %w", name, err)
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

// ResolveAllByName resolves all services registered under the given name.
// Returns an empty slice if no services are found.
func (c *Container) ResolveAllByName(name string) ([]any, error) {
	c.mu.RLock()
	wrappers, ok := c.services[name]
	c.mu.RUnlock()

	if !ok || len(wrappers) == 0 {
		return []any{}, nil
	}

	var results []any
	chain := c.getChain()

	// Check for cycle on the group name itself if relevant, but really we care about individual instances.
	// However, if we are resolving "foo", and "foo" depends on "foo" list, that is a cycle.
	for _, seen := range chain {
		if seen == name {
			cycle := append(append([]string{}, chain...), name)
			return nil, fmt.Errorf("%w: %s (in ResolveAllByName)", ErrCycle, strings.Join(cycle, " -> "))
		}
	}

	for _, wrapper := range wrappers {
		// We use the wrapper's name (which matches 'name' here) for cycle tracking.
		c.pushChain(name)
		instance, err := wrapper.GetInstance(c, nil)
		c.popChain()

		if err != nil {
			return nil, fmt.Errorf("di: resolving element of %s: %w", name, err)
		}
		results = append(results, instance)
	}
	return results, nil
}

// ResolveGroup resolves all services belonging to the specified group.
// Returns an empty slice if no services are found.
func (c *Container) ResolveGroup(group string) ([]any, error) {
	c.mu.RLock()
	var candidates []ServiceWrapper
	for _, wrappers := range c.services {
		for _, wrapper := range wrappers {
			for _, g := range wrapper.Groups() {
				if g == group {
					candidates = append(candidates, wrapper)
					break
				}
			}
		}
	}
	c.mu.RUnlock()

	if len(candidates) == 0 {
		return []any{}, nil
	}

	var results []any
	chain := c.getChain()

	for _, wrapper := range candidates {
		name := wrapper.Name()

		// Cycle detection per item
		for _, seen := range chain {
			if seen == name {
				cycle := append(append([]string{}, chain...), name)
				return nil, fmt.Errorf("%w: %s (in ResolveGroup)", ErrCycle, strings.Join(cycle, " -> "))
			}
		}

		c.pushChain(name)
		instance, err := wrapper.GetInstance(c, nil)
		c.popChain()

		if err != nil {
			return nil, fmt.Errorf("di: resolving group candidate %s: %w", name, err)
		}
		results = append(results, instance)
	}
	return results, nil
}

// ResolveAllByType resolves all services that are assignable to the given type.
// This scans all registered services regardless of their registration name.
func (c *Container) ResolveAllByType(t reflect.Type) ([]any, error) {
	c.mu.RLock()
	// Snapshot the services to avoid holding lock during resolution
	var candidates []ServiceWrapper
	for _, wrappers := range c.services {
		for _, wrapper := range wrappers {
			// Check if the service type implements/assigns to T
			if wrapper.ServiceType().AssignableTo(t) {
				candidates = append(candidates, wrapper)
			}
		}
	}
	c.mu.RUnlock()

	if len(candidates) == 0 {
		return []any{}, nil
	}

	var results []any
	chain := c.getChain()

	for _, wrapper := range candidates {
		name := wrapper.Name()

		// Cycle detection per item
		for _, seen := range chain {
			if seen == name {
				cycle := append(append([]string{}, chain...), name)
				return nil, fmt.Errorf("%w: %s (in ResolveAllByType)", ErrCycle, strings.Join(cycle, " -> "))
			}
		}

		c.pushChain(name)
		instance, err := wrapper.GetInstance(c, nil)
		c.popChain()

		if err != nil {
			return nil, fmt.Errorf("di: resolving candidate %s for type %v: %w", name, t, err)
		}
		results = append(results, instance)
	}
	return results, nil
}
