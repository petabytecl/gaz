# Phase 25: Configuration Harmonization - Research

**Researched:** 2026-01-30
**Domain:** Go struct-based config unmarshaling, custom struct tags, Viper/mapstructure integration
**Confidence:** HIGH

## Summary

This phase adds struct-based config resolution via `ProviderValues.Unmarshal()` methods. The implementation leverages Viper's existing `UnmarshalKey` functionality with custom mapstructure `DecoderConfig` options to support the `gaz` struct tag.

Key findings:
1. **Custom tag support is built-in**: Viper's `Unmarshal` and `UnmarshalKey` accept variadic `DecoderConfigOption` functions that configure mapstructure's `DecoderConfig.TagName`
2. **mapstructure v2 is already a dependency**: The project uses `github.com/go-viper/mapstructure/v2` (v2.4.0) via Viper
3. **Validator integration pattern exists**: The codebase already uses `go-playground/validator` with `RegisterTagNameFunc` for custom tag name resolution

**Primary recommendation:** Wrap Viper's `UnmarshalKey` with a `DecoderConfigOption` that sets `TagName = "gaz"`, and add methods to `ProviderValues` that delegate to this wrapped implementation.

## Standard Stack

The established libraries/tools for this domain:

### Core (Already in use)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/spf13/viper | v1.21.0 | Config management | Industry standard, already used |
| github.com/go-viper/mapstructure/v2 | v2.4.0 | Struct decoding | Viper's blessed dependency |
| github.com/go-playground/validator/v10 | v10.30.1 | Struct validation | Already integrated in config/validation.go |

### Supporting (No new dependencies needed)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| reflect (stdlib) | - | Tag parsing | Custom tag extraction |
| errors (stdlib) | - | Sentinel errors | ErrKeyNotFound |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Viper UnmarshalKey | Direct mapstructure | Loses Viper's config source abstraction |
| gaz tag | mapstructure tag | Less control for future extensions |

**Installation:**
```bash
# No new dependencies required
# All libraries already present in go.mod
```

## Architecture Patterns

### Recommended Implementation Structure
```
config/
├── backend.go          # Add UnmarshalKeyWithOptions to Backend interface (optional)
├── errors.go           # Add ErrKeyNotFound sentinel
├── unmarshal.go        # NEW: Unmarshaling utilities with gaz tag support
└── viper/
    └── backend.go      # Wrap UnmarshalKey with DecoderConfigOption

provider_config.go      # Add Unmarshal/UnmarshalKey to ProviderValues
```

### Pattern 1: DecoderConfigOption for Custom Tag
**What:** Use Viper's variadic options to configure mapstructure
**When to use:** Every Unmarshal call that should use `gaz` tag
**Example:**
```go
// Source: https://github.com/spf13/viper/blob/master/UPGRADE.md
import "github.com/go-viper/mapstructure/v2"

func gazTagOption() viper.DecoderConfigOption {
    return func(config *mapstructure.DecoderConfig) {
        config.TagName = "gaz"
    }
}

// Usage in Backend
func (b *Backend) UnmarshalKeyWithGazTag(key string, target any) error {
    return b.v.UnmarshalKey(key, target, gazTagOption())
}
```

### Pattern 2: Sentinel Error for Missing Key
**What:** Check `IsSet` before unmarshaling to return typed error
**When to use:** When callers need to distinguish "not found" from "unmarshal error"
**Example:**
```go
// Source: Idiomatic Go error patterns
var ErrKeyNotFound = errors.New("config: key not found")

func (pv *ProviderValues) UnmarshalKey(key string, target any) error {
    if !pv.backend.IsSet(key) {
        return fmt.Errorf("%w: %s", ErrKeyNotFound, key)
    }
    return pv.backend.UnmarshalKey(key, target)
}
```

### Pattern 3: Consistent Tag Name Across Libraries
**What:** Register same tag name with both mapstructure and validator
**When to use:** Error messages should reference `gaz` tag, not field names
**Example:**
```go
// For mapstructure (via DecoderConfig)
config.TagName = "gaz"

// For validator (via RegisterTagNameFunc)
v.RegisterTagNameFunc(func(fld reflect.StructField) string {
    name, _, _ := strings.Cut(fld.Tag.Get("gaz"), ",")
    if name != "-" && name != "" {
        return name
    }
    // Fallback to mapstructure, json, field name
    ...
})
```

