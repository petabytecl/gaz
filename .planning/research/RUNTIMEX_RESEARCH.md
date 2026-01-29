# RuntimeX Research: Feature Extraction for GAZ Apps

**Date:** 2026-01-29
**Source:** `_tmp_trust/runtimex/`
**Purpose:** Extract patterns and features from RuntimeX (Uber's fx-based runtime) to improve GAZ apps

---

## Executive Summary

RuntimeX is a production-grade Go application runtime built on top of **Uber's fx** dependency injection framework and **Cobra** CLI. It implements a **Builder pattern** for composing applications with modular dependencies, lifecycle hooks, and provider patterns. This research identifies features that can enhance GAZ's application framework.

---

## Source Code Analysis

### Core Components

#### 1. Builder (`builder.go`)

The central orchestrator for application construction:

```go
type Builder struct {
    Cmd     *cobra.Command   // Associated CLI command
    Provide []any            // Constructor functions
    Invoke  []fx.Option      // Functions to invoke at startup
    Options []fx.Option      // Additional fx options (modules)
}
```

**Key Methods:**
| Method | Purpose |
|--------|---------|
| `NewBuilder(cmd)` | Creates builder with cobra command |
| `WithConstructor(...)` | Adds dependency constructors |
| `WithProvider(p)` | Registers a Provider (flags + constructor + invoke) |
| `WithModule(opt)` | Adds fx.Option modules |
| `WithInvoke(funcs)` | Adds startup invoke functions |
| `RegisterFlags(fn)` | Registers command flags |
| `RegisterCmdPreRun(fn)` | Pre-execution hook |
| `RegisterCmdPostRun(fn)` | Post-execution hook |
| `Build()` | Creates `*fx.App` |
| `BuildTest(tb)` | Creates `*fxtest.App` for testing |

**Key Pattern - Command Args Injection:**
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

This pattern makes CLI arguments available as a named dependency throughout the app.

---

#### 2. Provider (`provider.go`)

Encapsulates a module's dependencies:

```go
type FlagRegistration func(set *pflag.FlagSet)
type CommandHookRegistration = cobra.PositionalArgs

type Provider struct {
    FlagFunc     FlagRegistration      // Flag registration
    CmdHookFunc  CommandHookRegistration
    Dependencies fx.Option             // Sub-dependencies
    Constructor  any                   // Main constructor
    InvokeFunc   any                   // Startup function
}
```

**Design Insight:** Providers bundle everything a module needs:
- Flag definitions
- Constructor functions
- Invoke functions (side effects at startup)
- Nested dependencies

This is a **self-contained module pattern** where each provider knows:
1. What flags it needs
2. What it constructs
3. What it needs to do at startup

---

#### 3. Shutdowner (`shutdowner.go`)

Graceful shutdown mechanism:

```go
var (
    shutdowner   fx.Shutdowner
    shutdownLock sync.Once
)

func Shutdown() {
    shutdownLock.Do(func() {
        if err := shutdowner.Shutdown(); err != nil {
            _, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
            os.Exit(1)
        }
    })
}
```

**Pattern:** Global shutdown function with `sync.Once` for idempotency.

---

#### 4. Build Info (`buildinfo.go`)

Build metadata injection:

```go
var (
    RepoURL    = defaultBuildInfo
    Commit     = defaultBuildInfo
    BranchName = defaultBuildInfo
    Revision   = defaultBuildInfo
    BuildUser  = defaultBuildInfo
)

type info struct {
    serviceName string
    repoURL     string
    commit      string
    branchName  string
    revision    string
    buildUser   string
}

func RegisterBuildInfo(cmd *cobra.Command, name string) {
    BuildInfo(name)
    cmd.SetVersionTemplate(CobraTemplate)
    cmd.Version = CobraVersion()
}
```

**Integration with Bazel stamping:** Variables are set at compile time via ldflags.

---

#### 5. Frame Utilities (`frame.go`)

Runtime introspection for debugging:

```go
func GetCurrentFunctionName() string {
    return getFrame(1).Function
}

func GetCallerFunctionName() string {
    return getFrame(2).Function
}

func getFrame(skipFrames int) runtime.Frame {
    // Uses runtime.Callers to get stack frame information
}
```

Useful for logging and debugging contexts.

---

#### 6. Service Builder (`service/builder.go`)

Higher-level builder for services:

```go
func Builder(cmd *cobra.Command, schema []byte, cliName string) *runtimex.Builder {
    // Init build info
    runtimex.RegisterBuildInfo(cmd, cliName)

    // Convert cliName to uppercase env prefix
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

**Pattern:** Service builder pre-configures common providers (logging, config) with:
- JSON schema for configuration validation
- Environment prefix for config loading

---

## Comparison: RuntimeX vs GAZ vs GAZx

| Feature | RuntimeX (fx) | GAZ (current) | GAZx (tmp) |
|---------|---------------|---------------|------------|
| DI Framework | Uber fx | Custom (di/) | Custom (dibx) |
| CLI Integration | Cobra direct | WithCobra() | BindCommand() |
| Provider Pattern | Provider struct | For[T]().Provider() | ModuleProvider |
| Flag Registration | FlagFunc | Via ConfigManager | FlagRegistrar |
| Lifecycle | fx lifecycle | Starter/Stopper interfaces | Lifecycle interface |
| Shutdown | fx.Shutdowner | App.Stop() | Shutdowner interface |
| Build Info | Bazel stamps | Not present | internal.GetInfo() |
| Workers | Not present | Not present | WorkerGroup |
| Health Checks | Not present | health/ package | HealthManager |
| Events | Not present | eventbus/ | EventBus |
| Testing | fxtest.App | Test helpers | RunTest() |

---

## Features to Port to GAZ

### Priority 1: High Impact, Low Effort

#### 1.1 Build Info Integration

**What:** Inject build metadata (version, commit, branch) into apps.

**Why:** Critical for debugging production issues, version tracking.

**Implementation:**
```go
// gaz/buildinfo/info.go
type BuildInfo struct {
    ServiceName    string
    Version        string
    Commit         string
    Branch         string
    BuildUser      string
    BuildTimestamp time.Time
}

var (
    Version   = "dev"
    Commit    = "unknown"
    Branch    = "unknown"
    BuildUser = "unknown"
    BuildTime = ""
)

func Get() BuildInfo {
    return BuildInfo{
        Version:   Version,
        Commit:    Commit,
        Branch:    Branch,
        BuildUser: BuildUser,
    }
}

// Integration with Cobra
func RegisterVersion(cmd *cobra.Command, serviceName string) {
    info := Get()
    info.ServiceName = serviceName
    cmd.Version = info.String()
}
```

**Integration with GAZ:**
```go
app := gaz.New().
    WithBuildInfo(buildinfo.Get()).
    WithCobra(rootCmd)
```

#### 1.2 Command Arguments Injection

**What:** Make CLI arguments available as dependencies.

**Why:** Commands often need to access positional args in services.

**Implementation:**
```go
// In gaz/cobra.go
const ArgsKey = "gaz.cmd.args"

func (a *App) bootstrap(ctx context.Context, cmd *cobra.Command) error {
    // Register command and args as dependencies
    gaz.Supply(a.container, cmd)
    gaz.SupplyNamed(a.container, ArgsKey, os.Args[1:])
    // ... existing bootstrap logic
}

// Helper functions
func GetArgs(c *Container) []string {
    return ResolveNamed[[]string](c, ArgsKey)
}
```

#### 1.3 Pre/Post Run Hooks

**What:** Allow registering hooks before and after command execution.

**Why:** Common patterns: setup connections before, cleanup after.

**Implementation:**
```go
type App struct {
    preRunHooks  []func(context.Context) error
    postRunHooks []func(context.Context) error
}

func (a *App) PreRun(fn func(context.Context) error) *App {
    a.preRunHooks = append(a.preRunHooks, fn)
    return a
}

func (a *App) PostRun(fn func(context.Context) error) *App {
    a.postRunHooks = append(a.postRunHooks, fn)
    return a
}
```

---

### Priority 2: Medium Impact

#### 2.1 Frame Introspection Utilities

**What:** Helper functions to get caller information.

**Why:** Useful for logging, error reporting, debugging.

**Implementation:**
```go
// gaz/debug/frame.go
package debug

import "runtime"

func CurrentFunction() string {
    return getFrame(1).Function
}

func CallerFunction() string {
    return getFrame(2).Function
}

func CallerFile() (string, int) {
    frame := getFrame(2)
    return frame.File, frame.Line
}
```

#### 2.2 Service Builder Pattern

**What:** Pre-configured builder for common service patterns.

**Why:** Reduce boilerplate for standard services.

**Implementation:**
```go
// gaz/service/builder.go
func NewService(name string, cmd *cobra.Command) *ServiceBuilder {
    app := gaz.New()
    
    // Register standard providers
    app.Module("core",
        logger.Provider(),    // Structured logging
        health.Provider(),    // Health checks
        config.Provider(),    // Configuration
    )
    
    // Setup version info
    buildinfo.RegisterVersion(cmd, name)
    
    return &ServiceBuilder{
        app: app,
        cmd: cmd,
    }
}

// Usage:
svc := service.NewService("my-api", rootCmd)
svc.WithProvider(database.Provider())
svc.Run()
```

---

### Priority 3: Architectural Improvements

#### 3.1 Unified Provider Interface

**What:** Combine GAZ's current patterns into a cohesive Provider struct.

**Current GAZ Pattern:**
```go
gaz.For[*Database](app).Provider(NewDatabase)
```

**Proposed Enhanced Pattern:**
```go
type Provider struct {
    Name         string
    Flags        func(*pflag.FlagSet)
    Constructor  any
    Dependencies []Provider
    OnStart      func(context.Context) error
    OnStop       func(context.Context) error
}

func (p *Provider) Register(app *gaz.App) {
    if p.Flags != nil {
        app.RegisterFlags(p.Flags)
    }
    for _, dep := range p.Dependencies {
        dep.Register(app)
    }
    gaz.For[any](app).Provider(p.Constructor)
    // Register lifecycle hooks if present
}
```

#### 3.2 Enhanced Test Builder

**What:** First-class testing support like fxtest.

**Implementation:**
```go
// gaz/testing/builder.go
type TestApp struct {
    *gaz.App
    t       testing.TB
    cleanup func()
}

func NewTest(t testing.TB) *TestApp {
    app := gaz.New(
        gaz.WithShutdownTimeout(5*time.Second),
    )
    
    ta := &TestApp{
        App: app,
        t:   t,
    }
    
    t.Cleanup(func() {
        if err := ta.Stop(context.Background()); err != nil {
            t.Errorf("cleanup failed: %v", err)
        }
    })
    
    return ta
}

func (ta *TestApp) RequireStart() {
    if err := ta.Build(); err != nil {
        ta.t.Fatalf("build failed: %v", err)
    }
    if err := ta.Start(context.Background()); err != nil {
        ta.t.Fatalf("start failed: %v", err)
    }
}
```

---

## Feature Comparison Matrix

| Feature | RuntimeX | Recommended for GAZ | Priority | Effort |
|---------|----------|---------------------|----------|--------|
| Build Info | Yes | **YES** | HIGH | Low |
| Args Injection | Yes | **YES** | HIGH | Low |
| Pre/Post Hooks | Yes | **YES** | HIGH | Low |
| Frame Utils | Yes | Optional | LOW | Very Low |
| Service Builder | Yes | **YES** | MEDIUM | Medium |
| Provider Bundle | Yes | Consider | MEDIUM | Medium |
| Test Support | fxtest | **Improve** | MEDIUM | Medium |
| Bazel Integration | Yes | If using Bazel | LOW | Medium |

---

## Anti-Patterns to Avoid

### From RuntimeX:
1. **Global Shutdowner** - RuntimeX uses a global var for shutdown. GAZ should keep this instance-based.
2. **Panic on Hook Conflict** - RuntimeX panics if pre/post hooks are registered twice. Use error returns instead.

### From fx (underlying):
1. **Implicit Ordering** - fx starts things based on dependency order which can be surprising. GAZ should keep explicit ordering.
2. **Reflection Heavy** - fx uses heavy reflection. GAZ's generics-based approach is preferred.

---

## Implementation Roadmap

### Phase 1: Quick Wins (1-2 days)
- [ ] Add `buildinfo` package with ldflags integration
- [ ] Inject command args as named dependency
- [ ] Add Pre/Post run hooks to `App`

### Phase 2: Developer Experience (3-5 days)
- [ ] Create `gaz/service` package with ServiceBuilder
- [ ] Enhance test utilities in `gaz/testing`
- [ ] Add frame introspection utilities

### Phase 3: Provider Consolidation (5-7 days)
- [ ] Unified Provider type (flags + constructor + lifecycle)
- [ ] Module registry to prevent duplicate registration
- [ ] Enhanced configuration with schema validation

---

## GAZx Learnings (from `_tmp/gazx/`)

The `_tmp/gazx/` codebase represents an evolution that **already incorporates** many RuntimeX concepts:

| GAZx Feature | Status | Should Port to Main GAZ |
|--------------|--------|------------------------|
| ModuleProvider | Implemented | **YES** |
| WorkerGroup | Implemented | **YES** |
| WorkerGroupManager | Implemented | **YES** |
| EventBus | Implemented | Consider (eventbus/ exists) |
| HealthManager | Implemented | Merge with health/ |
| LifecycleManager | Implemented | **YES** |
| Shutdowner | Implemented | **YES** |
| automaxprocs | Implemented | **YES** |
| memlimit | Implemented | **YES** |
| ModuleRegistry | Implemented | **YES** |

**Recommendation:** Merge GAZx patterns into main GAZ codebase, prioritizing:
1. WorkerGroup/WorkerGroupManager (background tasks)
2. LifecycleManager (ordered start/stop)
3. ModuleProvider (unified provider pattern)
4. Auto GOMAXPROCS and memory limits

---

## Conclusion

RuntimeX provides battle-tested patterns for production Go applications. The key takeaways for GAZ:

1. **Build Info** - Essential for production debugging
2. **Args Injection** - Clean way to access CLI args in services
3. **Provider Pattern** - Bundle flags + constructors + lifecycle
4. **Service Builder** - Reduce boilerplate for common service types
5. **Enhanced Testing** - First-class test builder support

The `_tmp/gazx/` codebase has already implemented many of these patterns and should be the primary source for consolidation into the main GAZ framework.

---

## References

- RuntimeX Source: `_tmp_trust/runtimex/`
- GAZx Source: `_tmp/gazx/`
- Uber fx Documentation: https://uber-go.github.io/fx/
- Cobra Documentation: https://cobra.dev/
