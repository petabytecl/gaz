# Phase 7: Validation Engine - Research

**Researched:** 2026-01-27
**Domain:** Go struct validation with tags for configuration validation
**Confidence:** HIGH

## Summary

This research investigated how to implement struct tag-based validation for gaz's configuration system. The goal is to validate config structs at load time using `validate` tags (e.g., `validate:"required"`), collect all errors, and prevent application startup if validation fails.

The standard approach in Go is to use **go-playground/validator** (v10), the de facto library for struct validation with tags. It's used by 23,000+ projects including gin, has excellent cross-field validation support via tags like `required_if`, handles nested structs automatically, and provides rich error details including field paths. The library is thread-safe, singleton-friendly, and has minimal overhead (28ns per field on success).

The integration point is clear: after `viper.Unmarshal()` populates the config struct, call `validator.Struct()` on it. Validation errors from go-playground/validator provide namespace (full path like `Config.Database.Host`), field name, tag name, and actual value - exactly what's needed for actionable error messages.

**Primary recommendation:** Use go-playground/validator v10 with `WithRequiredStructEnabled()` option, integrate validation into `ConfigManager.Load()` after unmarshal but before calling user's `Validate()` method, and format errors as `field.path: message (validate:"tag")`.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| go-playground/validator | v10.30.1 | Struct tag validation | 19.6k stars, 23k+ importers, default in gin, comprehensive tag support |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Standard library reflect | - | Field name extraction | Use `RegisterTagNameFunc` to extract json/mapstructure names for errors |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-playground/validator | Custom validation | Much more code, miss cross-field support, reinvent wheels |
| go-playground/validator | asaskevich/govalidator | Less popular, simpler API, fewer features |
| go-playground/validator | go-ozzo/ozzo-validation | Different API (not tag-based), good but unfamiliar pattern |

**Installation:**
```bash
go get github.com/go-playground/validator/v10
```

## Architecture Patterns

### Integration Point in ConfigManager

```
ConfigManager.Load()
    |
    +-> viper.ReadInConfig()
    +-> viper.Unmarshal(target)
    +-> target.Default() (if Defaulter)
    +-> validateWithTags(target)    <-- NEW: struct tag validation
    +-> target.Validate() (if Validator) <-- existing custom validation
    +-> return
```

The tag validation runs AFTER defaults are applied but BEFORE user's custom Validate() method, allowing:
1. Defaults fill in empty values
2. Tag validation catches structural issues
3. Custom validation handles business logic

### Recommended Project Structure
```
gaz/
├── config_manager.go     # Add validateWithTags() call in Load()
├── validation.go         # NEW: validator instance, error formatting
└── validation_test.go    # NEW: comprehensive test coverage
```

### Pattern 1: Singleton Validator
**What:** Create one `*validator.Validate` instance and reuse it
**When to use:** Always - validator caches struct info per type
**Example:**
```go
// Source: go-playground/validator official documentation
package gaz

import (
    "github.com/go-playground/validator/v10"
)

// Package-level singleton - thread-safe, caches struct info
var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
    // Register tag name function to use mapstructure tags for field names
    validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
        name := strings.SplitN(fld.Tag.Get("mapstructure"), ",", 2)[0]
        if name == "-" || name == "" {
            return fld.Name
        }
        return name
    })
}
```

### Pattern 2: Error Collection and Formatting
**What:** Collect all validation errors (not fail-fast) and format them
**When to use:** Config validation - show all errors at once
**Example:**
```go
// Source: go-playground/validator official examples
func formatValidationErrors(err error) string {
    var errs validator.ValidationErrors
    if !errors.As(err, &errs) {
        return err.Error()
    }
    
    var messages []string
    for _, e := range errs {
        // e.Namespace() = "Config.Database.Host"
        // e.Tag() = "required"
        // e.Field() = "Host"
        msg := fmt.Sprintf("%s: %s (validate:\"%s\")",
            e.Namespace(),
            humanizeTag(e.Tag(), e.Param()),
            e.Tag())
        messages = append(messages, msg)
    }
    return strings.Join(messages, "\n")
}

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
    default:
        return fmt.Sprintf("failed %s validation", tag)
    }
}
```

