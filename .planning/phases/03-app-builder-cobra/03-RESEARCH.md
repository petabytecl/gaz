# Phase 3: App Builder + Cobra - Research

**Researched:** 2026-01-26
**Domain:** Go DI framework fluent API + Cobra CLI integration
**Confidence:** HIGH

## Summary

This phase builds the developer-facing fluent API for the `gaz` DI framework. Research focused on functional options patterns, error aggregation, module composition patterns, and Cobra CLI integration. The design decisions are already locked: `gaz.New(opts ...Option)` with functional options, separate `Build()`/`Run()` phases, modules as functions returning `[]Provider`, and `WithCobra(rootCmd)` for Cobra integration.

The functional options pattern is the established Go idiom for configurable APIs (Dave Cheney 2014, Rob Pike 2014, uber-go/fx). Go 1.20+'s `errors.Join()` is the standard for aggregating multiple errors during Build() validation. Cobra's `PersistentPreRunE` and context passing via `ExecuteContext()` are the canonical patterns for integrating app lifecycle with CLI commands.

**Primary recommendation:** Use Dave Cheney's functional options pattern for `gaz.New()`, standard library `errors.Join()` for multi-error Build() validation, and Cobra's `PersistentPreRunE` hook with `ExecuteContext()` for lifecycle integration.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/spf13/cobra | v1.9.1+ | CLI framework | Industry standard (Kubernetes, Hugo, GitHub CLI) |
| errors (std) | go1.20+ | Error aggregation with errors.Join | Built-in, works with errors.Is/errors.As |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| context (std) | - | Context propagation for lifecycle | Always for timeouts and cancellation |
| sync (std) | - | Concurrency primitives | App state management |

### Not Needed This Phase
| Library | Why Not |
|---------|---------|
| go.uber.org/multierr | errors.Join in Go 1.20+ replaces this |
| github.com/hashicorp/go-multierror | Same - use stdlib errors.Join |

**Installation:**
```bash
go get github.com/spf13/cobra@latest
```

## Architecture Patterns

### Recommended Project Structure
```
gaz/
├── app.go           # App struct, New(), Build(), Run(), Stop()
├── app_options.go   # AppOption type and option functions
├── provider.go      # Provider type, scope methods
├── module.go        # Module() helper function
├── cobra.go         # WithCobra() and Cobra integration
├── errors.go        # Sentinel errors (existing)
└── container.go     # Core DI container (existing)
```

### Pattern 1: Functional Options Pattern
**What:** Configuration via variadic function parameters
**When to use:** Any constructor or function with multiple optional parameters
**Source:** Dave Cheney (2014), Rob Pike (2014), uber-go/fx

```go
// Source: Dave Cheney - Functional options for friendly APIs
// https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis

// Option modifies App configuration
type Option func(*App)

// WithShutdownTimeout sets graceful shutdown timeout
func WithShutdownTimeout(d time.Duration) Option {
    return func(a *App) {
        a.shutdownTimeout = d
    }
}

// WithLogger sets the application logger
func WithLogger(l Logger) Option {
    return func(a *App) {
        a.logger = l
    }
}

// New creates an App with defaults, then applies options
func New(opts ...Option) *App {
    app := &App{
        shutdownTimeout: 30 * time.Second,
        logger:          defaultLogger(),
    }
    for _, opt := range opts {
        opt(app)
    }
    return app
}

// Usage: clean, expressive, no nil or empty values needed
app := gaz.New(
    gaz.WithShutdownTimeout(10 * time.Second),
    gaz.WithLogger(myLogger),
)
```

### Pattern 2: Builder Pattern (Mutable - Recommended)
**What:** Fluent method chaining that mutates and returns same receiver
**When to use:** For Provider/Registration configuration
**Why Mutable:** Go convention for builders; immutable creates unexpected allocation

