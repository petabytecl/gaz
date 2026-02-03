// Package http provides a production-ready HTTP server with DI integration.
//
// The HTTP server is designed to work with gaz's dependency injection container
// and lifecycle management. It provides configurable timeouts for security
// (preventing slow loris attacks) and graceful shutdown support.
//
// # Basic Usage
//
// The simplest way to use the HTTP server is via the module:
//
//	app := gaz.New()
//	app.UseDI(http.NewModule())
//	app.Run()
//
// This starts an HTTP server on port 8080 with sensible defaults.
//
// # Custom Configuration
//
// You can customize the server via module options:
//
//	app.UseDI(http.NewModule(
//	    http.WithPort(3000),
//	    http.WithReadTimeout(15*time.Second),
//	    http.WithHandler(myHandler),
//	))
//
// # Handler Integration
//
// By default, the server uses http.NotFoundHandler(). For Gateway integration
// (Phase 39), the handler will be set by the Gateway module to proxy HTTP
// requests to gRPC services.
//
// # Lifecycle
//
// The HTTPServer implements di.Starter and di.Stopper interfaces:
//   - OnStart: Starts the HTTP server in a background goroutine
//   - OnStop: Gracefully shuts down the server using http.Server.Shutdown
//
// The server is registered as Eager, so it starts automatically when the
// application starts.
package http
