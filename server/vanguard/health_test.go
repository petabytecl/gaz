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

	// Liveness always 200; readiness/startup return 503 with no checks (StatusUnknown).
	expected := map[string]int{
		"/live":    http.StatusOK,
		"/ready":   http.StatusServiceUnavailable,
		"/startup": http.StatusServiceUnavailable,
	}
	for path, wantCode := range expected {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		s.Equalf(wantCode, rec.Code, "GET %s should return %d", path, wantCode)
	}
}

func (s *HealthTestSuite) TestBuildHealthMux_NilManager() {
	mux := buildHealthMux(nil, nil)
	s.Nil(mux, "Nil manager should return nil mux")
}

func (s *HealthTestSuite) TestMountHealthEndpoints_OnMux() {
	mgr := health.NewManager()
	mux := http.NewServeMux()

	cfg := &health.Config{
		ReadinessPath: "/custom-ready",
		LivenessPath:  "/custom-live",
		StartupPath:   "/custom-startup",
	}
	mountHealthEndpoints(mux, mgr, cfg)

	expected := map[string]int{
		"/custom-live":    http.StatusOK,
		"/custom-ready":   http.StatusServiceUnavailable,
		"/custom-startup": http.StatusServiceUnavailable,
	}
	for path, wantCode := range expected {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		s.Equalf(wantCode, rec.Code, "Health endpoint %s should respond with %d", path, wantCode)
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

	expected := map[string]int{
		"/l": http.StatusOK,
		"/r": http.StatusServiceUnavailable,
		"/s": http.StatusServiceUnavailable,
	}
	for path, wantCode := range expected {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		s.Equalf(wantCode, rec.Code, "GET %s should return %d", path, wantCode)
	}
}
