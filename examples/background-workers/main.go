// Package main demonstrates background worker patterns with gaz.
//
// This example shows:
//   - Implementing the worker.Worker interface (OnStart/OnStop with context)
//   - Registering workers as DI providers with Eager() for auto-start
//   - Multiple workers running concurrently
//   - Graceful shutdown with context cancellation
//
// Run with: go run .
// Stop with: Ctrl+C (SIGINT)
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/worker"
	workermod "github.com/petabytecl/gaz/worker/module"
)

// EmailWorker processes an email queue in the background.
// It implements worker.Worker interface for lifecycle integration.
type EmailWorker struct {
	name string
	done chan struct{}
	wg   sync.WaitGroup
}

// NewEmailWorker creates a new email worker.
func NewEmailWorker() *EmailWorker {
	return &EmailWorker{name: "email-worker"}
}

// Name returns the worker's unique identifier (used for logging/debugging).
func (w *EmailWorker) Name() string { return w.name }

// OnStart begins the worker's background processing.
// This method must be non-blocking - the worker spawns its own goroutine.
func (w *EmailWorker) OnStart(ctx context.Context) error {
	fmt.Printf("[%s] starting\n", w.name)

	w.done = make(chan struct{})
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("[%s] context cancelled, stopping\n", w.name)
				return
			case <-w.done:
				fmt.Printf("[%s] received stop signal\n", w.name)
				return
			case <-ticker.C:
				fmt.Printf("[%s] processing emails...\n", w.name)
			}
		}
	}()

	return nil
}

// OnStop signals the worker to shut down gracefully.
// It waits for the background goroutine to exit.
func (w *EmailWorker) OnStop(ctx context.Context) error {
	fmt.Printf("[%s] stopping...\n", w.name)
	close(w.done)
	w.wg.Wait()
	fmt.Printf("[%s] stopped\n", w.name)
	return nil
}

// NotificationWorker sends push notifications in the background.
type NotificationWorker struct {
	name string
	done chan struct{}
	wg   sync.WaitGroup
}

// NewNotificationWorker creates a new notification worker.
func NewNotificationWorker() *NotificationWorker {
	return &NotificationWorker{name: "notification-worker"}
}

func (w *NotificationWorker) Name() string { return w.name }

func (w *NotificationWorker) OnStart(ctx context.Context) error {
	fmt.Printf("[%s] starting\n", w.name)

	w.done = make(chan struct{})
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Printf("[%s] context cancelled, stopping\n", w.name)
				return
			case <-w.done:
				fmt.Printf("[%s] received stop signal\n", w.name)
				return
			case <-ticker.C:
				fmt.Printf("[%s] sending push notifications...\n", w.name)
			}
		}
	}()

	return nil
}

func (w *NotificationWorker) OnStop(ctx context.Context) error {
	fmt.Printf("[%s] stopping...\n", w.name)
	close(w.done)
	w.wg.Wait()
	fmt.Printf("[%s] stopped\n", w.name)
	return nil
}

// Compile-time interface checks.
var (
	_ worker.Worker = (*EmailWorker)(nil)
	_ worker.Worker = (*NotificationWorker)(nil)
)

func run(ctx context.Context) error {
	app := gaz.New()

	// Register worker module (provides worker.Manager)
	app.Use(workermod.New())

	// Register EmailWorker as an eager singleton.
	// Eager() ensures OnStart() is called during app.Run().
	// The worker.Worker interface is auto-detected by gaz.
	if err := gaz.For[*EmailWorker](app.Container()).
		Eager().
		Provider(func(c *gaz.Container) (*EmailWorker, error) {
			return NewEmailWorker(), nil
		}); err != nil {
		return fmt.Errorf("failed to register email worker: %w", err)
	}

	// Register NotificationWorker similarly.
	if err := gaz.For[*NotificationWorker](app.Container()).
		Eager().
		Provider(func(c *gaz.Container) (*NotificationWorker, error) {
			return NewNotificationWorker(), nil
		}); err != nil {
		return fmt.Errorf("failed to register notification worker: %w", err)
	}

	// Build the application (validates and prepares services)
	if err := app.Build(); err != nil {
		return fmt.Errorf("failed to build app: %w", err)
	}

	fmt.Println("Starting workers (Ctrl+C to stop)...")

	// Run blocks until shutdown signal (SIGINT/SIGTERM) or context cancellation.
	// During Run:
	// 1. Eager services are instantiated and OnStart() is called
	// 2. App waits for shutdown signal
	// 3. OnStop() is called for all services (reverse order)
	if err := app.Run(ctx); err != nil {
		return fmt.Errorf("application error: %w", err)
	}

	return nil
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Shutdown complete")
}
