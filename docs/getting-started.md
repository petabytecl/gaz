# Getting Started

Build your first gaz application in 5 minutes.

## Prerequisites

- Go 1.25 or later
- A terminal

## Installation

```bash
go get github.com/petabytecl/gaz
```

## Create a Project

```bash
mkdir myapp && cd myapp
go mod init myapp
go get github.com/petabytecl/gaz
```

## Write main.go

Create `main.go` with a complete working application:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/petabytecl/gaz"
)

// Greeter is a simple service with lifecycle hooks.
type Greeter struct {
    Name string
}

// OnStart is called when the application starts.
func (g *Greeter) OnStart(ctx context.Context) error {
    fmt.Printf("Hello, %s! Service starting...\n", g.Name)
    return nil
}

// OnStop is called during graceful shutdown.
func (g *Greeter) OnStop(ctx context.Context) error {
    fmt.Printf("Goodbye, %s! Service stopping...\n", g.Name)
    return nil
}

func main() {
    // Create the application
    app := gaz.New()

    // Register a singleton service with a provider function
    err := gaz.For[*Greeter](app.Container()).Provider(func(c *gaz.Container) (*Greeter, error) {
        return &Greeter{Name: "World"}, nil
    })
    if err != nil {
        log.Fatalf("Registration failed: %v", err)
    }

    // Build validates registrations and instantiates eager services
    if err := app.Build(); err != nil {
        log.Fatalf("Build failed: %v", err)
    }

    // Run starts services, waits for SIGTERM/SIGINT, then shuts down gracefully
    if err := app.Run(context.Background()); err != nil {
        log.Fatalf("Run failed: %v", err)
    }
}
```

## Run It

```bash
go run main.go
```

Expected output:

```
Hello, World! Service starting...
```

Press `Ctrl+C` to trigger graceful shutdown:

```
Goodbye, World! Service stopping...
```

## What Happened

The gaz lifecycle works in three phases:

1. **Build Phase** (`app.Build()`)
   - Validates all service registrations
   - Instantiates eager services
   - Detects dependency cycles

2. **Start Phase** (`app.Run()` calls `Start()` internally)
   - Computes startup order from dependency graph
   - Calls `OnStart(ctx)` for services implementing `Starter`
   - Starts services layer by layer (dependencies first)

3. **Shutdown Phase** (triggered by SIGTERM, SIGINT, or context cancellation)
   - Calls `OnStop(ctx)` for services implementing `Stopper`
   - Shuts down in reverse dependency order
   - Enforces per-hook timeouts (default: 10s)

## Resolving Dependencies

Services can resolve their dependencies in provider functions:

```go
type Database struct{}

func NewDatabase(c *gaz.Container) (*Database, error) {
    return &Database{}, nil
}

type UserRepo struct {
    db *Database
}

func NewUserRepo(c *gaz.Container) (*UserRepo, error) {
    // Resolve the Database dependency
    db, err := gaz.Resolve[*Database](c)
    if err != nil {
        return nil, err
    }
    return &UserRepo{db: db}, nil
}

func main() {
    app := gaz.New()

    // Register services using the type-safe For[T]() API
    if err := gaz.For[*Database](app.Container()).Provider(NewDatabase); err != nil {
        log.Fatal(err)
    }
    if err := gaz.For[*UserRepo](app.Container()).Provider(NewUserRepo); err != nil {
        log.Fatal(err)
    }

    if err := app.Build(); err != nil {
        log.Fatal(err)
    }

    // UserRepo is available with its Database dependency resolved
    repo, _ := gaz.Resolve[*UserRepo](app.Container())
    _ = repo
}
```

## Next Steps

- [Concepts](concepts.md) - Understand DI fundamentals, scopes, and lifecycle
- [Configuration](configuration.md) - Load config from files and environment
- [Validation](validation.md) - Validate configuration with struct tags
- [Advanced](advanced.md) - Modules, testing, and Cobra integration
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
