package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/health"
)

func TestHealthIntegration(t *testing.T) {
	// Configure app
	cfg := health.DefaultConfig()
	cfg.Port = 9093

	app := gaz.New(

		health.WithHealthChecks(cfg),
	)

	// Start app in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(ctx)
	}()

	// Wait for server to be ready
	url := fmt.Sprintf("http://localhost:%d/live", cfg.Port)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(2 * time.Second)

	ready := false
	for !ready {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for health server to start")
		case <-ticker.C:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					ready = true
				}
			}
		case err := <-errCh:
			if err != nil {
				t.Fatalf("App run failed: %v", err)
			}
		}
	}

	// Verify endpoints
	endpoints := []string{
		cfg.LivenessPath,
		cfg.ReadinessPath,
		cfg.StartupPath,
	}

	for _, path := range endpoints {
		fullURL := fmt.Sprintf("http://localhost:%d%s", cfg.Port, path)
		resp, err := http.Get(fullURL)
		if err != nil {
			t.Errorf("GET %s failed: %v", fullURL, err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("GET %s status: got %d, want 200", fullURL, resp.StatusCode)
		}
	}

	// Stop app
	cancel()

	// Wait for Run to return
	select {
	case <-errCh:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for app to stop")
	}
}
