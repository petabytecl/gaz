package di

// ResolveOption modifies resolution behavior.
// Options are passed to Resolve[T]() to customize the resolution process.
type ResolveOption func(*resolveOptions)

// resolveOptions holds resolution configuration.
type resolveOptions struct {
	name string // Custom name to resolve (empty = use type name)
}

// Named resolves a service by its registered name instead of type.
// Use this when you have multiple registrations of the same type.
//
// Example:
//
//	primaryDB, err := di.Resolve[*sql.DB](c, di.Named("primary"))
//	replicaDB, err := di.Resolve[*sql.DB](c, di.Named("replica"))
func Named(name string) ResolveOption {
	return func(o *resolveOptions) {
		o.name = name
	}
}

// applyOptions creates resolveOptions from variadic options.
func applyOptions(opts []ResolveOption) *resolveOptions {
	o := &resolveOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
