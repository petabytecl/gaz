package di

import (
	"fmt"
	"regexp"
)

var (
	// nameRegex validates service/module/worker names (alphanumeric + hyphens/underscores).
	nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// configKeyRegex validates config keys (alphanumeric + dots/hyphens/underscores).
	// Dots are allowed for nested keys like "database.host".
	configKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// ErrInvalidName is returned when a name fails validation.
var ErrInvalidName = fmt.Errorf("di: invalid name")

// validateServiceName validates a service registration name.
// Names must be alphanumeric with hyphens or underscores only.
func validateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidName)
	}
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("%w: %q (must be alphanumeric with hyphens/underscores)", ErrInvalidName, name)
	}
	return nil
}

// ValidateModuleName validates a module name.
// Uses the same pattern as service names.
// Exported for use by gaz package.
func ValidateModuleName(name string) error {
	return validateServiceName(name)
}

// validateWorkerName validates a worker name.
// Uses the same pattern as service names.
func validateWorkerName(name string) error {
	return validateServiceName(name)
}

// ValidateConfigKey validates a configuration key.
// Keys can contain dots for nesting (e.g., "database.host").
// Exported for use by config package.
func ValidateConfigKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: config key cannot be empty", ErrInvalidName)
	}
	if !configKeyRegex.MatchString(key) {
		return fmt.Errorf("%w: config key %q (must be alphanumeric with dots/hyphens/underscores)", ErrInvalidName, key)
	}
	// Prevent path traversal attempts
	if key == ".." || key == "." || containsPathTraversal(key) {
		return fmt.Errorf("%w: config key %q contains invalid path characters", ErrInvalidName, key)
	}
	return nil
}

// containsPathTraversal checks for path traversal patterns.
func containsPathTraversal(key string) bool {
	return regexp.MustCompile(`\.\./|\.\.\\|/\.\.|\\\.\.`).MatchString(key)
}