### Pattern 3: Opt-in Validation Mode
**What:** Only validate if config struct has validate tags
**When to use:** Default behavior - validate when tags present
**Example:**
```go
// Validation always runs on struct - if no validate tags, it's a no-op
// This is the simplest approach and matches user expectations
func validateConfig(cfg any) error {
    return validate.Struct(cfg)
}
```

### Anti-Patterns to Avoid
- **Creating validator per call:** Wastes cache, use singleton
- **Returning on first error:** Users want all errors at once
- **Generic error messages:** Include field path, tag, and constraint value
- **Validating before defaults:** Missing values will fail required checks incorrectly
- **Using os.Exit in library code:** Return error, let caller decide (recommendation)

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tag parsing | Custom tag parser | validator's built-in | Tags have complex syntax (comma, pipe, params) |
| Required check | `if field == ""` | `validate:"required"` | Handles zero values for all types correctly |
| Cross-field validation | Manual field comparison | `required_if`, `eqfield` | Edge cases, nested structs, array diving |
| Field path building | Manual string concatenation | `FieldError.Namespace()` | Handles nested structs, maps, slices |
| Nested struct walking | reflect recursion | validator auto-dive | Complex pointer handling, cycles |

**Key insight:** Validation looks trivial ("just check if empty") but has edge cases: pointers, zero values vs nil, nested structs, slices of structs, maps, cross-field dependencies. go-playground/validator handles all of these with tested code.

## Common Pitfalls

### Pitfall 1: Required on Pointers vs Values
**What goes wrong:** `required` on `*string` passes when pointer is non-nil but empty
**Why it happens:** Validator checks "is zero value" - non-nil pointer isn't zero
**How to avoid:** Use `required` on value types, or `omitnil` + value validation
**Warning signs:** Empty strings passing required validation when using pointer types

### Pitfall 2: Validation Before Defaults
**What goes wrong:** Required fields fail because defaults haven't been applied yet
**Why it happens:** Call order: unmarshal -> validate -> defaults
**How to avoid:** Always apply defaults before validation: unmarshal -> defaults -> validate
**Warning signs:** Tests pass with explicit values but fail with defaults

### Pitfall 3: Wrong Field Names in Errors
**What goes wrong:** Errors show Go field names (Host) instead of config keys (host)
**Why it happens:** Validator uses struct field names by default
**How to avoid:** Register TagNameFunc to extract from mapstructure/json tags
**Warning signs:** Users can't find the config key mentioned in error

### Pitfall 4: Cross-Field Validation Syntax
**What goes wrong:** `required_if=OtherField value` doesn't work as expected
**Why it happens:** Syntax is `required_if=OtherField value` (space not equals)
**How to avoid:** Follow docs exactly: `validate:"required_if=AuthType local"`
**Warning signs:** Cross-field validation always passes or always fails

### Pitfall 5: Nested Struct Validation Not Running
**What goes wrong:** Nested struct validation is skipped
**Why it happens:** By default, validator requires `WithRequiredStructEnabled()` for nested struct validation
**How to avoid:** Always use `validator.New(validator.WithRequiredStructEnabled())`
**Warning signs:** Nested struct fields aren't validated

## Code Examples

Verified patterns from official sources:

### Basic Struct Validation
```go
// Source: https://pkg.go.dev/github.com/go-playground/validator/v10
type DatabaseConfig struct {
    Host     string `mapstructure:"host" validate:"required"`
    Port     int    `mapstructure:"port" validate:"required,min=1,max=65535"`
    User     string `mapstructure:"user" validate:"required"`
    Password string `mapstructure:"password" validate:"required"`
    Name     string `mapstructure:"name" validate:"required,min=1,max=63"`
}

type Config struct {
    Database DatabaseConfig `mapstructure:"database" validate:"required"`
    LogLevel string         `mapstructure:"log_level" validate:"oneof=debug info warn error"`
}
```

