package connect

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/suite"
)

// ConnectRegistrarTestSuite tests the ConnectRegistrar interface contract.
type ConnectRegistrarTestSuite struct {
	suite.Suite
}

func TestConnectRegistrarTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectRegistrarTestSuite))
}

// mockConnectRegistrar implements ConnectRegistrar for testing.
type mockConnectRegistrar struct {
	receivedOpts []connect.HandlerOption
}

func (m *mockConnectRegistrar) RegisterConnect(opts ...connect.HandlerOption) (string, http.Handler) {
	m.receivedOpts = opts
	return "/test.v1.TestService/", http.NewServeMux()
}

func (s *ConnectRegistrarTestSuite) TestInterfaceCompliance() {
	// Verify mockConnectRegistrar satisfies the Registrar interface.
	s.Require().Implements((*Registrar)(nil), &mockConnectRegistrar{})
}

func (s *ConnectRegistrarTestSuite) TestRegisterConnectReturnValues() {
	var registrar Registrar = &mockConnectRegistrar{}

	path, handler := registrar.RegisterConnect()

	s.Require().NotEmpty(path, "Service path must not be empty")
	s.Require().NotNil(handler, "HTTP handler must not be nil")
}

func (s *ConnectRegistrarTestSuite) TestRegisterConnectPathFormat() {
	registrar := &mockConnectRegistrar{}

	path, _ := registrar.RegisterConnect()

	// Connect-Go service paths follow the /package.Service/ pattern.
	s.Require().Contains(path, "/", "Service path should contain forward slashes")
}

func (s *ConnectRegistrarTestSuite) TestRegisterConnectReceivesHandlerOptions() {
	registrar := &mockConnectRegistrar{}

	// Pass handler options (e.g., interceptors) to verify they are received.
	noopInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return next
	})
	opts := []connect.HandlerOption{connect.WithInterceptors(noopInterceptor)}

	registrar.RegisterConnect(opts...)

	s.Require().Len(registrar.receivedOpts, 1, "Handler options should be forwarded to RegisterConnect")
}

func (s *ConnectRegistrarTestSuite) TestRegisterConnectNoOptions() {
	registrar := &mockConnectRegistrar{}

	// Variadic allows calling with no options.
	path, handler := registrar.RegisterConnect()

	s.NotEmpty(path)
	s.NotNil(handler)
	s.Empty(registrar.receivedOpts, "No options should be received when none are passed")
}
