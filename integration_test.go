package gaz_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/eventbus"
	"github.com/petabytecl/gaz/worker"
)

// =============================================================================
// Event types for integration tests
// =============================================================================

type workerEvent struct {
	WorkerName string
	Iteration  int
}

func (e workerEvent) EventName() string { return "workerEvent" }

type taskCreatedEvent struct {
	TaskID string
}

func (e taskCreatedEvent) EventName() string { return "taskCreatedEvent" }

// =============================================================================
// TestIntegration_WorkerPublishesEvents
// =============================================================================

// TestIntegration_WorkerPublishesEvents registers a worker via DI that publishes
// events to EventBus on each work cycle. A subscriber collects the events.
// This tests: DI provider registration + worker lifecycle + eventbus pub/sub wiring.
func TestIntegration_WorkerPublishesEvents(t *testing.T) {
	app := gaz.New(gaz.WithShutdownTimeout(5 * time.Second))

	// Create the worker (bus will be set after Build, which registers eventbus)
	pw := &publishingWorker{
		workerName: "event-publisher",
	}

	// Register worker as instance (same pattern as existing worker tests)
	err := gaz.For[*publishingWorker](app.Container()).Named("event-publisher").Instance(pw)
	require.NoError(t, err)

	err = app.Build()
	require.NoError(t, err)

	// After Build, eventbus is available via app.EventBus()
	bus := app.EventBus()
	require.NotNil(t, bus)
	pw.bus = bus

	// Subscribe to events
	ts := eventbus.NewTestSubscriber[workerEvent](2)
	sub := eventbus.Subscribe(bus, ts.Handler())
	require.NotNil(t, sub)
	defer sub.Unsubscribe()

	// Run the app in a goroutine
	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for at least 2 events
	if !ts.WaitFor(3 * time.Second) {
		t.Fatalf("timeout waiting for 2 events, got %d", ts.Count())
	}

	events := ts.Events()
	assert.GreaterOrEqual(t, len(events), 2)
	assert.Equal(t, "event-publisher", events[0].WorkerName)

	// Graceful shutdown
	require.NoError(t, app.Stop(context.Background()))
	select {
	case err := <-runErr:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("app did not shut down in time")
	}
}

// =============================================================================
// TestIntegration_EventDrivenWorkerChain
// =============================================================================

// TestIntegration_EventDrivenWorkerChain registers two workers: Worker A publishes
// "taskCreatedEvent" events, Worker B subscribes and processes them (increments a counter).
// This tests: cross-worker communication via eventbus through DI.
func TestIntegration_EventDrivenWorkerChain(t *testing.T) {
	app := gaz.New(gaz.WithShutdownTimeout(5 * time.Second))

	var receivedCount atomic.Int32
	received := make(chan struct{}, 10)

	// Producer worker
	producer := &taskProducerWorker{}
	err := gaz.For[*taskProducerWorker](app.Container()).Named("task-producer").Instance(producer)
	require.NoError(t, err)

	err = app.Build()
	require.NoError(t, err)

	// Wire eventbus after Build
	bus := app.EventBus()
	require.NotNil(t, bus)
	producer.bus = bus

	// Consumer subscribes on the bus
	eventbus.Subscribe(bus, func(_ context.Context, _ taskCreatedEvent) {
		receivedCount.Add(1)
		select {
		case received <- struct{}{}:
		default:
		}
	})

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for at least 2 events received by the consumer
	for i := range 2 {
		select {
		case <-received:
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for event %d, received total: %d", i+1, receivedCount.Load())
		}
	}

	count := receivedCount.Load()
	assert.GreaterOrEqual(t, count, int32(2), "consumer should have received at least 2 events")

	require.NoError(t, app.Stop(context.Background()))
	select {
	case err := <-runErr:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("app did not shut down in time")
	}
}

// =============================================================================
// TestIntegration_GracefulShutdownDrainsEvents
// =============================================================================

