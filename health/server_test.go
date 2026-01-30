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
	shutdownCheck := NewShutdownCheck()

	server := NewManagementServer(config, manager, shutdownCheck)

	// Start
	ctx := context.Background()
	if err := server.OnStart(ctx); err != nil {
		t.Fatalf("OnStart failed: %v", err)
	}

	// Verify it's running
	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	url := fmt.Sprintf("http://localhost:%d/live", config.Port)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
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
	if stopErr := server.OnStop(stopCtx); stopErr != nil {
		t.Errorf("OnStop failed: %v", stopErr)
	}
}
