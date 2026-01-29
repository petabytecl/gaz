# Configuration

Loading and managing application configuration with gaz.

## Overview

Gaz provides two main approaches to configuration:

1. **ConfigProvider Pattern** - Services declare their config requirements via interface (recommended)
2. **Standalone config Package** - Direct configuration loading for simpler use cases

## ConfigProvider Pattern (Recommended)

The ConfigProvider pattern allows services to declare their configuration needs declaratively. The framework automatically handles loading, environment variable binding, and validation.

### Implementing ConfigProvider

Create a struct that implements `ConfigNamespace()` and `ConfigFlags()`:

```go
import "github.com/petabytecl/gaz"

// ServerConfig implements ConfigProvider to declare its configuration requirements.
type ServerConfig struct {
    pv *gaz.ProviderValues
}

// ConfigNamespace returns the namespace prefix for all config keys.
// Keys from ConfigFlags() are prefixed with this namespace.
func (s *ServerConfig) ConfigNamespace() string {
    return "server"
}

// ConfigFlags declares the configuration flags this provider needs.
func (s *ServerConfig) ConfigFlags() []gaz.ConfigFlag {
    return []gaz.ConfigFlag{
        {Key: "host", Type: gaz.ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
        {Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 8080, Description: "Server port"},
        {Key: "debug", Type: gaz.ConfigFlagTypeBool, Default: false, Description: "Debug mode"},
    }
}

// NewServerConfig creates the config with injected ProviderValues.
func NewServerConfig(c *gaz.Container) (*ServerConfig, error) {
    pv, err := gaz.Resolve[*gaz.ProviderValues](c)
    if err != nil {
        return nil, err
    }
    return &ServerConfig{pv: pv}, nil
}

// Accessor methods for type-safe config access
func (s *ServerConfig) Host() string { return s.pv.GetString("server.host") }
func (s *ServerConfig) Port() int    { return s.pv.GetInt("server.port") }
func (s *ServerConfig) Debug() bool  { return s.pv.GetBool("server.debug") }
```

### Registering a ConfigProvider

Register the provider using the fluent API:

```go
app := gaz.New()

if err := gaz.For[*ServerConfig](app.Container()).Provider(NewServerConfig); err != nil {
    log.Fatal(err)
}

if err := app.Build(); err != nil {
    log.Fatal(err)
}

cfg := gaz.MustResolve[*ServerConfig](app.Container())
fmt.Printf("Server: %s:%d\n", cfg.Host(), cfg.Port())
```

### How It Works

During `Build()`, the framework:

1. Registers `ProviderValues` early (before providers run)
2. Instantiates providers (can inject `ProviderValues`)
3. Collects `ConfigNamespace()` and `ConfigFlags()` from providers
4. Registers defaults and binds environment variables
5. Validates required flags are set

## ConfigFlag Types

| Type | Go Type | Example |
|------|---------|---------|
| `ConfigFlagTypeString` | `string` | `"localhost"` |
| `ConfigFlagTypeInt` | `int` | `8080` |
| `ConfigFlagTypeBool` | `bool` | `true` |
| `ConfigFlagTypeDuration` | `time.Duration` | `"30s"` |
| `ConfigFlagTypeFloat` | `float64` | `3.14` |

### Required Flags

Mark a flag as required:

```go
{Key: "password", Type: gaz.ConfigFlagTypeString, Required: true, Description: "Database password"}
```

If a required flag is not set, `Build()` returns an error:

```
provider "database": required config key "database.password" is not set
```

## ProviderValues Access

`ProviderValues` provides typed access to configuration values:

```go
func NewRedisClient(c *gaz.Container) (*RedisClient, error) {
    pv := gaz.MustResolve[*gaz.ProviderValues](c)
    
    host := pv.GetString("redis.host")
    port := pv.GetInt("redis.port")
    password := pv.GetString("redis.password")
    timeout := pv.GetDuration("redis.timeout")
    
    return &RedisClient{
        Addr:     fmt.Sprintf("%s:%d", host, port),
        Password: password,
        Timeout:  timeout,
    }, nil
}
```

**Available methods:**

- `GetString(key string) string`
- `GetInt(key string) int`
- `GetBool(key string) bool`
- `GetDuration(key string) time.Duration`
- `GetFloat(key string) float64`

## Environment Variables

Provider config keys are automatically bound to environment variables using single underscore separation:

| Config Key | Environment Variable |
|------------|---------------------|
| `server.host` | `SERVER_HOST` |
| `server.port` | `SERVER_PORT` |
| `redis.password` | `REDIS_PASSWORD` |

