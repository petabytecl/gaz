package gaz

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// contextKey is used to store App in context.
type contextKey struct{}

// FromContext retrieves the App from a context.
// Returns nil if no App is found.
// Use this in Cobra command handlers to access the DI container.
//
// Example:
//
//	var serveCmd = &cobra.Command{
//	    Use: "serve",
//	    RunE: func(cmd *cobra.Command, args []string) error {
//	        app := gaz.FromContext(cmd.Context())
//	        server, err := gaz.Resolve[*HTTPServer](app.Container())
//	        if err != nil {
//	            return err
//	        }
//	        return server.ListenAndServe()
//	    },
//	}
func FromContext(ctx context.Context) *App {
	if app, ok := ctx.Value(contextKey{}).(*App); ok {
		return app
	}
	return nil
}

// WithCobra attaches the App lifecycle to a Cobra command.
// This hooks into PersistentPreRunE to Build() and Start() the app,
// and into PersistentPostRunE to Stop() the app.
//
// The App is stored in the command's context, accessible via FromContext().
//
// Existing hooks on the command are preserved and chained (not replaced).
//
// Example:
//
//	rootCmd := &cobra.Command{Use: "myapp"}
//	app := gaz.New().
//	    ProvideSingleton(NewDatabase).
//	    WithCobra(rootCmd)
//
//	// In subcommand:
//	app := gaz.FromContext(cmd.Context())
//	db, _ := gaz.Resolve[*Database](app.Container())
func (a *App) WithCobra(cmd *cobra.Command) *App {
	// Preserve existing hooks
	originalPreRunE := cmd.PersistentPreRunE
	originalPostRunE := cmd.PersistentPostRunE

	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		// Chain original hook first
		if originalPreRunE != nil {
			if err := originalPreRunE(c, args); err != nil {
				return err
			}
		}

		// Bind flags if ConfigManager is available
		if a.configManager != nil {
			if err := a.configManager.BindFlags(c.Flags()); err != nil {
				return fmt.Errorf("failed to bind flags: %w", err)
			}
		}

		// Build the app (validates registrations)
		if err := a.Build(); err != nil {
			return fmt.Errorf("app build failed: %w", err)
		}

		// Get context from command (Cobra provides background if none)
		ctx := c.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// Start lifecycle hooks
		if err := a.Start(ctx); err != nil {
			return fmt.Errorf("app start failed: %w", err)
		}

		// Store app in context for subcommand access
		c.SetContext(context.WithValue(ctx, contextKey{}, a))

		return nil
	}

	cmd.PersistentPostRunE = func(c *cobra.Command, args []string) error {
		// Stop with configured timeout
		stopCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()

		stopErr := a.Stop(stopCtx)

		// Chain original hook
		if originalPostRunE != nil {
			if err := originalPostRunE(c, args); err != nil {
				return errors.Join(stopErr, err)
			}
		}

		return stopErr
	}

	return a
}

// Start initiates the application lifecycle.
// This is called automatically by WithCobra() or can be called manually.
// It executes OnStart hooks for all services in dependency order.
func (a *App) Start(ctx context.Context) error {
	// Ensure Build() was called first
	a.mu.Lock()
	if !a.built {
		a.mu.Unlock()
		if err := a.Build(); err != nil {
			return err
		}
		a.mu.Lock()
	}
	a.mu.Unlock()

	// Compute startup order
	graph := a.container.getGraph()
	services := make(map[string]serviceWrapper)
	a.container.mu.RLock()
	for k, v := range a.container.services {
		if w, ok := v.(serviceWrapper); ok {
			services[k] = w
		}
	}
	a.container.mu.RUnlock()

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		return err
	}

	// Start services layer by layer
	for _, layer := range startupOrder {
		for _, name := range layer {
			svc := services[name]
			if startErr := svc.start(ctx); startErr != nil {
				return fmt.Errorf("starting service %s: %w", name, startErr)
			}
		}
	}

	return nil
}
