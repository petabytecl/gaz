package vanguard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/petabytecl/gaz/health"
)

// HealthTestSuite tests the health endpoint mounting.
type HealthTestSuite struct {
	suite.Suite
}

func TestHealthTestSuite(t *testing.T) {
	suite.Run(t, new(HealthTestSuite))
}

func (s *HealthTestSuite) TestBuildHealthMux_AllPaths() {
	mgr := health.NewManager()
	mux := buildHealthMux(mgr)
	s.Require().NotNil(mux)

	// Verify /healthz, /readyz, /livez all respond.
	for _, path := range []string{"/healthz", "/readyz", "/livez"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		s.Equalf(http.StatusOK, rec.Code, "GET %s should return 200", path)
	}
}

func (s *HealthTestSuite) TestBuildHealthMux_NilManager() {
	mux := buildHealthMux(nil)
	s.Nil(mux, "Nil manager should return nil mux")
}

func (s *HealthTestSuite) TestMountHealthEndpoints_OnMux() {
	// Verify health endpoints are accessible via the built mux.
	mgr := health.NewManager()

	mux := buildHealthMux(mgr)
	s.Require().NotNil(mux)

	paths := []string{"/healthz", "/readyz", "/livez"}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		s.Equalf(http.StatusOK, rec.Code, "Health endpoint %s should respond with 200", path)
	}
}
