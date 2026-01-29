# gaz/worker

Background worker lifecycle management for Go.

## Installation

```bash
go get github.com/petabytecl/gaz/worker
```

## Quick Start

Implement the Worker interface:

```go
type MyWorker struct {
    stop chan struct{}
}

func (w *MyWorker) Name() string { return "my-worker" }

func (w *MyWorker) Start() {
    w.stop = make(chan struct{})
    go func() {
        for {
            select {
            case <-w.stop:
                return
            case <-time.After(time.Second):
                // Do periodic work
            }
        }
    }()
}

func (w *MyWorker) Stop() {
    close(w.stop)
}
```

## Worker Interface

The Worker interface defines three methods for lifecycle management:

- **Start()** - Begins the worker. Returns immediately; worker spawns its own goroutine.
- **Stop()** - Signals shutdown. Worker should exit gracefully.
- **Name()** - Returns a unique identifier for logging and debugging.

## Features

- **Supervised restart with backoff** - Workers are automatically restarted on panic
- **Circuit breaker for crash loops** - Prevents runaway restart loops
- **Integration with gaz.App lifecycle** - Workers start after Starter hooks, stop before Stopper hooks

## Registration Options

Workers can be registered with options:

```go
worker.WithPoolSize(4)        // Multiple worker instances
worker.WithCritical(true)     // App crashes if worker exhausts retries
worker.WithStableRunPeriod(5*time.Minute)  // Duration before backoff resets
worker.WithMaxRestarts(10)    // Max restarts before circuit trips
worker.WithCircuitWindow(time.Minute)      // Circuit breaker window
```

## Backoff Configuration

Workers use exponential backoff with jitter for restarts:

- **Min:** 1 second (first retry delay)
- **Max:** 5 minutes (cap on retry delay)
- **Factor:** 2 (exponential multiplier)
- **Jitter:** true (randomization to prevent thundering herd)

See [gaz framework](../README.md) for full documentation.
