# Configuration

Loading and managing application configuration with gaz.

## Config Structs

Define configuration as a Go struct with `mapstructure` tags for field mapping:

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

## Loading Config

Use `ConfigManager` to load configuration from files:

```go
cfg := &Config{}
cm := gaz.NewConfigManager(cfg,
    gaz.WithSearchPaths(".", "./config", "/etc/myapp"),
    gaz.WithName("config"),        // Looks for config.yaml, config.json, etc.
    gaz.WithFileType("yaml"),      // Default file type
)

if err := cm.Load(); err != nil {
    log.Fatal(err)
}
```

**Supported file formats:**

- YAML (`.yaml`, `.yml`)
- JSON (`.json`)
- TOML (`.toml`)
- HCL (`.hcl`)

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

## Environment Variables

Bind environment variables with a prefix:

```go
cm := gaz.NewConfigManager(cfg,
    gaz.WithEnvPrefix("MYAPP"),
)
```

Environment variables are translated automatically:

| Struct Field | Environment Variable |
|--------------|---------------------|
| `server.host` | `MYAPP__SERVER__HOST` |
| `server.port` | `MYAPP__SERVER__PORT` |
| `database.url` | `MYAPP__DATABASE__URL` |
| `debug` | `MYAPP__DEBUG` |

**Note:** Double underscore (`__`) separates nested fields.

Environment variables override file values:

```bash
export MYAPP__SERVER__PORT=9090
export MYAPP__DEBUG=true
```

## Profiles

Use profiles for environment-specific configuration:

```go
cm := gaz.NewConfigManager(cfg,
    gaz.WithProfileEnv("APP_ENV"),  // Read profile from APP_ENV
)
```

**How profiles work:**

1. Base config loaded from `config.yaml`
2. Profile config merged from `config.{profile}.yaml`
3. Profile values override base values

**Example:**

```bash
export APP_ENV=production
```

Files loaded:

1. `config.yaml` (base)
2. `config.production.yaml` (profile overrides)

**config.yaml:**

```yaml
server:
  port: 8080
debug: true
```

**config.production.yaml:**

```yaml
server:
  port: 443
debug: false
```

Result: `port=443`, `debug=false`

## Defaults

Set default values in two ways.

### WithDefaults Option

```go
cm := gaz.NewConfigManager(cfg,
    gaz.WithDefaults(map[string]any{
        "server.host": "localhost",
        "server.port": 8080,
        "database.max_conns": 10,
    }),
)
```

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

## Injecting Config

Register config as a singleton for DI resolution:

```go
app := gaz.New()
cfg := &Config{}

app.WithConfig(cfg,
    gaz.WithSearchPaths("."),
    gaz.WithEnvPrefix("MYAPP"),
)

app.ProvideSingleton(func(c *gaz.Container) (*Server, error) {
    // Config is automatically available
    cfg, err := gaz.Resolve[*Config](c)
    if err != nil {
        return nil, err
    }
    return NewServer(cfg.Server.Host, cfg.Server.Port), nil
})
```

## Provider Config

For reusable providers that need configuration, implement `ConfigProvider`:

```go
type RedisProvider struct{}

func (r *RedisProvider) ConfigNamespace() string {
    return "redis"
}

func (r *RedisProvider) ConfigFlags() []gaz.ConfigFlag {
    return []gaz.ConfigFlag{
        {
            Key:         "host",
            Type:        gaz.ConfigFlagTypeString,
            Default:     "localhost",
            Description: "Redis server host",
        },
        {
            Key:         "port",
            Type:        gaz.ConfigFlagTypeInt,
            Default:     6379,
            Description: "Redis server port",
        },
        {
            Key:         "password",
            Type:        gaz.ConfigFlagTypeString,
            Required:    true,
            Description: "Redis password",
        },
    }
}
```

**Config flag types:**

| Type | Go Type | Example |
|------|---------|---------|
| `ConfigFlagTypeString` | `string` | `"localhost"` |
| `ConfigFlagTypeInt` | `int` | `6379` |
| `ConfigFlagTypeBool` | `bool` | `true` |
| `ConfigFlagTypeDuration` | `time.Duration` | `"30s"` |
| `ConfigFlagTypeFloat` | `float64` | `3.14` |

**Accessing provider config values:**

```go
func NewRedisClient(c *gaz.Container) (*RedisClient, error) {
    pv := gaz.MustResolve[*gaz.ProviderValues](c)
    
    host := pv.GetString("redis.host")
    port := pv.GetInt("redis.port")
    password := pv.GetString("redis.password")
    
    return &RedisClient{
        Addr:     fmt.Sprintf("%s:%d", host, port),
        Password: password,
    }, nil
}
```

**Environment variable binding:**

Provider config keys are automatically bound to environment variables:

| Config Key | Environment Variable |
|------------|---------------------|
| `redis.host` | `REDIS_HOST` |
| `redis.port` | `REDIS_PORT` |
| `redis.password` | `REDIS_PASSWORD` |

**Validation at Build time:**

If a required provider config flag is not set, `Build()` returns an error:

```
provider "redis": required config key "redis.password" is not set
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/petabytecl/gaz"
)

type Config struct {
    Server struct {
        Host string `mapstructure:"host" validate:"required"`
        Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
    } `mapstructure:"server"`
    Debug bool `mapstructure:"debug"`
}

func (c *Config) Default() {
    if c.Server.Host == "" {
        c.Server.Host = "localhost"
    }
    if c.Server.Port == 0 {
        c.Server.Port = 8080
    }
}

func main() {
    app := gaz.New()
    cfg := &Config{}

    app.WithConfig(cfg,
        gaz.WithSearchPaths(".", "./config"),
        gaz.WithEnvPrefix("MYAPP"),
        gaz.WithProfileEnv("APP_ENV"),
    )

    app.ProvideSingleton(func(c *gaz.Container) (*http.Server, error) {
        cfg := gaz.MustResolve[*Config](c)
        addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
        return &http.Server{Addr: addr}, nil
    })

    if err := app.Build(); err != nil {
        log.Fatal(err)
    }

    app.Run(context.Background())
}
```
