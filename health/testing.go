package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/petabytecl/gaz/health/internal"
)

// TestConfig returns a health.Config with safe defaults for testing.
// Uses port 0 for random available port to avoid port conflicts in parallel tests.
func TestConfig() Config {
	return Config{
		Port:          0, // Random available port
		LivenessPath:  "/live",
		ReadinessPath: "/ready",
		StartupPath:   "/startup",
	}
}

// NewTestConfig returns a test config with custom options applied.
// Example:
//
//	cfg := health.NewTestConfig(func(c *health.Config) {
//	    c.Port = 8080
//	})
func NewTestConfig(opts ...func(*Config)) Config {
	cfg := TestConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// MockRegistrar is a test double for health.Registrar.
// It uses testify/mock for expectation setting and verification.
type MockRegistrar struct {
	mock.Mock
}

// NewMockRegistrar creates a MockRegistrar with default expectations.
// All Add* methods accept any arguments and return without error.
func NewMockRegistrar() *MockRegistrar {
	m := &MockRegistrar{}
	m.On("AddLivenessCheck", mock.Anything, mock.Anything).Return()
	m.On("AddReadinessCheck", mock.Anything, mock.Anything).Return()
	m.On("AddStartupCheck", mock.Anything, mock.Anything).Return()
	return m
}

// AddLivenessCheck records a liveness check registration.
func (m *MockRegistrar) AddLivenessCheck(name string, check CheckFunc) {
	m.Called(name, check)
}

// AddReadinessCheck records a readiness check registration.
func (m *MockRegistrar) AddReadinessCheck(name string, check CheckFunc) {
	m.Called(name, check)
}

// AddStartupCheck records a startup check registration.
func (m *MockRegistrar) AddStartupCheck(name string, check CheckFunc) {
	m.Called(name, check)
}

// TestManager creates a Manager with no checks registered, suitable for testing.
// The manager is ready to have checks added and can be used to create health checkers.
func TestManager() *Manager {
	return NewManager()
}

// RequireHealthy checks that all readiness checks pass.
// Uses testing.TB for compatibility with both tests and benchmarks.
// Note: An empty checker returns StatusUnknown, not StatusUp. Use this only
// when you have registered at least one passing check.
func RequireHealthy(tb testing.TB, m *Manager) {
	tb.Helper()
	checker := m.ReadinessChecker()
	result := checker.Check(context.Background())
	if result.Status != internal.StatusUp {
		tb.Fatalf("RequireHealthy: expected status 'up', got '%s'", result.Status)
	}
}

// RequireUnhealthy checks that at least one readiness check fails.
func RequireUnhealthy(tb testing.TB, m *Manager) {
	tb.Helper()
	checker := m.ReadinessChecker()
	result := checker.Check(context.Background())
	if result.Status == internal.StatusUp {
		tb.Fatalf("RequireUnhealthy: expected status not 'up', got '%s'", result.Status)
	}
}

// RequireLivenessCheckRegistered verifies a liveness check with the given name is registered.
func RequireLivenessCheckRegistered(tb testing.TB, m *Manager, name string) {
	tb.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.livenessChecks {
		if c.Name == name {
			return
		}
	}
	tb.Fatalf("RequireLivenessCheckRegistered: check %q not found", name)
}

// RequireReadinessCheckRegistered verifies a readiness check with the given name is registered.
func RequireReadinessCheckRegistered(tb testing.TB, m *Manager, name string) {
	tb.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.readinessChecks {
		if c.Name == name {
			return
		}
	}
	tb.Fatalf("RequireReadinessCheckRegistered: check %q not found", name)
}

// RequireStartupCheckRegistered verifies a startup check with the given name is registered.
func RequireStartupCheckRegistered(tb testing.TB, m *Manager, name string) {
	tb.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.startupChecks {
		if c.Name == name {
			return
		}
	}
	tb.Fatalf("RequireStartupCheckRegistered: check %q not found", name)
}
