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

	// Verify endpoints
	endpoints := []string{
		cfg.Health.LivenessPath,
		cfg.Health.ReadinessPath,
		cfg.Health.StartupPath,
	}

	for _, path := range endpoints {
		fullURL := fmt.Sprintf("http://localhost:%d%s", port, path)
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

			return resp.StatusCode == http.StatusOK
		}, 2*time.Second, 50*time.Millisecond, "endpoint %s not reachable", fullURL)
	}
}
