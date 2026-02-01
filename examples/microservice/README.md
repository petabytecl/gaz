# Microservice Example

Demonstrates a complete microservice with gaz using health checks, workers, and event bus.

## What This Demonstrates

- Health check endpoints (/live, /ready) on port 9090
- Background workers for async processing
- Event bus for internal pub/sub messaging
- Full lifecycle management with graceful shutdown

## Run

```bash
go run .
```

Check health:
```bash
curl http://localhost:9090/ready
```

Stop with Ctrl+C.

## Expected Output

```
=================================================
Microservice starting...
Health check: http://localhost:9090/ready
Press Ctrl+C to stop
=================================================
[order-processor] starting
[order-simulator] starting
[notification-subscriber] starting
[order-simulator] creating order ORD-001
[order-processor] processing order ORD-001: $50.00 for customer CUST-002
[order-processor] order ORD-001 processed
[notification-subscriber] sending notification: order ORD-001 processed at 15:04:05
^C
[order-simulator] stopping...
[order-simulator] stopped
[notification-subscriber] stopping...
[notification-subscriber] stopped
[order-processor] stopping...
[order-processor] stopped
Shutdown complete
```

## Architecture

```
+------------------+     +-------------------+
|  OrderSimulator  |---->|  EventBus         |
|  (creates fake   |     |  (pub/sub)        |
|   orders)        |     +-------------------+
+------------------+            |
                                | OrderCreatedEvent
                                v
                     +-------------------+
                     |  OrderProcessor   |
                     |  (processes       |
                     |   orders)         |
                     +-------------------+
                                |
                                | OrderProcessedEvent
                                v
                     +-------------------+
                     | NotificationSub   |
                     | (sends alerts)    |
                     +-------------------+
```

## Key Patterns

1. **Health Module:** `health.NewModule()` provides readiness/liveness probes
2. **Event-Driven:** Workers communicate via typed events
3. **Lifecycle Integration:** Workers implement `worker.Worker` interface
4. **Dependency Injection:** EventBus is injected into workers

## Configuration

The example includes a `config.yaml` for demonstration:

```yaml
server:
  port: 8080

health:
  port: 9090
```

## What's Next

- See [background-workers](../background-workers) for worker patterns
- See [http-server](../http-server) for HTTP server with health checks
