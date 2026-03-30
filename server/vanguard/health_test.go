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
	mux := buildHealthMux(mgr, nil)
	s.Require().NotNil(mux)

	// Verify default health paths respond.
	for _, path := range []string{"/ready", "/live", "/startup"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		s.Equalf(http.StatusOK, rec.Code, "GET %s should return 200", path)
	}
}

func (s *HealthTestSuite) TestBuildHealthMux_NilManager() {
	mux := buildHealthMux(nil, nil)
	s.Nil(mux, "Nil manager should return nil mux")
}

func (s *HealthTestSuite) TestMountHealthEndpoints_OnMux() {
	mgr := health.NewManager()
	mux := http.NewServeMux()

	// Mount with custom config paths
	cfg := &health.Config{
		ReadinessPath: "/custom-ready",
		LivenessPath:  "/custom-live",
		StartupPath:   "/custom-startup",
	}
	mountHealthEndpoints(mux, mgr, cfg)

	for _, path := range []string{"/custom-ready", "/custom-live", "/custom-startup"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		s.Equalf(http.StatusOK, rec.Code, "Health endpoint %s should respond with 200", path)
	}
}

func (s *HealthTestSuite) TestMountHealthEndpoints_NilManager() {
	mux := http.NewServeMux()
	// Should not panic with nil manager
	mountHealthEndpoints(mux, nil, nil)
}

func (s *HealthTestSuite) TestBuildHealthMux_CustomConfig() {
	mgr := health.NewManager()
	cfg := &health.Config{
		ReadinessPath: "/r",
		LivenessPath:  "/l",
		StartupPath:   "/s",
	}
	mux := buildHealthMux(mgr, cfg)
	s.Require().NotNil(mux)

	for _, path := range []string{"/r", "/l", "/s"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		s.Equalf(http.StatusOK, rec.Code, "GET %s should return 200", path)
	}
}
