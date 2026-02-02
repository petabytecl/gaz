package health

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/health/internal"
)

func TestTestConfig(t *testing.T) {
	cfg := TestConfig()

	assert.Equal(t, 0, cfg.Port, "Port should be 0 for random assignment")
	assert.Equal(t, "/live", cfg.LivenessPath)
	assert.Equal(t, "/ready", cfg.ReadinessPath)
	assert.Equal(t, "/startup", cfg.StartupPath)
}

func TestNewTestConfig(t *testing.T) {
	cfg := NewTestConfig(func(c *Config) {
		c.Port = 8080
		c.LivenessPath = "/healthz"
	})

	assert.Equal(t, 8080, cfg.Port, "Port should be overridden")
	assert.Equal(t, "/healthz", cfg.LivenessPath, "LivenessPath should be overridden")
	assert.Equal(t, "/ready", cfg.ReadinessPath, "ReadinessPath should keep default")
}

func TestMockRegistrar(t *testing.T) {
	m := NewMockRegistrar()

	// Should accept all check registrations without error
	m.AddLivenessCheck("db", func(ctx context.Context) error { return nil })
	m.AddReadinessCheck("cache", func(ctx context.Context) error { return nil })
	m.AddStartupCheck("migrations", func(ctx context.Context) error { return nil })

	// Verify all calls were made
	m.AssertCalled(t, "AddLivenessCheck", "db", mock.Anything)
	m.AssertCalled(t, "AddReadinessCheck", "cache", mock.Anything)
	m.AssertCalled(t, "AddStartupCheck", "migrations", mock.Anything)
}

func TestTestManager(t *testing.T) {
	m := TestManager()
	require.NotNil(t, m)

	// Should start with no checks
	checker := m.ReadinessChecker()
	result := checker.Check(context.Background())

	// Empty checker returns StatusUp (matches alexliesenfeld/health behavior)
	assert.Equal(t, internal.StatusUp, result.Status)
}

func TestRequireHealthy(t *testing.T) {
	m := TestManager()

	// No checks = healthy (matches alexliesenfeld/health behavior)
	RequireHealthy(t, m)

	// Add a passing check - still healthy
	m.AddReadinessCheck("ok", func(ctx context.Context) error { return nil })
	RequireHealthy(t, m)
}

func TestRequireUnhealthy(t *testing.T) {
	m := TestManager()
	m.AddReadinessCheck("failing", func(ctx context.Context) error {
		return errors.New("service unavailable")
	})

	RequireUnhealthy(t, m)
}

func TestRequireCheckRegistered(t *testing.T) {
	m := TestManager()

	m.AddLivenessCheck("db-live", func(ctx context.Context) error { return nil })
	m.AddReadinessCheck("db-ready", func(ctx context.Context) error { return nil })
	m.AddStartupCheck("db-startup", func(ctx context.Context) error { return nil })

	RequireLivenessCheckRegistered(t, m, "db-live")
	RequireReadinessCheckRegistered(t, m, "db-ready")
	RequireStartupCheckRegistered(t, m, "db-startup")
}
