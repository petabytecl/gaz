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

	// Start resolution with empty chain for cycle detection
	instance, err := c.resolveByName(name, nil)
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
