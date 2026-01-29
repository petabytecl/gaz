# Feature Landscape: v2.1 API Enhancement

**Domain:** Go DI Framework - Interface Auto-Detection and RuntimeX-Inspired Features  
**Researched:** 2026-01-29  
**Research Confidence:** HIGH (Context7 + Official Docs + Existing Codebase Analysis)

---

## Executive Summary

This research documents feature patterns for GAZ's v2.1 API Enhancement milestone, focusing on 8 specific features that build upon GAZ's existing `For[T]()` fluent API, lifecycle management, and Cobra integration. The findings are drawn from:

- **uber-go/fx**: Lifecycle hooks, fxtest, fx.Module bundling
- **samber/do v2**: Shutdowner interface auto-detection patterns
- **spf13/cobra**: PreRun/PostRun hooks, command context
- **Internal `_tmp_trust/runtimex`**: Build info, frame introspection, service builder patterns

GAZ already has strong foundations. These enhancements add developer experience improvements without architectural rewrites.

---

## GAZ Current State (What's Already Built)

Understanding what GAZ already has is critical for scoping new features:

| Component | Current Implementation | Location |
|-----------|------------------------|----------|
| DI Registration | `gaz.For[T](c).Provider(fn)` fluent API | `container.go` |
| Lifecycle Interfaces | `Starter`, `Stopper` interfaces defined | `lifecycle.go:28-38` |
| Lifecycle Execution | `App.Start()`, `App.Stop()` ordered by dependency graph | `app.go`, `cobra.go` |
| Hook Registration | `OnStart(fn)`, `OnStop(fn)` function-based | Per-service via `di.ServiceWrapper` |
| Cobra Integration | `WithCobra(cmd)`, `FromContext(ctx)`, `RegisterCobraFlags` | `cobra.go` |
| Config | `ConfigProvider` interface, `ProviderValues` DI | `config_manager.go` |
| Workers | `worker.Worker` interface, auto-discovery | `app.go:419-444` |
| Health | Health check package with server | `health/` |

---

## Feature 1: Interface Auto-Detection for Starter/Stopper

### Current Pattern (GAZ Today)
```go
// User must explicitly register lifecycle hooks
gaz.For[*HTTPServer](app).
    Provider(NewHTTPServer).
    OnStart(func(ctx context.Context) error { return srv.Start(ctx) }).
    OnStop(func(ctx context.Context) error { return srv.Stop(ctx) })
```

### Proposed Pattern (v2.1)
```go
// If HTTPServer implements gaz.Starter and/or gaz.Stopper, 
// hooks are registered automatically
gaz.For[*HTTPServer](app).Provider(NewHTTPServer) // Auto-detects OnStart/OnStop
```

### How This Works in the Ecosystem

| Framework | Pattern | Mechanism |
|-----------|---------|-----------|
| **uber-go/fx** | `fx.Lifecycle.Append()` in constructor | Explicit hook registration via injected lifecycle |
| **samber/do v2** | `Shutdowner` interface variants | Auto-detected at shutdown time, not registration |
| **Spring (Java)** | `@PostConstruct`, `@PreDestroy` | Reflection on annotations |

**samber/do's Interface Hierarchy (Shutdowner variants):**
```go
type Shutdowner interface { Shutdown() }
type ShutdownerWithContext interface { Shutdown(context.Context) }
type ShutdownerWithError interface { Shutdown() error }
type ShutdownerWithContextAndError interface { Shutdown(context.Context) error }
```
*Source: Context7 - samber/do v2 documentation (HIGH confidence)*

### GAZ Implementation Recommendation

**Use reflection at registration time, not runtime:**

```go
// In For[T]().Provider() registration flow:
func (b *FluentBuilder[T]) Provider(constructor any) error {
    // ... existing provider registration ...
    
    // After instance type is known, check for interfaces
    var zeroT T
    instanceType := reflect.TypeOf(zeroT)
    
    if instanceType.Implements(starterInterface) {
        b.service.SetAutoStart(true)
    }
    if instanceType.Implements(stopperInterface) {
        b.service.SetAutoStop(true)
    }
}
```

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Complexity | **Medium** |
| Breaking changes | None - additive, existing explicit hooks still work |
| Dependencies on existing | Requires modification to `di.ServiceWrapper` |
| Risk | Low - Go's interface satisfaction is compile-time safe |

