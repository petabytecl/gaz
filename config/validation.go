package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// configValidator provides a singleton validator instance for config validation.
// Thread-safe and caches struct info for performance.
//
//nolint:gochecknoglobals // Singleton pattern for validator efficiency
var configValidator = newConfigValidator()

// newConfigValidator creates and configures the validator instance.
func newConfigValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	// Register tag name function to use mapstructure tags for field names in error messages.
	// Falls back to json tag, then to Go field name.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("mapstructure"), ",")
		if name != "-" && name != "" {
			return name
		}
		name, _, _ = strings.Cut(fld.Tag.Get("json"), ",")
		if name != "-" && name != "" {
			return name
		}
		return fld.Name
	})

	return v
}

// ValidateStruct validates a config struct using validate tags.
// Returns nil if validation passes, or a ValidationErrors if validation fails.
//
// This function uses go-playground/validator for struct tag validation.
// Common validation tags include:
//   - required: field must not be empty
//   - min=N, max=N: minimum/maximum values or lengths
//   - oneof=a b c: value must be one of the specified options
//   - email, url, ip: format validators
//
// Example:
//
//	type Config struct {
//	    Host string `mapstructure:"host" validate:"required"`
//	    Port int    `mapstructure:"port" validate:"min=1,max=65535"`
//	}
//
//	if err := config.ValidateStruct(&cfg); err != nil {
//	    // Handle validation error
//	}
func ValidateStruct(cfg any) error {
	err := configValidator.Struct(cfg)
	if err == nil {
		return nil
	}

	// Handle invalid validation input (programming error)
	var invalidValidationError *validator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return fmt.Errorf("config: invalid validation input: %w", err)
	}

	// Handle validation errors
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		return formatValidationErrors(validationErrors)
	}

	// Wrap unknown errors from validator
	return fmt.Errorf("config: validation error: %w", err)
}

// formatValidationErrors converts validator.ValidationErrors into our ValidationErrors type.
func formatValidationErrors(errs validator.ValidationErrors) error {
	fieldErrors := make([]FieldError, 0, len(errs))
	for _, e := range errs {
		fieldErrors = append(fieldErrors, FieldError{
			Namespace: e.Namespace(),
			Tag:       e.Tag(),
			Param:     e.Param(),
			Message:   humanizeTag(e.Tag(), e.Param()),
		})
	}

	return NewValidationErrors(fieldErrors)
}

// humanizeTag converts validation tag names to human-readable messages.
func humanizeTag(tag, param string) string {
	switch tag {
	case "required":
		return "required field cannot be empty"
	case "min":
		return fmt.Sprintf("must be at least %s", param)
	case "max":
		return fmt.Sprintf("must be at most %s", param)
	case "oneof":
		return fmt.Sprintf("must be one of: %s", param)
	case "required_if":
		return fmt.Sprintf("required when %s", param)
	case "required_unless":
		return fmt.Sprintf("required unless %s", param)
	case "required_with":
		return fmt.Sprintf("required when %s is present", param)
	case "required_without":
		return fmt.Sprintf("required when %s is absent", param)
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", param)
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", param)
	case "gt":
		return fmt.Sprintf("must be greater than %s", param)
	case "lt":
		return fmt.Sprintf("must be less than %s", param)
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "ip":
		return "must be a valid IP address"
	case "ipv4":
		return "must be a valid IPv4 address"
	case "ipv6":
		return "must be a valid IPv6 address"
	default:
		return fmt.Sprintf("failed %s validation", tag)
	}
}
