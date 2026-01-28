package di

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// tagOptions holds parsed gaz struct tag options.
type tagOptions struct {
	inject   bool   // Has "inject" keyword
	name     string // Custom name (from name=xxx)
	optional bool   // Allow missing service
}

// parseTag parses a gaz struct tag value into tagOptions.
// Tag format: "inject" or "inject,name=foo" or "inject,optional" or "inject,name=foo,optional".
func parseTag(tag string) tagOptions {
	opts := tagOptions{}
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case part == "inject":
			opts.inject = true
		case part == "optional":
			opts.optional = true
		case strings.HasPrefix(part, "name="):
			opts.name = strings.TrimPrefix(part, "name=")
		}
	}
	return opts
}

// injectStruct populates tagged fields of a struct with resolved services.
// target must be a pointer to a struct. If not, injection is skipped silently.
// chain is the current resolution chain for cycle detection.
//
// Fields tagged with gaz:"inject" are resolved by type name.
// Fields tagged with gaz:"inject,name=foo" are resolved by the given name.
// Fields tagged with gaz:"inject,optional" are left as zero value if not registered.
//
// Returns ErrNotSettable if an unexported field has the gaz tag.
// Returns wrapped errors if dependency resolution fails.
func injectStruct(c *Container, target any, chain []string) error {
	val := reflect.ValueOf(target)

	// Only inject into struct pointers
	if val.Kind() != reflect.Pointer || val.Elem().Kind() != reflect.Struct {
		return nil // Not a struct pointer, skip injection silently
	}

	structVal := val.Elem()
	structType := structVal.Type()

	for i := range structVal.NumField() {
		field := structType.Field(i)
		fieldVal := structVal.Field(i)

		tagValue, hasTag := field.Tag.Lookup("gaz")
		if !hasTag {
			continue
		}

		opts := parseTag(tagValue)
		if !opts.inject {
			continue
		}

		// Check if field is settable (exported)
		if !fieldVal.CanSet() {
			return fmt.Errorf("%w: field %s.%s is unexported",
				ErrNotSettable, structType.Name(), field.Name)
		}

		// Determine service name
		serviceName := opts.name
		if serviceName == "" {
			serviceName = typeName(field.Type)
		}

		// Resolve the dependency
		instance, err := c.resolveByName(serviceName, chain)
		if err != nil {
			if opts.optional && errors.Is(err, ErrNotFound) {
				continue // Leave as zero value
			}
			return fmt.Errorf("injecting field %s.%s: %w",
				structType.Name(), field.Name, err)
		}

		// Type check and assign
		instanceVal := reflect.ValueOf(instance)
		if !instanceVal.Type().AssignableTo(fieldVal.Type()) {
			return fmt.Errorf("%w: cannot assign %s to field %s.%s (%s)",
				ErrTypeMismatch, instanceVal.Type(), structType.Name(),
				field.Name, fieldVal.Type())
		}

		fieldVal.Set(instanceVal)
	}

	return nil
}
