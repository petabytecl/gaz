# Phase 13: Config Package - Research

**Researched:** 2026-01-28
**Domain:** Go config package extraction, viper abstraction, validation patterns
**Confidence:** HIGH

## Summary

This phase extracts configuration functionality from the root `gaz` package into a standalone `gaz/config` subpackage, following the same extraction pattern successfully used in Phase 12 (DI package). The primary challenge is designing a Backend interface that abstracts viper while providing full feature access including file watching, environment variable binding, and config writing.

The research identified that the industry standard "Provider Interface Pattern" for 2025 aligns with the locked decisions: define an interface for the config accessor, implement it with viper in a separate subpackage, and allow standalone usage without the full framework. The existing gaz validation infrastructure using go-playground/validator v10 should be moved to the config package with minimal changes.

**Primary recommendation:** Create `gaz/config` package with composed interfaces (core `Backend` + optional `Watcher`/`Writer`/`EnvBinder`), place `ViperBackend` in `gaz/config/viper` subpackage to isolate the viper dependency, and add generic `Get[T]()` accessor for typed value retrieval.

## Standard Stack

This is a code reorganization phase - no new dependencies required.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/spf13/viper | v1.21.0 | Configuration management | Already used, industry standard for Go config |
| github.com/go-playground/validator/v10 | v10.30.1 | Struct tag validation | Already used, most popular Go validator |
| Standard library | Go 1.25.6 | reflect, sync, errors | Core implementation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/fsnotify/fsnotify | v1.9.0 | File system notifications | Used by viper for WatchConfig |
| github.com/go-viper/mapstructure/v2 | v2.4.0 | Struct unmarshaling | Used by viper for Unmarshal |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Viper | koanf | Cleaner API but would require migration |
| Viper | cleanenv | Lighter but fewer features |
| Full interface | Minimal interface | Minimal limits testing scenarios |

## Architecture Patterns

### Recommended Project Structure
```
gaz/
├── config/                      # NEW: Standalone config package
│   ├── config.go                # ConfigManager, New(), LoadInto()
│   ├── backend.go               # Backend interface and composed interfaces
│   ├── options.go               # ConfigOption, WithName(), WithSearchPaths(), etc.
│   ├── types.go                 # Defaulter, Validator interfaces
│   ├── validation.go            # validateConfigTags, ValidationErrors
│   ├── errors.go                # ErrConfigValidation and config-specific errors
│   ├── accessor.go              # Generic Get[T]() accessor
│   ├── doc.go                   # Package documentation
│   └── viper/                   # Viper implementation subpackage
│       ├── backend.go           # ViperBackend implementing Backend + Watcher + Writer + EnvBinder
│       └── doc.go               # Package documentation
├── app.go                       # App uses config.ConfigManager
├── options.go                   # App options (WithConfig removed, or delegates to config)
└── ...
```

### Pattern 1: Composed Interfaces
**What:** Core Backend interface with optional capability interfaces
**When to use:** When backends have varying feature support
**Example:**
```go
// Source: CONTEXT.md locked decision + Go interface composition pattern
package config

// Backend is the core interface for configuration access.
// All backends must implement at minimum Get/Set/Unmarshal operations.
type Backend interface {
    // Get returns the value for a key. Keys use dot notation (e.g., "database.host").
    Get(key string) any
    
    // GetString returns a string value for the key.
    GetString(key string) string
    
    // GetInt returns an int value for the key.
    GetInt(key string) int
    
    // GetBool returns a bool value for the key.
    GetBool(key string) bool
    
    // GetDuration returns a time.Duration value for the key.
    GetDuration(key string) time.Duration
    
    // GetFloat64 returns a float64 value for the key.
    GetFloat64(key string) float64
    
    // Set explicitly sets a value for a key.
    Set(key string, value any)
    
    // SetDefault sets a default value for a key.
    SetDefault(key string, value any)
    
    // IsSet checks if a key has been set.
    IsSet(key string) bool
    
    // Unmarshal unmarshals config into a struct.
    Unmarshal(target any) error
    
    // UnmarshalKey unmarshals a specific key into a struct.
    UnmarshalKey(key string, target any) error
}

// Watcher is implemented by backends that support file watching.
type Watcher interface {
    WatchConfig()
    OnConfigChange(callback func(event any))
}

// Writer is implemented by backends that can write config to files.
type Writer interface {
    WriteConfig() error
    WriteConfigAs(filename string) error
    SafeWriteConfig() error
    SafeWriteConfigAs(filename string) error
}

// EnvBinder is implemented by backends that support environment variable binding.
type EnvBinder interface {
    SetEnvPrefix(prefix string)
    AutomaticEnv()
    BindEnv(keys ...string) error
    SetEnvKeyReplacer(replacer StringReplacer)
}

// StringReplacer is used for env key replacement.
type StringReplacer interface {
    Replace(s string) string
}
```

