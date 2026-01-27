# HTTP Server Example

This example demonstrates building an HTTP server with gaz, featuring:

- **HTTP Server with Lifecycle Hooks**: Server starts in a goroutine and shuts down gracefully
- **Graceful Shutdown**: Uses `http.Server.Shutdown()` for connection draining
- **Health Checks**: Integrated health module with readiness/liveness probes
- **Dependency Injection**: Server, Handler, and Config wired through gaz

## What It Demonstrates

1. **Lifecycle Management**: The `Server` type has `OnStart` and `OnStop` methods registered as hooks
2. **Graceful Shutdown**: On SIGINT/SIGTERM, the server waits for active connections to complete
3. **Health Endpoints**: Health module provides `/ready`, `/live`, `/startup` on port 9090
4. **Service Registration**: Shows both `ProvideSingleton` (app-level) and `For[T]` (container-level) APIs

## Running

```bash
cd examples/http-server
go run .
```

## Testing

```bash
# Main endpoint
curl http://localhost:8080/
# {"service":"http-server-example","status":"running"}

# Hello endpoint
curl http://localhost:8080/hello
# {"message":"Hello, World!"}

curl http://localhost:8080/hello?name=Gaz
# {"message":"Hello, Gaz!"}

# Health endpoints (port 9090)
curl http://localhost:9090/ready
curl http://localhost:9090/live
curl http://localhost:9090/startup
```

## Graceful Shutdown

Press `Ctrl+C` to trigger graceful shutdown:

```
HTTP server listening on :8080
^C
Shutting down gracefully... (hint: Ctrl+C again to force)
Shutting down HTTP server...
```

The server:
1. Stops accepting new connections
2. Waits for in-flight requests to complete (up to 30s)
3. Calls all `OnStop` hooks in reverse dependency order
4. Exits cleanly

Press `Ctrl+C` twice for immediate force exit.

## Key Patterns

### Server Lifecycle

```go
func (s *Server) OnStart(_ context.Context) error {
    go func() {
        if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Printf("HTTP server error: %v", err)
        }
    }()
    return nil // Return immediately, server runs in background
}

func (s *Server) OnStop(ctx context.Context) error {
    return s.httpServer.Shutdown(ctx) // Uses context deadline for graceful drain
}
```

### Health Integration

```go
app := gaz.New(
    gaz.WithShutdownTimeout(30*time.Second),
    health.WithHealthChecks(health.DefaultConfig()),
)
```

This registers the health module which:
- Starts a management server on port 9090
- Provides `/ready`, `/live`, `/startup` endpoints
- Automatically marks shutting down when stop begins
