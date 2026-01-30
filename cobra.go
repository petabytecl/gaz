package gaz

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/petabytecl/gaz/di"
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
//	app := gaz.New()
//	gaz.For[*Database](app.Container()).Provider(NewDatabase)
//	app.WithCobra(rootCmd)
//
//	// In subcommand:
//	app := gaz.FromContext(cmd.Context())
//	db, _ := gaz.Resolve[*Database](app.Container())
func (a *App) WithCobra(cmd *cobra.Command) *App {
	// Store command reference for module flags integration
	a.cobraCmd = cmd

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

		// Get context from command (Cobra provides background if none)
		ctx := c.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		if err := a.bootstrap(ctx, c, args); err != nil {
			return err
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

func (a *App) bootstrap(ctx context.Context, cmd *cobra.Command, args []string) error {
	// Register CommandArgs
	_ = For[*CommandArgs](a.container).Instance(&CommandArgs{
		Command: cmd,
		Args:    args,
	})

	// Bind flags if ConfigManager is available
	if a.configMgr != nil {
		if err := a.configMgr.BindFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("failed to bind flags: %w", err)
		}
	}

	// Build the app (validates registrations)
	if err := a.Build(); err != nil {
		return fmt.Errorf("app build failed: %w", err)
	}

	// Start lifecycle hooks
	if err := a.Start(ctx); err != nil {
		return fmt.Errorf("app start failed: %w", err)
	}

	return nil
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
	graph := a.container.GetGraph()
	services := make(map[string]di.ServiceWrapper)
	a.container.ForEachService(func(name string, svc di.ServiceWrapper) {
		services[name] = svc
	})

	startupOrder, err := ComputeStartupOrder(graph, services)
	if err != nil {
		return err
	}

	// Start services layer by layer
	for _, layer := range startupOrder {
		for _, name := range layer {
			svc := services[name]
			if startErr := svc.Start(ctx); startErr != nil {
				return fmt.Errorf("starting service %s: %w", name, startErr)
			}
		}
	}

	return nil
}
