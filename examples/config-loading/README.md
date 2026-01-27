# Config Loading Example

Demonstrates gaz configuration management with file and environment variable loading.

## What This Demonstrates

- Loading configuration from YAML files
- Environment variable overrides
- Struct-based configuration with validation
- Validation tags (`required`, `min`, `max`)

## Run

With default config file:

```bash
go run .
```

With environment variable override:

```bash
APP_SERVER__PORT=9090 APP_DEBUG=false go run .
```

## Expected Output

Default config:
```
Configuration loaded:
  Server: localhost:8080
  Debug:  true
```

With env override:
```
Configuration loaded:
  Server: localhost:9090
  Debug:  false
```

## Configuration Sources

Configuration is loaded from multiple sources (later sources override earlier):

1. **Defaults** - Via `Defaulter` interface
2. **Config file** - `config.yaml` in search paths
3. **Profile config** - `config.{profile}.yaml` if profile env var set
4. **Environment variables** - Prefixed with `APP_`

## Environment Variable Mapping

With `WithEnvPrefix("APP")`:

| Config Key | Environment Variable |
|------------|---------------------|
| `server.port` | `APP_SERVER__PORT` |
| `server.host` | `APP_SERVER__HOST` |
| `debug` | `APP_DEBUG` |

Note: Nested keys use double underscore (`__`) as separator.

## Validation Tags

```go
type Config struct {
    Port int `validate:"required,min=1,max=65535"`
}
```

Available validation tags:
- `required` - Field must be non-zero
- `min=N` - Minimum value (numbers) or length (strings)
- `max=N` - Maximum value or length
- `oneof=a b c` - Value must be one of the listed options

## What's Next

- See [lifecycle](../lifecycle) for startup/shutdown hooks
- See [basic](../basic) for minimal DI example
