// Package main demonstrates gaz lifecycle hooks (OnStart/OnStop).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/petabytecl/gaz"
)

// Server implements Starter and Stopper interfaces for lifecycle management.
// When registered with gaz, OnStart is called during app.Run() and
// OnStop is called when the application shuts down.
type Server struct {
	port int
}

// OnStart is called when the application starts.
// Use this for initialization: opening connections, starting listeners, etc.
func (s *Server) OnStart(ctx context.Context) error {
	fmt.Printf("Server starting on port %d\n", s.port)
	// In a real app, you would start listening here:
	// listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	return nil
}

// OnStop is called when the application shuts down.
// Use this for cleanup: closing connections, flushing buffers, etc.
func (s *Server) OnStop(ctx context.Context) error {
	fmt.Println("Server stopping...")
	// In a real app, you would close listeners and connections here
	return nil
}

func main() {
	app := gaz.New()

	// Register the server as a singleton.
	// gaz automatically detects that Server implements Starter and Stopper
	// and will call OnStart during Run() and OnStop during shutdown.
	app.ProvideSingleton(func(c *gaz.Container) (*Server, error) {
		return &Server{port: 8080}, nil
	})

	if err := app.Build(); err != nil {
		log.Fatal(err)
	}

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived shutdown signal")
		cancel()
	}()

	// Run blocks until context is cancelled or shutdown signal received.
	// During Run():
	// 1. OnStart is called for all services implementing Starter
	// 2. App waits for shutdown signal
	// 3. OnStop is called for all services implementing Stopper (reverse order)
	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Shutdown complete")
}
