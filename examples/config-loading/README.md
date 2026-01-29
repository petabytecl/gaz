# Config Loading Example

Demonstrates the ConfigProvider pattern for flag-based configuration in gaz.

## What This Demonstrates

- **ConfigProvider interface** - Services declare config requirements via `ConfigNamespace()` + `ConfigFlags()`
- **ProviderValues injection** - Config values injected in provider constructors during Build()
- **Config source precedence** - Environment variables > config file > defaults

## Run

With default values:

```bash
go run .
```

With environment variable overrides:

```bash
SERVER_HOST=0.0.0.0 SERVER_PORT=9090 go run .
```

## Expected Output

```
Configuration loaded via ConfigProvider pattern:
  Server: localhost:8080
  Debug:  false

Key pattern: ProviderValues injected in provider constructor
  - NewServerConfig receives *ProviderValues via DI
  - Accessor methods (Host, Port, Debug) use stored ProviderValues
  - No need to resolve ProviderValues in main()

Config sources (in priority order):
  1. Environment variables (e.g., SERVER_HOST, SERVER_PORT)
  2. Config file (config.yaml)
  3. Defaults from ConfigFlags()
```

With env overrides:

```
Configuration loaded via ConfigProvider pattern:
  Server: 0.0.0.0:9090
  Debug:  false
...
```

## ConfigProvider Pattern

### 1. Implement ConfigProvider Interface

```go
type ServerConfig struct {
    pv *gaz.ProviderValues
}

// ConfigNamespace returns the prefix for all config keys
func (s *ServerConfig) ConfigNamespace() string {
    return "server"
}

// ConfigFlags declares typed config keys with defaults
func (s *ServerConfig) ConfigFlags() []gaz.ConfigFlag {
    return []gaz.ConfigFlag{
        {Key: "host", Type: gaz.ConfigFlagTypeString, Default: "localhost", Description: "Server host"},
        {Key: "port", Type: gaz.ConfigFlagTypeInt, Default: 8080, Description: "Server port"},
        {Key: "debug", Type: gaz.ConfigFlagTypeBool, Default: false, Description: "Debug mode"},
    }
}
```

### 2. Inject ProviderValues in Constructor

```go
func NewServerConfig(c *gaz.Container) (*ServerConfig, error) {
    pv, err := gaz.Resolve[*gaz.ProviderValues](c)
    if err != nil {
        return nil, err
    }
    return &ServerConfig{pv: pv}, nil
}
```

### 3. Create Typed Accessor Methods

```go
func (s *ServerConfig) Host() string { return s.pv.GetString("server.host") }
func (s *ServerConfig) Port() int    { return s.pv.GetInt("server.port") }
func (s *ServerConfig) Debug() bool  { return s.pv.GetBool("server.debug") }
```

## Environment Variable Mapping

Environment variables use single underscore and uppercase:

| Config Key | Environment Variable |
|------------|---------------------|
| `server.host` | `SERVER_HOST` |
| `server.port` | `SERVER_PORT` |
| `server.debug` | `SERVER_DEBUG` |

## Config Sources (Priority Order)

1. **Environment variables** - Highest priority, always win
2. **Config file** - `config.yaml` in current directory
3. **Defaults from ConfigFlags()** - Lowest priority

## Key Concepts

- **ConfigProvider** combines `ConfigNamespace()` + `ConfigFlags()` interfaces
- **ProviderValues** is registered BEFORE providers run during `Build()`
- This allows constructors to inject `*ProviderValues` as a dependency
- Accessor methods provide typed access to config values

## What's Next

- See [lifecycle](../lifecycle) for startup/shutdown hooks
- See [basic](../basic) for minimal DI example