### Table Stakes vs Differentiator

**Differentiator** - Most Go DI frameworks (fx, wire) require explicit hook registration. samber/do has auto-detection but only at shutdown. GAZ's approach (detection at registration, execution at lifecycle) is cleaner.

---

## Feature 2: Build Info Package

### What It Does

Injects build metadata (version, commit, branch, build time) into the application, available via DI and Cobra `--version` flag.

### Ecosystem Patterns

| Framework | Pattern | Injection Method |
|-----------|---------|------------------|
| **runtimex** | Package-level vars with ldflags | `RegisterBuildInfo(cmd, name)` |
| **go-zero** | `version.Full()` function | Build-time generation |
| **runtime/debug** | `debug.ReadBuildInfo()` | VCS info from Go module |

**Go 1.18+ runtime/debug.BuildInfo:**
```go
type BuildInfo struct {
    GoVersion string         // e.g., "go1.23.0"
    Path      string         // Module path of main package
    Main      Module         // Main module info
    Deps      []*Module      // Dependencies
    Settings  []BuildSetting // Build settings (vcs.revision, vcs.time, etc.)
}
```
*Source: pkg.go.dev/runtime/debug - Go 1.23 documentation (HIGH confidence)*

**Key BuildSettings from Go toolchain:**
- `vcs.revision` - Git commit SHA
- `vcs.time` - Commit timestamp (RFC3339)
- `vcs.modified` - "true" if working tree is dirty

### GAZ Implementation Recommendation

**Combine runtime/debug (automatic) with ldflags (explicit):**

```go
// gaz/buildinfo/info.go
package buildinfo

import (
    "runtime/debug"
    "time"
)

// Set via ldflags: -X github.com/petabytecl/gaz/buildinfo.Version=v1.0.0
var (
    Version   = "dev"
    Commit    = ""
    Branch    = ""
    BuildTime = ""
    BuildUser = ""
)

type Info struct {
    ServiceName string
    Version     string
    Commit      string
    Branch      string
    BuildTime   time.Time
    BuildUser   string
    GoVersion   string
    Dirty       bool
}

func Get(serviceName string) Info {
    info := Info{
        ServiceName: serviceName,
        Version:     Version,
        Commit:      Commit,
        Branch:      Branch,
        BuildUser:   BuildUser,
    }
    
    // Enrich with runtime/debug (automatic from VCS)
    if bi, ok := debug.ReadBuildInfo(); ok {
        info.GoVersion = bi.GoVersion
        for _, setting := range bi.Settings {
            switch setting.Key {
            case "vcs.revision":
                if info.Commit == "" {
                    info.Commit = setting.Value
                }
            case "vcs.time":
                // Parse and use if BuildTime not set
            case "vcs.modified":
                info.Dirty = setting.Value == "true"
            }
        }
    }
    
    return info
}

// Cobra integration
func RegisterVersion(cmd *cobra.Command, serviceName string) {
    info := Get(serviceName)
    cmd.Version = info.String()
}
```

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Complexity | **Low** |
| Breaking changes | None - new package |
| Dependencies on existing | None |
| Risk | Very low |

### Table Stakes vs Differentiator

**Table Stakes** - Every production service needs version info. Without this, debugging production is difficult.

---

## Feature 3: Command Arguments Injection

### What It Does

Makes CLI positional arguments (`args []string`) available as a DI dependency.

### Ecosystem Patterns

**runtimex Pattern:**
```go
const ArgsKey = "cmd.args"

func RunBuilder(builder *Builder) func(_ *cobra.Command, args []string) {
    return func(_ *cobra.Command, args []string) {
        argsSupplier := fx.Annotated{
            Name:   ArgsKey,
            Target: args,
        }
        builder = builder.WithModule(fx.Supply(argsSupplier))
        builder.Build().Run()
    }
}
```
*Source: Internal `_tmp_trust/runtimex/builder.go` (HIGH confidence)*

**uber-go/fx Pattern:**
- Uses `fx.Supply()` with `fx.Annotated{Name: ...}` for named values
- Consumer uses `fx.In` struct with `name:"..."` tag

### GAZ Implementation Recommendation

