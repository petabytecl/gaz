package gaz

import (
	"errors"
	"fmt"
	"strings"

	"github.com/petabytecl/gaz/di"
)

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
	// ErrDuplicate is an alias for di.ErrDuplicate for backward compatibility.
	// Deprecated: Use ErrDIDuplicate instead.
	ErrDuplicate = di.ErrDuplicate

	// ErrInvalidProvider is an alias for di.ErrInvalidProvider for backward compatibility.
	// Deprecated: Use ErrDIInvalidProvider instead.
	ErrInvalidProvider = di.ErrInvalidProvider

	// ErrDuplicateModule is an alias for ErrModuleDuplicate.
	// Deprecated: Use ErrModuleDuplicate instead.
	ErrDuplicateModule = ErrModuleDuplicate

	// ErrNotFound is an alias for di.ErrNotFound for backward compatibility.
	// Deprecated: Use ErrDINotFound instead.
	ErrNotFound = di.ErrNotFound

	// ErrCycle is an alias for di.ErrCycle for backward compatibility.
	// Deprecated: Use ErrDICycle instead.
	ErrCycle = di.ErrCycle

	// ErrNotSettable is an alias for di.ErrNotSettable for backward compatibility.
	// Deprecated: Use ErrDINotSettable instead.
	ErrNotSettable = di.ErrNotSettable

	// ErrTypeMismatch is an alias for di.ErrTypeMismatch for backward compatibility.
	// Deprecated: Use ErrDITypeMismatch instead.
	ErrTypeMismatch = di.ErrTypeMismatch

	// ErrAlreadyBuilt is an alias for di.ErrAlreadyBuilt for backward compatibility.
	// Deprecated: Use ErrDIAlreadyBuilt instead.
	ErrAlreadyBuilt = di.ErrAlreadyBuilt
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
type ValidationError struct {
	// Errors is the list of individual field validation failures.
	Errors []FieldError
}

// Error implements the error interface.
func (ve ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return ErrConfigValidation.Error()
	}

	msgs := make([]string, len(ve.Errors))
	for i, e := range ve.Errors {
		msgs[i] = e.String()
	}
	return fmt.Sprintf("%s:\n%s", ErrConfigValidation.Error(), strings.Join(msgs, "\n"))
}

// Unwrap returns the underlying ErrConfigValidation sentinel error.
// This allows errors.Is(err, ErrConfigValidation) to work correctly.
func (ve ValidationError) Unwrap() error {
	return ErrConfigValidation
}

// FieldError represents a single field validation failure.
type FieldError struct {
	// Namespace is the full path to the field (e.g., "Config.database.host").
	Namespace string

	// Tag is the validation tag that failed (e.g., "required", "min").
	Tag string

	// Param is the parameter for the validation tag (e.g., "5" for min=5).
	Param string

	// Message is a human-readable error message.
	Message string
}

// String returns a formatted string representation of the field error.
func (fe FieldError) String() string {
	if fe.Tag != "" {
		return fmt.Sprintf("%s: %s (validate:\"%s\")", fe.Namespace, fe.Message, fe.Tag)
	}
	return fmt.Sprintf("%s: %s", fe.Namespace, fe.Message)
}

// NewFieldError creates a new FieldError with the given parameters.
func NewFieldError(namespace, tag, param, message string) FieldError {
	return FieldError{
		Namespace: namespace,
		Tag:       tag,
		Param:     param,
		Message:   message,
	}
}

// NewValidationError creates a ValidationError from a slice of FieldErrors.
func NewValidationError(errs []FieldError) ValidationError {
	return ValidationError{Errors: errs}
}
