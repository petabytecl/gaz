# AGENTS.md - AI Coding Agent Guidelines

This document provides essential context for AI coding agents working in the `gaz` repository.

**Project:** gaz - Type-safe dependency injection framework for Go with lifecycle management  
**Module:** `github.com/petabytecl/gaz`  
**Go Version:** 1.25+

## Build, Lint, and Test Commands

### Quick Reference

| Command        | Description                                          |
|----------------|------------------------------------------------------|
| `make test`    | Run all tests with race detection                    |
| `make cover`   | Run tests with coverage (90% threshold enforced)     |
| `make lint`    | Run golangci-lint                                    |
| `make fmt`     | Format code with gofmt + goimports                   |
| `make fmt-check` | Check formatting without modifying                 |

### Running Tests

```bash
# Run all tests
make test
go test -race ./...

# Run tests in a specific package
go test -race -v ./di/...
go test -race -v ./health/...
go test -race -v ./config/...

# Run a single test function
go test -race -v ./... -run TestFunctionName

# Run a specific test suite method
go test -race -v ./... -run "TestAppTestSuite/TestRunAndStop"

# Run tests matching a pattern
go test -race -v ./... -run "Test.*Config.*"

# Run tests with coverage
make cover
go test -race -coverprofile=coverage.out -covermode=atomic ./...
```

### Linting

```bash
# Run linter
make lint
golangci-lint run

# Auto-fix issues
golangci-lint run --fix
```

## Code Style Guidelines

### Import Ordering

Imports MUST be organized in three groups (enforced by gci):

```go
import (
    // 1. Standard library
    "context"
    "errors"
    "fmt"

    // 2. External dependencies
    "github.com/spf13/cobra"
    "github.com/stretchr/testify/require"

    // 3. Local packages
    "github.com/petabytecl/gaz"
    "github.com/petabytecl/gaz/di"
)
```

### Naming Conventions

**Packages:** Lowercase, short, no underscores
```go
package health     // Good
package health_check // Bad
```

**Types:** PascalCase for exported, camelCase for unexported
```go
type ModuleBuilder struct {}  // Exported
type builtModule struct {}    // Unexported
```

**Interfaces:** `-er` suffix for single-method, nouns for multi-method
```go
type Starter interface { OnStart(context.Context) error }
type Backend interface { Get(key string) any; Set(key string, value any) }
```

**Constructors:**
- `New()` - single primary type per package (avoids stutter)
- `NewX()` - multiple types in package
- `NewXWithY()` - variant constructors

```go
// Single type: use New()
func New(opts ...Option) *Manager { ... }

// Multiple types: use NewX()
func NewManager() *Manager { ... }
func NewShutdownCheck() *ShutdownCheck { ... }
```

### Error Handling

**Sentinel errors:** `Err` prefix, `"pkg: description"` format
```go
var (
    ErrNotFound  = errors.New("di: service not found")
    ErrCycle     = errors.New("di: circular dependency detected")
    ErrDuplicate = errors.New("di: service already registered")
)
```

**Error wrapping:** Use `%w`, lowercase context, no trailing punctuation
```go
if err := doSomething(); err != nil {
    return fmt.Errorf("register manager: %w", err)
}
```

**Error types (if needed):** `Error` suffix
```go
type ValidationError struct {
    Field   string
    Message string
}
```

### DI Registration Pattern

Use type-safe generic functions:

```go
// Registration
gaz.For[*Database](c).Provider(NewDatabase)
gaz.For[*Service](c).Transient().Provider(NewService)
gaz.For[*Pool](c).Eager().Provider(NewPool)

// Resolution
db, err := gaz.Resolve[*Database](c)
db := gaz.MustResolve[*Database](c)

// Type checking
if gaz.Has[*Database](c) { ... }
```

### Module Pattern

Module factory functions use signature `func Module(c *gaz.Container) error`:

```go
func Module(c *gaz.Container) error {
    if err := gaz.For[*Manager](c).Provider(NewManager); err != nil {
        return fmt.Errorf("register manager: %w", err)
    }
    return nil
}
```

For complex modules, use the builder:

```go
module := gaz.NewModule("database").
    Provide(DBProvider).
    Flags(func(fs *pflag.FlagSet) {
        fs.String("db-host", "localhost", "Database host")
    }).
    Build()
```

### Lifecycle Interfaces

Services implementing these get automatic lifecycle management:

```go
type Starter interface { OnStart(context.Context) error }
type Stopper interface { OnStop(context.Context) error }
```

## Testing Guidelines

### Test Framework

Use `github.com/stretchr/testify` with test suites:

```go
type MyTestSuite struct {
    suite.Suite
}

func TestMyTestSuite(t *testing.T) {
    suite.Run(t, new(MyTestSuite))
}

func (s *MyTestSuite) TestSomething() {
    s.Require().NoError(err)
    s.Assert().Equal(expected, actual)
}
```

### Test Helpers (`gaztest` package)

```go
// Basic test app
app, err := gaztest.New(t).Build()
app.RequireStart()
defer app.RequireStop()

// With modules
app, err := gaztest.New(t).
    WithModules(myModule).
    Build()

// Type-safe resolution
svc := gaztest.RequireResolve[*MyService](t, app)
```

### Coverage Requirements

- **Minimum threshold: 90%**
- Coverage is enforced in CI via `make cover`

## Project Structure

```
gaz/
├── app.go, types.go, errors.go    # Core App and types
├── di/                            # DI container implementation
├── config/                        # Configuration loading
│   └── viper/                     # Viper backend
├── health/                        # Health checks
├── worker/                        # Background workers
├── cron/                          # Cron job scheduling
├── eventbus/                      # Pub/sub event bus
├── logger/                        # Structured logging
├── backoff/                       # Retry utilities
├── gaztest/                       # Test helpers
└── examples/                      # Reference implementations
```

## Linter Configuration

The `.golangci.yml` is comprehensive with 60+ linters. Key enforcements:

- **Complexity:** cyclop, gocognit, gocyclo
- **Error handling:** errcheck, errorlint, wrapcheck
- **Style:** revive, godot (comments end with period)
- **No globals:** gochecknoglobals
- **No init:** gochecknoinits

## Key Dependencies

| Package                     | Purpose                      |
|-----------------------------|------------------------------|
| `spf13/cobra`               | CLI framework                |
| `spf13/viper`               | Configuration management     |
| `go-playground/validator/v10` | Struct validation          |
| `stretchr/testify`          | Testing assertions/suites    |

## Common Patterns

### Provider with Dependencies

```go
func NewService(c *gaz.Container) (*Service, error) {
    db, err := gaz.Resolve[*Database](c)
    if err != nil {
        return nil, fmt.Errorf("resolve database: %w", err)
    }
    return &Service{db: db}, nil
}
```

### Configuration Provider

```go
func ConfigProvider[T any](c *gaz.Container) (*T, error) {
    mgr, err := gaz.Resolve[*config.Manager](c)
    if err != nil {
        return nil, err
    }
    var cfg T
    if err := mgr.UnmarshalKey("section", &cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    return &cfg, nil
}
```

## CI/CD

GitHub Actions runs on push/PR with:
1. Go 1.25 setup
2. golangci-lint v2.8
3. `make cover` (tests + 90% coverage threshold)
