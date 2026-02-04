package main

import (
	"context"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Uses default port 9090 (--health-port flag available via CLI)
	if err := run(ctx); err != nil {
		t.Fatalf("run() failed: %v", err)
	}
}
