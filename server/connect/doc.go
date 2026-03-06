// Package connect provides the ConnectRegistrar interface for auto-discovery
// of Connect-Go services within the gaz framework.
//
// # Overview
//
// This package defines the Registrar interface, which Connect-Go services
// implement to be automatically discovered and registered with the Vanguard
// server. The pattern mirrors the gRPC Registrar interface in the server/grpc
// package.
//
// # Service Registration
//
// Services implement the ConnectRegistrar interface to be auto-discovered:
//
//	type GreeterService struct {
//	    greetv1connect.UnimplementedGreeterServiceHandler
//	}
//
//	func (s *GreeterService) RegisterConnect() (string, http.Handler) {
//	    return greetv1connect.NewGreeterServiceHandler(s)
//	}
//
// Register the service in your module:
//
//	di.For[*GreeterService](c).Provider(NewGreeterService)
//
// The Vanguard server will discover all Registrar implementations
// automatically on startup via di.ResolveAll[connect.Registrar].
package connect
