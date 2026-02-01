package cronx

import (
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestWithLocation(t *testing.T) {
	c := New(WithLocation(time.UTC))
	if c.location != time.UTC {
		t.Errorf("expected UTC, got %v", c.location)
	}
}

func TestWithParser(t *testing.T) {
	parser := NewParser(Dow)
	c := New(WithParser(parser))
	if c.parser != parser {
		t.Error("expected provided parser")
	}
}

func TestWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := New(WithLogger(logger))

	_, err := c.AddFunc("@every 1s", func() {})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c.Start()
	time.Sleep(OneSecond)
	ctx := c.Stop()
	if ctx == nil {
		t.Error("expected non-nil context")
	}
}

func TestWithChain(t *testing.T) {
	var called bool
	wrapper := func(j Job) Job {
		return FuncJob(func() {
			called = true
			j.Run()
		})
	}

	c := New(WithParser(secondParser), WithChain(wrapper))
	c.AddFunc("* * * * * ?", func() {})
	c.Start()
	time.Sleep(OneSecond)
	c.Stop()

	if !called {
		t.Error("expected wrapper to be called")
	}
}

func TestWithSeconds(t *testing.T) {
	c := New(WithSeconds())

	// 6-field spec should work with seconds enabled
	_, err := c.AddFunc("* * * * * ?", func() {})
	if err != nil {
		t.Errorf("expected 6-field spec to work with WithSeconds, got: %v", err)
	}
}