### Pattern 2: Generic Typed Accessor
**What:** `Get[T]()` function for type-safe config value retrieval
**When to use:** When caller knows expected type at compile time
**Example:**
```go
// Source: Go generics pattern for config access (WebSearch 2025)
package config

// Get retrieves a typed value from the ConfigManager.
// Returns zero value if key not found or type assertion fails.
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

// GetOr retrieves a typed value with a fallback default.
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
```

### Pattern 3: Combined Load+Unmarshal
**What:** Single method that loads from sources AND unmarshals to struct
**When to use:** Common case where user wants config in a typed struct
**Example:**
```go
// Source: CONTEXT.md locked decision
package config

// LoadInto loads configuration from all sources and unmarshals into target.
// It applies defaults (via Defaulter interface), validates (via validate tags
// and Validator interface), and returns any errors.
func (m *Manager) LoadInto(target any) error {
    // 1. Load from file(s)
    if err := m.loadFromFile(); err != nil {
        return err
    }
    
    // 2. Unmarshal into target
    if err := m.backend.Unmarshal(target); err != nil {
        return fmt.Errorf("unmarshal config: %w", err)
    }
    
    // 3. Apply defaults
    if d, ok := target.(Defaulter); ok {
        d.Default()
    }
    
    // 4. Validate struct tags
    if err := validateConfigTags(target); err != nil {
        return err
    }
    
    // 5. Custom validation
    if v, ok := target.(Validator); ok {
        if err := v.Validate(); err != nil {
            return fmt.Errorf("config validation: %w", err)
        }
    }
    
    return nil
}
```

### Pattern 4: Standalone Config Manager
**What:** `config.New()` returns Manager without requiring gaz.App
**When to use:** Using config outside the gaz framework
**Example:**
```go
// Source: DI package pattern from Phase 12
package config

import "github.com/petabytecl/gaz/config/viper"

// New creates a new ConfigManager with the default ViperBackend.
func New(opts ...Option) *Manager {
    m := &Manager{
        backend:     viper.New(),
        fileName:    "config",
        fileType:    "yaml",
        searchPaths: []string{"."},
    }
    
    for _, opt := range opts {
        opt(m)
    }
    
    return m
}

// NewWithBackend creates a ConfigManager with a custom backend.
func NewWithBackend(backend Backend, opts ...Option) *Manager {
    m := &Manager{
        backend:     backend,
        fileName:    "config",
        fileType:    "yaml",
        searchPaths: []string{"."},
    }
    
    for _, opt := range opts {
        opt(m)
    }
    
    return m
}
```