```go
// Source: Existing gaz pattern + Go standard library patterns

// Mutable builder - returns *self for chaining
type App struct {
    providers []Provider
    modules   []module
    // ...
}

func (a *App) ProvideSingleton(p Provider) *App {
    a.providers = append(a.providers, withScope(p, Singleton))
    return a
}

func (a *App) ProvideTransient(p Provider) *App {
    a.providers = append(a.providers, withScope(p, Transient))
    return a
}

// Chaining works naturally
app.ProvideSingleton(NewDB).
    ProvideSingleton(NewCache).
    ProvideTransient(NewRequest)
```

**Note on immutability:** Immutable builders in Go are unusual and create extra allocations. The Go ecosystem favors mutable builders (http.Request.WithContext returns new, but most builders mutate). For gaz, **use mutable builder** as it matches expectations.

### Pattern 3: Error Aggregation with errors.Join (Go 1.20+)
**What:** Collect multiple errors during validation
**When to use:** Build() phase validation
**Source:** Go standard library errors package

```go
// Source: https://pkg.go.dev/errors#Join

func (a *App) Build() error {
    var errs []error
    
    // Validate all registrations
    for _, p := range a.providers {
        if err := a.validateProvider(p); err != nil {
            errs = append(errs, err)
        }
    }
    
    // Check for cycles
    if cycles := a.detectCycles(); len(cycles) > 0 {
        for _, cycle := range cycles {
            errs = append(errs, fmt.Errorf("%w: %s", ErrCyclicDependency, cycle))
        }
    }
    
    // Check for missing dependencies
    for _, missing := range a.findMissingDeps() {
        errs = append(errs, fmt.Errorf("%w: %s", ErrMissingDependency, missing))
    }
    
    if len(errs) > 0 {
        return errors.Join(errs...) // Go 1.20+
    }
    
    a.built = true
    return nil
}
```

**errors.Join behavior:**
- Returns nil if all errors are nil
- Single error returned unwrapped
- Multiple errors joined with newlines in Error()
- Works with errors.Is() and errors.As() for any contained error

### Pattern 4: Module as Function Pattern
**What:** Modules are functions returning providers, not types
**When to use:** Organizing related services
**Source:** uber-go/fx Module pattern (simplified for gaz)

```go
// Source: uber-go/fx Module pattern, adapted for gaz

// Module groups related providers with a name for debugging
func Module(name string, providers ...Provider) []Provider {
    // Tag each provider with module name for debugging/error messages
    tagged := make([]Provider, len(providers))
    for i, p := range providers {
        tagged[i] = withModuleName(p, name)
    }
    return tagged
}

// Usage: composable, simple function returns
func DatabaseModule(cfg DatabaseConfig) []Provider {
    return gaz.Module("database",
        gaz.ProvideWith(func() *sql.DB { return openDB(cfg) }),
        gaz.Provide[UserRepository](NewUserRepo),
        gaz.Provide[PostRepository](NewPostRepo),
    )
}

// In app setup
app := gaz.New().
    Provide(DatabaseModule(dbConfig)...).
    Provide(HTTPModule()...)
```

### Pattern 5: Cobra Integration via PersistentPreRunE
**What:** Hook app lifecycle into Cobra command tree
**When to use:** CLI applications with DI
**Source:** Cobra official documentation + user guide

