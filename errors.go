package gaz

import (
	"errors"
	"fmt"
	"strings"

	"github.com/petabytecl/gaz/config"
	"github.com/petabytecl/gaz/cron"
	"github.com/petabytecl/gaz/di"
	"github.com/petabytecl/gaz/worker"
)

// =============================================================================
// Sentinel Errors
// =============================================================================

// DI subsystem errors.
// These are re-exports from the di package with standardized ErrDI* naming.
// Use errors.Is(err, gaz.ErrDI*) to check for these errors.
//
// Note: Due to Go's import cycle constraints (gaz imports di), the canonical
// source of DI errors is di/errors.go. These are aliases that point to the
// same error values, ensuring errors.Is compatibility.
var (
	// ErrDINotFound is returned when a requested service is not registered in the container.
	// Check with: errors.Is(err, gaz.ErrDINotFound).
	ErrDINotFound = di.ErrNotFound

	// ErrDICycle is returned when a circular dependency is detected during resolution.
	// Check with: errors.Is(err, gaz.ErrDICycle).
	ErrDICycle = di.ErrCycle

	// ErrDIDuplicate is returned when attempting to register a service that already exists.
	// Check with: errors.Is(err, gaz.ErrDIDuplicate).
	ErrDIDuplicate = di.ErrDuplicate

	// ErrDINotSettable is returned when a struct field cannot be set during injection.
	// Check with: errors.Is(err, gaz.ErrDINotSettable).
	ErrDINotSettable = di.ErrNotSettable

	// ErrDITypeMismatch is returned when a resolved service cannot be assigned to the target type.
	// Check with: errors.Is(err, gaz.ErrDITypeMismatch).
	ErrDITypeMismatch = di.ErrTypeMismatch

	// ErrDIAlreadyBuilt is returned when attempting to register after Build() was called.
	// Check with: errors.Is(err, gaz.ErrDIAlreadyBuilt).
	ErrDIAlreadyBuilt = di.ErrAlreadyBuilt

	// ErrDIInvalidProvider is returned when a provider function has invalid signature.
	// Check with: errors.Is(err, gaz.ErrDIInvalidProvider).
	ErrDIInvalidProvider = di.ErrInvalidProvider
)

// Config subsystem errors.
// These are re-exports from the config package with standardized ErrConfig* naming.
// Use errors.Is(err, gaz.ErrConfig*) to check for these errors.
//
// Note: Due to Go's import cycle constraints, the canonical source of config
// errors is config/errors.go. These are aliases that point to the same error
// values, ensuring errors.Is compatibility.
var (
	// ErrConfigValidation is returned when config struct validation fails.
	// Check with: errors.Is(err, gaz.ErrConfigValidation) or errors.Is(err, config.ErrConfigValidation).
	ErrConfigValidation = config.ErrConfigValidation

	// ErrConfigNotFound is returned when a config key/namespace doesn't exist.
	// Check with: errors.Is(err, gaz.ErrConfigNotFound) or errors.Is(err, config.ErrKeyNotFound).
	ErrConfigNotFound = config.ErrKeyNotFound
)

// Worker subsystem errors.
// These are re-exports from the worker package with standardized ErrWorker* naming.
// Use errors.Is(err, gaz.ErrWorker*) to check for these errors.
//
// Note: Due to Go's import cycle constraints, the canonical source of worker
// errors is worker/errors.go. These are aliases that point to the same error
// values, ensuring errors.Is compatibility.
var (
	// ErrWorkerCircuitTripped indicates a worker exhausted its restart attempts
	// within the configured circuit window. The worker will not be restarted
	// until the application is restarted.
	ErrWorkerCircuitTripped = worker.ErrCircuitBreakerTripped

	// ErrWorkerStopped indicates a worker stopped normally without error.
	// This is not an error condition - it signals clean shutdown.
	ErrWorkerStopped = worker.ErrWorkerStopped

	// ErrWorkerCriticalFailed indicates a critical worker failed and exhausted
	// its restart attempts. This error triggers application shutdown.
	ErrWorkerCriticalFailed = worker.ErrCriticalWorkerFailed

	// ErrWorkerManagerRunning indicates an attempt to register a worker
	// after the manager has started.
	ErrWorkerManagerRunning = worker.ErrManagerAlreadyRunning
)

