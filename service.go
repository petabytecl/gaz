package gaz

import (
	"context"
	"fmt"
	"sync"
)

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

	// start executes the OnStart hooks for the service.
	start(context.Context) error

	// stop executes the OnStop hooks for the service.
	stop(context.Context) error

	// hasLifecycle returns true if the service has any lifecycle hooks registered.
	hasLifecycle() bool
}

// baseService implements common functionality for all service wrappers.
// It handles metadata (name, type) and lifecycle hook execution.
type baseService struct {
	serviceName     string
	serviceTypeName string
	startHooks      []func(context.Context, any) error
	stopHooks       []func(context.Context, any) error
}

func (s *baseService) name() string {
	return s.serviceName
}

func (s *baseService) typeName() string {
	return s.serviceTypeName
}

func (s *baseService) hasLifecycle() bool {
	return len(s.startHooks) > 0 || len(s.stopHooks) > 0
}

func (s *baseService) runStartHooks(ctx context.Context, instance any) error {
	for _, hook := range s.startHooks {
		if err := hook(ctx, instance); err != nil {
			return err
		}
	}
	return nil
}

func (s *baseService) runStopHooks(ctx context.Context, instance any) error {
	for i := len(s.stopHooks) - 1; i >= 0; i-- {
		if err := s.stopHooks[i](ctx, instance); err != nil {
			return err
		}
	}
	return nil
}

func (s *baseService) runStartLifecycle(ctx context.Context, instance any) error {
	if err := s.runStartHooks(ctx, instance); err != nil {
		return err
	}

	if starter, ok := instance.(Starter); ok {
		if err := starter.OnStart(ctx); err != nil {
			return fmt.Errorf("service %s: start failed: %w", s.serviceName, err)
		}
	}
	return nil
}

func (s *baseService) runStopLifecycle(ctx context.Context, instance any) error {
	if stopper, ok := instance.(Stopper); ok {
		if err := stopper.OnStop(ctx); err != nil {
			return fmt.Errorf("service %s: stop failed: %w", s.serviceName, err)
		}
	}

	return s.runStopHooks(ctx, instance)
}

// lazySingleton is the default service type - creates instance on first resolve,
// then caches it for all subsequent calls.
type lazySingleton[T any] struct {
	baseService
	provider func(*Container) (T, error)

	mu       sync.Mutex
	instance T
	built    bool
}

// newLazySingleton creates a new lazy singleton service wrapper.
func newLazySingleton[T any](
	name, typeName string,
	provider func(*Container) (T, error),
	startHooks, stopHooks []func(context.Context, any) error,
) *lazySingleton[T] {
	return &lazySingleton[T]{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			startHooks:      startHooks,
			stopHooks:       stopHooks,
		},
		provider: provider,
	}
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
	if err = injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	s.instance = instance
	s.built = true
	return instance, nil
}

func (s *lazySingleton[T]) start(ctx context.Context) error {
	if !s.built {
		return nil
	}
	return s.runStartHooks(ctx, s.instance)
}

func (s *lazySingleton[T]) stop(ctx context.Context) error {
	if !s.built {
		return nil
	}
	return s.runStopHooks(ctx, s.instance)
}

// transientService creates a new instance on every resolve call.
// No caching is performed.
type transientService[T any] struct {
	serviceName     string
	serviceTypeName string
	provider        func(*Container) (T, error)
}

// newTransient creates a new transient service wrapper.
func newTransient[T any](
	name, typeName string,
	provider func(*Container) (T, error),
) *transientService[T] {
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
	if err = injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	return instance, nil
}

func (s *transientService[T]) start(context.Context) error { return nil }
func (s *transientService[T]) stop(context.Context) error  { return nil }
func (s *transientService[T]) hasLifecycle() bool          { return false }

// eagerSingleton is like lazySingleton but instantiates at Build() time.
// The isEager() method returns true so Build() knows to instantiate it.
type eagerSingleton[T any] struct {
	baseService
	provider func(*Container) (T, error)

	mu       sync.Mutex
	instance T
	built    bool
}

// newEagerSingleton creates a new eager singleton service wrapper.
func newEagerSingleton[T any](
	name, typeName string,
	provider func(*Container) (T, error),
	startHooks, stopHooks []func(context.Context, any) error,
) *eagerSingleton[T] {
	return &eagerSingleton[T]{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			startHooks:      startHooks,
			stopHooks:       stopHooks,
		},
		provider: provider,
	}
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
	if err = injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	s.instance = instance
	s.built = true
	return instance, nil
}

func (s *eagerSingleton[T]) start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.built {
		// Should have been built, but just in case
		return nil
	}

	return s.runStartLifecycle(ctx, s.instance)
}

func (s *eagerSingleton[T]) stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.built {
		return nil
	}

	return s.runStopLifecycle(ctx, s.instance)
}

func (s *eagerSingleton[T]) hasLifecycle() bool {
	return true
}

// instanceService wraps a pre-built value. No provider is called.
// Used by .Instance(val) registration.
type instanceService[T any] struct {
	baseService
	value T
}

// newInstanceService creates a new instance service wrapper with a pre-built value.
func newInstanceService[T any](
	name, typeName string,
	value T,
	startHooks, stopHooks []func(context.Context, any) error,
) *instanceService[T] {
	return &instanceService[T]{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			startHooks:      startHooks,
			stopHooks:       stopHooks,
		},
		value: value,
	}
}

