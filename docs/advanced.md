# Advanced

Modules, testing patterns, Cobra integration, and best practices.

## Modules

Group related providers into reusable modules:

```go
app := gaz.New()

app.Module("database",
    func(c *gaz.Container) error {
        return gaz.For[*sql.DB](c).Provider(NewDatabase)
    },
    func(c *gaz.Container) error {
        return gaz.For[*UserRepo](c).Provider(NewUserRepo)
    },
    func(c *gaz.Container) error {
        return gaz.For[*PostRepo](c).Provider(NewPostRepo)
    },
)

app.Module("http",
    func(c *gaz.Container) error {
        return gaz.For[*Router](c).Provider(NewRouter)
    },
    func(c *gaz.Container) error {
        return gaz.For[*Server](c).Provider(NewServer)
    },
)
```

**Module benefits:**

- Logical grouping of related services
- Clearer error messages (prefixed with module name)
- Duplicate module detection

**Module error handling:**

```go
// Duplicate module names cause error at Build()
app.Module("database", ...)
app.Module("database", ...)  // Error: ErrDuplicateModule
```

### Reusable Modules

Create module functions for reuse across applications:

```go
// pkg/database/module.go
package database

func Module(c *gaz.Container) error {
    if err := gaz.For[*sql.DB](c).Provider(NewDB); err != nil {
        return err
    }
    return gaz.For[*Migrator](c).Provider(NewMigrator)
}

// main.go
app := gaz.New()
app.Module("database", database.Module)
```

## Testing

gaz is designed for testability. Use the Container directly for isolated test scopes.

### Basic Test Pattern

```go
func TestUserService(t *testing.T) {
    c := gaz.NewContainer()
    
    // Register mock dependencies
    gaz.For[*Database](c).Instance(&MockDatabase{
        users: map[string]*User{
            "1": {ID: "1", Name: "Alice"},
        },
    })
    
    // Register service under test
    gaz.For[*UserService](c).Provider(NewUserService)
    
    if err := c.Build(); err != nil {
        t.Fatal(err)
    }
    
    // Resolve and test
    svc, err := gaz.Resolve[*UserService](c)
    if err != nil {
        t.Fatal(err)
    }
    
    user, err := svc.GetUser("1")
    if err != nil {
        t.Fatal(err)
    }
    if user.Name != "Alice" {
        t.Errorf("expected Alice, got %s", user.Name)
    }
}
```

### Replace for Integration Tests

Use `Replace()` to override production registrations:

```go
func TestWithRealDatabase(t *testing.T) {
    c := gaz.NewContainer()
    
    // Production registrations
    gaz.For[*Database](c).Provider(NewDatabase)
    gaz.For[*UserService](c).Provider(NewUserService)
    
    // Override with test database
    gaz.For[*Database](c).Replace().Instance(testDB)
    
    c.Build()
    // ... test with real service, test database
}
```

### Testing with Lifecycle

Test services with startup/shutdown hooks:

```go
func TestServerLifecycle(t *testing.T) {
    c := gaz.NewContainer()
    
    gaz.For[*Server](c).Provider(NewServer)
    c.Build()
    
    ctx := context.Background()
    
    // Resolve (instantiates singleton)
    server, _ := gaz.Resolve[*Server](c)
    
    // Manually call lifecycle hooks
    if err := server.OnStart(ctx); err != nil {
        t.Fatal(err)
    }
    
    // Test server...
    
    if err := server.OnStop(ctx); err != nil {
        t.Fatal(err)
    }
}
```

### Table-Driven Tests

```go
func TestUserValidation(t *testing.T) {
    tests := []struct {
        name    string
        user    User
        wantErr bool
    }{
        {"valid", User{Name: "Alice", Email: "alice@example.com"}, false},
        {"missing name", User{Email: "alice@example.com"}, true},
        {"invalid email", User{Name: "Alice", Email: "invalid"}, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c := gaz.NewContainer()
            gaz.For[*UserValidator](c).Provider(NewUserValidator)
            c.Build()
            
            v, _ := gaz.Resolve[*UserValidator](c)
            err := v.Validate(tt.user)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Cobra Integration

Integrate gaz with Cobra for CLI applications.

### Basic Setup

```go
package main

import (
    "github.com/petabytecl/gaz"
    "github.com/spf13/cobra"
)

func main() {
    rootCmd := &cobra.Command{
        Use:   "myapp",
        Short: "My application",
    }
    
    app := gaz.New()
    app.ProvideSingleton(NewDatabase)
    app.ProvideSingleton(NewServer)
    app.WithCobra(rootCmd)
    
    // Add subcommands
    rootCmd.AddCommand(serveCmd)
    rootCmd.AddCommand(migrateCmd)
    
    rootCmd.Execute()
}
```

### Accessing App in Commands

Use `gaz.FromContext()` to access the app in command handlers:

```go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the server",
    RunE: func(cmd *cobra.Command, args []string) error {
        app := gaz.FromContext(cmd.Context())
        
        server, err := gaz.Resolve[*Server](app.Container())
        if err != nil {
            return err
        }
        
        return server.ListenAndServe()
    },
}
```

### WithCobra Lifecycle

`WithCobra()` hooks into Cobra's lifecycle:

1. **PersistentPreRunE**: Calls `Build()` and `Start()`
2. **Command RunE**: Your command handler runs
3. **PersistentPostRunE**: Calls `Stop()` with graceful shutdown

```go
app.WithCobra(rootCmd)

