// Package gateway provides an HTTP-to-gRPC gateway using grpc-gateway.
//
// The Gateway translates RESTful HTTP/JSON requests into gRPC calls,
// enabling a single gRPC service to serve both gRPC and HTTP clients.
// It uses grpc-gateway's runtime.ServeMux for request translation
// and rs/cors for CORS handling.
//
// # Auto-Discovery
//
// Services that want HTTP exposure implement the Registrar interface:
//
//	type Registrar interface {
//	    RegisterGateway(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
//	}
//
// The Gateway auto-discovers all registered Registrar implementations
// via di.ResolveAll and registers them during startup.
//
// # Usage
//
// Basic usage with NewModule:
//
//	app := gaz.New()
//	app.Use(grpc.NewModule())
//	app.Use(http.NewModule(http.WithPort(8080)))
//	app.Use(gateway.NewModule())
//
//	// After modules, wire the handler
//	gateway := gaz.MustResolve[*gateway.Gateway](app.Container())
//	httpServer := gaz.MustResolve[*http.Server](app.Container())
//	httpServer.SetHandler(gateway.Handler())
//
// With CLI flags:
//
//	app := gaz.New()
//	app.Use(gateway.NewModuleWithFlags(rootCmd.Flags()))
//	// --gateway-port, --gateway-grpc-target, --gateway-dev-mode available
//
// # CORS Configuration
//
// In development mode (WithDevMode(true)), CORS is permissive:
//   - AllowedOrigins: ["*"]
//   - AllowedHeaders: ["*"]
//   - AllowCredentials: false
//
// In production mode, CORS must be explicitly configured:
//
//	gateway.NewModule(gateway.WithCORS(gateway.CORSConfig{
//	    AllowedOrigins:   []string{"https://example.com"},
//	    AllowedMethods:   []string{"GET", "POST"},
//	    AllowCredentials: true,
//	}))
//
// # Error Responses
//
// Errors are returned in RFC 7807 Problem Details format:
//
//	{
//	    "type": "https://grpc.io/docs/guides/status-codes/#not_found",
//	    "title": "NOT_FOUND",
//	    "status": 404,
//	    "detail": "resource not found",
//	    "instance": "req-123"
//	}
//
// In production, the detail field contains only generic HTTP status text.
// In development mode, it includes the actual gRPC error message.
package gateway
