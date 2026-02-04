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
// This is an Option passed to gaz.New() that hooks into:
// - PersistentPreRunE: applies stored flags, Build() and Start() the app
// - PersistentPostRunE: Stop() the app
//
// The App is stored in the command's context, accessible via FromContext().
//
// Existing hooks on the command are preserved and chained (not replaced).
//
// Example:
//
//	rootCmd := &cobra.Command{Use: "myapp"}
//	app := gaz.New(gaz.WithCobra(rootCmd))
//	gaz.For[*Database](app.Container()).Provider(NewDatabase)
//
//	// In subcommand:
//	app := gaz.FromContext(cmd.Context())
//	db, _ := gaz.Resolve[*Database](app.Container())
func WithCobra(cmd *cobra.Command) Option {
	return func(a *App) {
		a.cobraCmd = cmd

		// Apply any flags that were already registered before WithCobra() was called
		for _, fn := range a.flagFns {
			fn(cmd.PersistentFlags())
		}

		// Preserve existing hooks
		originalPreRunE := cmd.PersistentPreRunE
		originalPostRunE := cmd.PersistentPostRunE

		cmd.PersistentPreRunE = a.makePreRunE(originalPreRunE)
		cmd.PersistentPostRunE = a.makePostRunE(originalPostRunE)

		// Inject default RunE if no Run/RunE is defined
		if cmd.Run == nil && cmd.RunE == nil {
			cmd.RunE = func(c *cobra.Command, _ []string) error {
				return a.waitForShutdownSignal(c.Context())
			}
		}
	}
}

// makePreRunE creates the PersistentPreRunE hook that bootstraps the app.
func (a *App) makePreRunE(original func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		if original != nil {
			if err := original(c, args); err != nil {
				return err
			}
		}

		ctx := c.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		if err := a.bootstrap(ctx, c, args); err != nil {
			return err
		}

		c.SetContext(context.WithValue(ctx, contextKey{}, a))
		return nil
	}
}

// makePostRunE creates the PersistentPostRunE hook that stops the app.
func (a *App) makePostRunE(original func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		stopCtx, cancel := context.WithTimeout(context.Background(), a.opts.ShutdownTimeout)
		defer cancel()

		stopErr := a.Stop(stopCtx)

		a.mu.Lock()
		a.running = false
		a.mu.Unlock()

		if original != nil {
			if err := original(c, args); err != nil {
				return errors.Join(stopErr, err)
			}
		}

		return stopErr
	}
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

	// Initialize run state similar to App.Run
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return errors.New("app is already running")
	}
	a.stopCh = make(chan struct{})
	a.running = true
	a.mu.Unlock()

	// Ensure we clean up if start fails
	success := false
	defer func() {
		if !success {
			a.mu.Lock()
			a.running = false
			a.mu.Unlock()
		}
	}()

	// Build the app (validates registrations)
	if err := a.Build(); err != nil {
		return fmt.Errorf("app build failed: %w", err)
	}

	// Start lifecycle hooks
	if err := a.Start(ctx); err != nil {
		return fmt.Errorf("app start failed: %w", err)
	}

	success = true
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
