package connect

import (
	"net/http"
	"testing"

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
type mockConnectRegistrar struct{}

func (m *mockConnectRegistrar) RegisterConnect() (string, http.Handler) {
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
