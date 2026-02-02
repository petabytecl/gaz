package disk_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/petabytecl/gaz/health/checks/disk"
)

func TestNew_EmptyPath(t *testing.T) {
	check := disk.New(disk.Config{
		ThresholdPercent: 90,
	})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if got := err.Error(); got != "disk: path is empty" {
		t.Errorf("got %q, want %q", got, "disk: path is empty")
	}
}

func TestNew_InvalidThreshold_Zero(t *testing.T) {
	check := disk.New(disk.Config{
		Path:             "/",
		ThresholdPercent: 0,
	})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for zero threshold")
	}
	if got := err.Error(); got != "disk: threshold must be between 0 and 100, got 0.0" {
		t.Errorf("got %q, want threshold error", got)
	}
}

func TestNew_InvalidThreshold_Negative(t *testing.T) {
	check := disk.New(disk.Config{
		Path:             "/",
		ThresholdPercent: -10,
	})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for negative threshold")
	}
}

func TestNew_InvalidThreshold_TooHigh(t *testing.T) {
	check := disk.New(disk.Config{
		Path:             "/",
		ThresholdPercent: 150,
	})
	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for threshold > 100")
	}
}

func TestNew_SuccessWithHighThreshold(t *testing.T) {
	// Use 99.9% threshold - will pass unless disk is literally full
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:"
	}

	check := disk.New(disk.Config{
		Path:             path,
		ThresholdPercent: 99.9,
	})

	err := check(context.Background())
	if err != nil {
		t.Errorf("expected success with 99.9%% threshold, got error: %v", err)
	}
}

func TestNew_FailWithLowThreshold(t *testing.T) {
	// Use 0.1% threshold - will fail unless disk is almost empty
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:"
	}

	check := disk.New(disk.Config{
		Path:             path,
		ThresholdPercent: 0.1,
	})

	err := check(context.Background())
	if err == nil {
		t.Fatal("expected failure with 0.1% threshold")
	}
	// Should contain "exceeds threshold" in the error message
	if got := err.Error(); len(got) == 0 {
		t.Error("expected non-empty error message")
	}
}

func TestNew_InvalidPath(t *testing.T) {
	check := disk.New(disk.Config{
		Path:             "/nonexistent/path/that/does/not/exist",
		ThresholdPercent: 90,
	})

	err := check(context.Background())
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestNew_ContextCancellation(t *testing.T) {
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:"
	}

	check := disk.New(disk.Config{
		Path:             path,
		ThresholdPercent: 99.9,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Note: gopsutil may or may not respect cancelled context immediately
	// This test just ensures no panic
	_ = check(ctx)
}