### Anti-Patterns to Avoid
- **Rolling your own tag parser**: mapstructure already handles all edge cases (squash, remain, omitempty)
- **Modifying Backend interface signature**: Keep it compatible, add ProviderValues methods instead
- **Storing parsed config in ProviderValues**: Let Viper/mapstructure do the work fresh each time

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tag parsing | Manual reflect.StructField.Tag.Lookup | mapstructure.DecoderConfig.TagName | Handles comma-separated options, squash, remain, omitempty |
| Nested struct mapping | Manual recursive walk | mapstructure's decoder | Handles pointer types, embedded structs, slices correctly |
| Duration parsing | Custom string-to-duration | mapstructure.StringToTimeDurationHookFunc | Already a decode hook |
| Type conversion | Manual type switches | mapstructure.WeaklyTypedInput or DecodeHooks | Handles all standard conversions |

**Key insight:** mapstructure is battle-tested with extensive edge case handling. The `DecoderConfig` provides all needed customization without reimplementing decoding logic.

## Common Pitfalls

### Pitfall 1: Viper's IsSet Doesn't Check Nested Keys
**What goes wrong:** `v.IsSet("redis")` returns false even if `redis.host` is set
**Why it happens:** Viper's IsSet only checks explicit keys, not parent paths
**How to avoid:** Check if `v.Sub(key)` returns nil for namespace existence
**Warning signs:** ErrKeyNotFound returned even when nested keys exist

```go
// Wrong
if !v.IsSet(key) { return ErrKeyNotFound }

// Better - check if any keys exist under namespace
if v.Sub(key) == nil && !v.IsSet(key) {
    return fmt.Errorf("%w: %s", ErrKeyNotFound, key)
}
```

### Pitfall 2: Tag Name Must Be Set Before Decode
**What goes wrong:** Tag configuration ignored, uses default "mapstructure"
**Why it happens:** DecoderConfig is used at decode time, not registration time
**How to avoid:** Always pass DecoderConfigOption to each Unmarshal call
**Warning signs:** Struct fields not populated despite correct gaz tags

### Pitfall 3: Embedded Structs Default to Nested (Not Squashed)
**What goes wrong:** Config key `Name` expected at root, but mapper looks for `EmbeddedType.Name`
**Why it happens:** mapstructure treats embedded structs as nested by default
**How to avoid:** Use `gaz:",squash"` or set `DecoderConfig.Squash = true`
**Warning signs:** Fields from embedded structs always zero-valued

### Pitfall 4: Validator Tag Name Registration Separate from Mapstructure
**What goes wrong:** Validation errors show Go field names instead of gaz tag names
**Why it happens:** validator's `RegisterTagNameFunc` is separate from mapstructure's `TagName`
**How to avoid:** Register "gaz" tag in both systems
**Warning signs:** Error messages like "Config.DatabaseHost" instead of "database.host"

### Pitfall 5: Partial Unmarshal Leaves Struct Fields at Zero Value
**What goes wrong:** Expected merge behavior, got replacement
**Why it happens:** mapstructure replaces slices/maps by default, doesn't merge
**How to avoid:** This is correct behavior per CONTEXT.md ("partial fill is OK")
**Warning signs:** None - this is expected behavior

## Code Examples

Verified patterns from official sources:

### Custom Tag Name with Viper
```go
// Source: https://github.com/spf13/viper/blob/master/UPGRADE.md
import (
    "github.com/go-viper/mapstructure/v2"
    "github.com/spf13/viper"
)

type RedisConfig struct {
    Host string `gaz:"host"`
    Port int    `gaz:"port"`
}

func unmarshalWithGazTag(v *viper.Viper, key string, target any) error {
    return v.UnmarshalKey(key, target, func(dc *mapstructure.DecoderConfig) {
        dc.TagName = "gaz"
    })
}
```

### Checking Key Existence Before Unmarshal
```go
// Source: Viper documentation + Go idioms
func (pv *ProviderValues) UnmarshalKey(key string, target any) error {
    // Check if namespace has any values
    // Note: Sub returns nil if key doesn't exist OR has no sub-keys
    if !pv.backend.IsSet(key) {
        // Check if it might be a parent key with children
        if sub, ok := pv.backend.(interface{ Sub(string) *viper.Viper }); ok {
            if sub.Sub(key) == nil {
                return fmt.Errorf("%w: %s", config.ErrKeyNotFound, key)
            }
        } else {
            return fmt.Errorf("%w: %s", config.ErrKeyNotFound, key)
        }
    }
    return pv.unmarshalKey(key, target)
}
```

