package main

import (
	"context"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := run(ctx); err != nil {
		t.Fatalf("run() failed: %v", err)
	}
}