**Register args during WithCobra bootstrap:**

```go
// gaz/cobra.go - enhanced bootstrap
type CommandArgs struct {
    Args    []string
    Command *cobra.Command
}

func (a *App) bootstrap(ctx context.Context, cmd *cobra.Command) error {
    // Get args from command
    args := cmd.Flags().Args()
    
    // Register as dependency
    cmdArgs := &CommandArgs{
        Args:    args,
        Command: cmd,
    }
    
    if err := For[*CommandArgs](a.container).Instance(cmdArgs); err != nil {
        return err
    }
    
    // ... existing bootstrap logic ...
}

// Helper for consumers
func GetArgs(c *Container) []string {
    cmdArgs, _ := Resolve[*CommandArgs](c)
    if cmdArgs == nil {
        return nil
    }
    return cmdArgs.Args
}
```

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Complexity | **Low** |
| Breaking changes | None - additive |
| Dependencies on existing | Modifies `cobra.go:bootstrap()` |
| Risk | Very low |

### Table Stakes vs Differentiator

**Table Stakes** - Commands need access to positional args. Currently requires manual parsing.

---

## Feature 4: Pre/Post Run Hooks

### What It Does

Allows registering hooks that run before/after command execution, independent of lifecycle hooks.

### Ecosystem Patterns

**Cobra Native Hooks:**
```go
type Command struct {
    PersistentPreRun      func(cmd *Command, args []string)
    PersistentPreRunE     func(cmd *Command, args []string) error
    PreRun                func(cmd *Command, args []string)
    PreRunE               func(cmd *Command, args []string) error
    Run                   func(cmd *Command, args []string)
    RunE                  func(cmd *Command, args []string) error
    PostRun               func(cmd *Command, args []string)
    PostRunE              func(cmd *Command, args []string) error
    PersistentPostRun     func(cmd *Command, args []string)
    PersistentPostRunE    func(cmd *Command, args []string) error
}
```
*Source: Context7 - spf13/cobra documentation (HIGH confidence)*

**Execution Order:**
1. `PersistentPreRun` (parent commands, cascading)
2. `PreRun` (this command only)
3. `Run` (main execution)
4. `PostRun` (this command only)
5. `PersistentPostRun` (parent commands, cascading)

**Cobra Global Hooks:**
```go
func OnInitialize(y ...func()) // Runs when Execute() is called
func OnFinalize(y ...func())   // Runs when Execute() finishes (v1.6.0+)
```

### GAZ Current State

GAZ's `WithCobra()` already uses `PersistentPreRunE` and `PersistentPostRunE`:
- PreRunE: Calls `bootstrap()` (Build, Start)
- PostRunE: Calls `Stop()`

**Problem:** This consumes the hooks. Users can't add their own.

### GAZ Implementation Recommendation

**Chain user hooks with internal hooks:**

```go
// gaz/app.go
type App struct {
    // ... existing fields ...
    preRunHooks  []func(context.Context) error
    postRunHooks []func(context.Context) error
}

// Register before WithCobra()
func (a *App) PreRun(fn func(context.Context) error) *App {
    a.preRunHooks = append(a.preRunHooks, fn)
    return a
}

func (a *App) PostRun(fn func(context.Context) error) *App {
    a.postRunHooks = append(a.postRunHooks, fn)
    return a
}

// In bootstrap() - run user preRunHooks AFTER Build but BEFORE Start
func (a *App) bootstrap(ctx context.Context, cmd *cobra.Command) error {
    // ... flag binding ...
    
    if err := a.Build(); err != nil {
        return err
    }
    
    // Run user pre-run hooks (after DI is ready)
    for _, fn := range a.preRunHooks {
        if err := fn(ctx); err != nil {
            return err
        }
    }
    
    if err := a.Start(ctx); err != nil {
        return err
    }
    
    return nil
}

// PostRun hooks run in PersistentPostRunE, BEFORE Stop()
```

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Complexity | **Low** |
| Breaking changes | None - additive |
| Dependencies on existing | Modifies `WithCobra()` behavior |
| Risk | Low |

### Table Stakes vs Differentiator

**Table Stakes** - Common pattern for setup/teardown outside lifecycle (e.g., database migrations, cache warming).

---

