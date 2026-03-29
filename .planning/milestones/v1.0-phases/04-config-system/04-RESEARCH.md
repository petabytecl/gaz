# Research: Phase 4 Config System

## 1. Objective
Implement a robust configuration system that satisfies:
- Loading from Environment Variables (with prefixes/delimiters).
- Loading from Files (YAML, JSON, TOML).
- Loading from CLI Flags (Cobra integration).
- Support for Defaults and Validation via interfaces.
- Immutability after startup.

## 2. Technology Selection
**Recommendation: `spf13/viper` (Instance Mode)**

While `koanf` is cleaner, `spf13/viper` is chosen for its native integration with `spf13/cobra`, which is already the chosen CLI framework. The requirement "Developer can load config from CLI flags with Cobra integration" is best satisfied by Viper's `BindPFlags`.

**Constraint Check:**
- We will avoid the Global State pattern (`viper.Get`) and instead use `viper.New()` to create a scoped configuration instance per `App`.
- This respects the "Clean break" and "No legacy constraints" decisions by using the library idiomatically for a modern codebase.

## 3. Architecture & API Design

### 3.1. New Interfaces
To support the "Requirements" regarding logic-based defaults and validation:

```go
// Defaulter allows a config struct to set its own default values.
// This runs after unmarshaling but before validation.
type Defaulter interface {
    Default()
}

// Validator allows a config struct to validate its own state.
// This runs after defaults are applied.
type Validator interface {
    Validate() error
}
```

### 3.2. App Structure Updates
The `App` struct needs to hold the configuration state and target.

```go
type App struct {
    // ... existing fields
    configTarget any           // Pointer to user's config struct
    viper        *viper.Viper  // The underlying viper instance
    configOpts   ConfigOptions // Options for loading (paths, name, env prefix)
}

type ConfigOptions struct {
    Name       string   // e.g. "config"
    Type       string   // e.g. "yaml"
    Paths      []string // e.g. [".", "/etc/app"]
    EnvPrefix  string   // e.g. "GAZ"
    ProfileEnv string   // e.g. "APP_ENV" to determine profile (dev, prod)
}
```

### 3.3. API Flow
The user experience should be fluent:

```go
type Config struct {
    Port int `mapstructure:"port"`
}

func (c *Config) Default() {
    if c.Port == 0 { c.Port = 8080 }
}

func main() {
    var cfg Config
    
    app := gaz.New().
        WithConfig(&cfg, gaz.ConfigOpts{
            EnvPrefix: "APP",
            Paths: []string{"."},
        }).
        ProvideSingleton(func() *Config { return &cfg }). // Auto-registration?
        WithCobra(rootCmd)
    
    // ...
}
```

**Refinement:** `gaz` should probably *auto-register* the config struct as a Singleton if `WithConfig` is used. This removes boilerplate for the user.

### 3.4. Lifecycle Integration
The configuration loading must happen **before** `Build()` because eager services might depend on the config.

- **Current Flow:** `WithCobra` -> `PreRun` -> `Build()` -> `Start()`
- **New Flow:** `WithCobra` -> `PreRun` -> `LoadConfig()` -> `Build()` -> `Start()`

## 4. Implementation Details

### 4.1. Loading Strategy (Precedence)
1.  **Defaults:** Set via `viper.SetDefault` (if using map) or `Defaulter` interface (post-load). *Decision:* Use `Defaulter` interface on the struct as the primary mechanism for logic-based defaults.
2.  **File (Base):** `viper.ReadInConfig()`.
3.  **File (Profile):** Check env var (e.g. `APP_ENV=prod`). If set, `viper.SetConfigName("config.prod")`, then `viper.MergeInConfig()`.
4.  **Environment:** `viper.AutomaticEnv()`.
    - `SetEnvPrefix("GAZ")`
    - `SetEnvKeyReplacer(strings.NewReplacer(".", "__"))`
5.  **Flags:** `viper.BindPFlags(cmd.Flags())`.
    - This must happen inside the `PreRun` hook where the `cmd` is available.

### 4.2. Validation
After `viper.Unmarshal(app.configTarget)`:
1.  Cast `configTarget` to `Defaulter` (if applicable) and call `Default()`.
2.  Cast `configTarget` to `Validator` (if applicable) and call `Validate()`.
    - If validation fails, abort startup.

### 4.3. Array Handling
Viper's `Unmarshal` behavior naturally replaces slices in the struct if the key exists in the source. This satisfies the "Replace strategy" requirement.

### 4.4. Dependency
Need to add `github.com/spf13/viper` to `go.mod`.

## 5. Potential Pitfalls
- **Cobra Flags:** `BindPFlags` binds *all* flags. We need to ensure we don't accidentally bind flags that shouldn't be in config (though usually, if it's a flag, it's config).
- **Mapstructure Tags:** Users need to use `mapstructure:"key"` tags, not `json` or `yaml`, because Viper uses mapstructure internally. We must document this clearly.
- **Profiles:** `MergeInConfig` can be tricky if the base config and profile config have different array lengths. But with "Replace" strategy, this is actually simpler (the profile wins).

## 6. Action Plan
1.  Add `spf13/viper` dependency.
2.  Define `ConfigOptions` and Interfaces (`Validator`, `Defaulter`).
3.  Add `WithConfig(target any, opts ConfigOptions)` to `App`.
4.  Implement `App.loadConfig(cmd *cobra.Command)` method:
    - Init Viper.
    - Bind Flags (if cmd provided).
    - Load Files (Base + Profile).
    - Unmarshal.
    - Default & Validate.
    - Register `target` into DI Container (as Instance).
5.  Update `WithCobra` implementation in `gaz/cobra.go` to call `loadConfig`.
6.  Add manual `LoadConfig()` method for non-Cobra usage (testing).