### Anti-Patterns to Avoid
- **Global config state:** Avoid package-level viper instance. Always use instance methods.
- **Import cycles:** config package must NOT import gaz package. Only gaz imports config.
- **Leaking viper types:** Backend interface should use standard Go types, not viper-specific types.
- **Validation in wrong layer:** Validation should happen in config package, not repeated in App.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Struct tag parsing | Custom tag parser | mapstructure (via viper) | Handles all edge cases, nested structs |
| Validation framework | Custom validators | go-playground/validator | Industry standard, rich tag vocabulary |
| File watching | Custom fsnotify | viper.WatchConfig() | Already handles debouncing, errors |
| Env var binding | Manual os.Getenv | viper.AutomaticEnv() | Handles prefix, key replacement |
| Multiple error collection | []error slice | errors.Join() | Go 1.20+ standard pattern |

**Key insight:** This is a reorganization phase, not a rewrite. Move code, add interface layer, don't reimplement viper features.

## Common Pitfalls

### Pitfall 1: Import Cycles Between gaz and config
**What goes wrong:** Root gaz package imports config, but config imports something from gaz
**Why it happens:** Sharing types or error definitions between packages
**How to avoid:**
- config package has zero imports from parent gaz package
- Move ALL config-related types to config package
- App references config.Manager, not vice versa
**Warning signs:** Go import cycle compile error; any `import "github.com/petabytecl/gaz"` in config/ files

### Pitfall 2: Incomplete Backend Interface
**What goes wrong:** Backend interface missing methods that App needs
**Why it happens:** Only considering happy path, forgetting edge cases
**How to avoid:**
- Review ALL viper methods currently used in gaz
- Include IsSet, SetDefault, Get* variants
- Design for testing (mock implementations)
**Warning signs:** App can't call needed methods after extraction

