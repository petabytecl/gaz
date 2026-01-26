package gaz

import "sync"

// serviceWrapper is the internal interface for all service types.
// It abstracts the differences between lazy singletons, transient services,
// eager singletons, and pre-built instances.
type serviceWrapper interface {
	// name returns the service registration name (type name or explicit name).
	name() string

	// typeName returns the full type name for error messages.
	typeName() string

	// isEager returns true if this service should be instantiated at Build() time.
	isEager() bool

	// getInstance returns the service instance, creating it if necessary.
	// The chain parameter tracks the resolution path for cycle detection.
	getInstance(c *Container, chain []string) (any, error)
}

// lazySingleton is the default service type - creates instance on first resolve,
// then caches it for all subsequent calls.
type lazySingleton[T any] struct {
	serviceName     string
	serviceTypeName string
	provider        func(*Container) (T, error)

	mu       sync.Mutex
	instance T
	built    bool
}

// newLazySingleton creates a new lazy singleton service wrapper.
func newLazySingleton[T any](name, typeName string, provider func(*Container) (T, error)) *lazySingleton[T] {
	return &lazySingleton[T]{
		serviceName:     name,
		serviceTypeName: typeName,
		provider:        provider,
	}
}

func (s *lazySingleton[T]) name() string {
	return s.serviceName
}

func (s *lazySingleton[T]) typeName() string {
	return s.serviceTypeName
}

func (s *lazySingleton[T]) isEager() bool {
	return false
}

func (s *lazySingleton[T]) getInstance(c *Container, chain []string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.built {
		return s.instance, nil
	}

	instance, err := s.provider(c)
	if err != nil {
		return nil, err
	}

	// Auto-inject struct fields tagged with gaz:"inject"
	if err := injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	s.instance = instance
	s.built = true
	return instance, nil
}

// transientService creates a new instance on every resolve call.
// No caching is performed.
type transientService[T any] struct {
	serviceName     string
	serviceTypeName string
	provider        func(*Container) (T, error)
}

// newTransient creates a new transient service wrapper.
func newTransient[T any](name, typeName string, provider func(*Container) (T, error)) *transientService[T] {
	return &transientService[T]{
		serviceName:     name,
		serviceTypeName: typeName,
		provider:        provider,
	}
}

func (s *transientService[T]) name() string {
	return s.serviceName
}

func (s *transientService[T]) typeName() string {
	return s.serviceTypeName
}

func (s *transientService[T]) isEager() bool {
	return false
}

func (s *transientService[T]) getInstance(c *Container, chain []string) (any, error) {
	// Always call provider - no caching for transient services
	instance, err := s.provider(c)
	if err != nil {
		return nil, err
	}

	// Auto-inject struct fields tagged with gaz:"inject"
	if err := injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	return instance, nil
}

// eagerSingleton is like lazySingleton but instantiates at Build() time.
// The isEager() method returns true so Build() knows to instantiate it.
type eagerSingleton[T any] struct {
	serviceName     string
	serviceTypeName string
	provider        func(*Container) (T, error)

	mu       sync.Mutex
	instance T
	built    bool
}

// newEagerSingleton creates a new eager singleton service wrapper.
func newEagerSingleton[T any](name, typeName string, provider func(*Container) (T, error)) *eagerSingleton[T] {
	return &eagerSingleton[T]{
		serviceName:     name,
		serviceTypeName: typeName,
		provider:        provider,
	}
}

func (s *eagerSingleton[T]) name() string {
	return s.serviceName
}

func (s *eagerSingleton[T]) typeName() string {
	return s.serviceTypeName
}

func (s *eagerSingleton[T]) isEager() bool {
	return true
}

func (s *eagerSingleton[T]) getInstance(c *Container, chain []string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.built {
		return s.instance, nil
	}

	instance, err := s.provider(c)
	if err != nil {
		return nil, err
	}

	// Auto-inject struct fields tagged with gaz:"inject"
	if err := injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	s.instance = instance
	s.built = true
	return instance, nil
}

// instanceService wraps a pre-built value. No provider is called.
// Used by .Instance(val) registration.
type instanceService[T any] struct {
	serviceName     string
	serviceTypeName string
	value           T
}

// newInstanceService creates a new instance service wrapper with a pre-built value.
func newInstanceService[T any](name, typeName string, value T) *instanceService[T] {
	return &instanceService[T]{
		serviceName:     name,
		serviceTypeName: typeName,
		value:           value,
	}
}

func (s *instanceService[T]) name() string {
	return s.serviceName
}

func (s *instanceService[T]) typeName() string {
	return s.serviceTypeName
}

func (s *instanceService[T]) isEager() bool {
	return false // Already instantiated, no need for Build()
}

func (s *instanceService[T]) getInstance(c *Container, chain []string) (any, error) {
	return s.value, nil
}