## Feature 5: Frame Introspection

### What It Does

Provides helper functions to get caller information (function name, file, line) for logging and debugging.

### Ecosystem Patterns

**Modern Go Idiom (2025+):**
```go
func getFrame(skipFrames int) runtime.Frame {
    targetFrameIndex := skipFrames + 2
    programCounters := make([]uintptr, targetFrameIndex+2)
    n := runtime.Callers(0, programCounters)
    
    if n > 0 {
        frames := runtime.CallersFrames(programCounters[:n])
        for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
            frame, m := frames.Next()
            more = m
            if frameIndex == targetFrameIndex {
                return frame
            }
        }
    }
    return runtime.Frame{Function: "unknown"}
}
```
*Source: WebSearch verified against Go runtime documentation (MEDIUM confidence)*

**Key Considerations:**
- Use `runtime.CallersFrames` not `runtime.FuncForPC` (handles inlining correctly)
- Package name extraction requires string parsing of `Frame.Function`
- `log/slog` with `AddSource: true` handles this automatically for logging

**runtimex Implementation:**
```go
func GetCurrentFunctionName() string { return getFrame(1).Function }
func GetCallerFunctionName() string { return getFrame(2).Function }
```
*Source: Internal `_tmp_trust/runtimex/frame.go` (HIGH confidence)*

### GAZ Implementation Recommendation

**Add as utility package, not core:**

```go
// gaz/debug/frame.go
package debug

import "runtime"

func CurrentFunction() string { return getFrame(1).Function }
func CallerFunction() string { return getFrame(2).Function }
func CallerLocation() (file string, line int) {
    frame := getFrame(2)
    return frame.File, frame.Line
}

// Package extraction
func CallerPackage() string {
    fn := getFrame(2).Function
    // Parse: "github.com/user/repo/pkg.FuncName" -> "github.com/user/repo/pkg"
    // ... string parsing logic ...
}
```

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Complexity | **Low** |
| Breaking changes | None - new package |
| Dependencies on existing | None |
| Risk | Very low |

### Table Stakes vs Differentiator

**Anti-Feature (borderline)** - Nice to have, but `log/slog` with `AddSource: true` handles the primary use case. Low priority.

---

## Feature 6: Service Builder

### What It Does

Pre-configured builder for common service patterns, reducing boilerplate.

### Ecosystem Patterns

**runtimex Service Builder:**
```go
func Builder(cmd *cobra.Command, schema []byte, cliName string) *runtimex.Builder {
    runtimex.RegisterBuildInfo(cmd, cliName)
    
    cliName = strings.ToUpper(strings.ReplaceAll(cliName, "-", "_"))
    if !strings.HasSuffix(cliName, "_") {
        cliName += "_"
    }
    
    return runtimex.NewBuilder(cmd).
        WithModule(fx.Supply(
            fx.Annotated{Name: "config.schema", Target: schema},
            fx.Annotated{Name: "config.env_prefix", Target: cliName},
        )).
        WithProvider(logx.NewProvider()).
        WithProvider(configx.NewProvider())
}
```
*Source: Internal `_tmp_trust/runtimex/service/builder.go` (HIGH confidence)*

### GAZ Implementation Recommendation

**Simple helper function, not new type:**

```go
// gaz/service/builder.go
package service

import (
    "github.com/petabytecl/gaz"
    "github.com/petabytecl/gaz/buildinfo"
    "github.com/petabytecl/gaz/health"
    "github.com/spf13/cobra"
)

type Config struct {
    Name        string
    EnvPrefix   string  // Defaults to uppercase NAME_
    HealthPort  int     // Defaults to 8081
    Options     []gaz.Option
}

func New(cmd *cobra.Command, cfg Config) *gaz.App {
    // Apply defaults
    if cfg.EnvPrefix == "" {
        cfg.EnvPrefix = strings.ToUpper(strings.ReplaceAll(cfg.Name, "-", "_")) + "_"
    }
    if cfg.HealthPort == 0 {
        cfg.HealthPort = 8081
    }
    
    // Register version
    buildinfo.RegisterVersion(cmd, cfg.Name)
    
    // Create app with standard options
    opts := []gaz.Option{
        gaz.WithShutdownTimeout(30 * time.Second),
    }
    opts = append(opts, cfg.Options...)
    
    app := gaz.New(opts...).
        WithConfig(nil, config.WithEnvPrefix(cfg.EnvPrefix)).
        WithCobra(cmd)
    
    // Register standard modules
    app.Module("health", health.Provider(cfg.HealthPort))
    
    return app
}
```

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Complexity | **Low-Medium** |
| Breaking changes | None - new package |
| Dependencies on existing | Depends on buildinfo, health |
| Risk | Low |

