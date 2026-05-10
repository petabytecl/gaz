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
		BindAddress:   "127.0.0.1",
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
	url := fmt.Sprintf("http://127.0.0.1:%d/live", port)
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

func TestManagementServer_Timeouts(t *testing.T) {
	config := DefaultConfig()
	config.Port = 0
	manager := NewManager()
	shutdownCheck := NewShutdownCheck()

	server := NewManagementServer(config, manager, shutdownCheck, nil)

	require.Equal(t, 10*time.Second, server.server.ReadTimeout, "ReadTimeout should be 10s")
	require.Equal(t, 10*time.Second, server.server.WriteTimeout, "WriteTimeout should be 10s")
	require.Equal(t, 60*time.Second, server.server.IdleTimeout, "IdleTimeout should be 60s")
}

func TestManagementServer_BindAddress(t *testing.T) {
	config := DefaultConfig()
	config.Port = 9090

	require.Equal(t, "127.0.0.1", config.BindAddress, "Default BindAddress should be 127.0.0.1")

	manager := NewManager()
	shutdownCheck := NewShutdownCheck()
	server := NewManagementServer(config, manager, shutdownCheck, nil)

	require.Equal(t, "127.0.0.1:9090", server.server.Addr)
}

func TestManagementServer_ShowErrorsDefault(t *testing.T) {
	config := DefaultConfig()
	require.False(t, config.ShowErrors, "ShowErrors should default to false")
}
