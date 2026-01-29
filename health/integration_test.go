package health_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/petabytecl/gaz"
	"github.com/petabytecl/gaz/health"
)

func TestWithHealthChecks(t *testing.T) {
	// Create custom config
	cfg := health.Config{
		Port:          18082, // Use non-standard port for test isolation
		LivenessPath:  "/healthz",
		ReadinessPath: "/ready",
		StartupPath:   "/startup",
	}

	// Create app with WithHealthChecks option
	app := gaz.New(health.WithHealthChecks(cfg))

	// Verify health.Config is registered
	c := app.Container()
	resolvedCfg, err := gaz.Resolve[health.Config](c)
	if err != nil {
		t.Fatalf("Config not resolved: %v", err)
	}
	if resolvedCfg.Port != cfg.Port {
		t.Errorf("Config port mismatch: got %d, want %d", resolvedCfg.Port, cfg.Port)
	}
	if resolvedCfg.LivenessPath != cfg.LivenessPath {
		t.Errorf("Config LivenessPath mismatch: got %s, want %s", resolvedCfg.LivenessPath, cfg.LivenessPath)
	}

	// Build and run app to start management server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Make HTTP request to health endpoint
	healthURL := fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.LivenessPath)
	resp, err := http.Get(healthURL)
	if err != nil {
		t.Fatalf("Failed to GET %s: %v", healthURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test ready endpoint
	readyURL := fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.ReadinessPath)
	resp, err = http.Get(readyURL)
	if err != nil {
		t.Fatalf("Failed to GET %s: %v", readyURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for ready, got %d", resp.StatusCode)
	}

	// Stop the app
	cancel()

	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("App did not stop in time")
	}
}

func TestWithHealthChecks_DefaultConfig(t *testing.T) {
	// Use default config
	cfg := health.DefaultConfig()

	// Create app with default config
	app := gaz.New(health.WithHealthChecks(cfg))

	// Verify config values are default
	c := app.Container()
	resolvedCfg, err := gaz.Resolve[health.Config](c)
	if err != nil {
		t.Fatalf("Config not resolved: %v", err)
	}

	if resolvedCfg.Port != health.DefaultPort {
		t.Errorf("Expected default port %d, got %d", health.DefaultPort, resolvedCfg.Port)
	}
	if resolvedCfg.LivenessPath != "/live" {
		t.Errorf("Expected default liveness path /live, got %s", resolvedCfg.LivenessPath)
	}
	if resolvedCfg.ReadinessPath != "/ready" {
		t.Errorf("Expected default readiness path /ready, got %s", resolvedCfg.ReadinessPath)
	}

	// Verify Module is registered (components are available)
	if err := app.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Verify health components are resolvable
	if _, err := gaz.Resolve[*health.Manager](c); err != nil {
		t.Errorf("Manager not resolved: %v", err)
	}
	if _, err := gaz.Resolve[*health.ShutdownCheck](c); err != nil {
		t.Errorf("ShutdownCheck not resolved: %v", err)
	}
	if _, err := gaz.Resolve[*health.ManagementServer](c); err != nil {
		t.Errorf("ManagementServer not resolved: %v", err)
	}
}
