package health

import (
	"context"
	"testing"
)

func TestShutdownCheck(t *testing.T) {
	c := NewShutdownCheck()
	ctx := context.Background()

	if err := c.Check(ctx); err != nil {
		t.Fatalf("expected nil error initially, got %v", err)
	}

	c.MarkShuttingDown()

	if err := c.Check(ctx); err == nil {
		t.Fatal("expected error after shutdown, got nil")
	}
}
