// Package server provides a unified transport layer for gaz applications,
// combining gRPC and Vanguard on a single port with lifecycle management.
//
// # Overview
//
// The server package coordinates gRPC and Vanguard to serve all supported
// protocols (gRPC, Connect, gRPC-Web, REST) on a single h2c port. gRPC
// registers services and interceptors without binding its own listener,
// while Vanguard wraps them via a transcoder and handles all connections.
//
// Startup order:
//   - gRPC registers services, interceptors, reflection, and health (no listener)
//   - Vanguard builds the transcoder and serves on a single h2c port
//
// Shutdown order:
//   - Vanguard stops first (drains HTTP connections)
//   - gRPC stops second (closes service registrations)
//
// # Usage
//
// Use NewModule to create a unified server module that registers both
// gRPC and Vanguard with correct configuration:
//
//	app := gaz.New()
//	app.Use(server.NewModule())
//	app.Run()
//
// The server module automatically sets gRPC SkipListener=true so that
// Vanguard handles all connections. No additional configuration is needed.
//
// # Configuration
//
// Configuration is handled via CLI flags and config files:
//
//   - Vanguard: "vanguard-address" (default ":8080"), timeouts, CORS, dev-mode
//   - gRPC: "grpc-reflection", "grpc-dev-mode", interceptor settings
//
// # Subpackages
//
// For more granular control, the subpackages can be used directly:
//
//   - server/grpc: gRPC server with interceptors, reflection, and service discovery
//   - server/http: Standalone HTTP server with configurable timeouts and lifecycle
//   - server/vanguard: Vanguard unified server (gRPC, Connect, gRPC-Web, REST transcoding)
//   - server/connect: Connect interceptor bundles (auth, logging, recovery, validation, rate-limit)
//
// # Lifecycle Integration
//
// Both gRPC and Vanguard servers implement di.Starter and di.Stopper interfaces,
// integrating with gaz's application lifecycle. The Vanguard server is registered
// as Eager, meaning it starts automatically when the application starts.
package server
