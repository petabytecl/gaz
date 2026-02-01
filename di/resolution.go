package di

import "fmt"

// Resolve retrieves a service of type T from the container.
// By default, it looks up the service by its type name.
// Use Named() option to resolve by a custom registration name.
//
// Returns (T, error) - the resolved instance or an error if:
//   - Service not found (ErrNotFound)
//   - Circular dependency detected (ErrCycle)
//   - Provider returns an error (wrapped with resolution context)
//   - Type assertion fails (ErrTypeMismatch)
//
// Example:
//
//	// Resolve by type
//	db, err := di.Resolve[*DatabasePool](c)
//	if errors.Is(err, di.ErrNotFound) {
//	    // Handle missing dependency
//	}
//
//	// Resolve by name
//	primaryDB, err := di.Resolve[*sql.DB](c, di.Named("primary"))
func Resolve[T any](c *Container, opts ...ResolveOption) (T, error) {
	options := applyOptions(opts)

	name := options.name
	if name == "" {
		name = TypeName[T]()
	}

	// Get current chain from container (may be non-nil if called from provider)
	chain := c.getChain()

	// Continue resolution with current chain for proper cycle detection
	instance, err := c.ResolveByName(name, chain)
	if err != nil {
		var zero T
		return zero, err
	}

	// Type assertion
	result, ok := instance.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("%w: expected %s, got %T", ErrTypeMismatch, TypeName[T](), instance)
	}

	return result, nil
}

// MustResolve resolves a service or panics if resolution fails.
// Use only in test setup or main() initialization where failure is fatal.
//
// Example:
//
//	func TestSomething(t *testing.T) {
//	    c := di.NewTestContainer()
//	    di.For[*MockDB](c).Instance(&MockDB{})
//	    db := di.MustResolve[*MockDB](c) // panics if not found
//	}
func MustResolve[T any](c *Container, opts ...ResolveOption) T {
	result, err := Resolve[T](c, opts...)
	if err != nil {
		panic(fmt.Sprintf("di.MustResolve[%s]: %v", TypeName[T](), err))
	}
	return result
}
