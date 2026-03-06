// Package connect provides the Registrar interface for auto-discovery of
// Connect-Go services and the ConnectInterceptorBundle interface for
// auto-discovered interceptor chains within the gaz framework.
//
// # Overview
//
// This package defines two key interfaces:
//
//   - [Registrar]: implemented by Connect-Go services for automatic discovery
//     and registration with the Vanguard server.
//   - [ConnectInterceptorBundle]: implemented by interceptor bundles for
//     automatic discovery, priority-based sorting, and chaining.
//
// The pattern mirrors the gRPC Registrar and InterceptorBundle interfaces
// in the server/grpc package.
//
// # Service Registration
//
// Services implement the Registrar interface to be auto-discovered. The
// RegisterConnect method accepts variadic connect.HandlerOption for interceptor
// injection by the Vanguard server:
//
//	type GreeterService struct {
//	    greetv1connect.UnimplementedGreeterServiceHandler
//	}
//
//	func (s *GreeterService) RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler) {
//	    return greetv1connect.NewGreeterServiceHandler(s, opts...)
//	}
//
// Register the service in your module:
//
//	di.For[*GreeterService](c).Provider(NewGreeterService)
//
// The Vanguard server will discover all Registrar implementations
// automatically on startup via di.ResolveAll[connect.Registrar].
//
// # Interceptor Bundles
//
// Interceptor bundles implement [ConnectInterceptorBundle] and are
// auto-discovered via di.ResolveAll[connect.ConnectInterceptorBundle].
// Each bundle declares a priority that determines its position in the
// interceptor chain (lower values run earlier).
//
// Five built-in bundles are provided:
//
//   - [LoggingBundle] (priority 0): logs procedure name, duration, and errors.
//   - [RateLimitBundle] (priority 25): enforces rate limits via [ConnectLimiter].
//   - [AuthBundle] (priority 50): validates credentials via [ConnectAuthFunc].
//   - [ValidationBundle] (priority 100): validates protobuf messages.
//   - [RecoveryBundle] (priority 1000): catches panics and returns errors.
//
// Custom interceptor bundles can be registered with priorities between 1 and 999.
package connect
