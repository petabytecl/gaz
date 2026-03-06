package connect

import "net/http"

// Registrar is implemented by Connect-Go services that want to be
// auto-discovered and registered with the Vanguard server.
//
// Implementations return the service path and HTTP handler. The signature
// matches Connect-Go generated NewXxxServiceHandler functions exactly:
//
//	type GreeterService struct {
//	    greetv1connect.UnimplementedGreeterServiceHandler
//	}
//
//	func (s *GreeterService) RegisterConnect() (string, http.Handler) {
//	    return greetv1connect.NewGreeterServiceHandler(s)
//	}
//
// Services are auto-discovered via di.ResolveAll[connect.Registrar] in the
// Vanguard server's OnStart method. Each returned (path, handler) pair is
// registered as a Vanguard service for protocol transcoding.
type Registrar interface {
	RegisterConnect() (string, http.Handler)
}
