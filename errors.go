package gaz

import (
	"errors"

	"github.com/petabytecl/gaz/config"
	"github.com/petabytecl/gaz/di"
)

// Re-export di errors for backward compatibility.
// Code that uses errors.Is(err, gaz.ErrNotFound) will work correctly
// since these are aliases to the actual error values in di.
var (
	// ErrNotFound is returned when a requested service is not registered in the container.
	ErrNotFound = di.ErrNotFound

	// ErrCycle is returned when a circular dependency is detected during resolution.
	ErrCycle = di.ErrCycle

	// ErrDuplicate is returned when attempting to register a service that already exists.
	ErrDuplicate = di.ErrDuplicate

	// ErrNotSettable is returned when a struct field cannot be set during injection.
	ErrNotSettable = di.ErrNotSettable

	// ErrTypeMismatch is returned when a resolved service cannot be assigned to the target type.
	ErrTypeMismatch = di.ErrTypeMismatch

	// ErrAlreadyBuilt is returned when attempting to register after Build() was called.
	ErrAlreadyBuilt = di.ErrAlreadyBuilt

	// ErrInvalidProvider is returned when a provider function has invalid signature.
	ErrInvalidProvider = di.ErrInvalidProvider

	// ErrDuplicateModule is returned when a module with the same name is registered twice.
	// This error is specific to gaz (not in di or config packages).
	ErrDuplicateModule = errors.New("gaz: duplicate module name")

	// ErrConfigKeyCollision is returned when two providers register the same config key.
	// This error is specific to gaz (not in di or config packages).
	ErrConfigKeyCollision = errors.New("gaz: config key collision")

	// ErrConfigValidation is returned when config struct validation fails.
	// This is an alias to config.ErrConfigValidation for backward compatibility.
	ErrConfigValidation = config.ErrConfigValidation
)
