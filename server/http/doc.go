// Package http provides a production-ready HTTP server with DI integration.
//
// # Overview
//
// The HTTP server integrates with gaz's dependency injection container
// and lifecycle management. It provides configurable timeouts for security
// (preventing slow loris attacks) and graceful shutdown support.
//
// # Quick Start
//
// Use the module to register the HTTP server with your application:
//
//	app := gaz.New()
//	app.Use(http.NewModule())
//	app.Run()
//
// This starts an HTTP server on port 8080 with sensible defaults.
//
// # Configuration
//
// Configuration can be provided via config file or module options:
//
//	servers:
//	  http:
//	    port: 8080
//	    read_timeout: 10s
//	    write_timeout: 30s
//	    idle_timeout: 120s
//	    read_header_timeout: 5s
//
// Or via module options:
//
//	app.Use(http.NewModule(
//	    http.WithPort(3000),
//	    http.WithReadTimeout(15*time.Second),
//	    http.WithHandler(myHandler),
//	))
//
// # Timeout Rationale
//
// Each timeout serves a specific security and performance purpose:
//
//   - ReadTimeout: Maximum duration for reading the entire request including body.
//     Prevents clients from keeping connections open indefinitely with slow uploads.
//
//   - WriteTimeout: Maximum duration for writing the response. Ensures the server
//     doesn't hang waiting for slow clients to receive data.
//
//   - IdleTimeout: Maximum duration keep-alive connections remain open between
//     requests. Manages connection pool lifecycle for efficient resource usage.
//
//   - ReadHeaderTimeout: Maximum duration for reading request headers. This is
//     the primary defense against slow loris attacks (5s recommended).
//
// # Security Considerations
//
// The default ReadHeaderTimeout of 5 seconds protects against slow loris attacks,
// where attackers send HTTP headers slowly to exhaust server connections. This
// is aligned with security research recommending timeouts between 5-10 seconds.
//
// For additional security in production:
//   - Consider using a reverse proxy (nginx, envoy) for TLS termination
//   - Implement rate limiting at the Gateway layer
//   - Monitor connection counts and request latencies
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
