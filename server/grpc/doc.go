// Package grpc provides a production-ready gRPC server with auto-discovery,
// interceptors, and lifecycle integration for the gaz framework.
//
// # Overview
//
// This package implements a gRPC server that integrates with gaz's DI container
// and lifecycle management. Services implementing the ServiceRegistrar interface
// are automatically discovered and registered on startup.
//
// # Quick Start
//
// Use the module to register the gRPC server with your application:
//
//	app := gaz.New()
//	app.Use(grpc.NewModule(
//	    grpc.WithPort(50051),
//	    grpc.WithReflection(true),
//	))
//
// # Service Registration
//
// Services implement the ServiceRegistrar interface to be auto-discovered:
//
//	type GreeterService struct {
//	    pb.UnimplementedGreeterServer
//	}
//
//	func (s *GreeterService) RegisterService(server grpc.ServiceRegistrar) {
//	    pb.RegisterGreeterServer(server, s)
//	}
//
// Register the service in your module:
//
//	di.For[*GreeterService](c).Provider(NewGreeterService)
//
// The gRPC server will discover and register all ServiceRegistrar implementations
// automatically on startup.
//
// # Interceptors
//
// The server includes built-in interceptors for:
//   - Logging: Request/response logging with duration and status
//   - Recovery: Panic recovery with stack trace logging
//
// # Reflection
//
// gRPC reflection is enabled by default, allowing tools like grpcurl to
// introspect available services:
//
//	grpcurl -plaintext localhost:50051 list
//	grpcurl -plaintext localhost:50051 describe mypackage.MyService
//
// Disable reflection in production if needed:
//
//	grpc.NewModule(grpc.WithReflection(false))
//
// # Configuration
//
// Configuration can be provided via config file or module options:
//
//	servers:
//	  grpc:
//	    port: 50051
//	    reflection: true
//	    max_recv_msg_size: 4194304
//	    max_send_msg_size: 4194304
package grpc
