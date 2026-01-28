package config

// Get retrieves a typed value from the Manager.
// Returns the zero value of T if the key is not found or type assertion fails.
//
// This function uses Go generics to provide type-safe configuration access
// without requiring explicit type assertions at the call site.
//
// Example:
//
//	port := config.Get[int](mgr, "server.port")
//	host := config.Get[string](mgr, "server.host")
//	debug := config.Get[bool](mgr, "debug")
func Get[T any](m *Manager, key string) T {
	val := m.backend.Get(key)
	if val == nil {
		var zero T
		return zero
	}

	typed, ok := val.(T)
	if !ok {
		var zero T
		return zero
	}

	return typed
}

// GetOr retrieves a typed value from the Manager with a fallback default.
// Returns the fallback value if the key is not found or type assertion fails.
//
// This is useful when you want to provide a default value inline rather than
// configuring defaults via WithDefaults option.
//
// Example:
//
//	port := config.GetOr(mgr, "server.port", 8080)
//	host := config.GetOr(mgr, "server.host", "localhost")
//	timeout := config.GetOr(mgr, "timeout", 30*time.Second)
func GetOr[T any](m *Manager, key string, fallback T) T {
	val := m.backend.Get(key)
	if val == nil {
		return fallback
	}

	typed, ok := val.(T)
	if !ok {
		return fallback
	}

	return typed
}

// MustGet retrieves a typed value from the Manager.
// Panics if the key is not found or type assertion fails.
// Use this only for values that are required and whose absence indicates
// a programming error.
//
// Example:
//
//	// Panics if "database.host" is not set or not a string
//	host := config.MustGet[string](mgr, "database.host")
func MustGet[T any](m *Manager, key string) T {
	val := m.backend.Get(key)
	if val == nil {
		panic("config: key not found: " + key)
	}

	typed, ok := val.(T)
	if !ok {
		var zero T
		panic("config: type assertion failed for key: " + key +
			" (expected " + typeNameOf(zero) + ")")
	}

	return typed
}

// typeNameOf returns a string representation of the type for error messages.
func typeNameOf(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case int:
		return "int"
	case int64:
		return "int64"
	case float64:
		return "float64"
	case bool:
		return "bool"
	default:
		return "unknown"
	}
}
