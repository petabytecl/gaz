# gaz

[![Go Reference](https://pkg.go.dev/badge/github.com/petabytecl/gaz.svg)](https://pkg.go.dev/github.com/petabytecl/gaz)
![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)

Simple, type-safe dependency injection with lifecycle management for Go applications. No code generation, no reflection magic.

## Installation

```bash
go get github.com/petabytecl/gaz
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/petabytecl/gaz"
)

type Server struct {
	addr string
}

func (s *Server) OnStart(ctx context.Context) error {
	fmt.Println("server starting on", s.addr)
	return nil
}

func (s *Server) OnStop(ctx context.Context) error {
	fmt.Println("server stopped")
	return nil
}

func main() {
	app := gaz.New()
	app.ProvideSingleton(func(c *gaz.Container) (*Server, error) {
		return &Server{addr: ":8080"}, nil
	})

	if err := app.Build(); err != nil {
		log.Fatal(err)
	}

	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

## Features

- **Type-safe container** - Compile-time type checking via generics
- **Singleton and transient scopes** - One instance or new instance per resolution
- **Lifecycle hooks** - `OnStart`/`OnStop` interfaces for startup/shutdown logic
- **Graceful shutdown** - Configurable timeout with per-hook limits
- **Configuration loading** - YAML/JSON/TOML files, environment variables, profiles
- **Struct validation** - `validate` tags with go-playground/validator
- **Health checks** - Readiness and liveness probes via `health` subpackage
- **Cobra CLI integration** - Build CLI apps with dependency injection
- **Module organization** - Group related providers into reusable modules

## Core Concepts

### App vs Container

`App` is the high-level API for building applications. It manages the container, configuration, lifecycle, and signal handling:

```go
app := gaz.New()
app.ProvideSingleton(NewDatabase)
app.Build()
app.Run(ctx)
```

`Container` is the low-level DI container. Use it directly for testing or advanced scenarios:

```go
c := gaz.NewContainer()
gaz.For[*Database](c).Provider(NewDatabase)
c.Build()
db, _ := gaz.Resolve[*Database](c)
```

### Service Scopes

```go
// Singleton (default): one instance for container lifetime
app.ProvideSingleton(NewDatabase)

// Transient: new instance on every resolution
app.ProvideTransient(NewRequest)

// Eager: singleton instantiated at Build() time
app.ProvideEager(NewConnectionPool)
```

### Lifecycle Hooks

Services implementing `Starter` or `Stopper` interfaces get automatic lifecycle management:

```go
type Starter interface {
	OnStart(context.Context) error
}

type Stopper interface {
	OnStop(context.Context) error
}
```

Hooks are called in dependency order (dependencies start first, stop last).

## Documentation

- [Getting Started](docs/getting-started.md) - First application walkthrough
- [Concepts](docs/concepts.md) - DI fundamentals, scopes, lifecycle
- [Configuration](docs/configuration.md) - Config loading, env vars, validation
- [Validation](docs/validation.md) - Struct tags and custom validators
- [Advanced](docs/advanced.md) - Modules, testing, Cobra integration

## Examples

See the [examples](examples/) directory:

- [basic](examples/basic/) - Minimal working application
- [http-server](examples/http-server/) - HTTP server with graceful shutdown
- [config-loading](examples/config-loading/) - Configuration files and environment variables
- [lifecycle](examples/lifecycle/) - Services with OnStart/OnStop hooks
- [modules](examples/modules/) - Organizing providers into modules
- [cobra-cli](examples/cobra-cli/) - CLI application with Cobra

## License

MIT
