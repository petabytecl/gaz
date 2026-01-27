package health

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestManagementServer_StartStop(t *testing.T) {
	// Setup
	config := Config{
		Port:          9091, // Use different port to avoid conflict
		LivenessPath:  "/live",
		ReadinessPath: "/ready",
		StartupPath:   "/startup",
	}
	manager := NewManager() // Assuming default manager works

	server := NewManagementServer(config, manager)

	// Start
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify it's running
	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/live", config.Port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to request liveness: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := server.Stop(stopCtx); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}
