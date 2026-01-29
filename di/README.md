# gaz/di

Standalone dependency injection container for Go.

## Installation

```bash
go get github.com/petabytecl/gaz/di
```

## Quick Start

```go
package main

import (
    "log"

    "github.com/petabytecl/gaz/di"
)

type Database struct {
    DSN string
}

func NewDatabase(c *di.Container) (*Database, error) {
    return &Database{DSN: "postgres://localhost/mydb"}, nil
}

func main() {
    c := di.New()

    di.For[*Database](c).Provider(NewDatabase)

    if err := c.Build(); err != nil {
        log.Fatal(err)
    }

    db, _ := di.Resolve[*Database](c)
    log.Printf("Connected to: %s", db.DSN)
}
```

## Features

- **Type-safe generics API** - `di.For[T]()` and `di.Resolve[T]()` for compile-time type checking
- **Singleton/Transient/Eager scopes** - Control instance lifetime
- **Named registrations** - Multiple instances of the same type
- **Lifecycle hooks** - `OnStart`/`OnStop` for startup and shutdown logic
- **Works standalone or with gaz.App** - Use di package directly or as part of the full framework

## Registration Patterns

```go
// Singleton (default): one instance for container lifetime
di.For[*Service](c).Provider(NewService)

// Transient: new instance on every resolution
di.For[*Request](c).Transient().Provider(NewRequest)

// Eager: singleton instantiated at Build() time
di.For[*Pool](c).Eager().Provider(NewPool)

// Instance: register a pre-built value
di.For[*Config](c).Instance(cfg)
```

## Named Services

```go
di.For[*sql.DB](c).Named("primary").Provider(NewPrimaryDB)
di.For[*sql.DB](c).Named("replica").Provider(NewReplicaDB)

primary, _ := di.Resolve[*sql.DB](c, di.Named("primary"))
replica, _ := di.Resolve[*sql.DB](c, di.Named("replica"))
```

## Lifecycle Hooks

```go
di.For[*Server](c).
    OnStart(func(ctx context.Context, s *Server) error {
        return s.ListenAndServe()
    }).
    OnStop(func(ctx context.Context, s *Server) error {
        return s.Shutdown(ctx)
    }).
    Provider(NewServer)
```

See [gaz framework](../README.md) for full documentation.
