# Troubleshooting

Common issues and solutions when using gaz.

## Container Errors

### "type not registered" on Resolve

**Problem:** Calling `gaz.Resolve[T]()` before registering T.

**Solution:**

```go
// Wrong: resolve before register
svc, err := gaz.Resolve[*MyService](c) // error: ErrNotFound

// Correct: register then resolve
gaz.For[*MyService](c).Provider(NewMyService)
c.Build()
svc, _ := gaz.Resolve[*MyService](c)
```

### "container not built" on Resolve

**Problem:** Resolving before calling `Build()`.

**Solution:**

```go
gaz.For[*MyService](c).Provider(...)
c.Build() // Don't forget!
svc, _ := gaz.Resolve[*MyService](c)
```

### "duplicate registration" error

**Problem:** Registering the same type twice without `Replace()`.

**Solution:**

```go
// Wrong: duplicate registration
gaz.For[*Database](c).Provider(NewDatabase)
gaz.For[*Database](c).Provider(NewOtherDatabase) // error: ErrDuplicate

// Correct: use Replace() to override
gaz.For[*Database](c).Provider(NewDatabase)
gaz.For[*Database](c).Replace().Provider(NewOtherDatabase) // OK
```

### "circular dependency detected"

**Problem:** Service A depends on B, and B depends on A.

**Solution:** Refactor to break the cycle:

```go
// Problem: A -> B -> A (cycle)
func NewA(c *gaz.Container) (*A, error) {
    b, _ := gaz.Resolve[*B](c)
    return &A{b: b}, nil
}

func NewB(c *gaz.Container) (*B, error) {
    a, _ := gaz.Resolve[*A](c) // Cycle!
    return &B{a: a}, nil
}

// Solution 1: Extract shared dependency
func NewA(c *gaz.Container) (*A, error) {
    shared, _ := gaz.Resolve[*Shared](c)
    return &A{shared: shared}, nil
}

func NewB(c *gaz.Container) (*B, error) {
    shared, _ := gaz.Resolve[*Shared](c)
    return &B{shared: shared}, nil
}

// Solution 2: Use events/callbacks instead of direct dependency
```

## Lifecycle Errors

### OnStart/OnStop not called

**Problem:** Service implements Starter/Stopper but methods not called.

**Causes:**

