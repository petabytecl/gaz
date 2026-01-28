# Basic Example

Minimal working gaz application demonstrating core DI concepts.

## What This Demonstrates

- Creating a gaz application with `gaz.New()`
- Registering a singleton service with `gaz.For[T]().Provider()`
- Building the application with `app.Build()`
- Resolving a service with `gaz.Resolve[T]()`

## Key Pattern

```go
// Register a singleton provider using the type-safe For[T]() API
err := gaz.For[*Greeter](app.Container()).Provider(func(c *gaz.Container) (*Greeter, error) {
    return &Greeter{Name: "World"}, nil
})
if err != nil {
    log.Fatal(err)
}
```

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
