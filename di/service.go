package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// ServiceWrapper is the interface for service lifecycle management.
// Exported for use by gaz.App.
type ServiceWrapper interface {
	// Name returns the service registration name (type name or explicit name).
	Name() string

	// TypeName returns the full type name for error messages.
	TypeName() string

	// IsEager returns true if this service should be instantiated at Build() time.
	IsEager() bool

	// IsTransient returns true if this service creates a new instance on every resolve.
	IsTransient() bool

	// GetInstance returns the service instance, creating it if necessary.
	// The chain parameter tracks the resolution path for cycle detection.
	GetInstance(c *Container, chain []string) (any, error)

	// Start executes the OnStart hooks for the service.
	Start(context.Context) error

	// Stop executes the OnStop hooks for the service.
	Stop(context.Context) error

	// HasLifecycle returns true if the service has any lifecycle hooks registered.
	HasLifecycle() bool

	// ServiceType returns the reflect.Type of the service.
	// Used for type-checking without instantiation (e.g., interface implementation checks).
	ServiceType() reflect.Type

	// Groups returns the list of groups this service belongs to.
	Groups() []string
}

// baseService implements common functionality for all service wrappers.
// It handles metadata (name, type) only. Lifecycle is detected via interfaces.
type baseService struct {
	serviceName     string
	serviceTypeName string
	groups          []string
}

func (s *baseService) Name() string {
	return s.serviceName
}

func (s *baseService) TypeName() string {
	return s.serviceTypeName
}

func (s *baseService) Groups() []string {
	return s.groups
}

func (s *baseService) runStartLifecycle(ctx context.Context, instance any) error {
	if starter, ok := instance.(Starter); ok {
		if err := starter.OnStart(ctx); err != nil {
			return fmt.Errorf("di: service %s start failed: %w", s.serviceName, err)
		}
	}
	return nil
}

func (s *baseService) runStopLifecycle(ctx context.Context, instance any) error {
	if stopper, ok := instance.(Stopper); ok {
		if err := stopper.OnStop(ctx); err != nil {
			return fmt.Errorf("di: service %s stop failed: %w", s.serviceName, err)
		}
	}
	return nil
}

// hasLifecycleImpl is a helper for generic service wrappers to check for
// Starter/Stopper interfaces on T or *T.
func hasLifecycleImpl[T any]() bool {
	// Check if T implements interfaces (e.g. T is *Service)
	var zero T
	if _, ok := any(zero).(Starter); ok {
		return true
	}
	if _, ok := any(zero).(Stopper); ok {
		return true
	}

	// Check if *T implements interfaces (e.g. T is Service struct, methods on *Service)
	ptr := new(T)
	if _, ok := any(ptr).(Starter); ok {
		return true
	}
	if _, ok := any(ptr).(Stopper); ok {
		return true
	}

	return false
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
	groups ...string,
) *lazySingleton[T] {
	return &lazySingleton[T]{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			groups:          groups,
		},
		provider: provider,
	}
}

func (s *lazySingleton[T]) IsEager() bool {
	return false
}

func (s *lazySingleton[T]) IsTransient() bool {
	return false
}

func (s *lazySingleton[T]) HasLifecycle() bool {
	return hasLifecycleImpl[T]()
}

