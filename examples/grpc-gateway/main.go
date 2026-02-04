// Package main demonstrates a Unified Server (gRPC + Gateway) using gaz and Cobra.
//
// This example shows:
//   - gRPC server with auto-discovery
//   - HTTP Gateway proxying to gRPC
//   - Native health checks
//   - Integration with Cobra for CLI flags
//
// Run with:
//
//	go run . serve --grpc.port 9090 --gateway.port 8080 --grpc.dev_mode
//
// Test gRPC: grpcurl -plaintext -d '{"name": "Developer"}' localhost:9090 hello.Greeter/SayHello
// Test HTTP: curl "http://localhost:8080/v1/example/echo?name=Gaz"
// Health: curl http://localhost:8080/health
package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/config"
	cfgviper "github.com/petabytecl/gaz/config/viper"
	"github.com/petabytecl/gaz/health"
	"github.com/petabytecl/gaz/server"
)

func main() {
	if err := execute(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}

func execute() error {
	rootCmd := &cobra.Command{
		Use:   "grpc-gateway-example",
		Short: "gRPC-Gateway Example",
	}

	// 1. Setup Viper manually to ensure flags are bound correctly
	v := viper.New()

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// Sync flags to viper to ensure UnmarshalKey works.
			// Viper's UnmarshalKey doesn't always pick up PFlags unless they are set in the map.
			v.Set("grpc.port", v.GetInt("grpc.port"))
			v.Set("grpc.reflection", v.GetBool("grpc.reflection"))
			v.Set("grpc.dev_mode", v.GetBool("grpc.dev_mode"))
			v.Set("gateway.port", v.GetInt("gateway.port"))
			v.Set("gateway.grpc_target", v.GetString("gateway.grpc_target"))
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			// App is already started by WithCobra's PreRun hook.
			// We just need to block until shutdown signal.
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

			slog.Info("Server running. Press Ctrl+C to stop.")
			<-sigCh
			slog.Info("Shutdown signal received")

			return nil
		},
	}

	// Add flags matching the config keys expected by server module
	serveCmd.Flags().Int("grpc.port", 50051, "gRPC server port")
	serveCmd.Flags().Bool("grpc.reflection", true, "Enable gRPC reflection")
	serveCmd.Flags().Bool("grpc.dev_mode", true, "Enable gRPC dev mode")
	serveCmd.Flags().Int("gateway.port", 8080, "Gateway server port")
	serveCmd.Flags().String("gateway.grpc_target", "localhost:50051", "gRPC target for gateway")

	if err := v.BindPFlags(serveCmd.Flags()); err != nil {
		return fmt.Errorf("bind flags: %w", err)
	}

	// 2. Create the gaz application with custom viper backend
	app := gaz.New()
	app.WithConfig(nil, config.WithBackend(cfgviper.NewWithViper(v)))

	// 3. Use modules
	app.Use(server.NewModule())
	app.UseDI(health.NewModule())

	// 4. Register our Greeter Service
	if err := gaz.For[*GreeterService](app.Container()).Provider(NewGreeterService); err != nil {
		return fmt.Errorf("register greeter service: %w", err)
	}

	// 5. Attach Cobra integration
	app.WithCobra(serveCmd)

	rootCmd.AddCommand(serveCmd)

	return rootCmd.Execute()
}
