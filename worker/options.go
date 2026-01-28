package worker

import "time"

// WorkerOptions holds configuration for worker registration.
type WorkerOptions struct {
	// PoolSize is the number of worker instances to create.
	// Each instance runs in its own goroutine with a unique name suffix.
	// Default: 1
	PoolSize int

	// Critical marks the worker as critical to application operation.
	// If a critical worker exhausts its restart attempts (circuit breaker trips),
	// the application will initiate shutdown.
	// Default: false
	Critical bool

	// StableRunPeriod is the duration a worker must run without panicking
	// before the backoff counter resets. This allows workers that recovered
	// successfully to restart quickly if they panic again later.
	// Default: 30 seconds
	StableRunPeriod time.Duration

	// MaxRestarts is the maximum number of restart attempts within
	// CircuitWindow before the circuit breaker trips and the worker
	// is considered failed.
	// Default: 5
	MaxRestarts int

	// CircuitWindow is the time window for tracking restart attempts.
	// If MaxRestarts is exceeded within this window, the circuit breaker
	// trips. The window resets after CircuitWindow duration passes.
	// Default: 10 minutes
	CircuitWindow time.Duration
}

// WorkerOption configures WorkerOptions.
type WorkerOption func(*WorkerOptions)

// DefaultWorkerOptions returns WorkerOptions with sensible defaults.
//
// Default values:
//   - PoolSize: 1
//   - Critical: false
//   - StableRunPeriod: 30 seconds
//   - MaxRestarts: 5
//   - CircuitWindow: 10 minutes
func DefaultWorkerOptions() *WorkerOptions {
	return &WorkerOptions{
		PoolSize:        1,
		Critical:        false,
		StableRunPeriod: 30 * time.Second,
		MaxRestarts:     5,
		CircuitWindow:   10 * time.Minute,
	}
}

// ApplyOptions applies the given options to the WorkerOptions.
func (o *WorkerOptions) ApplyOptions(opts ...WorkerOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithPoolSize sets the number of worker instances to create.
// Each instance runs in its own goroutine with a name suffix (e.g., "worker-1", "worker-2").
// Pool workers are useful for parallel processing of work queues.
//
// Example:
//
//	manager.Register(worker, WithPoolSize(4)) // Creates 4 instances
func WithPoolSize(n int) WorkerOption {
	return func(o *WorkerOptions) {
		if n > 0 {
			o.PoolSize = n
		}
	}
}

// WithCritical marks the worker as critical to application operation.
// If a critical worker exhausts its restart attempts, the application
// will initiate shutdown rather than continuing in a degraded state.
//
// Use this for workers that are essential to core functionality.
//
// Example:
//
//	manager.Register(paymentProcessor, WithCritical())
func WithCritical() WorkerOption {
	return func(o *WorkerOptions) {
		o.Critical = true
	}
}

// WithStableRunPeriod sets the duration a worker must run without panicking
// before the backoff counter resets.
//
// A longer stable period means the worker must run longer before being
// considered "recovered" and eligible for quick restarts again.
//
// Example:
//
//	manager.Register(worker, WithStableRunPeriod(time.Minute))
func WithStableRunPeriod(d time.Duration) WorkerOption {
	return func(o *WorkerOptions) {
		if d > 0 {
			o.StableRunPeriod = d
		}
	}
}

// WithMaxRestarts sets the maximum number of restart attempts within
// the circuit window before the worker is considered failed.
//
// Example:
//
//	manager.Register(worker, WithMaxRestarts(3)) // Allow 3 restarts
func WithMaxRestarts(n int) WorkerOption {
	return func(o *WorkerOptions) {
		if n > 0 {
			o.MaxRestarts = n
		}
	}
}

// WithCircuitWindow sets the time window for tracking restart attempts.
// The restart counter resets after this duration passes.
//
// Example:
//
//	manager.Register(worker, WithCircuitWindow(5*time.Minute))
func WithCircuitWindow(d time.Duration) WorkerOption {
	return func(o *WorkerOptions) {
		if d > 0 {
			o.CircuitWindow = d
		}
	}
}
