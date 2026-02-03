package main

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	buf := new(bytes.Buffer)
	// Pass buffer to capture output
	if err := run(ctx, 9090, buf); err != nil {
		t.Fatalf("run() failed: %v", err)
	}

	output := buf.String()
	assert.Contains(t, output, "Server starting on port 9090")
	// OnStop is called during shutdown
	assert.Contains(t, output, "Server stopping...")
}

func TestServerLifecycle(t *testing.T) {
	buf := new(bytes.Buffer)
	server := &Server{port: 8080, out: buf}

	// Test OnStart
	err := server.OnStart(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Server starting on port 8080")
	buf.Reset()

	// Test OnStop
	err = server.OnStop(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Server stopping...")
}
