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

	"github.com/spf13/cobra"

	"github.com/petabytecl/gaz"
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

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
	}

	// 1. Create the gaz application
	app := gaz.New()

	// 2. Use modules
	// server.NewModule() provides gRPC + Gateway + Health + Config
	app.Use(server.NewModule())

	// 3. Register our Greeter Service
	if err := gaz.For[*GreeterService](app.Container()).Provider(NewGreeterService); err != nil {
		return fmt.Errorf("register greeter service: %w", err)
	}

	// 4. Attach Cobra integration
	// This handles flags, lifecycle, and graceful shutdown
	app.WithCobra(serveCmd)

	rootCmd.AddCommand(serveCmd)

	return rootCmd.Execute()
}
