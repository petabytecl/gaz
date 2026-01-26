package gaz

import "reflect"

// TypeName returns the fully-qualified type name for T.
// It uses reflection to generate consistent string keys from generic type parameters.
// Examples:
//   - TypeName[string]() returns "string"
//   - TypeName[*Config]() returns "*github.com/petabytecl/gaz.Config"
//   - TypeName[[]byte]() returns "[]uint8"
func TypeName[T any]() string {
	var zero T
	return typeName(reflect.TypeOf(&zero).Elem())
}

// typeName returns a string representation of the given reflect.Type.
// It handles named types with package paths, pointers, slices, maps, and interfaces.
func typeName(t reflect.Type) string {
	if t == nil {
		return "nil"
	}

	// Named types: return package path + name
	if name := t.Name(); name != "" {
		if pkg := t.PkgPath(); pkg != "" {
			return pkg + "." + name
		}
		return name
	}

	// Unnamed types: handle by kind
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + typeName(t.Elem())
	case reflect.Slice:
		return "[]" + typeName(t.Elem())
	case reflect.Map:
		return "map[" + typeName(t.Key()) + "]" + typeName(t.Elem())
	default:
		// Fallback for interfaces and other unnamed types
		return t.String()
	}
}