```go
// Source: spf13/cobra user guide + pkg.go.dev/github.com/spf13/cobra

// WithCobra attaches app lifecycle to Cobra command
func (a *App) WithCobra(cmd *cobra.Command) *App {
    // Store app in command context for subcommand access
    originalPreRunE := cmd.PersistentPreRunE
    
    cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
        // Chain original if exists
        if originalPreRunE != nil {
            if err := originalPreRunE(c, args); err != nil {
                return err
            }
        }
        
        // Build the app (validation phase)
        if err := a.Build(); err != nil {
            return err
        }
        
        // Start lifecycle hooks
        ctx := c.Context()
        if ctx == nil {
            ctx = context.Background()
        }
        if err := a.Start(ctx); err != nil {
            return err
        }
        
        // Store app in context for resolution in subcommands
        c.SetContext(contextWithApp(ctx, a))
        return nil
    }
    
    // Cleanup in PersistentPostRunE
    originalPostRunE := cmd.PersistentPostRunE
    cmd.PersistentPostRunE = func(c *cobra.Command, args []string) error {
        // Stop lifecycle (reverse order)
        ctx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
        defer cancel()
        
        stopErr := a.Stop(ctx)
        
        // Chain original if exists
        if originalPostRunE != nil {
            if err := originalPostRunE(c, args); err != nil {
                return errors.Join(stopErr, err)
            }
        }
        
        return stopErr
    }
    
    return a
}

// Subcommands resolve from context
var serveCmd = &cobra.Command{
    Use: "serve",
    RunE: func(cmd *cobra.Command, args []string) error {
        app := gaz.FromContext(cmd.Context())
        server, err := gaz.Resolve[*HTTPServer](app)
        if err != nil {
            return err
        }
        return server.ListenAndServe()
    },
}
```

### Anti-Patterns to Avoid

- **Separate New() and Start():** Don't mix construction with side effects. Build() validates, Run() executes.
- **Nil options:** Never require nil or empty values. Variadic options eliminate this.
- **Global state for DI:** Don't use package-level vars for container access. Pass via context.
- **Ignoring Cobra context:** Always use cmd.Context() not context.Background() in commands.
- **Missing error collection:** Don't fail on first validation error - collect all for better DX.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Multi-error aggregation | Custom error slice type | errors.Join() | Standard library, works with Is/As |
| CLI structure | Custom arg parsing | github.com/spf13/cobra | Industry standard, completions, help |
| Context cancellation | Manual channel signals | context.WithTimeout | Standard, composable, well-understood |
| Shutdown signals | os.Signal handling | Already in gaz/app.go | Keep existing, proven implementation |
| Option parsing | Config struct | Functional options | Cleaner API, no nil/empty values |

**Key insight:** Go 1.20+ solved the multi-error problem with errors.Join(). Don't use external libraries or custom types for error aggregation.

## Common Pitfalls

### Pitfall 1: Immutable Builder Confusion
**What goes wrong:** Creating immutable builders that allocate on every method call
**Why it happens:** Coming from Java/functional programming backgrounds
**How to avoid:** Use mutable builders (mutate receiver, return *self)
**Warning signs:** Seeing `new(App)` in every method, unexpected allocations

### Pitfall 2: Context Loss in Cobra
**What goes wrong:** Subcommands don't have access to app/container
**Why it happens:** Not propagating context through command tree
**How to avoid:** Use cmd.SetContext() in PersistentPreRunE, cmd.Context() in RunE
**Warning signs:** "nil app" or "container not found" in subcommand handlers

### Pitfall 3: Build() After Run()
**What goes wrong:** Registration after Build() causes silent failures or panics
**Why it happens:** API allows calling Provide() after Build()
**How to avoid:** Add `built` flag, return error from Provide() if already built
**Warning signs:** Missing services, silent registration failures

### Pitfall 4: Single Error on Build Failure
**What goes wrong:** Build() returns first error, hiding other problems
**Why it happens:** Early return pattern without aggregation
**How to avoid:** Collect all errors with errors.Join(), validate everything
**Warning signs:** Fixing one error reveals another, poor developer experience

### Pitfall 5: Missing Lifecycle Cleanup on Error
**What goes wrong:** Start() fails but already-started services not stopped
**Why it happens:** No rollback logic when partial startup occurs
**How to avoid:** Track started services, call Stop() in reverse on error
**Warning signs:** Resource leaks, dangling connections after failed startup

