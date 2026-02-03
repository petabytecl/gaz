package di

import "errors"

// DI sentinel errors with standardized "di: action" format.
// These are the canonical source of truth for DI errors.
// The gaz package re-exports these as gaz.ErrDI* for convenience.
var (
	// ErrNotFound is returned when a requested service is not registered in the container.
	// Check with: errors.Is(err, di.ErrNotFound) or errors.Is(err, gaz.ErrDINotFound).
	ErrNotFound = errors.New("di: not found")

	// ErrCycle is returned when a circular dependency is detected during resolution.
	// Check with: errors.Is(err, di.ErrCycle) or errors.Is(err, gaz.ErrDICycle).
	ErrCycle = errors.New("di: circular dependency")

	// ErrDuplicate is returned when attempting to register a service that already exists.
	// Check with: errors.Is(err, di.ErrDuplicate) or errors.Is(err, gaz.ErrDIDuplicate).
	ErrDuplicate = errors.New("di: duplicate registration")

	// ErrNotSettable is returned when a struct field cannot be set during injection.
	// Check with: errors.Is(err, di.ErrNotSettable) or errors.Is(err, gaz.ErrDINotSettable).
	ErrNotSettable = errors.New("di: field not settable")

	// ErrTypeMismatch is returned when a resolved service cannot be assigned to the target type.
	// Check with: errors.Is(err, di.ErrTypeMismatch) or errors.Is(err, gaz.ErrDITypeMismatch).
	ErrTypeMismatch = errors.New("di: type mismatch")

	// ErrAlreadyBuilt is returned when attempting to register after Build() was called.
	// Check with: errors.Is(err, di.ErrAlreadyBuilt) or errors.Is(err, gaz.ErrDIAlreadyBuilt).
	ErrAlreadyBuilt = errors.New("di: already built")

	// ErrInvalidProvider is returned when a provider function has invalid signature.
	// Check with: errors.Is(err, di.ErrInvalidProvider) or errors.Is(err, gaz.ErrDIInvalidProvider).
	ErrInvalidProvider = errors.New("di: invalid provider")

	// ErrAmbiguous is returned when multiple services are registered for the same key.
	// Check with: errors.Is(err, di.ErrAmbiguous).
	ErrAmbiguous = errors.New("di: ambiguous resolution: multiple services registered")
)
