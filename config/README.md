# gaz/config

Standalone configuration management for Go.

## Installation

```bash
go get github.com/petabytecl/gaz/config
```

## Quick Start

```go
package main

import (
    "log"

    "github.com/petabytecl/gaz/config"
    _ "github.com/petabytecl/gaz/config/viper" // Backend implementation
)

type AppConfig struct {
    Server struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
    } `mapstructure:"server"`
    Debug bool `mapstructure:"debug"`
}

func main() {
    cfg := &AppConfig{}
    
    mgr := config.New(
        config.WithName("config"),
        config.WithSearchPaths(".", "./config"),
    )
    
    if err := mgr.LoadInto(cfg); err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
}
```

## Features

- **Backend interface** - Abstracts viper for flexibility
- **File loading** - YAML, JSON, TOML support
- **Environment variable binding** - Override config with env vars
- **Validation** - Struct tags with go-playground/validator
- **Defaulter/Validator interfaces** - Custom defaults and validation logic

## Backend Interface

The core `config` package defines interfaces. The viper implementation is in a subpackage to isolate the dependency:

```go
import (
    "github.com/petabytecl/gaz/config"
    _ "github.com/petabytecl/gaz/config/viper" // Register viper backend
)
```

Optional composed interfaces extend Backend capabilities:

- `Watcher` - Configuration file watching
- `Writer` - Writing configuration back to files
- `EnvBinder` - Environment variable binding

## Validation

Structs can implement `Defaulter` and `Validator` interfaces:

```go
func (c *AppConfig) Default() {
    c.Server.Host = "localhost"
    c.Server.Port = 8080
}

func (c *AppConfig) Validate() error {
    if c.Server.Port < 1 {
        return errors.New("port must be positive")
    }
    return nil
}
```

Struct tag validation using go-playground/validator is also supported:

```go
type Config struct {
    Port int `validate:"required,min=1,max=65535"`
}
```

See [gaz framework](../README.md) for full documentation.