### Table Stakes vs Differentiator

**Differentiator** - Reduces "time to first HTTP" significantly. Opinionated defaults are valuable.

---

## Feature 7: Unified Provider

### What It Does

Bundles multiple related registrations (constructor, flags, lifecycle hooks, dependencies) into a single reusable unit.

### Ecosystem Patterns

**uber-go/fx Module Pattern:**
```go
var HTTPModule = fx.Module("http",
    fx.Provide(
        NewHTTPServer,
        NewRouter,
        NewMiddleware,
    ),
    fx.Invoke(RegisterRoutes),
    fx.Decorate(func(log *zap.Logger) *zap.Logger {
        return log.Named("http")
    }),
)
```
*Source: Context7 - uber-go/fx Module documentation (HIGH confidence)*

**runtimex Provider Pattern:**
```go
type Provider struct {
    FlagFunc     FlagRegistration      // Flag registration
    CmdHookFunc  CommandHookRegistration
    Dependencies fx.Option             // Sub-dependencies
    Constructor  any                   // Main constructor
    InvokeFunc   any                   // Startup function
}
```

### GAZ Current State

GAZ already has `App.Module(name string, opts ...ModuleOption)` for grouping:
- `For[T]().Provider()` for registration
- Duplicate detection
- But no bundling of flags + constructor + lifecycle

### GAZ Implementation Recommendation

**Extend Module pattern, don't replace For[T]():**

```go
// gaz/module.go (enhanced)
type ModuleBuilder struct {
    name     string
    app      *App
    provides []func(*Container) error
    flags    func(*pflag.FlagSet)
}

func Module(name string) *ModuleBuilder {
    return &ModuleBuilder{name: name}
}

func (m *ModuleBuilder) Flags(fn func(*pflag.FlagSet)) *ModuleBuilder {
    m.flags = fn
    return m
}

func (m *ModuleBuilder) Provide(fns ...func(*Container) error) *ModuleBuilder {
    m.provides = append(m.provides, fns...)
    return m
}

func (m *ModuleBuilder) Register(app *App) error {
    if m.flags != nil {
        // Register with app's flag set (via Cobra or standalone)
        // This happens BEFORE Build()
    }
    for _, fn := range m.provides {
        if err := fn(app.Container()); err != nil {
            return err
        }
    }
    return nil
}

// Usage:
var HTTPModule = gaz.Module("http").
    Flags(func(fs *pflag.FlagSet) {
        fs.Int("http-port", 8080, "HTTP server port")
    }).
    Provide(
        func(c *gaz.Container) error {
            return gaz.For[*HTTPServer](c).Provider(NewHTTPServer)
        },
    )

// In main:
app := gaz.New()
HTTPModule.Register(app)
```

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Complexity | **Medium** |
| Breaking changes | None - additive to existing Module |
| Dependencies on existing | Extends `app_module.go` |
| Risk | Medium - API design requires iteration |

### Table Stakes vs Differentiator

**Differentiator** - Most Go DI frameworks don't have this level of bundling. Makes library packages cleaner.

---

## Feature 8: Test Builder (gaztest)

### What It Does

Provides test utilities similar to `fxtest` for testing GAZ applications.

### Ecosystem Patterns

**uber-go/fx fxtest:**
```go
func TestMyService(t *testing.T) {
    var svc *MyService
    
    app := fxtest.New(t,
        fx.Provide(NewMyService),
        fx.Populate(&svc),
    )
    
    app.RequireStart()
    defer app.RequireStop()
    
    assert.True(t, svc.Running)
}
```
*Source: WebSearch + Context7 (HIGH confidence)*