### Cross-Field Validation Tags
```go
// Source: https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Required_If
type AuthConfig struct {
    Type     string `mapstructure:"type" validate:"required,oneof=none basic oauth"`
    Username string `mapstructure:"username" validate:"required_if=Type basic"`
    Password string `mapstructure:"password" validate:"required_if=Type basic"`
    Token    string `mapstructure:"token" validate:"required_if=Type oauth"`
}
```

### Error Handling
```go
// Source: https://github.com/go-playground/validator/blob/master/_examples/simple/main.go
err := validate.Struct(cfg)
if err != nil {
    var invalidValidationError *validator.InvalidValidationError
    if errors.As(err, &invalidValidationError) {
        // Programming error - invalid input to Struct()
        return fmt.Errorf("invalid validation input: %w", err)
    }
    
    var validationErrors validator.ValidationErrors
    if errors.As(err, &validationErrors) {
        // Format and return all errors
        return formatValidationErrors(validationErrors)
    }
    
    return err
}
```

### Using RegisterTagNameFunc for Mapstructure Tags
```go
// Source: https://github.com/go-playground/validator/blob/master/_examples/struct-level/main.go
validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
    // Try mapstructure tag first (gaz uses this)
    name := strings.SplitN(fld.Tag.Get("mapstructure"), ",", 2)[0]
    if name != "-" && name != "" {
        return name
    }
    // Fall back to json tag
    name = strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
    if name != "-" && name != "" {
        return name
    }
    return fld.Name
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Default nested struct validation | Opt-in via `WithRequiredStructEnabled()` | v10 (will change in v11) | Must explicitly enable for nested struct validation |
| Type assertions for errors | errors.As() | Go 1.13+ | More idiomatic error handling |

**Deprecated/outdated:**
- validator v8/v9: Use v10 for latest features and Go module support
- Direct type assertion `err.(validator.ValidationErrors)`: Use `errors.As()` for safety

## Open Questions

Things that couldn't be fully resolved:

1. **Source info in error messages**
   - What we know: User wants errors to show where value came from (file, env var)
   - What's unclear: Viper doesn't track per-key source after unmarshal
   - Recommendation: For v1.1, show field path only; source tracking is a larger effort

2. **Opt-in validation mode**
   - What we know: User mentioned opt-in vs always-on as Claude's discretion
   - What's unclear: Whether "opt-in" means config option or just "has validate tags"
   - Recommendation: Always run validation - if no tags, it's a no-op (simple, predictable)

3. **Exit vs return error for validation failures**
   - What we know: User left this as Claude's discretion
   - What's unclear: Whether library should call os.Exit
   - Recommendation: Return error from Load(), caller (App.Run) already handles errors

## Sources

### Primary (HIGH confidence)
- /go-playground/validator - Context7 documentation query
- https://pkg.go.dev/github.com/go-playground/validator/v10 - Official Go docs
- https://github.com/go-playground/validator - Official README, v10.30.1

### Secondary (MEDIUM confidence)
- https://github.com/go-playground/validator/blob/master/_examples/simple/main.go - Official example
- https://github.com/go-playground/validator/blob/master/_examples/struct-level/main.go - Cross-field example

### Tertiary (LOW confidence)
- None - all findings verified with official sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - go-playground/validator is de facto standard, verified via Context7 and official docs
- Architecture: HIGH - integration point is clear, existing code reviewed
- Pitfalls: HIGH - documented in official examples and README

**Research date:** 2026-01-27
**Valid until:** 2026-02-27 (30 days - stable library, mature ecosystem)
