package gaz

import "errors"

// =============================================================================
// Sentinel Errors
// =============================================================================

// DI subsystem errors.
var (
	// ErrDINotFound is returned when a requested service is not registered in the container.
	ErrDINotFound = errors.New("di: not found")

	// ErrDICycle is returned when a circular dependency is detected during resolution.
	ErrDICycle = errors.New("di: circular dependency")

	// ErrDIDuplicate is returned when attempting to register a service that already exists.
	ErrDIDuplicate = errors.New("di: duplicate registration")

	// ErrDINotSettable is returned when a struct field cannot be set during injection.
	ErrDINotSettable = errors.New("di: field not settable")

	// ErrDITypeMismatch is returned when a resolved service cannot be assigned to the target type.
	ErrDITypeMismatch = errors.New("di: type mismatch")

	// ErrDIAlreadyBuilt is returned when attempting to register after Build() was called.
	ErrDIAlreadyBuilt = errors.New("di: already built")

	// ErrDIInvalidProvider is returned when a provider function has invalid signature.
	ErrDIInvalidProvider = errors.New("di: invalid provider")
)

// Config subsystem errors.
var (
	// ErrConfigValidation is returned when config struct validation fails.
	// Use errors.Is(err, ErrConfigValidation) to check for validation errors.
	ErrConfigValidation = errors.New("config: validation failed")

	// ErrConfigNotFound is returned when a config key/namespace doesn't exist.
	// Use errors.Is(err, ErrConfigNotFound) to check for missing keys.
	ErrConfigNotFound = errors.New("config: key not found")
)

// Worker subsystem errors.
var (
	// ErrWorkerCircuitTripped indicates a worker exhausted its restart attempts
	// within the configured circuit window. The worker will not be restarted
	// until the application is restarted.
	ErrWorkerCircuitTripped = errors.New("worker: circuit breaker tripped")

	// ErrWorkerStopped indicates a worker stopped normally without error.
	// This is not an error condition - it signals clean shutdown.
	ErrWorkerStopped = errors.New("worker: stopped normally")

	// ErrWorkerCriticalFailed indicates a critical worker failed and exhausted
	// its restart attempts. This error triggers application shutdown.
	ErrWorkerCriticalFailed = errors.New("worker: critical worker failed")

	// ErrWorkerManagerRunning indicates an attempt to register a worker
	// after the manager has started.
	ErrWorkerManagerRunning = errors.New("worker: manager already running")
)

// Cron subsystem errors.
var (
	// ErrCronNotRunning indicates an operation was attempted on a scheduler
	// that is not running.
	ErrCronNotRunning = errors.New("cron: scheduler not running")
)

// Module errors (gaz-specific).
var (
	// ErrModuleDuplicate is returned when a module with the same name is registered twice.
	ErrModuleDuplicate = errors.New("gaz: duplicate module")

	// ErrConfigKeyCollision is returned when two providers register the same config key.
	ErrConfigKeyCollision = errors.New("gaz: config key collision")
)

// =============================================================================
// Backward Compatibility Aliases
// =============================================================================

// These aliases preserve backward compatibility with existing code that uses
// the short error names. They will be removed in a future release when all
// code is migrated to use the namespaced names (ErrDI*, ErrConfig*, etc.).

var (
	// ErrDuplicate is an alias for ErrDIDuplicate.
	// Deprecated: Use ErrDIDuplicate instead.
	ErrDuplicate = ErrDIDuplicate

	// ErrInvalidProvider is an alias for ErrDIInvalidProvider.
	// Deprecated: Use ErrDIInvalidProvider instead.
	ErrInvalidProvider = ErrDIInvalidProvider

	// ErrDuplicateModule is an alias for ErrModuleDuplicate.
	// Deprecated: Use ErrModuleDuplicate instead.
	ErrDuplicateModule = ErrModuleDuplicate

	// ErrNotFound is an alias for ErrDINotFound.
	// Deprecated: Use ErrDINotFound instead.
	ErrNotFound = ErrDINotFound

	// ErrCycle is an alias for ErrDICycle.
	// Deprecated: Use ErrDICycle instead.
	ErrCycle = ErrDICycle

	// ErrNotSettable is an alias for ErrDINotSettable.
	// Deprecated: Use ErrDINotSettable instead.
	ErrNotSettable = ErrDINotSettable

	// ErrTypeMismatch is an alias for ErrDITypeMismatch.
	// Deprecated: Use ErrDITypeMismatch instead.
	ErrTypeMismatch = ErrDITypeMismatch

	// ErrAlreadyBuilt is an alias for ErrDIAlreadyBuilt.
	// Deprecated: Use ErrDIAlreadyBuilt instead.
	ErrAlreadyBuilt = ErrDIAlreadyBuilt
)
