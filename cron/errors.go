package cron

import "errors"

// Sentinel errors for cron package.
var (
	// ErrNotRunning indicates an operation was attempted on a scheduler
	// that is not running.
	ErrNotRunning = errors.New("cron: scheduler not running")
)