func (s *instanceService[T]) isEager() bool {
	return false // Already instantiated, no need for Build()
}

func (s *instanceService[T]) getInstance(_ *Container, _ []string) (any, error) {
	return s.value, nil
}

func (s *instanceService[T]) start(ctx context.Context) error {
	return s.runStartLifecycle(ctx, s.value)
}

func (s *instanceService[T]) stop(ctx context.Context) error {
	return s.runStopLifecycle(ctx, s.value)
}

func (s *instanceService[T]) hasLifecycle() bool {
	return true
}

// =============================================================================
// Non-generic service wrappers for reflection-based registration
// These are used by App.ProvideSingleton/ProvideTransient/etc. which use
// reflection to extract types from provider functions.
// =============================================================================

// lazySingletonAny is a non-generic version of lazySingleton for reflection-based registration.
type lazySingletonAny struct {
	baseService
	provider func(*Container) (any, error)

	mu       sync.Mutex
	instance any
	built    bool
}

func newLazySingletonAny(
	name, typeName string,
	provider func(*Container) (any, error),
	startHooks, stopHooks []func(context.Context, any) error,
) *lazySingletonAny {
	return &lazySingletonAny{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			startHooks:      startHooks,
			stopHooks:       stopHooks,
		},
		provider: provider,
	}
}

func (s *lazySingletonAny) isEager() bool { return false }

func (s *lazySingletonAny) getInstance(c *Container, chain []string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.built {
		return s.instance, nil
	}

	instance, err := s.provider(c)
	if err != nil {
		return nil, err
	}

	if err = injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	s.instance = instance
	s.built = true
	return instance, nil
}

func (s *lazySingletonAny) start(ctx context.Context) error {
	if !s.built {
		return nil
	}
	return s.runStartHooks(ctx, s.instance)
}

func (s *lazySingletonAny) stop(ctx context.Context) error {
	if !s.built {
		return nil
	}
	return s.runStopHooks(ctx, s.instance)
}

// transientServiceAny is a non-generic version of transientService.
type transientServiceAny struct {
	serviceName     string
	serviceTypeName string
	provider        func(*Container) (any, error)
}

func newTransientAny(
	name, typeName string,
	provider func(*Container) (any, error),
) *transientServiceAny {
	return &transientServiceAny{
		serviceName:     name,
		serviceTypeName: typeName,
		provider:        provider,
	}
}

func (s *transientServiceAny) name() string     { return s.serviceName }
func (s *transientServiceAny) typeName() string { return s.serviceTypeName }
func (s *transientServiceAny) isEager() bool    { return false }

func (s *transientServiceAny) getInstance(c *Container, chain []string) (any, error) {
	instance, err := s.provider(c)
	if err != nil {
		return nil, err
	}

	if err = injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	return instance, nil
}

func (s *transientServiceAny) start(context.Context) error { return nil }
func (s *transientServiceAny) stop(context.Context) error  { return nil }
func (s *transientServiceAny) hasLifecycle() bool          { return false }

// eagerSingletonAny is a non-generic version of eagerSingleton.
type eagerSingletonAny struct {
	baseService
	provider func(*Container) (any, error)

	mu       sync.Mutex
	instance any
	built    bool
}

func newEagerSingletonAny(
	name, typeName string,
	provider func(*Container) (any, error),
	startHooks, stopHooks []func(context.Context, any) error,
) *eagerSingletonAny {
	return &eagerSingletonAny{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			startHooks:      startHooks,
			stopHooks:       stopHooks,
		},
		provider: provider,
	}
}

func (s *eagerSingletonAny) isEager() bool { return true }

func (s *eagerSingletonAny) getInstance(c *Container, chain []string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.built {
		return s.instance, nil
	}

	instance, err := s.provider(c)
	if err != nil {
		return nil, err
	}

	if err = injectStruct(c, instance, chain); err != nil {
		return nil, err
	}

	s.instance = instance
	s.built = true
	return instance, nil
}

func (s *eagerSingletonAny) start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.built {
		return nil
	}

	return s.runStartLifecycle(ctx, s.instance)
}

func (s *eagerSingletonAny) stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.built {
		return nil
	}

	return s.runStopLifecycle(ctx, s.instance)
}

func (s *eagerSingletonAny) hasLifecycle() bool {
	return true
}

// instanceServiceAny is a non-generic version of instanceService.
type instanceServiceAny struct {
	baseService
	value any
}

func newInstanceServiceAny(
	name, typeName string,
	value any,
	startHooks, stopHooks []func(context.Context, any) error,
) *instanceServiceAny {
	return &instanceServiceAny{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			startHooks:      startHooks,
			stopHooks:       stopHooks,
		},
		value: value,
	}
}

func (s *instanceServiceAny) isEager() bool { return false }

func (s *instanceServiceAny) getInstance(_ *Container, _ []string) (any, error) {
	return s.value, nil
}

func (s *instanceServiceAny) start(ctx context.Context) error {
	return s.runStartLifecycle(ctx, s.value)
}

func (s *instanceServiceAny) stop(ctx context.Context) error {
	return s.runStopLifecycle(ctx, s.value)
}

func (s *instanceServiceAny) hasLifecycle() bool {
	return true
}
