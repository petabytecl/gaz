package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	if err := execute(context.Background(), []string{"version"}, buf); err != nil {
		t.Fatalf("execute(version) failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "myapp v1.0.0") {
		t.Errorf("expected version output, got: %q", output)
	}
}

func TestServerLifecycle(t *testing.T) {
	config := AppConfig{
		Debug:   true,
		Port:    8080,
		Host:    "localhost",
		Timeout: 30,
	}
	buf := new(bytes.Buffer)
	server := NewServer(config, buf)
	require.NotNil(t, server)

	// Test OnStart
	err := server.OnStart(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Initializing server on localhost:8080...")
	buf.Reset()

	// Test Start with cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	startErr := server.Start(ctx)
	assert.NoError(t, startErr)
	output := buf.String()
	assert.Contains(t, output, "Server starting on localhost:8080")
	assert.Contains(t, output, "Debug mode: true")
	assert.Contains(t, output, "Server shutting down...")
	buf.Reset()

	// Test OnStop
	err = server.OnStop(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Cleaning up server resources...")
}

func TestExecuteServe(t *testing.T) {
	buf := new(bytes.Buffer)

	// Use a context that cancels quickly to stop the server
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Pass arguments to override defaults and verify they are used
	args := []string{"serve", "--port", "9090", "--debug"}

	err := execute(ctx, args, buf)
	assert.NoError(t, err)

	output := buf.String()
	// Check if server started with correct config
	assert.Contains(t, output, "Server starting on localhost:9090")
	assert.Contains(t, output, "Debug mode: true")
	// The shutdown message might be printed after context cancellation
	// Since execute blocks until server stops, output should contain it.
	assert.Contains(t, output, "Server shutting down...")
}
