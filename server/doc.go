// Package server provides a unified transport layer for gaz applications,
// combining gRPC and HTTP servers with lifecycle management.
//
// # Overview
//
// The server package coordinates the startup and shutdown of both gRPC and HTTP
// servers in the correct order:
//
//   - Startup: gRPC first, then HTTP (Gateway depends on gRPC being up)
//   - Shutdown: HTTP first, then gRPC (reverse of startup)
//
// This ordering ensures that the HTTP Gateway can always proxy to gRPC services
// during normal operation and graceful shutdown.
//
// # Usage
//
// Use NewModule to create a unified server module that registers both servers:
//
//	app := gaz.New()
//	app.Use(server.NewModule(
//	    server.WithGRPCPort(50051),
//	    server.WithHTTPPort(8080),
//	))
//	app.Run()
//
// # Configuration Options
//
// The module supports various configuration options:
//
//   - WithGRPCPort: Set the gRPC server port (default: 50051)
//   - WithHTTPPort: Set the HTTP server port (default: 8080)
//   - WithGRPCReflection: Enable/disable gRPC reflection (default: true)
//   - WithHTTPHandler: Set a custom HTTP handler (default: NotFoundHandler)
//
// # Subpackages
//
// For more granular control, the subpackages can be used directly:
//
//   - server/grpc: gRPC server with interceptors, reflection, and service discovery
//   - server/http: HTTP server with configurable timeouts and lifecycle management
//
// # Lifecycle Integration
//
// Both servers implement di.Starter and di.Stopper interfaces, integrating with
// gaz's application lifecycle. Services are registered as Eager, meaning they
// start automatically when the application starts.
package server
