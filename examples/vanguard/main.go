// Package main demonstrates the Vanguard unified server with gaz.
//
// This example shows how to serve gRPC, Connect, gRPC-Web, and REST/JSON
// on a single port using server.NewModule(). The Greeter service is
// auto-discovered via the grpc.Registrar and connect.Registrar interfaces.
//
// Run with:
//
//	go run .
//
// Test endpoints (default port 8080):
//
//	# REST/JSON (via HTTP annotation: POST /v1/example/echo)
//	curl -X POST http://localhost:8080/v1/example/echo \
//	  -H "Content-Type: application/json" \
//	  -d '{"name": "World"}'
//
//	# Connect (unary RPC over HTTP)
//	curl -X POST http://localhost:8080/hello.Greeter/SayHello \
//	  -H "Content-Type: application/json" \
//	  -d '{"name": "World"}'
//
//	# gRPC (requires grpcurl or a gRPC client)
//	grpcurl -plaintext -d '{"name": "World"}' localhost:8080 hello.Greeter/SayHello
//
//	# gRPC-Web (use a gRPC-Web client or browser library)
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/server"
)

func run(ctx context.Context) error {
	app := gaz.New()

	// Use the unified server module: gRPC + Vanguard on a single port.
	app.Use(server.NewModule())

	// Register GreeterService as eager so it starts automatically.
	// The server module auto-discovers it via grpc.Registrar and connect.Registrar
	// interfaces using di.ResolveAll (reflection-based interface matching).
	if err := gaz.For[*GreeterService](app.Container()).
		Eager().
		Provider(NewGreeterService); err != nil {
		return fmt.Errorf("register greeter service: %w", err)
	}

	// Build the application (resolves all providers, validates config).
	if err := app.Build(); err != nil {
		return fmt.Errorf("build app: %w", err)
	}

	slog.Info("Vanguard example starting",
		"port", 8080,
		"protocols", "gRPC, Connect, gRPC-Web, REST",
	)
	slog.Info("Try: curl -X POST http://localhost:8080/v1/example/echo -H 'Content-Type: application/json' -d '{\"name\": \"World\"}'")

	// Run blocks until shutdown signal (SIGINT/SIGTERM) or context cancellation.
	if err := app.Run(ctx); err != nil {
		return fmt.Errorf("run app: %w", err)
	}

	return nil
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
