package gaz

import "errors"

var (
	// ErrNotFound is returned when a requested service is not registered in the container.
	ErrNotFound = errors.New("gaz: service not found")

	// ErrCycle is returned when a circular dependency is detected during resolution.
	ErrCycle = errors.New("gaz: circular dependency detected")

	// ErrDuplicate is returned when attempting to register a service that already exists.
	ErrDuplicate = errors.New("gaz: service already registered")

	// ErrNotSettable is returned when a struct field cannot be set during injection.
	ErrNotSettable = errors.New("gaz: field is not settable")

	// ErrTypeMismatch is returned when a resolved service cannot be assigned to the target type.
	ErrTypeMismatch = errors.New("gaz: type mismatch")

	// ErrAlreadyBuilt is returned when attempting to register after Build() was called.
	ErrAlreadyBuilt = errors.New("gaz: cannot register after Build()")

	// ErrInvalidProvider is returned when a provider function has invalid signature.
	ErrInvalidProvider = errors.New("gaz: invalid provider signature")

	// ErrDuplicateModule is returned when a module with the same name is registered twice.
	ErrDuplicateModule = errors.New("gaz: duplicate module name")
)
