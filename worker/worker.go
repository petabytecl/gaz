package worker

// Worker defines the interface for background workers with lifecycle management.
//
// Workers are long-running background tasks that integrate with gaz's lifecycle
// system. They auto-start with app.Run() and auto-stop on shutdown.
//
// # Contract
//
// Implementations must follow these rules:
//
//   - Start() must be non-blocking. The worker should spawn its own goroutine
//     internally for any long-running work. Start() should return immediately
//     after initiating the worker's background processing.
//
//   - Stop() signals the worker to shut down. The worker should exit gracefully,
//     completing or aborting any in-progress work. Stop() may block until the
//     worker has fully stopped, or it may return immediately if the worker uses
//     a channel-based shutdown signal.
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
//	func (p *Poller) Start() {
//	    p.done = make(chan struct{})
//	    p.wg.Add(1)
//	    go func() {
//	        defer p.wg.Done()
//	        ticker := time.NewTicker(p.interval)
//	        defer ticker.Stop()
//	        for {
//	            select {
//	            case <-p.done:
//	                return
//	            case <-ticker.C:
//	                // Poll for work
//	            }
//	        }
//	    }()
//	}
//
//	func (p *Poller) Stop() {
//	    close(p.done)
//	    p.wg.Wait() // Wait for goroutine to exit
//	}
type Worker interface {
	// Start begins the worker's background processing.
	//
	// This method must be non-blocking. The worker should spawn its own
	// goroutine internally for long-running work. The method should return
	// immediately after initiating the worker.
	//
	// Start may be called multiple times if the worker is restarted after
	// a panic. Implementations should handle this gracefully.
	Start()

	// Stop signals the worker to shut down.
	//
	// The worker should exit gracefully, completing or aborting any in-progress
	// work. This method may block until shutdown is complete, or return
	// immediately if using a channel-based signal.
	//
	// Stop is called during application shutdown and when the worker panics
	// (before restart). Implementations should be idempotent.
	Stop()

	// Name returns a unique identifier for this worker.
	//
	// The name is used for logging, debugging, and pool worker naming.
	// It must return a non-empty string. For pool workers, the manager
	// appends an index suffix (e.g., "queue-processor-1", "queue-processor-2").
	Name() string
}