1. Service not registered as Eager
2. Service never resolved (lazy singletons aren't instantiated until first use)
3. Using `Run()` without `Build()` first

**Solution:**

```go
// For services that must start immediately:
gaz.For[*Server](c).Eager().Provider(NewServer)

// Or explicitly resolve to trigger instantiation:
c.Build()
_, _ = gaz.Resolve[*Server](c) // Now it's instantiated
```

### Shutdown timeout

**Problem:** App takes too long to shut down.

**Solution:** Check OnStop implementations for blocking operations:

```go
func (s *Server) OnStop(ctx context.Context) error {
    // Use context deadline - don't block forever
    select {
    case <-s.done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### Services start in wrong order

**Problem:** Service A starts before its dependency B.

**Cause:** Dependencies not properly declared in providers.

**Solution:** Ensure providers resolve dependencies:

```go
// Wrong: No dependency tracking
func NewServer(c *gaz.Container) (*Server, error) {
    // Database never resolved, so gaz doesn't know about the dependency
    return &Server{}, nil
}

// Correct: Resolve dependency in provider
func NewServer(c *gaz.Container) (*Server, error) {
    db, err := gaz.Resolve[*Database](c) // Now gaz knows Server depends on Database
    if err != nil {
        return nil, err
    }
    return &Server{db: db}, nil
}
```

## Configuration Errors

### Config key not found

**Problem:** `ErrKeyNotFound` when accessing config.

**Causes:**

1. YAML key doesn't match struct tag
2. Config file not loaded or not found

**Solution:**

```go
// Check struct tags match config file keys
type Config struct {
    Port int `gaz:"port"` // matches "port" in YAML
}

// Verify config file is being loaded
fmt.Println(pv.GetString("server.port")) // debug

// Check file exists in search paths
mgr := config.New(
    config.WithName("config"),
    config.WithSearchPaths(".", "./config"),
)
```

### Validation fails on startup

**Problem:** App panics with validation error on Build().

**Solution:** Ensure config values meet validation rules:

```go
type ServerConfig struct {
    Port int `gaz:"port" validate:"required,min=1,max=65535"`
}

// Provide valid values in config.yaml:
// server:
//   port: 8080
```

### Environment variables not working

**Problem:** Env vars don't override config file values.

**Solution:** Check env var naming convention:

```go
// Config keys use dot notation: server.port
// Env vars use underscore: SERVER_PORT

// Check env prefix if set:
mgr := config.New(config.WithEnvPrefix("MYAPP"))
// Now use: MYAPP_SERVER_PORT
```

## Module Errors

### Import cycle with gaz package

**Problem:** `import cycle not allowed` when creating modules.

**Solution:** Use `di` package instead of `gaz` in subsystem modules:

```go
// Wrong: Import cycle
import "github.com/petabytecl/gaz"

func NewModule() gaz.Module { // Cycle!
    ...
}

// Correct: Use di package for module definitions
import "github.com/petabytecl/gaz/di"

func NewModule() di.Module {
    return di.NewModuleFunc(func(c *di.Container) error {
        // Register with di, not gaz
        return nil
    })
}
```

### Duplicate module name

**Problem:** `ErrDuplicateModule` when registering modules.

**Solution:** Use unique module names:

```go
// Wrong: Same name twice
app.Module("database", ...)
app.Module("database", ...) // ErrDuplicateModule

// Correct: Unique names
app.Module("database", ...)
app.Module("cache", ...)
```

## Worker Errors

### Worker not starting

**Problem:** Registered worker never starts.

**Cause:** Worker not properly registered or module not loaded.

**Solution:**

```go
// Register worker module
app.UseDI(worker.NewModule())

// Register your worker
gaz.For[worker.Worker](app.Container()).
    Named("my-worker").
    Provider(NewMyWorker)
```

### Worker panics crash the app

**Problem:** Panic in worker takes down entire application.

**Solution:** Workers have built-in panic recovery. Check logs for panic details:

```
ERROR worker "my-worker" panicked: runtime error: index out of range
```

The worker will restart according to its backoff policy.

## Testing Errors

### Test App already built

**Problem:** `ErrAlreadyBuilt` error in tests.

**Solution:** Create fresh app per test:

```go
func TestMyService(t *testing.T) {
    ta := gaztest.New(t) // Fresh app each test
    // ...
}
```

### Tests hang waiting for lifecycle

**Problem:** Tests never complete when using app with lifecycle.

**Solution:** Use context with timeout or cancel:

```go
func TestWithLifecycle(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    ta := gaztest.New(t)
    ta.Build()
    
    // Run in goroutine, cancel when done
    go func() {
        time.Sleep(100 * time.Millisecond)
        cancel()
    }()
    
    ta.Run(ctx) // Will stop when context cancels
}
```

### Mock not being used

**Problem:** Real implementation used instead of mock.

**Solution:** Register mock before building, use Replace() if needed:

```go
func TestWithMock(t *testing.T) {
    ta := gaztest.New(t)
    
    // Register mock BEFORE production registrations
    gaz.For[*Database](ta.App().Container()).Instance(&MockDatabase{})
    
    // Or use Replace() if production is already registered
    gaz.For[*Database](ta.App().Container()).Replace().Instance(&MockDatabase{})
    
    ta.Build()
}
```

## Health Check Errors

### Health checks not responding

**Problem:** `/health` endpoint returns no checkers.

**Cause:** Checkers not registered with health.Manager.

**Solution:**

```go
// Register your service as a health checker
func NewDatabase(c *gaz.Container) (*Database, error) {
    db := &Database{...}
    
    hm, _ := gaz.Resolve[*health.Manager](c)
    hm.Register("database", db) // db must implement health.Checker
    
    return db, nil
}

// Implement the Checker interface
func (d *Database) Check(ctx context.Context) error {
    return d.pool.PingContext(ctx)
}
```

## See Also

- [Getting Started](getting-started.md) - First app walkthrough
- [Concepts](concepts.md) - DI fundamentals
- [Configuration](configuration.md) - Config loading
- [Advanced](advanced.md) - Modules and testing
