// Package worker provides background worker lifecycle management for Go applications.
//
// This package defines the [Worker] interface and supporting types for managing
// long-running background tasks. Workers integrate with gaz's lifecycle system
// for automatic startup and graceful shutdown.
//
// # Worker Interface
//
// The [Worker] interface defines three methods for lifecycle management:
//
//   - OnStart(ctx) - Begins the worker. Returns immediately; worker spawns its own goroutine.
//     The context provides cancellation signals. Returns error if startup fails.
//   - OnStop(ctx) - Signals shutdown. Worker should exit gracefully. The context provides
//     a shutdown deadline. Returns error to log issues (shutdown continues regardless).
//   - Name() - Returns a unique identifier for logging and debugging.
//
// The OnStart/OnStop signature aligns with [di.Starter] and [di.Stopper] interfaces,
// enabling consistent lifecycle management across all gaz services.
//
// # Implementing a Worker
//
// Workers are responsible for their own goroutine management. OnStart() must be
// non-blocking; the worker should spawn its own goroutine for long-running work.
// OnStop() signals the worker to shut down; the worker decides when to return.
//
// Example of a simple worker:
//
//	type MyWorker struct {
//	    stop chan struct{}
//	}
//
//	func (w *MyWorker) Name() string { return "my-worker" }
//
//	func (w *MyWorker) OnStart(ctx context.Context) error {
//	    w.stop = make(chan struct{})
//	    go func() {
//	        for {
//	            select {
//	            case <-ctx.Done():
//	                return
//	            case <-w.stop:
//	                return
//	            case <-time.After(time.Second):
//	                // Do work
//	            }
//	        }
//	    }()
//	    return nil
//	}
//
//	func (w *MyWorker) OnStop(ctx context.Context) error {
//	    close(w.stop)
//	    return nil
//	}
//
// # Registration Options
//
// Workers can be registered with options via [WorkerOptions]:
//
//   - [WithPoolSize] - Creates multiple instances of a worker for parallel processing
//   - [WithCritical] - Marks a worker as critical; crashes the app if it exhausts retries
//   - [WithStableRunPeriod] - Duration of stable run before backoff resets
//   - [WithMaxRestarts] - Maximum restarts before circuit breaker trips
//   - [WithCircuitWindow] - Time window for circuit breaker tracking
//
// # Panic Recovery and Restart
//
// Workers are supervised by a WorkerManager (see manager.go in future plans).
// If a worker panics, it is recovered, logged, and restarted with exponential
// backoff. The [BackoffConfig] controls restart delays.
//
// # Backoff Configuration
//
// The [BackoffConfig] wraps the internal [backoff.ExponentialBackOff] with sensible defaults:
//
//   - Min: 1 second (first retry delay)
//   - Max: 5 minutes (cap on retry delay)
//   - Factor: 2 (exponential multiplier)
//   - Jitter: true (randomization to prevent thundering herd)
package worker
