package gaz

// serviceScope defines the lifecycle scope for a registered service.
type serviceScope int

const (
	// scopeSingleton creates one instance for the container lifetime (default).
	scopeSingleton serviceScope = iota
	// scopeTransient creates a new instance on every resolution.
	scopeTransient
)

// RegistrationBuilder provides a fluent API for configuring and registering services.
// Start with For[T]() and chain methods like Named(), Transient(), Eager(), Replace(),
// then terminate with Provider() or Instance().
type RegistrationBuilder[T any] struct {
	container    *Container
	name         string       // Registration key (default: type name)
	typeName     string       // Type name for errors
	scope        serviceScope // singleton or transient
	lazy         bool         // lazy (default) or eager
	allowReplace bool         // allow overwriting existing
}

// For returns a registration builder for type T.
// This is the entry point for registering services in the container.
//
// Example:
//
//	err := gaz.For[*MyService](c).Provider(func(c *gaz.Container) (*MyService, error) {
//	    return &MyService{}, nil
//	})
func For[T any](c *Container) *RegistrationBuilder[T] {
	name := TypeName[T]()
	return &RegistrationBuilder[T]{
		container:    c,
		name:         name,
		typeName:     name,
		scope:        scopeSingleton,
		lazy:         true,
		allowReplace: false,
	}
}

// Named sets a custom registration name for the service.
// This allows multiple registrations of the same type with different names.
//
// Example:
//
//	gaz.For[*sql.DB](c).Named("primary").Provider(NewPrimaryDB)
//	gaz.For[*sql.DB](c).Named("replica").Provider(NewReplicaDB)
func (b *RegistrationBuilder[T]) Named(name string) *RegistrationBuilder[T] {
	b.name = name
	return b
}

// Transient marks the service as transient scope.
// A new instance will be created on every resolution.
// By default, services are singletons (one instance per container).
func (b *RegistrationBuilder[T]) Transient() *RegistrationBuilder[T] {
	b.scope = scopeTransient
	return b
}

// Eager marks the service for instantiation at Build() time.
// By default, services are lazy (instantiated on first resolution).
// Eager services are useful for services that must start at application startup.
func (b *RegistrationBuilder[T]) Eager() *RegistrationBuilder[T] {
	b.lazy = false
	return b
}

// Replace allows overwriting an existing registration with the same name.
// Without Replace(), duplicate registrations return ErrDuplicate.
// This is primarily useful for testing scenarios.
func (b *RegistrationBuilder[T]) Replace() *RegistrationBuilder[T] {
	b.allowReplace = true
	return b
}

// Provider registers a provider function that creates the service instance.
// The provider receives the container for resolving dependencies.
// Returns an error if a service with the same name already exists (unless Replace() was called).
//
// Example:
//
//	err := gaz.For[*MyService](c).Provider(func(c *gaz.Container) (*MyService, error) {
//	    dep, err := gaz.Resolve[*Dependency](c)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return &MyService{dep: dep}, nil
//	})
func (b *RegistrationBuilder[T]) Provider(fn func(*Container) (T, error)) error {
	// Check for duplicate registration
	if !b.allowReplace && b.container.hasService(b.name) {
		return ErrDuplicate
	}

	// Create appropriate service wrapper based on scope and lazy settings
	var svc serviceWrapper
	switch {
	case b.scope == scopeTransient:
		svc = newTransient(b.name, b.typeName, fn)
	case !b.lazy:
		svc = newEagerSingleton(b.name, b.typeName, fn)
	default:
		svc = newLazySingleton(b.name, b.typeName, fn)
	}

	b.container.register(b.name, svc)
	return nil
}

// ProviderFunc registers a simple provider function that creates the service instance.
// Unlike Provider(), this variant does not return an error from the provider.
// Use this for providers that cannot fail.
//
// Example:
//
//	err := gaz.For[*Config](c).ProviderFunc(func(c *gaz.Container) *Config {
//	    return &Config{Debug: true}
//	})
func (b *RegistrationBuilder[T]) ProviderFunc(fn func(*Container) T) error {
	return b.Provider(func(c *Container) (T, error) {
		return fn(c), nil
	})
}

// Instance registers a pre-built value as the service.
// No provider is called - the value is returned directly on resolution.
// This is useful for configuration objects or external dependencies.
//
// Example:
//
//	cfg := &Config{Debug: true}
//	err := gaz.For[*Config](c).Instance(cfg)
func (b *RegistrationBuilder[T]) Instance(val T) error {
	// Check for duplicate registration
	if !b.allowReplace && b.container.hasService(b.name) {
		return ErrDuplicate
	}

	svc := newInstanceService(b.name, b.typeName, val)
	b.container.register(b.name, svc)
	return nil
}