### Pitfall 3: Validator Singleton State
**What goes wrong:** Global validator.Validate instance causes test interference
**Why it happens:** Current code uses package-level singleton for performance
**How to avoid:**
- Keep singleton pattern (it's thread-safe)
- Document that validator instance is shared
- Don't register custom validators at runtime
**Warning signs:** Race conditions in parallel tests

### Pitfall 4: Breaking Existing Config API
**What goes wrong:** Users' code breaks after extraction
**Why it happens:** Changing method signatures or removing functionality
**How to avoid:**
- WithConfig[T]() in App should work identically
- Keep ConfigOption functions compatible
- Add backward compatibility shims if needed
**Warning signs:** Existing tests fail; examples don't compile

### Pitfall 5: Viper Package Dependency Leak
**What goes wrong:** Users who don't want viper still get it transitively
**Why it happens:** ViperBackend in main config package
**How to avoid:**
- Put ViperBackend in `config/viper` subpackage
- Main config package has NO viper import
- App imports config/viper internally
**Warning signs:** `go mod graph` shows viper for users who only import config

## Code Examples

### Standalone Config Usage
```go
// Source: Research synthesis, CONTEXT.md pattern
package main

import (
    "fmt"
    "log"
    
    "github.com/petabytecl/gaz/config"
    _ "github.com/petabytecl/gaz/config/viper" // registers viper backend
)

type AppConfig struct {
    Server struct {
        Host string `mapstructure:"host" validate:"required"`
        Port int    `mapstructure:"port" validate:"min=1,max=65535"`
    } `mapstructure:"server"`
}

func main() {
    cfg := &AppConfig{}
    
    mgr := config.New(
        config.WithName("config"),
        config.WithSearchPaths(".", "./config"),
        config.WithEnvPrefix("APP"),
    )
    
    if err := mgr.LoadInto(cfg); err != nil {
        log.Fatalf("config load failed: %v", err)
    }
    
    fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
}
```

### Generic Accessor Usage
```go
// Source: Go generics pattern
package main

import "github.com/petabytecl/gaz/config"

func main() {
    mgr := config.New()
    mgr.LoadInto(&cfg)
    
    // Type-safe access for individual values
    port := config.Get[int](mgr, "server.port")
    host := config.GetOr(mgr, "server.host", "localhost")
    debug := config.Get[bool](mgr, "debug")
}
```

### App Integration
```go
// Source: Current gaz pattern, preserved
package main

import "github.com/petabytecl/gaz"

type Config struct {
    Debug bool `mapstructure:"debug"`
}

func main() {
    cfg := &Config{}
    app := gaz.New().WithConfig(cfg,
        gaz.WithName("config"),
        gaz.WithEnvPrefix("MYAPP"),
    )
    
    // Works exactly as before
    if err := app.Build(); err != nil {
        log.Fatal(err)
    }
}
```

### ValidationErrors Collection
```go
// Source: Current gaz validation.go + CONTEXT.md requirement
package config

// ValidationErrors holds multiple validation errors.
// Implements error interface and provides access to individual errors.
type ValidationErrors struct {
    Errors []FieldError
}

func (ve ValidationErrors) Error() string {
    var msgs []string
    for _, e := range ve.Errors {
        msgs = append(msgs, e.String())
    }
    return fmt.Sprintf("%w:\n%s", ErrConfigValidation, strings.Join(msgs, "\n"))
}

// FieldError represents a single field validation failure.
type FieldError struct {
    Namespace string // e.g., "Config.database.host"
    Tag       string // e.g., "required"
    Param     string // e.g., "5" for min=5
    Message   string // Human-readable message
}

func (fe FieldError) String() string {
    return fmt.Sprintf("%s: %s (validate:\"%s\")", fe.Namespace, fe.Message, fe.Tag)
}
```

### ViperBackend Implementation
```go
// Source: viper Context7 docs + current ConfigManager
package viper

import (
    "github.com/petabytecl/gaz/config"
    "github.com/spf13/viper"
)

// Backend implements config.Backend + config.Watcher + config.Writer + config.EnvBinder.
type Backend struct {
    v *viper.Viper
}

// New creates a new ViperBackend.
func New() *Backend {
    return &Backend{v: viper.New()}
}

// Get implements config.Backend.
func (b *Backend) Get(key string) any {
    return b.v.Get(key)
}

func (b *Backend) GetString(key string) string {
    return b.v.GetString(key)
}

func (b *Backend) GetInt(key string) int {
    return b.v.GetInt(key)
}

// ... other Backend methods

// WatchConfig implements config.Watcher.
func (b *Backend) WatchConfig() {
    b.v.WatchConfig()
}

func (b *Backend) OnConfigChange(callback func(event any)) {
    b.v.OnConfigChange(func(e fsnotify.Event) {
        callback(e)
    })
}

// WriteConfig implements config.Writer.
func (b *Backend) WriteConfig() error {
    return b.v.WriteConfig()
}

// ... other Writer methods

// SetEnvPrefix implements config.EnvBinder.
func (b *Backend) SetEnvPrefix(prefix string) {
    b.v.SetEnvPrefix(prefix)
}

// ... other EnvBinder methods
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Global viper state | Instance-based viper | viper v1.x | Already used in gaz |
| interface{} assertions | Generic Get[T]() | Go 1.18 (2022) | Implement in this phase |
| Single validation error | errors.Join() multi-error | Go 1.20 (2023) | Already used in gaz |
| Separate Load + Unmarshal | Combined LoadInto() | 2025 pattern | Implement per CONTEXT.md |

**Deprecated/outdated:**
- Global viper functions (viper.Get): Use instance methods instead
- Manual mapstructure decoding: Use viper.Unmarshal()
- interface{} for config values: Use generics when possible

## Open Questions

Things that couldn't be fully resolved:

1. **Backend nil-safety**
   - What we know: CONTEXT.md lists this as Claude's discretion
   - What's unclear: Should nil Backend panic or return zero values?
   - Recommendation: Require non-nil Backend in Manager constructor. Panic early with clear message if nil Backend passed.

2. **Defaulter/Validator interface location**
   - What we know: Currently in root gaz package as interfaces
   - What's unclear: Should config re-export or define its own?
   - Recommendation: Define in config package. App can type-alias or users import config directly.

3. **ConfigManager registration in DI**
   - What we know: CONTEXT.md says "ConfigManager is registered in DI container"
   - What's unclear: Does config package register itself, or does App register it?
   - Recommendation: App registers ConfigManager during Build(). Config package stays DI-agnostic.

## Sources

### Primary (HIGH confidence)
- `/spf13/viper` (Context7) - Get/Set/Unmarshal API, WatchConfig, env binding, WriteConfig
- `/go-playground/validator` (Context7) - ValidationErrors, struct validation
- Current gaz codebase - config.go, config_manager.go, validation.go, options.go

### Secondary (MEDIUM confidence)
- WebSearch "Go config package interface pattern 2025" - Provider Interface Pattern
- WebSearch "Go generics config Get T accessor" - Generic accessor patterns
- Phase 12 research - DI extraction patterns (applied to config)

### Tertiary (LOW confidence)
- None - all patterns verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Using existing dependencies, no new additions
- Architecture: HIGH - Based on locked decisions in CONTEXT.md and DI extraction precedent
- Pitfalls: HIGH - Based on Phase 12 experience and code analysis

**Research date:** 2026-01-28
**Valid until:** 2026-02-28 (stable domain, no external dependencies changing)

## Files to Move

Complete inventory of config-related code to extract:

| Current File | Target File | What Moves | Notes |
|--------------|-------------|------------|-------|
| config.go | config/types.go | Defaulter, Validator interfaces | Unchanged |
| config_manager.go | config/manager.go | ConfigManager struct, Load() | Rename to Manager, add LoadInto() |
| validation.go | config/validation.go | validateConfigTags, humanizeTag, configValidator | Unchanged logic |
| options.go | config/options.go | ConfigOption, WithName, WithType, WithEnvPrefix, WithSearchPaths, WithProfileEnv, WithDefaults | Keep in config package |
| errors.go | config/errors.go | ErrConfigValidation | Move only config errors |
| (new) | config/backend.go | Backend interface, Watcher, Writer, EnvBinder | New interfaces |
| (new) | config/accessor.go | Get[T](), GetOr[T]() | New generic accessors |
| (new) | config/viper/backend.go | ViperBackend struct + methods | New viper implementation |
| provider_config.go | (stays in gaz) | ConfigProvider, ConfigFlag, ProviderValues | Framework integration |

**Files staying in root gaz:**
- app.go - Uses config.Manager internally
- provider_config.go - Framework-specific provider config integration
- errors.go - Re-exports ErrConfigValidation for compatibility

**New file in root gaz:**
- (none needed) - App.WithConfig() delegates to config package

## Public API for config Package

### Exported (PUBLIC)
```go
// Core types
type Manager struct { ... }
type Option func(*Manager)

// Interfaces
type Backend interface { ... }
type Watcher interface { ... }
type Writer interface { ... }
type EnvBinder interface { ... }
type Defaulter interface { Default() }
type Validator interface { Validate() error }

// Constructors
func New(opts ...Option) *Manager
func NewWithBackend(backend Backend, opts ...Option) *Manager

// Manager methods
func (m *Manager) Load() error
func (m *Manager) LoadInto(target any) error
func (m *Manager) Backend() Backend

// Generic accessors
func Get[T any](m *Manager, key string) T
func GetOr[T any](m *Manager, key string, fallback T) T

// Options
func WithName(name string) Option
func WithType(t string) Option
func WithEnvPrefix(prefix string) Option
func WithSearchPaths(paths ...string) Option
func WithProfileEnv(envVar string) Option
func WithDefaults(defaults map[string]any) Option
func WithBackend(backend Backend) Option

// Errors
var ErrConfigValidation error
type ValidationErrors struct { ... }
type FieldError struct { ... }
```

### config/viper Subpackage
```go
// Viper implementation
type Backend struct { ... }

func New() *Backend
func NewWithOptions(opts ...viper.Option) *Backend
```
