package worker

import "context"

// Worker defines the interface for background workers with lifecycle management.
//
// Workers are long-running background tasks that integrate with gaz's lifecycle
// system. They auto-start with app.Run() and auto-stop on shutdown.
//
// The Worker interface aligns with di.Starter/di.Stopper patterns, enabling
// consistent lifecycle management across all gaz services.
//
// # Contract
//
// Implementations must follow these rules:
//
//   - OnStart(ctx) must be non-blocking. The worker should spawn its own goroutine
//     internally for any long-running work. OnStart should return immediately
//     after initiating the worker's background processing. The context provides
//     cancellation signals that the worker should respect. Return an error if
//     startup fails (this prevents the worker from running).
//
//   - OnStop(ctx) signals the worker to shut down. The worker should exit gracefully,
//     completing or aborting any in-progress work. OnStop may block until the
//     worker has fully stopped, or it may return immediately if the worker uses
//     a channel-based shutdown signal. The context provides a deadline for shutdown.
//     Return an error to log shutdown issues (shutdown continues regardless).
//
//   - Name() must return a non-empty, unique string identifier. This name is used
//     for logging, debugging, and pool worker naming (e.g., "queue-processor-1").
//
// # Example
//
//	type Poller struct {
//	    interval time.Duration
//	    done     chan struct{}
//	    wg       sync.WaitGroup
//	}
//
//	func (p *Poller) Name() string { return "poller" }
//
//	func (p *Poller) OnStart(ctx context.Context) error {
//	    p.done = make(chan struct{})
//	    p.wg.Add(1)
//	    go func() {
//	        defer p.wg.Done()
//	        ticker := time.NewTicker(p.interval)
//	        defer ticker.Stop()
//	        for {
//	            select {
//	            case <-ctx.Done():
//	                return
//	            case <-p.done:
//	                return
//	            case <-ticker.C:
//	                // Poll for work
//	            }
//	        }
//	    }()
//	    return nil
//	}
//
//	func (p *Poller) OnStop(ctx context.Context) error {
//	    close(p.done)
//	    p.wg.Wait() // Wait for goroutine to exit
//	    return nil
//	}
type Worker interface {
	// OnStart begins the worker's background processing.
	//
	// This method must be non-blocking. The worker should spawn its own
	// goroutine internally for long-running work. The method should return
	// immediately after initiating the worker.
	//
	// The context provides cancellation signals that the worker should
	// respect for graceful shutdown.
	//
	// OnStart may be called multiple times if the worker is restarted after
	// a panic. Implementations should handle this gracefully.
	//
	// Return an error if startup fails. A startup error prevents the worker
	// from running and triggers the restart logic.
	OnStart(ctx context.Context) error

	// OnStop signals the worker to shut down.
	//
	// The worker should exit gracefully, completing or aborting any in-progress
	// work. This method may block until shutdown is complete, or return
	// immediately if using a channel-based signal.
	//
	// The context provides a deadline for shutdown. Workers should respect
	// this deadline and abort cleanup if the context is cancelled.
	//
	// OnStop is called during application shutdown and when the worker panics
	// (before restart). Implementations should be idempotent.
	//
	// Return an error to log shutdown issues. Errors are logged but shutdown
	// continues regardless (stop errors are non-fatal).
	OnStop(ctx context.Context) error

	// Name returns a unique identifier for this worker.
	//
	// The name is used for logging, debugging, and pool worker naming.
	// It must return a non-empty string. For pool workers, the manager
	// appends an index suffix (e.g., "queue-processor-1", "queue-processor-2").
	Name() string
}
