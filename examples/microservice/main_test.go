package main

import (
	"context"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Use port 0 for random port to avoid conflicts
	if err := run(ctx, 0); err != nil {
		t.Fatalf("run() failed: %v", err)
	}
}
