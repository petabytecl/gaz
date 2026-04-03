package health

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManagementServer_StartStop(t *testing.T) {
	// Setup with port 0 for random available port
	config := Config{
		Port:          0,
		LivenessPath:  "/live",
		ReadinessPath: "/ready",
		StartupPath:   "/startup",
	}
	manager := NewManager()
	shutdownCheck := NewShutdownCheck()

	server := NewManagementServer(config, manager, shutdownCheck, nil)

	// Start
	ctx := context.Background()
	err := server.OnStart(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		require.NoError(t, server.OnStop(stopCtx))
	})

	// Use the actual bound port
	port := server.Port()
	require.NotZero(t, port)

	// Verify liveness endpoint is reachable
	url := fmt.Sprintf("http://localhost:%d/live", port)
	require.Eventually(t, func() bool {
		req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if reqErr != nil {
			return false
		}
		resp, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			return false
		}
		_ = resp.Body.Close()

		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 50*time.Millisecond)
}