// Cron subsystem errors.
// These are re-exports from the cron package with standardized ErrCron* naming.
// Use errors.Is(err, gaz.ErrCron*) to check for these errors.
//
// Note: Due to Go's import cycle constraints, the canonical source of cron
// errors is cron/errors.go. These are aliases that point to the same error
// values, ensuring errors.Is compatibility.
var (
	// ErrCronNotRunning indicates an operation was attempted on a scheduler
	// that is not running.
	ErrCronNotRunning = cron.ErrNotRunning
)

// Module errors (gaz-specific).
var (
	// ErrModuleDuplicate is returned when a module with the same name is registered twice.
	ErrModuleDuplicate = errors.New("gaz: duplicate module")

	// ErrConfigKeyCollision is returned when two providers register the same config key.
	ErrConfigKeyCollision = errors.New("gaz: config key collision")
)

// =============================================================================
// Typed Errors
// =============================================================================

// ResolutionError represents a DI resolution failure with context about
// which service failed and the resolution chain leading to the failure.
// Use errors.As to extract resolution context for debugging.
type ResolutionError struct {
	// ServiceName is the service that failed to resolve.
	ServiceName string

	// Chain is the resolution chain leading to the failure.
	// For example: ["App", "UserService", "Database"] shows that
	// App depends on UserService which depends on Database which failed.
	Chain []string

	// Cause is the underlying error that caused the resolution to fail.
	Cause error
}

// Error implements the error interface.
func (e *ResolutionError) Error() string {
	if len(e.Chain) == 0 {
		return fmt.Sprintf("di: failed to resolve %s: %v", e.ServiceName, e.Cause)
	}
	return fmt.Sprintf("di: failed to resolve %s (chain: %s): %v",
		e.ServiceName, strings.Join(e.Chain, " -> "), e.Cause)
}

// Unwrap returns the underlying cause, enabling errors.Is and errors.As
// to work through the error chain.
func (e *ResolutionError) Unwrap() error {
	return e.Cause
}

// LifecycleError represents a failure during service start or stop.
// Use errors.As to extract which service failed and in which phase.
type LifecycleError struct {
	// ServiceName is the service that failed during lifecycle.
	ServiceName string

	// Phase is the lifecycle phase that failed: "start" or "stop".
	Phase string

	// Cause is the underlying error.
	Cause error
}

// Error implements the error interface.
func (e *LifecycleError) Error() string {
	return fmt.Sprintf("lifecycle: %s failed during %s: %v", e.ServiceName, e.Phase, e.Cause)
}

// Unwrap returns the underlying cause.
func (e *LifecycleError) Unwrap() error {
	return e.Cause
}

// ValidationError holds multiple validation errors from config struct validation.
// It implements the error interface and provides access to individual field errors.
// Use errors.Is(err, ErrConfigValidation) to check for validation errors.
//
// This is a type alias for config.ValidationError, ensuring compatibility
// between gaz and config packages.
type ValidationError = config.ValidationError

// FieldError represents a single field validation failure.
// This is a type alias for config.FieldError.
type FieldError = config.FieldError

// NewFieldError creates a new FieldError with the given parameters.
// This is a convenience wrapper for config.NewFieldError.
func NewFieldError(namespace, tag, param, message string) FieldError {
	return config.NewFieldError(namespace, tag, param, message)
}

// NewValidationError creates a ValidationError from a slice of FieldErrors.
// This is a convenience wrapper for config.NewValidationError.
func NewValidationError(errs []FieldError) ValidationError {
	return config.NewValidationError(errs)
}
