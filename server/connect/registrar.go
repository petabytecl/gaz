package connect

import (
	"net/http"

	"connectrpc.com/connect"
)

// Registrar is implemented by Connect-Go services that want to be
// auto-discovered and registered with the Vanguard server.
//
// Implementations return the service path and HTTP handler. The opts parameter
// receives connect.HandlerOption values (e.g., connect.WithInterceptors) that
// the Vanguard server passes automatically for interceptor injection.
// The signature matches Connect-Go generated NewXxxServiceHandler functions:
//
//	type GreeterService struct {
//	    greetv1connect.UnimplementedGreeterServiceHandler
//	}
//
//	func (s *GreeterService) RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler) {
//	    return greetv1connect.NewGreeterServiceHandler(s, opts...)
//	}
//
// Services are auto-discovered via di.ResolveAll[connect.Registrar] in the
// Vanguard server's OnStart method. Each returned (path, handler) pair is
// registered as a Vanguard service for protocol transcoding.
type Registrar interface {
	RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler)
}