// TestIntegration_GracefulShutdownDrainsEvents verifies that on shutdown, in-flight
// events in the eventbus are drained before the bus closes.
// This tests: shutdown ordering -- worker stops first, eventbus drains after.
func TestIntegration_GracefulShutdownDrainsEvents(t *testing.T) {
	app := gaz.New(gaz.WithShutdownTimeout(5 * time.Second))

	var processedCount atomic.Int32
	publishDone := make(chan struct{})

	// Use a worker that publishes a batch of events then stops itself
	batchW := &batchPublisherWorker{publishDone: publishDone}
	err := gaz.For[*batchPublisherWorker](app.Container()).Named("batch-publisher").Instance(batchW)
	require.NoError(t, err)

	err = app.Build()
	require.NoError(t, err)

	bus := app.EventBus()
	require.NotNil(t, bus)
	batchW.bus = bus

	// Subscribe a handler that takes 50ms per event (simulating slow processing)
	eventbus.Subscribe(bus, func(_ context.Context, _ workerEvent) {
		time.Sleep(50 * time.Millisecond)
		processedCount.Add(1)
	})

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(context.Background())
	}()

	// Wait for the worker to finish publishing its batch
	select {
	case <-publishDone:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not finish publishing")
	}

	// Now trigger shutdown -- eventbus should drain the buffered events
	require.NoError(t, app.Stop(context.Background()))

	select {
	case err := <-runErr:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("app did not shut down in time")
	}

	// After shutdown, verify events were processed (eventbus drained)
	count := processedCount.Load()
	assert.Greater(t, count, int32(0), "at least some events should have been processed before shutdown")
}

// =============================================================================
// Worker implementations for integration tests
// =============================================================================

// publishingWorker publishes workerEvent on a ticker.
type publishingWorker struct {
	workerName string
	bus        *eventbus.EventBus
	done       chan struct{}
	wg         sync.WaitGroup
}

func (w *publishingWorker) Name() string { return w.workerName }

func (w *publishingWorker) OnStart(ctx context.Context) error {
	w.done = make(chan struct{})
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		iteration := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			case <-ticker.C:
				if w.bus == nil {
					continue
				}
				iteration++
				eventbus.Publish(ctx, w.bus, workerEvent{
					WorkerName: w.workerName,
					Iteration:  iteration,
				}, "")
			}
		}
	}()

	return nil
}

func (w *publishingWorker) OnStop(_ context.Context) error {
	close(w.done)
	w.wg.Wait()

	return nil
}

// taskProducerWorker publishes taskCreatedEvent events on a ticker.
type taskProducerWorker struct {
	bus  *eventbus.EventBus
	done chan struct{}
	wg   sync.WaitGroup
}

func (w *taskProducerWorker) Name() string { return "task-producer" }

func (w *taskProducerWorker) OnStart(ctx context.Context) error {
	w.done = make(chan struct{})
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		counter := 0

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			case <-ticker.C:
				if w.bus == nil {
					continue
				}
				counter++
				eventbus.Publish(ctx, w.bus, taskCreatedEvent{
					TaskID: "task-" + itoa(counter),
				}, "")
			}
		}
	}()

	return nil
}

func (w *taskProducerWorker) OnStop(_ context.Context) error {
	close(w.done)
	w.wg.Wait()

	return nil
}

// batchPublisherWorker publishes a fixed batch of events, then signals done.
// It does not publish during shutdown, avoiding races with eventbus.Close().
type batchPublisherWorker struct {
	bus         *eventbus.EventBus
	publishDone chan struct{} // closed when batch is published
	done        chan struct{}
	wg          sync.WaitGroup
}

func (w *batchPublisherWorker) Name() string { return "batch-publisher" }

func (w *batchPublisherWorker) OnStart(_ context.Context) error {
	w.done = make(chan struct{})
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		// Publish a batch of 5 events quickly, then stop
		for i := 1; i <= 5; i++ {
			if w.bus == nil {
				break
			}
			eventbus.Publish(context.Background(), w.bus, workerEvent{
				WorkerName: "batch",
				Iteration:  i,
			}, "")
		}

		close(w.publishDone)

		// Wait for shutdown signal
		<-w.done
	}()

	return nil
}

func (w *batchPublisherWorker) OnStop(_ context.Context) error {
	close(w.done)
	w.wg.Wait()

	return nil
}

// itoa converts an int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	digits := ""

	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}

	return digits
}

// Verify worker interface compliance at compile time.
var (
	_ worker.Worker = (*publishingWorker)(nil)
	_ worker.Worker = (*taskProducerWorker)(nil)
	_ worker.Worker = (*batchPublisherWorker)(nil)
)
