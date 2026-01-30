package config

import (
	"errors"
	"fmt"
	"strings"
)

// ErrConfigValidation is returned when config struct validation fails.
// Use errors.Is(err, ErrConfigValidation) to check for validation errors.
var ErrConfigValidation = errors.New("config: validation failed")

// ValidationError holds multiple validation errors.
// It implements the error interface and provides access to individual field errors.
type ValidationError struct {
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
