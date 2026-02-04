// Package main demonstrates Cobra CLI integration with gaz.
//
// This example shows:
//   - Root command with persistent flags
//   - Subcommands that use dependency injection
//   - gaz.WithCobra() for automatic lifecycle management
//
// Run with:
//
//	go run . serve --port 8080
//	go run . version
//	go run . --help
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/petabytecl/gaz"
)

// AppConfig holds application configuration.
// Fields are populated from flags.
type AppConfig struct {
	Debug   bool   `mapstructure:"debug"`
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
	Timeout int    `mapstructure:"timeout"`
}

// Server represents the application server.
type Server struct {
	config AppConfig
	out    io.Writer
}

// NewServer creates a new server with the given configuration.
func NewServer(config AppConfig, out io.Writer) *Server {
	if out == nil {
		out = os.Stdout
	}
	return &Server{config: config, out: out}
}

// Start begins the server operation.
func (s *Server) Start(ctx context.Context) error {
	fmt.Fprintf(s.out, "Server starting on %s:%d\n", s.config.Host, s.config.Port)
	fmt.Fprintf(s.out, "Debug mode: %v\n", s.config.Debug)
	fmt.Fprintf(s.out, "Request timeout: %ds\n", s.config.Timeout)

	// Simulate server running
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(s.out, "Server shutting down...")
			return nil
		case <-ticker.C:
			fmt.Fprintln(s.out, "Server is running... (Ctrl+C to stop)")
		}
	}
}

// OnStart is called when the application starts.
func (s *Server) OnStart(_ context.Context) error {
	fmt.Fprintf(s.out, "Initializing server on %s:%d...\n", s.config.Host, s.config.Port)
	return nil
}

// OnStop is called when the application stops.
func (s *Server) OnStop(_ context.Context) error {
	fmt.Fprintln(s.out, "Cleaning up server resources...")
	return nil
}

func main() {
	if err := execute(context.Background(), os.Args[1:], os.Stdout); err != nil {
		os.Exit(1)
	}
}

func execute(ctx context.Context, args []string, out io.Writer) error {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "myapp",
		Short: "Example CLI application with gaz DI",
		Long: `This example demonstrates integrating gaz dependency injection
with Cobra CLI. It shows:
- Persistent flags on root command
- Subcommands that access injected services
- Automatic lifecycle management via WithCobra()`,
	}

	rootCmd.SetOut(out)
	rootCmd.SetArgs(args)

	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().IntP("port", "p", 8080, "Server port")
	rootCmd.PersistentFlags().StringP("host", "H", "localhost", "Server host")
	rootCmd.PersistentFlags().IntP("timeout", "t", 30, "Request timeout in seconds")

	// Serve subcommand - the lifecycle-managed command
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		Long:  "Starts the application server with the configured settings.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get the app from context (set by gaz.WithCobra)
			app := gaz.FromContext(cmd.Context())
			if app == nil {
				return fmt.Errorf("app not found in context")
			}

			// Resolve the server and call Start
			server, err := gaz.Resolve[*Server](app.Container())
			if err != nil {
				return fmt.Errorf("failed to resolve server: %w", err)
			}

			// Run the server until context is cancelled
			return server.Start(cmd.Context())
		},
	}

	// Create the gaz application with Cobra integration
	// WithCobra as an Option handles lifecycle (Build/Start in PreRunE, Stop in PostRunE)
	app := gaz.New(
		gaz.WithShutdownTimeout(10*time.Second),
		gaz.WithCobra(serveCmd),
	)

	// Register flag functions to add module flags
	app.AddFlagsFn(func(fs *pflag.FlagSet) {
		// Module-specific flags would go here
	})

	// Register server with lifecycle hooks (Server implements di.Starter and di.Stopper)
	if err := gaz.For[*Server](app.Container()).
		Eager().
		Provider(func(c *gaz.Container) (*Server, error) {
			// Get command args to read flag values
			cmdArgs, err := gaz.Resolve[*gaz.CommandArgs](c)
			if err != nil {
				return nil, err
			}
			cmd := cmdArgs.Command

			// Read flag values from the command
			debug, _ := cmd.Flags().GetBool("debug")
			port, _ := cmd.Flags().GetInt("port")
			host, _ := cmd.Flags().GetString("host")
			timeout, _ := cmd.Flags().GetInt("timeout")

			config := AppConfig{
				Debug:   debug,
				Port:    port,
				Host:    host,
				Timeout: timeout,
			}

			return NewServer(config, out), nil
		}); err != nil {
		return fmt.Errorf("failed to register server: %w", err)
	}

	// Version subcommand (simple, no DI needed)
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(out, "myapp v1.0.0")
			fmt.Fprintln(out, "Built with gaz dependency injection")
		},
	}

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)

	return rootCmd.ExecuteContext(ctx)
}