### Pitfall 6: Cobra Hook Override
**What goes wrong:** WithCobra() replaces existing PersistentPreRunE
**Why it happens:** Direct assignment instead of chaining
**How to avoid:** Store original hook, call it first, then add app logic
**Warning signs:** Other hooks stop working after WithCobra()

## Code Examples

Verified patterns from official sources:

### Complete App with Functional Options
```go
// Full example of gaz.New() with functional options

// Option configures the App
type Option func(*App)

// App is the application container
type App struct {
    container       *Container
    shutdownTimeout time.Duration
    logger          Logger
    providers       []providerEntry
    modules         map[string]bool  // track module names
    built           bool
    running         bool
    mu              sync.Mutex
    stopCh          chan struct{}
}

// New creates a new App with sensible defaults
func New(opts ...Option) *App {
    app := &App{
        container:       newContainer(),
        shutdownTimeout: 30 * time.Second,
        logger:          defaultLogger(),
        providers:       make([]providerEntry, 0),
        modules:         make(map[string]bool),
    }
    
    for _, opt := range opts {
        opt(app)
    }
    
    return app
}

// WithShutdownTimeout configures graceful shutdown timeout
func WithShutdownTimeout(d time.Duration) Option {
    return func(a *App) {
        a.shutdownTimeout = d
    }
}

// WithLogger configures the application logger
func WithLogger(l Logger) Option {
    return func(a *App) {
        a.logger = l
    }
}
```

### Scope-Specific Provider Methods
```go
// Scope-specific registration methods

func (a *App) ProvideSingleton(provider any) *App {
    if a.built {
        panic("cannot add providers after Build()")
    }
    a.providers = append(a.providers, providerEntry{
        provider: provider,
        scope:    ScopeSingleton,
    })
    return a
}

func (a *App) ProvideTransient(provider any) *App {
    if a.built {
        panic("cannot add providers after Build()")
    }
    a.providers = append(a.providers, providerEntry{
        provider: provider,
        scope:    ScopeTransient,
    })
    return a
}

func (a *App) ProvideEager(provider any) *App {
    if a.built {
        panic("cannot add providers after Build()")
    }
    a.providers = append(a.providers, providerEntry{
        provider: provider,
        scope:    ScopeSingleton,
        eager:    true,
    })
    return a
}
```

### Build with Error Aggregation
```go
// Source: errors.Join from Go 1.20+

func (a *App) Build() error {
    a.mu.Lock()
    defer a.mu.Unlock()
    
    if a.built {
        return nil // idempotent
    }
    
    var errs []error
    
    // Register all providers with container
    for _, p := range a.providers {
        if err := a.registerProvider(p); err != nil {
            errs = append(errs, err)
        }
    }
    
    // Validate dependency graph
    if graphErrs := a.container.ValidateGraph(); len(graphErrs) > 0 {
        errs = append(errs, graphErrs...)
    }
    
    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    
    // Instantiate eager services
    if err := a.container.Build(); err != nil {
        return err
    }
    
    a.built = true
    return nil
}
```

### Module with Named Registration
```go
// Module groups providers with a name for debugging

type module struct {
    name      string
    providers []any
}

// Module creates a named group of providers
func Module(name string, providers ...any) *module {
    return &module{
        name:      name,
        providers: providers,
    }
}

// App.Module registers a named module
func (a *App) Module(name string, providers ...any) *App {
    if a.built {
        panic("cannot add modules after Build()")
    }
    
    // Check for duplicate module name
    if a.modules[name] {
        a.providers = append(a.providers, providerEntry{
            err: fmt.Errorf("%w: module %q", ErrDuplicateModule, name),
        })
        return a
    }
    a.modules[name] = true
    
    // Add all providers with module tag
    for _, p := range providers {
        a.providers = append(a.providers, providerEntry{
            provider:   p,
            moduleName: name,
        })
    }
    
    return a
}
```

