package internal

import (
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

func TestWithLocation(t *testing.T) {
	t.Parallel()
	c := New(WithLocation(time.UTC))
	if c.location != time.UTC {
		t.Errorf("expected UTC, got %v", c.location)
	}
}

func TestWithParser(t *testing.T) {
	t.Parallel()
	parser := NewParser(Dow)
	c := New(WithParser(parser))
	if c.parser != parser {
		t.Error("expected provided parser")
	}
}

func TestWithLogger(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := New(WithLogger(logger))

	ran := make(chan struct{}, 1)
	_, err := c.AddFunc("@every 1s", func() {
		select {
		case ran <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c.Start()
	// Wait for the job to fire at least once
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not run")
	}
	ctx := c.Stop()
	if ctx == nil {
		t.Error("expected non-nil context")
	}
}

func TestWithChain(t *testing.T) {
	t.Parallel()
	var called atomic.Bool
	wrapper := func(j Job) Job {
		return FuncJob(func() {
			called.Store(true)
			j.Run()
		})
	}

	c := New(WithParser(secondParser), WithChain(wrapper))
	ran := make(chan struct{}, 1)
	_, _ = c.AddFunc("* * * * * ?", func() {
		select {
		case ran <- struct{}{}:
		default:
		}
	})
	c.Start()
	// Wait for the job to fire at least once
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not run")
	}
	c.Stop()

	if !called.Load() {
		t.Error("expected wrapper to be called")
	}
}

func TestWithSeconds(t *testing.T) {
	t.Parallel()
	c := New(WithSeconds())

	// 6-field spec should work with seconds enabled
	_, err := c.AddFunc("* * * * * ?", func() {})
	if err != nil {
		t.Errorf("expected 6-field spec to work with WithSeconds, got: %v", err)
	}
}