### Validator Tag Name Alignment
```go
// Source: config/validation.go (existing pattern)
v.RegisterTagNameFunc(func(fld reflect.StructField) string {
    // Try gaz tag first (our custom tag)
    name, _, _ := strings.Cut(fld.Tag.Get("gaz"), ",")
    if name != "-" && name != "" {
        return name
    }
    // Fallback chain: mapstructure -> json -> field name
    name, _, _ = strings.Cut(fld.Tag.Get("mapstructure"), ",")
    if name != "-" && name != "" {
        return name
    }
    name, _, _ = strings.Cut(fld.Tag.Get("json"), ",")
    if name != "-" && name != "" {
        return name
    }
    return fld.Name
})
```

### Complete ProviderValues.UnmarshalKey Implementation
```go
// provider_config.go addition
import (
    "fmt"
    "github.com/go-viper/mapstructure/v2"
    "github.com/petabytecl/gaz/config"
)

// UnmarshalKey unmarshals config at the given key into target struct.
// Uses "gaz" struct tags for field mapping.
// Returns config.ErrKeyNotFound if the key/namespace doesn't exist.
func (pv *ProviderValues) UnmarshalKey(key string, target any) error {
    if !pv.hasKey(key) {
        return fmt.Errorf("%w: %s", config.ErrKeyNotFound, key)
    }
    
    // Type assert to access UnmarshalKey with options
    if vu, ok := pv.backend.(interface {
        UnmarshalKeyWithOptions(string, any, ...func(*mapstructure.DecoderConfig)) error
    }); ok {
        return vu.UnmarshalKeyWithOptions(key, target, gazDecoderOption)
    }
    
    // Fallback: use standard UnmarshalKey (uses mapstructure tag)
    return pv.backend.UnmarshalKey(key, target)
}

func gazDecoderOption(dc *mapstructure.DecoderConfig) {
    dc.TagName = "gaz"
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| mitchellh/mapstructure | go-viper/mapstructure/v2 | 2024 | Import path change only |
| Viper without options | Viper with DecoderConfigOption | v1.15+ | Enables custom tag support |
| validator without tag name | RegisterTagNameFunc | v8+ | Custom error field names |

**Deprecated/outdated:**
- `github.com/mitchellh/mapstructure`: Archived, use go-viper/mapstructure/v2
- Viper v1.x without mapstructure v2: Use v1.18+ for full v2 support

## Open Questions

Things that couldn't be fully resolved:

1. **Viper.Sub() behavior with empty parent**
   - What we know: Sub returns nil if key doesn't exist
   - What's unclear: Does it return nil if key exists but has no children?
   - Recommendation: Test during implementation, use IsSet as primary check

2. **DecodeHook composition**
   - What we know: Viper provides StringToTimeDurationHookFunc
   - What's unclear: Should we add custom decode hooks for env binding?
   - Recommendation: Start without, add if needed for env override complexity

3. **Namespace collision detection timing**
   - What we know: CONTEXT.md says "at config registration time"
   - What's unclear: Exact mechanism for tracking registered namespaces
   - Recommendation: Store registered namespaces in Manager, error on duplicate

## Sources

### Primary (HIGH confidence)
- Context7 `/spf13/viper` - UnmarshalKey, DecoderConfigOption, custom TagName
- go-viper/mapstructure v2.5.0 - DecoderConfig struct, TagName field
- https://github.com/spf13/viper/blob/master/UPGRADE.md - Custom tag example

### Secondary (MEDIUM confidence)
- pkg.go.dev/github.com/go-playground/validator/v10 - RegisterTagNameFunc
- Existing codebase: config/validation.go - Current tag name registration pattern

### Tertiary (LOW confidence)
- None - all critical patterns verified with official documentation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in go.mod, versions verified
- Architecture: HIGH - Patterns directly from official docs and existing codebase
- Pitfalls: MEDIUM - Some derived from general Viper/mapstructure experience

**Research date:** 2026-01-30
**Valid until:** 60 days (stable libraries, unlikely to change)
