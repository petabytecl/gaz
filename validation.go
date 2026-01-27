package gaz

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

// validateConfigTags validates a config struct using validate tags.
// Returns nil if validation passes, or a formatted error if validation fails.
func validateConfigTags(cfg any) error {
	err := configValidator.Struct(cfg)
	if err == nil {
		return nil
	}

	// Handle invalid validation input (programming error)
	var invalidValidationError *validator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return fmt.Errorf("invalid validation input: %w", err)
	}

	// Handle validation errors
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		return formatValidationErrors(validationErrors)
	}

	// Wrap unknown errors from validator
	return fmt.Errorf("validation error: %w", err)
}

// formatValidationErrors converts validator.ValidationErrors into a human-readable error.
func formatValidationErrors(errs validator.ValidationErrors) error {
	messages := make([]string, 0, len(errs))
	for _, e := range errs {
		// e.Namespace() = "Config.database.host"
		// e.Tag() = "required"
		// e.Param() = constraint parameter (e.g., "5" for min=5)
		msg := fmt.Sprintf("%s: %s (validate:\"%s\")",
			e.Namespace(),
			humanizeTag(e.Tag(), e.Param()),
			e.Tag())
		messages = append(messages, msg)
	}

	return fmt.Errorf("%w:\n%s", ErrConfigValidation, strings.Join(messages, "\n"))
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
