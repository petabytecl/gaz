// Package main demonstrates Cobra CLI integration with gaz.
//
// This example shows:
//   - Root command with persistent flags
//   - Subcommands that use dependency injection
//   - Flag binding to configuration via viper
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
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/petabytecl/gaz"
)

// AppConfig holds application configuration.
// Fields are populated from flags, env vars, and config files via viper.
type AppConfig struct {
	Debug   bool   `mapstructure:"debug"`
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
	Timeout int    `mapstructure:"timeout"`
}

// Server represents the application server.
type Server struct {
	config AppConfig
}

// NewServer creates a new server with the given configuration.
func NewServer(config AppConfig) *Server {
	return &Server{config: config}
}

// Start begins the server operation.
func (s *Server) Start(ctx context.Context) error {
	fmt.Printf("Server starting on %s:%d\n", s.config.Host, s.config.Port)
	fmt.Printf("Debug mode: %v\n", s.config.Debug)
	fmt.Printf("Request timeout: %ds\n", s.config.Timeout)

	// Simulate server running
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Server shutting down...")
			return nil
		case <-ticker.C:
			fmt.Println("Server is running... (Ctrl+C to stop)")
		}
	}
}

// OnStart is called when the application starts.
func (s *Server) OnStart(_ context.Context) error {
	fmt.Printf("Initializing server on %s:%d...\n", s.config.Host, s.config.Port)
	return nil
}

// OnStop is called when the application stops.
func (s *Server) OnStop(_ context.Context) error {
	fmt.Println("Cleaning up server resources...")
	return nil
}

func main() {
	if err := execute(); err != nil {
		os.Exit(1)
	}
}

func execute() error {
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

	// Persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().IntP("port", "p", 8080, "Server port")
	rootCmd.PersistentFlags().StringP("host", "H", "localhost", "Server host")
	rootCmd.PersistentFlags().IntP("timeout", "t", 30, "Request timeout in seconds")

	// Serve subcommand
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		Long:  "Starts the application server with the configured settings.",
		RunE:  runServe,
	}

	// Version subcommand (simple, no DI needed)
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("myapp v1.0.0")
			fmt.Println("Built with gaz dependency injection")
		},
	}

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)

	return rootCmd.Execute()
}

// runServe is the handler for the "serve" subcommand.
func runServe(cmd *cobra.Command, _ []string) error {
	// Read flag values
	debug, _ := cmd.Flags().GetBool("debug")
	port, _ := cmd.Flags().GetInt("port")
	host, _ := cmd.Flags().GetString("host")
	timeout, _ := cmd.Flags().GetInt("timeout")

	// Create configuration from flags
	config := AppConfig{
		Debug:   debug,
		Port:    port,
		Host:    host,
		Timeout: timeout,
	}

	// Create the gaz application
	app := gaz.New(
		gaz.WithShutdownTimeout(10 * time.Second),
	)

	// Register configuration as instance
	app.ProvideInstance(config)

	// Register server with lifecycle hooks
	if err := gaz.For[*Server](app.Container()).
		OnStart(func(ctx context.Context, s *Server) error {
			return s.OnStart(ctx)
		}).
		OnStop(func(ctx context.Context, s *Server) error {
			return s.OnStop(ctx)
		}).
		Eager().
		Provider(func(c *gaz.Container) (*Server, error) {
			cfg, err := gaz.Resolve[AppConfig](c)
			if err != nil {
				return nil, err
			}
			return NewServer(cfg), nil
		}); err != nil {
		return fmt.Errorf("failed to register server: %w", err)
	}

	// Attach gaz lifecycle to the Cobra command
	app.WithCobra(cmd)

	// The lifecycle is now managed:
	// - PersistentPreRunE: Build() and Start() are called
	// - PersistentPostRunE: Stop() is called

	// Start the server (blocks until signal)
	server, err := gaz.Resolve[*Server](app.Container())
	if err != nil {
		return fmt.Errorf("failed to resolve server: %w", err)
	}

	// Run the server with context from Cobra
	return server.Start(cmd.Context())
}
