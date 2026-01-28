package worker

import "errors"

// Sentinel errors for worker package.
var (
	// ErrCircuitBreakerTripped indicates a worker exhausted its restart attempts
	// within the configured circuit window. The worker will not be restarted
	// until the application is restarted.
	ErrCircuitBreakerTripped = errors.New("worker: circuit breaker tripped, max restarts exceeded")

	// ErrWorkerStopped indicates a worker stopped normally without error.
	// This is not an error condition - it signals clean shutdown.
	ErrWorkerStopped = errors.New("worker: stopped normally")

	// ErrCriticalWorkerFailed indicates a critical worker failed and exhausted
	// its restart attempts. This error triggers application shutdown.
	ErrCriticalWorkerFailed = errors.New("worker: critical worker failed, initiating shutdown")

	// ErrManagerAlreadyRunning indicates an attempt to register a worker
	// after the manager has started.
	ErrManagerAlreadyRunning = errors.New("worker: cannot register worker after manager has started")
)
