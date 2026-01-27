# Basic Example

Minimal working gaz application demonstrating core DI concepts.

## What This Demonstrates

- Creating a gaz application with `gaz.New()`
- Registering a singleton service with `ProvideSingleton`
- Building the application with `app.Build()`
- Resolving a service with `gaz.Resolve[T]()`

## Run

```bash
go run .
```

## Expected Output

```
Hello, World!
```

## What's Next

- See [lifecycle](../lifecycle) for startup/shutdown hooks
- See [config-loading](../config-loading) for configuration management
