// Package disk provides a health check for disk space using gopsutil.
package disk

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/disk"
)

// Config configures the disk space health check.
type Config struct {
	// Path is the filesystem path to check. Required.
	// Use "/" for root filesystem on Unix, "C:" for Windows.
	Path string
	// ThresholdPercent is the maximum usage percentage allowed (0-100). Required.
	// Check fails if current usage exceeds this threshold.
	ThresholdPercent float64
}

// New creates a new disk space health check.
// Uses gopsutil for cross-platform disk usage reporting.
//
// Returns nil if usage is under threshold, error if exceeded.
func New(cfg Config) func(context.Context) error {
	return func(ctx context.Context) error {
		if cfg.Path == "" {
			return fmt.Errorf("disk: path is empty")
		}
		if cfg.ThresholdPercent <= 0 || cfg.ThresholdPercent > 100 {
			return fmt.Errorf("disk: threshold must be between 0 and 100, got %.1f", cfg.ThresholdPercent)
		}

		usage, err := disk.UsageWithContext(ctx, cfg.Path)
		if err != nil {
			return fmt.Errorf("disk: failed to get usage: %w", err)
		}
		if usage.UsedPercent > cfg.ThresholdPercent {
			return fmt.Errorf("disk: usage %.1f%% exceeds threshold %.1f%%",
				usage.UsedPercent, cfg.ThresholdPercent)
		}
		return nil
	}
}
