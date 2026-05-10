package di

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// injectFieldCache caches parsed inject field metadata per reflect.Type.
// This avoids re-parsing struct tags on every injection call for the same type.
//
//nolint:gochecknoglobals // Package-level for reflect type caching.
var injectFieldCache sync.Map // map[reflect.Type][]injectField

// injectField holds parsed metadata for a single struct field with a gaz:"inject" tag.
type injectField struct {
	index       int    // field index in the struct
	serviceName string // resolved service name (from name= or type name)
	optional    bool   // allow missing service
}

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

// getInjectFields returns the cached inject field metadata for the given struct type.
// On first call for a type, it parses all fields with gaz:"inject" tags, caches the
// result, and returns it. Subsequent calls return the cached value.
//
// Returns an error if an unexported field has the gaz:"inject" tag.
func getInjectFields(t reflect.Type) ([]injectField, error) {
	if cached, ok := injectFieldCache.Load(t); ok {
		return cached.([]injectField), nil //nolint:forcetypeassert,errcheck // sync.Map stores []injectField exclusively
	}

	var fields []injectField
	for i := range t.NumField() {
		field := t.Field(i)

		tagValue, hasTag := field.Tag.Lookup("gaz")
		if !hasTag {
			continue
		}

		opts := parseTag(tagValue)
		if !opts.inject {
			continue
		}

		// Skip unexported fields — they cannot be set via reflect
		if !field.IsExported() {
			return nil, fmt.Errorf("%w: field %s.%s is unexported",
				ErrNotSettable, t.Name(), field.Name)
		}

		serviceName := opts.name
		if serviceName == "" {
			serviceName = typeName(field.Type)
		}

		fields = append(fields, injectField{
			index:       i,
			serviceName: serviceName,
			optional:    opts.optional,
		})
	}

	injectFieldCache.Store(t, fields)
	return fields, nil
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

	fields, err := getInjectFields(structType)
	if err != nil {
		return err
	}

	for _, f := range fields {
		fieldVal := structVal.Field(f.index)
		field := structType.Field(f.index)

		// Resolve the dependency
		instance, resolveErr := c.ResolveByName(f.serviceName, chain)
		if resolveErr != nil {
			if f.optional && errors.Is(resolveErr, ErrNotFound) {
				continue // Leave as zero value
			}
			return fmt.Errorf("di: injecting field %s.%s: %w",
				structType.Name(), field.Name, resolveErr)
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
