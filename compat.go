package gaz

import (
	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/worker"
)

// =============================================================================
// Backward Compatibility Layer
// Re-exports DI types so existing gaz code continues to work unchanged.
// =============================================================================

// Container is a type alias for di.Container.
// Use di.Container directly for new code.
type Container = di.Container

// NewContainer creates a new empty Container.
// For standalone DI usage, you may also use di.New() directly.
func NewContainer() *Container {
	return di.New()
}

// For returns a registration builder for type T.
// This wraps di.For[T] for backward compatibility with gaz.For[T].
//
// Example:
//
//	gaz.For[*MyService](c).Provider(NewMyService)
func For[T any](c *Container) *di.RegistrationBuilder[T] {
	return di.For[T](c)
}

// Resolve retrieves a service of type T from the container.
// This wraps di.Resolve[T] for backward compatibility.
func Resolve[T any](c *Container, opts ...di.ResolveOption) (T, error) {
	return di.Resolve[T](c, opts...)
}

// MustResolve resolves a service or panics if resolution fails.
// This wraps di.MustResolve[T] for backward compatibility.
func MustResolve[T any](c *Container, opts ...di.ResolveOption) T {
	return di.MustResolve[T](c, opts...)
}

// Has returns true if a service of type T is registered.
// This wraps di.Has[T] for backward compatibility.
func Has[T any](c *Container) bool {
	return di.Has[T](c)
}

// TypeName returns the fully-qualified type name for T.
// This wraps di.TypeName[T] for backward compatibility.
func TypeName[T any]() string {
	return di.TypeName[T]()
}

// typeName returns a string representation of the given reflect.Type.
// Internal use only - used by registerInstance for reflection-based registration.
func typeName(t any) string {
	return di.TypeNameReflect(t)
}

// Named resolves a service by its registered name instead of type.
// This wraps di.Named for backward compatibility.
func Named(name string) di.ResolveOption {
	return di.Named(name)
}

// =============================================================================
// Re-exported Types
// =============================================================================

// ResolveOption modifies resolution behavior.
type ResolveOption = di.ResolveOption

// RegistrationBuilder provides a fluent API for configuring services.
type RegistrationBuilder[T any] = di.RegistrationBuilder[T]

// ServiceWrapper is the interface for service lifecycle management.
type ServiceWrapper = di.ServiceWrapper

// =============================================================================
// Worker Interface Re-export
// =============================================================================

// Worker is a background task that runs continuously.
// Alias for worker.Worker for convenience.
type Worker = worker.Worker

// =============================================================================
// CronJob Interface Re-export
// =============================================================================

// CronJob is a type alias for cron.CronJob for convenience.
// Users can import this from the root gaz package instead of gaz/cron.
type CronJob = cron.CronJob