func (s *lazySingleton[T]) ServiceType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (s *lazySingleton[T]) GetInstance(c *Container, chain []string) (any, error) {
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

func (s *lazySingleton[T]) Start(ctx context.Context) error {
	if !s.built {
		return nil
	}
	return s.runStartLifecycle(ctx, s.instance)
}

func (s *lazySingleton[T]) Stop(ctx context.Context) error {
	if !s.built {
		return nil
	}
	return s.runStopLifecycle(ctx, s.instance)
}

// transientService creates a new instance on every resolve call.
// No caching is performed.
type transientService[T any] struct {
	serviceName     string
	serviceTypeName string
	groups          []string
	provider        func(*Container) (T, error)
}

// newTransient creates a new transient service wrapper.
func newTransient[T any](
	name, typeName string,
	provider func(*Container) (T, error),
	groups ...string,
) *transientService[T] {
	return &transientService[T]{
		serviceName:     name,
		serviceTypeName: typeName,
		groups:          groups,
		provider:        provider,
	}
}

func (s *transientService[T]) Name() string {
	return s.serviceName
}

func (s *transientService[T]) TypeName() string {
	return s.serviceTypeName
}

func (s *transientService[T]) Groups() []string {
	return s.groups
}

func (s *transientService[T]) IsEager() bool {
	return false
}

func (s *transientService[T]) IsTransient() bool {
	return true
}

func (s *transientService[T]) GetInstance(c *Container, chain []string) (any, error) {
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

func (s *transientService[T]) Start(context.Context) error { return nil }
func (s *transientService[T]) Stop(context.Context) error  { return nil }
func (s *transientService[T]) HasLifecycle() bool          { return false }

func (s *transientService[T]) ServiceType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

// eagerSingleton is like lazySingleton but instantiates at Build() time.
// The IsEager() method returns true so Build() knows to instantiate it.
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
	groups ...string,
) *eagerSingleton[T] {
	return &eagerSingleton[T]{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			groups:          groups,
		},
		provider: provider,
	}
}

func (s *eagerSingleton[T]) IsEager() bool {
	return true
}

func (s *eagerSingleton[T]) IsTransient() bool {
	return false
}

func (s *eagerSingleton[T]) GetInstance(c *Container, chain []string) (any, error) {
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

func (s *eagerSingleton[T]) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.built {
		// Should have been built, but just in case
		return nil
	}

	return s.runStartLifecycle(ctx, s.instance)
}

func (s *eagerSingleton[T]) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.built {
		return nil
	}

	return s.runStopLifecycle(ctx, s.instance)
}

func (s *eagerSingleton[T]) HasLifecycle() bool {
	return hasLifecycleImpl[T]()
}

func (s *eagerSingleton[T]) ServiceType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
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
	groups ...string,
) *instanceService[T] {
	return &instanceService[T]{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			groups:          groups,
		},
		value: value,
	}
}

func (s *instanceService[T]) IsEager() bool {
	return false // Already instantiated, no need for Build()
}

func (s *instanceService[T]) IsTransient() bool {
	return false
}

func (s *instanceService[T]) GetInstance(_ *Container, _ []string) (any, error) {
	return s.value, nil
}

func (s *instanceService[T]) Start(ctx context.Context) error {
	return s.runStartLifecycle(ctx, s.value)
}

func (s *instanceService[T]) Stop(ctx context.Context) error {
	return s.runStopLifecycle(ctx, s.value)
}

func (s *instanceService[T]) HasLifecycle() bool {
	return hasLifecycleImpl[T]()
}

func (s *instanceService[T]) ServiceType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

// =============================================================================
// Internal instance registration helper
// Used by App.registerInstance() for runtime type registration (WithConfig, etc.)
// =============================================================================

// instanceServiceAny is a non-generic version of instanceService for reflection-based registration.
type instanceServiceAny struct {
	baseService
	value any
}

// NewInstanceServiceAny creates a non-generic instance service wrapper.
// Exported for use by gaz.App for reflection-based registration (WithConfig, etc.)
func NewInstanceServiceAny(
	name, typeName string,
	value any,
	groups ...string,
) *instanceServiceAny {
	return &instanceServiceAny{
		baseService: baseService{
			serviceName:     name,
			serviceTypeName: typeName,
			groups:          groups,
		},
		value: value,
	}
}

func (s *instanceServiceAny) IsEager() bool     { return false }
func (s *instanceServiceAny) IsTransient() bool { return false }

func (s *instanceServiceAny) GetInstance(_ *Container, _ []string) (any, error) {
	return s.value, nil
}

func (s *instanceServiceAny) Start(ctx context.Context) error {
	return s.runStartLifecycle(ctx, s.value)
}

func (s *instanceServiceAny) Stop(ctx context.Context) error {
	return s.runStopLifecycle(ctx, s.value)
}

func (s *instanceServiceAny) HasLifecycle() bool {
	// Runtime check for Starter/Stopper interfaces
	if _, ok := s.value.(Starter); ok {
		return true
	}
	if _, ok := s.value.(Stopper); ok {
		return true
	}
	return false
}

func (s *instanceServiceAny) ServiceType() reflect.Type {
	return reflect.TypeOf(s.value)
}