**Key fxtest Features:**
- `fxtest.New(t, ...)` - Wraps testing.TB for automatic failure reporting
- `app.RequireStart()` - Start with `t.Fatal()` on error
- `app.RequireStop()` - Stop with `t.Fatal()` on error
- `t.Cleanup()` integration - Automatic cleanup
- `fx.Replace()` - Swap real implementations with mocks
- `fxtest.NewLifecycle(t)` - Isolated lifecycle for unit tests

### GAZ Implementation Recommendation

```go
// gaz/gaztest/gaztest.go
package gaztest

import (
    "context"
    "testing"
    "time"
    
    "github.com/petabytecl/gaz"
)

type App struct {
    *gaz.App
    t testing.TB
}

func New(t testing.TB, opts ...gaz.Option) *App {
    // Default shorter timeout for tests
    testOpts := []gaz.Option{
        gaz.WithShutdownTimeout(5 * time.Second),
        gaz.WithPerHookTimeout(2 * time.Second),
    }
    testOpts = append(testOpts, opts...)
    
    app := &App{
        App: gaz.New(testOpts...),
        t:   t,
    }
    
    // Automatic cleanup
    t.Cleanup(func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := app.Stop(ctx); err != nil {
            t.Logf("cleanup stop error: %v", err)
        }
    })
    
    return app
}

func (a *App) RequireStart() {
    a.t.Helper()
    if err := a.Build(); err != nil {
        a.t.Fatalf("build failed: %v", err)
    }
    if err := a.Start(context.Background()); err != nil {
        a.t.Fatalf("start failed: %v", err)
    }
}

func (a *App) RequireStop() {
    a.t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := a.Stop(ctx); err != nil {
        a.t.Fatalf("stop failed: %v", err)
    }
}

// Replace allows swapping a dependency for testing
func (a *App) Replace(instance any) *App {
    a.t.Helper()
    // Uses the same reflection logic as registerInstance
    if err := a.App.ReplaceInstance(instance); err != nil {
        a.t.Fatalf("replace failed: %v", err)
    }
    return a
}

// Lifecycle for unit testing individual services
type Lifecycle struct {
    t      testing.TB
    starts []gaz.HookFunc
    stops  []gaz.HookFunc
}

func NewLifecycle(t testing.TB) *Lifecycle {
    return &Lifecycle{t: t}
}

func (l *Lifecycle) Append(hook gaz.Hook) {
    if hook.OnStart != nil {
        l.starts = append(l.starts, hook.OnStart)
    }
    if hook.OnStop != nil {
        l.stops = append(l.stops, hook.OnStop)
    }
}

func (l *Lifecycle) MustStart(ctx context.Context) {
    l.t.Helper()
    for _, fn := range l.starts {
        if err := fn(ctx); err != nil {
            l.t.Fatalf("start failed: %v", err)
        }
    }
}

func (l *Lifecycle) MustStop(ctx context.Context) {
    l.t.Helper()
    for i := len(l.stops) - 1; i >= 0; i-- {
        if err := l.stops[i](ctx); err != nil {
            l.t.Fatalf("stop failed: %v", err)
        }
    }
}
```

### Complexity Assessment

| Aspect | Assessment |
|--------|------------|
| Complexity | **Medium** |
| Breaking changes | None - new package |
| Dependencies on existing | Needs `App.ReplaceInstance()` for Replace functionality |
| Risk | Low |

### Table Stakes vs Differentiator

**Table Stakes** - Testing DI applications is painful without proper utilities. `fxtest` is the gold standard.

---

## Feature Summary Table

| # | Feature | Category | Complexity | Depends On |
|---|---------|----------|------------|------------|
| 1 | Interface Auto-Detection | **Differentiator** | Medium | di.ServiceWrapper changes |
| 2 | Build Info Package | **Table Stakes** | Low | None |
| 3 | Command Args Injection | **Table Stakes** | Low | cobra.go:bootstrap |
| 4 | Pre/Post Run Hooks | **Table Stakes** | Low | WithCobra modification |
| 5 | Frame Introspection | Anti-Feature | Low | None |
| 6 | Service Builder | **Differentiator** | Low-Medium | buildinfo, health |
| 7 | Unified Provider | **Differentiator** | Medium | Module system |
| 8 | Test Builder (gaztest) | **Table Stakes** | Medium | App.ReplaceInstance |

---

## Anti-Features (Things to NOT Build)

