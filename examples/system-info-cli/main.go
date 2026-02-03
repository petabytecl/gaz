// Package main provides a system information CLI demonstrating gaz framework features.
//
// This example showcases the full gaz pattern:
//   - Dependency Injection: For[T]() and Resolve[T]() patterns
//   - ConfigProvider: Flag-based configuration with ProviderValues
//   - Workers: Background data collection with lifecycle integration
//   - Cobra Integration: RegisterCobraFlags for CLI flag visibility in --help
//
// Usage:
//
//	go run . run --sysinfo-once              # One-shot mode
//	go run . run --sysinfo-format json       # JSON output
//	go run . run                             # Continuous monitoring (Ctrl+C to stop)
//	go run . run --sysinfo-refresh 10s       # Custom refresh interval
//	go run . run --help                      # View available flags
//	go run . version                         # Print version
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/petabytecl/gaz"
)

func main() {
	if err := execute(os.Args[1:], os.Stdout); err != nil {
		os.Exit(1)
	}
}

func execute(args []string, out io.Writer) error {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "sysinfo",
		Short: "System information CLI - gaz framework demo",
		Long: `System Information CLI demonstrates gaz DI framework features:

  - Dependency Injection: For[T]() and Resolve[T]() patterns
  - ConfigProvider: Flag-based configuration with ProviderValues  
  - Workers: Background data collection with lifecycle integration
  - Cobra Integration: RegisterCobraFlags for CLI flag visibility

Examples:
  sysinfo run                         # Continuous monitoring (Ctrl+C to stop)
  sysinfo run --sysinfo-once          # Display system info and exit
  sysinfo run --sysinfo-format json   # JSON output
  sysinfo run --sysinfo-refresh 10s   # Custom refresh interval
  sysinfo version                     # Print version`,
	}

	rootCmd.SetOut(out)
	rootCmd.SetArgs(args)

	// Create gaz application with shutdown timeout
	app := gaz.New(gaz.WithShutdownTimeout(5 * time.Second))

	// Register ConfigProvider - declares config flags (sysinfo.refresh, sysinfo.format, sysinfo.once)
	if err := gaz.For[*SystemInfoConfig](app.Container()).Provider(NewSystemInfoConfig); err != nil {
		return fmt.Errorf("failed to register config: %w", err)
	}

	// Register Collector service
	if err := gaz.For[*Collector](app.Container()).Provider(NewCollector); err != nil {
		return fmt.Errorf("failed to register collector: %w", err)
	}

	// CRITICAL: Register flags BEFORE Execute() for --help visibility
	// This exposes --sysinfo-refresh, --sysinfo-format, --sysinfo-once flags
	if err := app.RegisterCobraFlags(rootCmd); err != nil {
		return fmt.Errorf("failed to register flags: %w", err)
	}

	// Run subcommand - main operation
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run system info collection",
		Long:  "Start system information collection in one-shot or continuous mode.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSysInfo(cmd, app)
		},
	}
	rootCmd.AddCommand(runCmd)

	// Version subcommand - simple, no DI needed
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(out, "sysinfo v1.0.0")
			fmt.Fprintln(out, "gaz framework system info example")
		},
	}
	rootCmd.AddCommand(versionCmd)

	return rootCmd.Execute()
}

// runSysInfo handles the "run" subcommand for system info collection.
func runSysInfo(cmd *cobra.Command, app *gaz.App) error {
	// Attach gaz lifecycle to Cobra command
	app.WithCobra(cmd)

	// Resolve the ConfigProvider
	cfg, err := gaz.Resolve[*SystemInfoConfig](app.Container())
	if err != nil {
		return fmt.Errorf("failed to resolve config: %w", err)
	}

	// One-shot mode: collect once and exit
	if cfg.Once() {
		collector, err := gaz.Resolve[*Collector](app.Container())
		if err != nil {
			return fmt.Errorf("failed to resolve collector: %w", err)
		}

		info, err := collector.Collect()
		if err != nil {
			return fmt.Errorf("failed to collect system info: %w", err)
		}

		return collector.Display(info)
	}

	// Continuous mode: register worker for periodic refresh
	collector, err := gaz.Resolve[*Collector](app.Container())
	if err != nil {
		return fmt.Errorf("failed to resolve collector: %w", err)
	}

	// Create RefreshWorker with config values
	worker := NewRefreshWorker(
		"sysinfo-worker",
		cfg.RefreshInterval(),
		cfg.Format(),
		collector,
	)

	// Register worker via Instance() - enables auto-discovery during Build()
	if err := gaz.For[*RefreshWorker](app.Container()).Named("sysinfo-worker").Instance(worker); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Print startup message
	fmt.Printf("Starting continuous monitoring (refresh every %s, Ctrl+C to stop)...\n\n", cfg.RefreshInterval())

	// Run the application - handles lifecycle, signal handling, graceful shutdown
	return app.Run(cmd.Context())
}