// Equivalent to:
// rootCmd.PersistentPreRunE = func(...) {
//     app.Build()
//     app.Start(ctx)
// }
// rootCmd.PersistentPostRunE = func(...) {
//     app.Stop(ctx)
// }
```

### Flag Binding

Cobra flags are automatically bound to configuration:

```go
rootCmd.PersistentFlags().Int("port", 8080, "Server port")
rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")

app.WithConfig(cfg,
    gaz.WithSearchPaths("."),
)
app.WithCobra(rootCmd)

// Flags override config file values
// --port=9090 overrides config.yaml server.port
```

## Health Checks

Register health checkers for liveness and readiness probes.

### Basic Health Check

```go
import "github.com/petabytecl/gaz/health"

app := gaz.New()

// Register health module
app.Module("health", health.Module())

// Register checkers
app.ProvideSingleton(func(c *gaz.Container) (*Database, error) {
    db := NewDatabase()
    
    // Register health check
    hm := gaz.MustResolve[*health.Manager](c)
    hm.Register("database", db)  // db implements health.Checker
    
    return db, nil
})
```

### Checker Interface

```go
type Checker interface {
    Check(context.Context) error
}

// Example implementation
type Database struct {
    pool *sql.DB
}

func (d *Database) Check(ctx context.Context) error {
    return d.pool.PingContext(ctx)
}
```

### HTTP Health Endpoint

```go
app.ProvideSingleton(func(c *gaz.Container) (*http.ServeMux, error) {
    hm := gaz.MustResolve[*health.Manager](c)
    
    mux := http.NewServeMux()
    mux.HandleFunc("/health", hm.Handler())
    mux.HandleFunc("/ready", hm.ReadyHandler())
    mux.HandleFunc("/live", hm.LiveHandler())
    
    return mux, nil
})
```

## Graceful Shutdown

gaz handles shutdown automatically with configurable timeouts.

### Default Behavior

- Global timeout: 30 seconds
- Per-hook timeout: 10 seconds
- Signal handling: SIGTERM, SIGINT

### Configuration

```go
app := gaz.New(
    gaz.WithShutdownTimeout(60 * time.Second),  // Global timeout
    gaz.WithPerHookTimeout(15 * time.Second),   // Per-service timeout
)
```

### Per-Hook Timeout

Override timeout for specific services:

```go
gaz.For[*SlowService](c).
    OnStop(func(ctx context.Context, s *SlowService) error {
        return s.Cleanup(ctx)
    }, gaz.WithHookTimeout(30 * time.Second)).
    Provider(NewSlowService)
```

### Double-SIGINT Behavior

- First SIGINT: Begins graceful shutdown
- Second SIGINT: Forces immediate exit (no cleanup)

```
^C
Shutting down gracefully... (hint: Ctrl+C again to force)
^C
Received second interrupt, forcing exit
```

### Blame Logging

When hooks exceed timeout, gaz logs which service is slow:

```
ERROR shutdown: *database.Pool exceeded 10s timeout (elapsed: 10.001s)
```

## Best Practices

### Constructor Injection

Prefer resolving dependencies in providers (constructor injection):

```go
// Good: Dependencies explicit in provider
func NewUserService(c *gaz.Container) (*UserService, error) {
    db, err := gaz.Resolve[*Database](c)
    if err != nil {
        return nil, err
    }
    return &UserService{db: db}, nil
}

// Avoid: Service locator pattern
type UserService struct {
    container *gaz.Container  // Don't store container
}

func (s *UserService) GetUser(id string) (*User, error) {
    db, _ := gaz.Resolve[*Database](s.container)  // Late resolution
    return db.FindUser(id)
}
```

### Interface Dependencies

Depend on interfaces, not concrete types:

```go
// Good: Interface dependency
type UserService struct {
    store UserStore  // Interface
}

// Avoid: Concrete dependency
type UserService struct {
    db *PostgresDatabase  // Concrete type
}
```

### Single Responsibility

Each service should have one reason to change:

```go
// Good: Separate concerns
type UserRepository struct{ db *sql.DB }
type UserValidator struct{ rules []Rule }
type UserNotifier struct{ mailer *Mailer }

// Avoid: God service
type UserService struct {
    db     *sql.DB
    mailer *Mailer
    rules  []Rule
    cache  *Cache
    // ... does everything
}
```

### Avoid Circular Dependencies

gaz detects cycles at resolution time:

```go
// This causes ErrCycle
func NewA(c *gaz.Container) (*A, error) {
    b, _ := gaz.Resolve[*B](c)  // A depends on B
    return &A{b: b}, nil
}

func NewB(c *gaz.Container) (*B, error) {
    a, _ := gaz.Resolve[*A](c)  // B depends on A - cycle!
    return &B{a: a}, nil
}
```

**Solutions:**

1. Extract shared dependency
2. Use events/callbacks instead of direct dependency
3. Restructure to break the cycle

### Eager for Infrastructure

Use eager loading for infrastructure that must start early:

```go
app.ProvideEager(NewConnectionPool)  // Validate connection at startup
app.ProvideEager(NewMigrator)        // Run migrations before serving
```

### Transient for Stateless

Use transient scope for stateless, per-request services:

```go
app.ProvideTransient(NewRequestHandler)  // New instance per resolution
app.ProvideTransient(NewCommandHandler)  // Stateless, no shared state
```