### 1. Global Shutdowner

**Pattern from runtimex:**
```go
var (
    shutdowner   fx.Shutdowner
    shutdownLock sync.Once
)

func Shutdown() {
    shutdownLock.Do(func() {
        shutdowner.Shutdown()
    })
}
```

**Why NOT:** Global state breaks testability. GAZ's instance-based `app.Stop()` is correct.

### 2. Panic on Hook Conflict

**Pattern from runtimex:**
```go
func (b *Builder) RegisterCmdPreRun(preFunc ...) *Builder {
    if b.Cmd.PreRunE != nil {
        panic("cannot register pre run hook, already registered")
    }
    // ...
}
```

**Why NOT:** Panics in configuration are bad DX. Return errors instead.

### 3. Heavy Reflection for Interface Detection

**Pattern to avoid:**
```go
// Checking interfaces at EVERY resolve, not just registration
func (c *Container) Resolve(name string) any {
    instance := c.get(name)
    // BAD: Checking interfaces on every call
    if starter, ok := instance.(Starter); ok {
        starter.OnStart(ctx) // Wrong: lifecycle already handled
    }
    return instance
}
```

**Why NOT:** Interface detection should happen at registration, execution at lifecycle. Mixing causes unexpected behavior.

### 4. Automatic Config Schema Validation

**Pattern from runtimex:**
```go
fx.Supply(
    fx.Annotated{Name: "config.schema", Target: schema},
)
```

**Why NOT (for now):** Adds complexity. GAZ's `ConfigProvider` pattern with required field validation is simpler. Consider for v3+.

### 5. Spring-Style Auto-Wiring

**Pattern to avoid:**
```go
// Auto-detecting all struct fields and injecting them
type MyService struct {
    Logger   *slog.Logger   `inject:""`
    Database *sql.DB        `inject:""`
}
```

**Why NOT:** "Magic" injection hides dependencies, breaks refactoring. Explicit `For[T]().Provider(NewT)` is cleaner.

---

## Feature Dependencies Graph

```
                    +-----------------+
                    |  Build Info (2) |
                    +--------+--------+
                             |
                    +--------v--------+
                    | Service Builder |
                    |       (6)       |
                    +-----------------+
                             
+--------------+    +-----------------+    +------------------+
|  Args        |    | Pre/Post Hooks  |    | Interface Auto   |
| Injection(3) |    |      (4)        |    |  Detection (1)   |
+------+-------+    +--------+--------+    +--------+---------+
       |                     |                      |
       +---------------------+----------------------+
                             |
                    +--------v--------+
                    |  Test Builder   |
                    |      (8)        |
                    +-----------------+
                             
+------------------+
| Unified Provider |
|       (7)        |---- (Independent, but benefits from 1)
+------------------+

[Frame Introspection (5) - Standalone, low priority]
```

---

## Recommended Implementation Order

### Phase 1: Core Enhancements (Low Risk)
1. **Build Info Package** - No dependencies, immediate value
2. **Command Args Injection** - Simple addition to bootstrap
3. **Pre/Post Run Hooks** - Simple addition to WithCobra

### Phase 2: DI Enhancements (Medium Risk)
4. **Interface Auto-Detection** - Core DI change, test thoroughly
5. **Test Builder (gaztest)** - Enables testing of Phase 2 features

### Phase 3: Developer Experience (Polish)
6. **Service Builder** - Depends on buildinfo
7. **Unified Provider** - API design iteration needed

### Defer/Drop
- **Frame Introspection** - Low value, slog handles logging use case

---

## Sources

| Source | Type | Confidence |
|--------|------|------------|
| uber-go/fx Context7 | Official Documentation | HIGH |
| samber/do v2 Context7 | Official Documentation | HIGH |
| spf13/cobra Context7 | Official Documentation | HIGH |
| pkg.go.dev/runtime/debug | Official Go Documentation | HIGH |
| `_tmp_trust/runtimex/*` | Internal Implementation | HIGH |
| GAZ codebase (`lifecycle.go`, `cobra.go`, `app.go`) | Internal | HIGH |
| WebSearch (fxtest patterns) | Community Sources | MEDIUM |
| WebSearch (runtime.Callers) | Community Sources | MEDIUM |