Environment variables override config file values.

## Standalone Config Usage

For simpler use cases or when not using the full framework, use the config package directly:

```go
import "github.com/petabytecl/gaz/config"

type AppConfig struct {
    Server struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"server"`
}

func main() {
    cfg := &AppConfig{}
    mgr := config.New(
        config.WithName("config"),
        config.WithSearchPaths(".", "./config", "/etc/myapp"),
    )
    
    if err := mgr.LoadInto(cfg); err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
}
```

**Supported file formats:**

- YAML (`.yaml`, `.yml`)
- JSON (`.json`)
- TOML (`.toml`)
- HCL (`.hcl`)

### Config Options

```go
mgr := config.New(
    config.WithName("config"),           // Looks for config.yaml, config.json, etc.
    config.WithSearchPaths(".", "./config"),
    config.WithEnvPrefix("MYAPP"),        // Bind env vars with prefix
)
```

## Config Structs

Define configuration as a Go struct with `mapstructure` tags:

```go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Debug    bool           `mapstructure:"debug"`
}

type ServerConfig struct {
    Host string `mapstructure:"host"`
    Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
    URL         string `mapstructure:"url"`
    MaxConns    int    `mapstructure:"max_conns"`
    IdleTimeout string `mapstructure:"idle_timeout"`
}
```

**Example config.yaml:**

```yaml
server:
  host: localhost
  port: 8080
database:
  url: postgres://localhost/myapp
  max_conns: 10
  idle_timeout: 5m
debug: false
```

## Defaults

### Defaulter Interface

Implement `Defaulter` for programmatic defaults:

```go
type Config struct {
    Server ServerConfig `mapstructure:"server"`
}

func (c *Config) Default() {
    if c.Server.Port == 0 {
        c.Server.Port = 8080
    }
    if c.Server.Host == "" {
        c.Server.Host = "localhost"
    }
}
```

The `Default()` method is called automatically after loading.

### ConfigFlag Defaults

When using ConfigProvider, defaults are specified per-flag:

```go
{Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 8080, Description: "Server port"}
```

## Validation

See [Validation](validation.md) for struct tag validation and custom validators.

## Complete Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/petabytecl/gaz"
)

// ServerConfig implements ConfigProvider
type ServerConfig struct {
    pv *gaz.ProviderValues
}

func (s *ServerConfig) ConfigNamespace() string { return "server" }

func (s *ServerConfig) ConfigFlags() []gaz.ConfigFlag {
    return []gaz.ConfigFlag{
        {Key: "host", Type: gaz.ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
        {Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 8080, Description: "Server port"},
        {Key: "debug", Type: gaz.ConfigFlagTypeBool, Default: false, Description: "Debug mode"},
    }
}

func NewServerConfig(c *gaz.Container) (*ServerConfig, error) {
    pv, err := gaz.Resolve[*gaz.ProviderValues](c)
    if err != nil {
        return nil, err
    }
    return &ServerConfig{pv: pv}, nil
}

func (s *ServerConfig) Host() string { return s.pv.GetString("server.host") }
func (s *ServerConfig) Port() int    { return s.pv.GetInt("server.port") }
func (s *ServerConfig) Debug() bool  { return s.pv.GetBool("server.debug") }

func main() {
    app := gaz.New()

    // Register the ConfigProvider
    if err := gaz.For[*ServerConfig](app.Container()).Provider(NewServerConfig); err != nil {
        log.Fatalf("Failed to register config provider: %v", err)
    }

    // Build triggers config loading and provider instantiation
    if err := app.Build(); err != nil {
        log.Fatalf("Failed to build app: %v", err)
    }

    // Get the ServerConfig - ProviderValues already injected
    cfg := gaz.MustResolve[*ServerConfig](app.Container())

    fmt.Println("Configuration loaded via ConfigProvider pattern:")
    fmt.Printf("  Server: %s:%d\n", cfg.Host(), cfg.Port())
    fmt.Printf("  Debug:  %v\n", cfg.Debug())
    fmt.Println()
    fmt.Println("Config sources (in priority order):")
    fmt.Println("  1. Environment variables (e.g., SERVER_HOST, SERVER_PORT)")
    fmt.Println("  2. Config file (config.yaml)")
    fmt.Println("  3. Defaults from ConfigFlags()")
}
```

**config.yaml:**

```yaml
server:
  host: 0.0.0.0
  port: 3000
  debug: true
```

Run with environment override:

```bash
export SERVER_PORT=9090
go run main.go
# Output: Server: 0.0.0.0:9090
```
