package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Manager coordinates multiple workers, providing registration, startup,
// and graceful shutdown. It wraps each worker in a supervisor that handles
// panic recovery and restart logic.
//
// Workers are registered before Start() is called. After Start(), new
// registrations are rejected. All workers start concurrently when Start()
// is called and stop concurrently when Stop() is called.
//
// Example:
//
//	mgr := worker.NewManager(logger)
//	mgr.Register(myWorker)
//	mgr.Register(queueProcessor, worker.WithPoolSize(4), worker.WithCritical())
//
//	if err := mgr.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Later, during shutdown:
//	mgr.Stop()
type Manager struct {
	logger      *slog.Logger
	supervisors []*supervisor

	mu      sync.Mutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	done    chan struct{}

	// Callback for critical worker failure (signals app shutdown)
	onCriticalFail func()
}

// NewManager creates a new worker manager with the given logger.
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		logger:      logger.With(slog.String("component", "worker.Manager")),
		supervisors: make([]*supervisor, 0),
		done:        make(chan struct{}),
	}
}

// SetCriticalFailHandler sets the callback invoked when a critical worker's
// circuit breaker trips. This is typically used by App to trigger graceful
// shutdown when an essential worker fails.
func (m *Manager) SetCriticalFailHandler(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onCriticalFail = fn
}

// Register adds a worker to the manager with the given options.
// For pool workers (WithPoolSize > 1), multiple supervisors are created
// with indexed names (e.g., "worker-1", "worker-2").
//
// Register must be called before Start(). Calling Register after Start()
// returns ErrManagerAlreadyRunning.
func (m *Manager) Register(w Worker, opts ...WorkerOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return ErrManagerAlreadyRunning
	}

	// Apply options to defaults
	options := DefaultWorkerOptions()
	options.ApplyOptions(opts...)

	// Create supervisors (multiple for pool workers)
	if options.PoolSize > 1 {
		for i := 1; i <= options.PoolSize; i++ {
			poolWorker := &pooledWorker{
				delegate: w,
				name:     fmt.Sprintf("%s-%d", w.Name(), i),
			}
			sup := newSupervisor(poolWorker, options, m.logger, m.handleCriticalFail)
			m.supervisors = append(m.supervisors, sup)
		}
	} else {
		sup := newSupervisor(w, options, m.logger, m.handleCriticalFail)
		m.supervisors = append(m.supervisors, sup)
	}

	m.logger.Debug("worker registered",
		slog.String("worker", w.Name()),
		slog.Int("pool_size", options.PoolSize),
		slog.Bool("critical", options.Critical),
	)

	return nil
}

// Start begins all registered workers concurrently.
// It returns immediately after spawning supervisor goroutines.
// The context controls the lifetime of all workers.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil // Already running, idempotent
	}

	m.running = true
	m.ctx, m.cancel = context.WithCancel(ctx)

	m.logger.Info("starting workers", slog.Int("count", len(m.supervisors)))

	// Start all supervisors concurrently
	for _, sup := range m.supervisors {
		m.wg.Add(1)
		go func(s *supervisor) {
			defer m.wg.Done()
			s.start(m.ctx)
			// Wait for supervisor to fully stop
			<-s.wait()
		}(sup)
	}

	// Watch for all workers to complete and close done channel
	go func() {
		m.wg.Wait()
		close(m.done)
	}()

	return nil
}

// Stop signals all workers to stop and waits for them to complete.
// It cancels the context and waits for all supervisor goroutines to exit.
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil // Not running
	}
	m.running = false
	m.mu.Unlock()

	m.logger.Info("stopping workers", slog.Int("count", len(m.supervisors)))

	// Cancel context to signal all supervisors
	if m.cancel != nil {
		m.cancel()
	}

	// Wait for all supervisors to complete
	m.wg.Wait()

	m.logger.Info("all workers stopped")
	return nil
}

// Done returns a channel that closes when all workers have stopped.
// This is useful for external shutdown verification.
func (m *Manager) Done() <-chan struct{} {
	return m.done
}

// handleCriticalFail is called by supervisors when a critical worker's
// circuit breaker trips.
func (m *Manager) handleCriticalFail() {
	m.mu.Lock()
	fn := m.onCriticalFail
	m.mu.Unlock()

	if fn != nil {
		fn()
	}
}

// pooledWorker wraps a worker with a custom name for pool instances.
type pooledWorker struct {
	delegate Worker
	name     string
}

func (p *pooledWorker) Start()       { p.delegate.Start() }
func (p *pooledWorker) Stop()        { p.delegate.Stop() }
func (p *pooledWorker) Name() string { return p.name }
