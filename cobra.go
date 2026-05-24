package gaz

import (
	"context"

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

		session := a.lifecycleSession()
		cmd.PersistentPreRunE = session.makePreRunE(originalPreRunE)
		cmd.PersistentPostRunE = session.makePostRunE(originalPostRunE)

		// Inject default RunE if no Run/RunE is defined
		if cmd.Run == nil && cmd.RunE == nil {
			cmd.RunE = func(c *cobra.Command, _ []string) error {
				return session.waitForShutdownSignal(c.Context())
			}
		}
	}
}
