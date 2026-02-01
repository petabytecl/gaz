# Background Workers Example

Demonstrates background worker patterns with gaz lifecycle integration.

## What This Demonstrates

- Implementing the `worker.Worker` interface (OnStart/OnStop with context)
- Registering workers as DI providers with `Eager()` for auto-start
- Multiple workers running concurrently
- Graceful shutdown with context cancellation

## Run

```bash
go run .
```

Then press Ctrl+C to trigger graceful shutdown.

## Expected Output

```
Starting workers (Ctrl+C to stop)...
[email-worker] starting
[notification-worker] starting
[email-worker] processing emails...
[notification-worker] sending push notifications...
[email-worker] processing emails...
^C
[email-worker] stopping...
[email-worker] received stop signal
[email-worker] stopped
[notification-worker] stopping...
[notification-worker] received stop signal
[notification-worker] stopped
Shutdown complete
```

## Worker Interface

```go
type Worker interface {
    // OnStart begins background processing (must be non-blocking)
    OnStart(ctx context.Context) error
    
    // OnStop signals graceful shutdown
    OnStop(ctx context.Context) error
    
    // Name returns unique identifier for logging
    Name() string
}
```

## Key Patterns

1. **Non-blocking OnStart:** Spawn goroutines internally, return immediately
2. **Graceful OnStop:** Signal worker to stop, wait for goroutine to exit
3. **Context Cancellation:** Respect `ctx.Done()` for shutdown signals
4. **Eager Registration:** Use `Eager()` to auto-start workers during `app.Run()`

## What's Next

- See [lifecycle](../lifecycle) for basic lifecycle hooks
- See [http-server](../http-server) for HTTP server with health checks
