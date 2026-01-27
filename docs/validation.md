# Validation

Validating configuration with struct tags using go-playground/validator.

## Validation Tags

Add `validate` tags to config struct fields:

```go
type Config struct {
    Server ServerConfig `mapstructure:"server"`
}

type ServerConfig struct {
    Host string `mapstructure:"host" validate:"required"`
    Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
}
```

Validation runs automatically when config loads. If validation fails, the application exits immediately with a clear error message.

## Common Validators

### Required Fields

```go
Name string `validate:"required"`           // Must be non-zero
Email string `validate:"required,email"`    // Must be valid email
```

### Numeric Constraints

```go
Port int `validate:"min=1,max=65535"`       // Range check
Workers int `validate:"gte=1,lte=100"`      // Greater/less than or equal
Timeout int `validate:"gt=0,lt=3600"`       // Greater/less than (exclusive)
```

### String Validators

```go
Email string `validate:"email"`             // Valid email format
URL string `validate:"url"`                 // Valid URL format
IP string `validate:"ip"`                   // Valid IP address
IPv4 string `validate:"ipv4"`               // Valid IPv4 address
IPv6 string `validate:"ipv6"`               // Valid IPv6 address
```

### Enumeration

```go
LogLevel string `validate:"oneof=debug info warn error"`
Env string `validate:"oneof=development staging production"`
```

## Cross-field Validation

Validate fields based on other fields.

### required_if

Field required if another field has specific value:

```go
type Config struct {
    UseSSL   bool   `mapstructure:"use_ssl"`
    CertFile string `mapstructure:"cert_file" validate:"required_if=UseSSL true"`
    KeyFile  string `mapstructure:"key_file" validate:"required_if=UseSSL true"`
}
```

### required_unless

Field required unless another field has specific value:

```go
type Config struct {
    Env    string `mapstructure:"env"`
    Secret string `mapstructure:"secret" validate:"required_unless=Env development"`
}
```

### required_with

Field required when another field is present:

```go
type Config struct {
    Username string `mapstructure:"username"`
    Password string `mapstructure:"password" validate:"required_with=Username"`
}
```

### required_without

Field required when another field is absent:

```go
type Config struct {
    APIKey   string `mapstructure:"api_key"`
    Username string `mapstructure:"username" validate:"required_without=APIKey"`
    Password string `mapstructure:"password" validate:"required_without=APIKey"`
}
```

### eqfield

Field must equal another field:

```go
type Config struct {
    Password        string `mapstructure:"password"`
    ConfirmPassword string `mapstructure:"confirm_password" validate:"eqfield=Password"`
}
```

## Nested Structs

Nested structs are validated recursively:

```go
type Config struct {
    Database DatabaseConfig `mapstructure:"database"`
}

type DatabaseConfig struct {
    Host     string `mapstructure:"host" validate:"required"`
    Port     int    `mapstructure:"port" validate:"required,min=1,max=65535"`
    Username string `mapstructure:"username" validate:"required"`
    Password string `mapstructure:"password" validate:"required"`
}
```

### Slices with dive

Use `dive` to validate slice elements:

```go
type Config struct {
    Servers []ServerConfig `mapstructure:"servers" validate:"required,dive"`
}

type ServerConfig struct {
    Host string `mapstructure:"host" validate:"required"`
    Port int    `mapstructure:"port" validate:"required,min=1"`
}
```

### Maps with dive

```go
type Config struct {
    Headers map[string]string `mapstructure:"headers" validate:"dive,required"`
}
```

## Custom Validators

Implement the `Validator` interface for complex validation logic:

```go
type Config struct {
    StartDate string `mapstructure:"start_date"`
    EndDate   string `mapstructure:"end_date"`
}

func (c *Config) Validate() error {
    start, err := time.Parse("2006-01-02", c.StartDate)
    if err != nil {
        return fmt.Errorf("invalid start_date: %w", err)
    }
    
    end, err := time.Parse("2006-01-02", c.EndDate)
    if err != nil {
        return fmt.Errorf("invalid end_date: %w", err)
    }
    
    if end.Before(start) {
        return errors.New("end_date must be after start_date")
    }
    
    return nil
}
```

The `Validate()` method is called after struct tag validation passes.

## Error Messages

Validation errors are formatted with field path and constraint:

```
config validation failed:
Config.server.host: required field cannot be empty (validate:"required")
Config.server.port: must be at least 1 (validate:"min")
Config.database.url: must be a valid URL (validate:"url")
```

**Error message mapping:**

| Tag | Message |
|-----|---------|
| `required` | required field cannot be empty |
| `min=N` | must be at least N |
| `max=N` | must be at most N |
| `oneof=a b c` | must be one of: a b c |
| `email` | must be a valid email address |
| `url` | must be a valid URL |
| `ip` | must be a valid IP address |
| `required_if` | required when [condition] |
| `required_unless` | required unless [condition] |

## Startup Behavior

Validation failures cause immediate exit:

```go
app := gaz.New()
cfg := &Config{}

app.WithConfig(cfg,
    gaz.WithSearchPaths("."),
)

// If validation fails, Build() returns error
if err := app.Build(); err != nil {
    log.Fatal(err)  // Exits with validation error message
}
```

**Design philosophy:** Fail fast. Invalid configuration should never reach runtime. Catch errors at startup, not in production.

## Complete Example

```go
package main

import (
    "context"
    "errors"
    "log"
    "time"

    "github.com/petabytecl/gaz"
)

type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Cache    CacheConfig    `mapstructure:"cache"`
}

type ServerConfig struct {
    Host         string `mapstructure:"host" validate:"required"`
    Port         int    `mapstructure:"port" validate:"required,min=1,max=65535"`
    ReadTimeout  string `mapstructure:"read_timeout" validate:"required"`
    WriteTimeout string `mapstructure:"write_timeout" validate:"required"`
}

type DatabaseConfig struct {
    URL      string `mapstructure:"url" validate:"required,url"`
    MaxConns int    `mapstructure:"max_conns" validate:"min=1,max=100"`
}

type CacheConfig struct {
    Enabled bool   `mapstructure:"enabled"`
    TTL     string `mapstructure:"ttl" validate:"required_if=Enabled true"`
    Size    int    `mapstructure:"size" validate:"required_if=Enabled true,min=1"`
}

func (c *Config) Validate() error {
    // Custom validation: timeouts must be parseable
    if _, err := time.ParseDuration(c.Server.ReadTimeout); err != nil {
        return errors.New("server.read_timeout must be a valid duration")
    }
    if _, err := time.ParseDuration(c.Server.WriteTimeout); err != nil {
        return errors.New("server.write_timeout must be a valid duration")
    }
    return nil
}

func main() {
    app := gaz.New()
    cfg := &Config{}

    app.WithConfig(cfg,
        gaz.WithSearchPaths("."),
        gaz.WithEnvPrefix("MYAPP"),
    )

    if err := app.Build(); err != nil {
        log.Fatalf("Configuration error: %v", err)
    }

    app.Run(context.Background())
}
```

**config.yaml:**

```yaml
server:
  host: localhost
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  url: postgres://localhost/myapp
  max_conns: 10

cache:
  enabled: true
  ttl: 5m
  size: 1000
```
