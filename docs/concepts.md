# Concepts

Core concepts for understanding gaz dependency injection.

## Dependency Injection

Dependency injection (DI) inverts the control of object creation. Instead of services creating their dependencies, dependencies are provided from outside.

**Without DI:**

```go
type UserService struct {
    db *Database
}

func NewUserService() *UserService {
    return &UserService{
        db: NewDatabase(), // Hardcoded dependency
    }
}
```

**With DI:**

```go
func NewUserService(c *gaz.Container) (*UserService, error) {
    db, err := gaz.Resolve[*Database](c)
    if err != nil {
        return nil, err
    }
    return &UserService{db: db}, nil
}
```

Benefits:

- **Testability** - Inject mocks instead of real dependencies
- **Flexibility** - Swap implementations without changing consumers
- **Clarity** - Dependencies are explicit in the provider signature

## The Container

The `Container` is the central registry holding all service registrations. It:

- Stores provider functions keyed by type
- Tracks instantiated singletons
- Maintains the dependency graph for lifecycle ordering
- Detects circular dependencies at resolution time

Create a container directly for library or testing use:

```go
c := gaz.NewContainer()
gaz.For[*MyService](c).Provider(NewMyService)
c.Build()

svc, _ := gaz.Resolve[*MyService](c)
```

For applications, prefer the `App` wrapper which adds lifecycle and signal handling.

## Providers

A provider is a factory function that creates a service instance. Providers receive the container for resolving dependencies.

**Provider with error:**

```go
func NewDatabase(c *gaz.Container) (*Database, error) {
    cfg, err := gaz.Resolve[*Config](c)
    if err != nil {
        return nil, err
    }
    return &Database{dsn: cfg.DatabaseURL}, nil
}
```

**Provider without error:**

```go
gaz.For[*Config](c).ProviderFunc(func(c *gaz.Container) *Config {
    return &Config{Debug: true}
})
```

**Instance registration (no provider):**

```go
cfg := &Config{Debug: true}
gaz.For[*Config](c).Instance(cfg)
```

## Service Scopes

gaz supports three service scopes that control instantiation behavior.

### Singleton (Default)

One instance per container lifetime. Created on first resolution, reused thereafter.

```go
gaz.For[*Database](c).Provider(NewDatabase)
```

### Transient

New instance on every resolution. Use for request-scoped or stateless services.

```go
gaz.For[*RequestHandler](c).Transient().Provider(NewRequestHandler)
```

### Eager

Singleton instantiated at `Build()` time instead of first resolution. Use for services that must start immediately.

```go
gaz.For[*ConnectionPool](c).Eager().Provider(NewConnectionPool)
```

**When to use each scope:**

| Scope | Instance Count | Created At | Use Case |
|-------|----------------|------------|----------|
| Singleton | 1 | First resolution | Database pools, HTTP clients, caches |
| Transient | Many | Each resolution | Request handlers, short-lived workers |
| Eager | 1 | Build time | Services requiring immediate startup |

## Lifecycle

Services can implement lifecycle interfaces for startup/shutdown hooks.

### Starter Interface

```go
type Starter interface {
    OnStart(context.Context) error
}
```

Called during `app.Run()` or `app.Start()` after dependencies are resolved. Services start in dependency order (dependencies first).

### Stopper Interface

```go
type Stopper interface {
    OnStop(context.Context) error
}
```

Called during graceful shutdown. Services stop in reverse dependency order (dependents first).

**Example with both:**

```go
type Server struct {
    listener net.Listener
}

func (s *Server) OnStart(ctx context.Context) error {
    ln, err := net.Listen("tcp", ":8080")
    if err != nil {
        return err
    }
    s.listener = ln
    go s.serve()
    return nil
}

func (s *Server) OnStop(ctx context.Context) error {
    return s.listener.Close()
}
```

### Startup Order

gaz computes startup order using topological sort on the dependency graph:

1. Services with no dependencies start first
2. Services wait for their dependencies to start
3. Independent services in the same layer start in parallel

### Shutdown Order

Shutdown reverses the startup order:

1. Services with no dependents stop first
2. Dependencies stop after all their dependents
3. Per-hook timeout enforced (default: 10s)

## Resolution

`gaz.Resolve[T]()` retrieves a service instance from the container.

```go
db, err := gaz.Resolve[*Database](c)
if err != nil {
    // ErrNotFound - service not registered
    // ErrCycle - circular dependency detected
    return nil, err
}
```

**Automatic dependency resolution:**

When a provider calls `Resolve[T]()`, gaz:

1. Checks if instance exists (for singletons)
2. Checks for cycles in the resolution chain
3. Calls the provider if needed
4. Records the dependency in the graph
5. Returns the instance

**Convenience functions:**

```go
// Panics on error (use only when you know the service exists)
db := gaz.MustResolve[*Database](c)
```

## App vs Container

gaz provides two APIs for different use cases.

### App (Applications)

Use `App` for applications that run, wait for signals, and shut down:

```go
app := gaz.New(gaz.WithShutdownTimeout(30 * time.Second))

// Register services using the type-safe For[T]() API
gaz.For[*Database](app.Container()).Provider(NewDatabase)

app.Build()
app.Run(ctx) // Blocks until shutdown signal
```

Features:

- Signal handling (SIGTERM, SIGINT)
- Graceful shutdown with timeout
- Configuration loading
- Cobra integration
- Structured logging

### Container (Libraries/Testing)

Use `Container` directly for libraries or test scenarios:

```go
c := gaz.NewContainer()
gaz.For[*Service](c).Provider(NewService)
c.Build()

svc, _ := gaz.Resolve[*Service](c)
// Use svc...
```

Features:

- Lightweight
- No signal handling
- No logging
- Manual lifecycle control

**In tests:**

```go
func TestUserService(t *testing.T) {
    c := gaz.NewContainer()
    
    // Register mock
    gaz.For[*Database](c).Instance(&MockDatabase{})
    
    // Register service under test
    gaz.For[*UserService](c).Provider(NewUserService)
    
    c.Build()
    
    svc, _ := gaz.Resolve[*UserService](c)
    // Test svc with mock database...
}
```

## Named Registrations

Register multiple implementations of the same type:

```go
gaz.For[*sql.DB](c).Named("primary").Provider(NewPrimaryDB)
gaz.For[*sql.DB](c).Named("replica").Provider(NewReplicaDB)
```

Resolve by name:

```go
primary, _ := gaz.ResolveNamed[*sql.DB](c, "primary")
replica, _ := gaz.ResolveNamed[*sql.DB](c, "replica")
```

## Replace (Testing)

Override registrations in tests:

```go
// Production registration
gaz.For[*EmailSender](c).Provider(NewSMTPSender)

// Override in test
gaz.For[*EmailSender](c).Replace().Instance(&MockEmailSender{})
```

Without `Replace()`, duplicate registrations return `ErrDuplicate`.