### Cobra Integration with Context Propagation
```go
// Source: spf13/cobra ExecuteContext and PersistentPreRunE

type contextKey struct{}

// FromContext retrieves the App from context
func FromContext(ctx context.Context) *App {
    if app, ok := ctx.Value(contextKey{}).(*App); ok {
        return app
    }
    return nil
}

// WithCobra attaches app lifecycle to a Cobra command
func (a *App) WithCobra(cmd *cobra.Command) *App {
    // Preserve existing hooks
    originalPreRunE := cmd.PersistentPreRunE
    originalPostRunE := cmd.PersistentPostRunE
    
    cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
        // Chain original hook
        if originalPreRunE != nil {
            if err := originalPreRunE(c, args); err != nil {
                return err
            }
        }
        
        // Build validates and wires
        if err := a.Build(); err != nil {
            return fmt.Errorf("app build failed: %w", err)
        }
        
        // Get context (Cobra sets background if none)
        ctx := c.Context()
        
        // Start lifecycle hooks
        if err := a.Start(ctx); err != nil {
            return fmt.Errorf("app start failed: %w", err)
        }
        
        // Make app available to subcommands via context
        c.SetContext(context.WithValue(ctx, contextKey{}, a))
        
        return nil
    }
    
    cmd.PersistentPostRunE = func(c *cobra.Command, args []string) error {
        // Stop with timeout
        stopCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
        defer cancel()
        
        stopErr := a.Stop(stopCtx)
        
        // Chain original hook
        if originalPostRunE != nil {
            if err := originalPostRunE(c, args); err != nil {
                return errors.Join(stopErr, err)
            }
        }
        
        return stopErr
    }
    
    return a
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Custom multi-error types | errors.Join() | Go 1.20 (2023) | Use stdlib, no external deps |
| Config struct for options | Functional options | 2014+ | Cleaner API, no nil values |
| go.uber.org/multierr | errors.Join() | Go 1.20 (2023) | Fewer dependencies |

**Deprecated/outdated:**
- `go.uber.org/multierr`: Replaced by stdlib errors.Join() for new code
- Config struct with pointers for optional fields: Functional options pattern preferred

## Open Questions

Things that couldn't be fully resolved:

1. **Manual lifecycle override for commands**
   - What we know: Some commands (like `version`) shouldn't trigger full lifecycle
   - What's unclear: Best API for opting out (`WithCobra(cmd, Options{SkipLifecycle: true})` vs annotations)
   - Recommendation: Start with `cmd.Annotations["gaz:skipLifecycle"] = "true"` check in PreRunE

2. **Module nesting depth**
   - What we know: Modules are functions returning providers, can compose
   - What's unclear: Should nested modules inherit parent name? (`database.users` vs just `users`)
   - Recommendation: Keep it simple - flat module names, user can use dots if desired

3. **Provider type assertion vs generics**
   - What we know: Need to accept various provider signatures
   - What's unclear: Best balance between type safety and flexibility
   - Recommendation: Use existing `For[T]()` pattern, add `ProvideFunc` helpers

## Sources

### Primary (HIGH confidence)
- `/spf13/cobra` via Context7 - PersistentPreRunE, ExecuteContext, hook order
- `/uber-go/fx` via Context7 - Module pattern, lifecycle hooks, Option interface
- Go stdlib errors package - errors.Join documentation and examples
- Dave Cheney "Functional options for friendly APIs" (2014) - https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
- Cobra User Guide - https://github.com/spf13/cobra/blob/main/site/content/user_guide.md

### Secondary (MEDIUM confidence)
- uber-go/fx source code (app.go) - Option implementation patterns
- Existing gaz codebase - Current patterns to extend

### Tertiary (LOW confidence)
- None - all findings verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Cobra and stdlib are definitive choices
- Architecture: HIGH - Patterns verified from official sources
- Pitfalls: HIGH - Common issues documented in multiple sources

**Research date:** 2026-01-26
**Valid until:** 60 days (stable patterns, established libraries)
