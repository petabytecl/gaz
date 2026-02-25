package di

// ContainerOptions configures resource limits for the container.
type ContainerOptions struct {
	// MaxServices is the maximum number of services that can be registered.
	// Default: 1000
	MaxServices int
}

// DefaultContainerOptions returns ContainerOptions with sensible defaults.
func DefaultContainerOptions() *ContainerOptions {
	return &ContainerOptions{
		MaxServices: 1000,
	}
}

// ResolveOption modifies resolution behavior.
type ResolveOption func(*resolveOptions)

// resolveOptions holds resolution configuration.
type resolveOptions struct {
	name string // Custom name to resolve (empty = use type name)
}

// Named resolves a service by its registered name instead of type.
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
