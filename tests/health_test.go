package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/health"
	healthmod "github.com/petabytecl/gaz/health/module"
)

// testConfig implements health.HealthConfigProvider for auto-registration.
type testConfig struct {
	Health health.Config
}

// HealthConfig returns the health configuration.
func (c *testConfig) HealthConfig() health.Config {
	return c.Health
}

func TestHealthIntegration(t *testing.T) {
	// Configure app with HealthConfigProvider using port 0 for random available port
	cfg := &testConfig{
		Health: health.DefaultConfig(),
	}
	cfg.Health.Port = 0

	app := gaz.New()
	app.WithConfig(cfg)

	// Build and start (instead of Run) so we can resolve the server's actual port
	ctx := context.Background()

	err := app.Start(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, app.Stop(stopCtx))
	})

	// Resolve the ManagementServer to get the actual bound port
	mgmtServer, err := gaz.Resolve[*health.ManagementServer](app.Container())
	require.NoError(t, err)

	port := mgmtServer.Port()
	require.NotZero(t, port)

	// Verify endpoints are reachable and return expected status codes.
	// Liveness/readiness return 200 (readiness has auto-registered checks).
	// Startup returns 503 when no startup checks registered (StatusUnknown).
	expectedStatus := map[string]int{
		cfg.Health.LivenessPath:  http.StatusOK,
		cfg.Health.ReadinessPath: http.StatusOK,
		cfg.Health.StartupPath:   http.StatusServiceUnavailable,
	}

	for path, wantCode := range expectedStatus {
		fullURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
		require.Eventually(t, func() bool {
			req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, fullURL, nil)
			if reqErr != nil {
				return false
			}
			resp, doErr := http.DefaultClient.Do(req)
			if doErr != nil {
				return false
			}
			_ = resp.Body.Close()

			return resp.StatusCode == wantCode
		}, 2*time.Second, 50*time.Millisecond, "endpoint %s expected %d", fullURL, wantCode)
	}
}

func TestHealthExplicitModulePlusConfigProviderDoesNotDoubleApply(t *testing.T) {
	cfg := &testConfig{
		Health: health.DefaultConfig(),
	}

	app := gaz.New()
	app.Use(healthmod.New())
	app.WithConfig(cfg)

	// Build (not Start) is sufficient — we only need to verify the module
	// was applied exactly once without duplicate errors.
	err := app.Build()
	require.NoError(t, err)

	_, err = gaz.Resolve[*health.ManagementServer](app.Container())
	require.NoError(t, err)

	_, err = gaz.Resolve[*health.Manager](app.Container())
	require.NoError(t, err)

	_, err = gaz.Resolve[*health.ShutdownCheck](app.Container())
	require.NoError(t, err)
}
