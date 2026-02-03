package main

import (
	"context"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cfg := DefaultAppConfig()
	cfg.Server.Port = 0 // Let OS choose port
	cfg.Health.Port = 0 // Let OS choose port

	if err := run(ctx, cfg); err != nil {
		t.Fatalf("run() failed: %v", err)
	}
}
