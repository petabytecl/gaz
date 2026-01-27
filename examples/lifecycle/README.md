# Lifecycle Example

Demonstrates gaz lifecycle hooks for graceful startup and shutdown.

## What This Demonstrates

- Implementing `Starter` interface (`OnStart` method)
- Implementing `Stopper` interface (`OnStop` method)
- Graceful shutdown on Ctrl+C
- Service start/stop ordering

## Run

```bash
go run .
```

Then press Ctrl+C to trigger graceful shutdown.

## Expected Output

```
Server starting on port 8080
^C
Received shutdown signal
Server stopping...
Shutdown complete
```

## Lifecycle Interfaces

```go
// Starter is called during app.Run()
type Starter interface {
    OnStart(context.Context) error
}

// Stopper is called during shutdown
type Stopper interface {
    OnStop(context.Context) error
}
```

## Startup/Shutdown Ordering

When you have multiple services with dependencies:

- **Startup:** Services start in dependency order (dependencies first)
- **Shutdown:** Services stop in reverse order (dependents first)

This ensures a database connection starts before services that need it, and shuts down after those services have stopped.

## What's Next

- See [config-loading](../config-loading) for configuration management
- See [basic](../basic) for minimal DI example
