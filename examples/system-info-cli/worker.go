// Package main provides a background worker for periodic system info collection.
//
// This file implements the Worker interface for continuous monitoring mode.
// The RefreshWorker collects and displays system information at configured intervals.
package main

import (
	"context"
	"sync"
	"time"
)

// RefreshWorker implements the Worker interface for periodic system info refresh.
// It collects system information at regular intervals and displays it.
type RefreshWorker struct {
	name      string
	interval  time.Duration
	format    string
	collector *Collector
	done      chan struct{}
	wg        sync.WaitGroup
}

// NewRefreshWorker creates a new RefreshWorker.
//
// Parameters:
//   - name: Worker name for logging and identification
//   - interval: Time between data collection cycles
//   - format: Output format ("text" or "json")
//   - collector: Collector service for gathering system info
func NewRefreshWorker(name string, interval time.Duration, format string, collector *Collector) *RefreshWorker {
	return &RefreshWorker{
		name:      name,
		interval:  interval,
		format:    format,
		collector: collector,
	}
}

// Name returns the worker's name for logging and debugging.
func (w *RefreshWorker) Name() string {
	return w.name
}

// OnStart begins the worker's background processing.
//
// This method is non-blocking. It spawns a goroutine that:
//   - Performs initial collection and display on start
//   - Creates a ticker for periodic collection
//   - Collects and displays system info on each tick
//   - Exits gracefully when done channel is closed
//
// The context can be used for cancellation signals.
// Returns nil as worker startup doesn't fail.
func (w *RefreshWorker) OnStart(ctx context.Context) error {
	w.done = make(chan struct{})
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		// Initial collection on start
		w.collectAndDisplay()

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.done:
				return
			case <-ticker.C:
				w.collectAndDisplay()
			}
		}
	}()
	return nil
}

// OnStop signals the worker to shut down and waits for completion.
//
// This method blocks until the background goroutine has fully stopped.
// It closes the done channel to signal shutdown and waits on the WaitGroup.
// The context provides a deadline for shutdown (not currently used).
// Returns nil as worker stop doesn't fail.
func (w *RefreshWorker) OnStop(ctx context.Context) error {
	close(w.done)
	w.wg.Wait()
	return nil
}

// collectAndDisplay gathers and outputs system information.
func (w *RefreshWorker) collectAndDisplay() {
	info, err := w.collector.Collect()
	if err != nil {
		// Log error but continue - don't crash the worker
		return
	}
	_ = w.collector.Display(info)
}
