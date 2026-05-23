package gaz

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz/health"
)

// healthTestConfig implements health.HealthConfigProvider for testing auto-registration.
type healthTestConfig struct {
	Health health.Config
}

func (c *healthTestConfig) HealthConfig() health.Config {
	return c.Health
}

// nonHealthTestConfig does NOT implement HealthConfigProvider.
type nonHealthTestConfig struct {
	Name string
}

func TestAutoRegisterHealth_WithProvider(t *testing.T) {
	cfg := &healthTestConfig{Health: health.DefaultConfig()}
	cfg.Health.Port = 0

	app := New()
	app.configTarget = cfg

	err := app.autoRegisterHealth()
	require.NoError(t, err)
	require.True(t, app.modules[health.ModuleName])

	require.NoError(t, app.container.Build())

	_, err = Resolve[health.Config](app.container)
	require.NoError(t, err)

	_, err = Resolve[*health.Manager](app.container)
	require.NoError(t, err)

	_, err = Resolve[*health.ShutdownCheck](app.container)
	require.NoError(t, err)

	_, err = Resolve[*health.ManagementServer](app.container)
	require.NoError(t, err)
}

func TestAutoRegisterHealth_WithoutProvider(t *testing.T) {
	cfg := &nonHealthTestConfig{Name: "test"}

	app := New()
	app.configTarget = cfg

	err := app.autoRegisterHealth()
	require.NoError(t, err)
	require.False(t, app.modules[health.ModuleName])
}

func TestAutoRegisterHealth_NilConfigTarget(t *testing.T) {
	app := New()

	err := app.autoRegisterHealth()
	require.NoError(t, err)
	require.False(t, app.modules[health.ModuleName])
}

func TestAutoRegisterHealth_ExplicitModuleSkipsAutoRegistration(t *testing.T) {
	cfg := &healthTestConfig{Health: health.DefaultConfig()}
	cfg.Health.Port = 0

	app := New()
	app.configTarget = cfg

	// Simulate an explicit health module already applied.
	app.modules[health.ModuleName] = true

	err := app.autoRegisterHealth()
	require.NoError(t, err)

	// Config should NOT be registered in container (auto-registration was skipped).
	_, err = Resolve[health.Config](app.container)
	require.Error(t, err)
}

func TestAutoRegisterHealth_PropagatesConfigRegistrationError(t *testing.T) {
	cfg := &healthTestConfig{Health: health.DefaultConfig()}

	app := New()
	app.configTarget = cfg

	// Force container into built state so Instance registration fails
	// with ErrAlreadyBuilt — verifying error context is preserved.
	require.NoError(t, app.container.Build())

	err := app.autoRegisterHealth()
	require.Error(t, err)
	require.Contains(t, err.Error(), "auto-register health config")
}
