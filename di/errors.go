package di

import "errors"

var (
	// ErrNotFound is returned when a requested service is not registered in the container.
	ErrNotFound = errors.New("di: service not found")

	// ErrCycle is returned when a circular dependency is detected during resolution.
	ErrCycle = errors.New("di: circular dependency detected")

	// ErrDuplicate is returned when attempting to register a service that already exists.
	ErrDuplicate = errors.New("di: service already registered")

	// ErrNotSettable is returned when a struct field cannot be set during injection.
	ErrNotSettable = errors.New("di: field is not settable")

	// ErrTypeMismatch is returned when a resolved service cannot be assigned to the target type.
	ErrTypeMismatch = errors.New("di: type mismatch")

	// ErrAlreadyBuilt is returned when attempting to register after Build() was called.
	ErrAlreadyBuilt = errors.New("di: cannot register after Build()")

	// ErrInvalidProvider is returned when a provider function has invalid signature.
	ErrInvalidProvider = errors.New("di: invalid provider signature")
)
