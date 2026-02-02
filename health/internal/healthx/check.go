package healthx

import (
	"context"
	"time"
)

// Check configures a health check.
type Check struct {
	// Name must be unique among all checks. Required.
	Name string

	// Check is the function that performs the health check.
	// Must return nil if healthy, error if unhealthy.
	// The context includes a timeout - implementations should respect ctx.Done().
	Check func(ctx context.Context) error

	// Timeout overrides the default timeout for this check.
	// Zero means use default (5s).
	Timeout time.Duration

	// Critical determines if this check affects overall status.
	// When true (or unset), a failing check causes StatusDown for the overall result.
	// When false, the check is a "warning" that reports independently without
	// affecting the aggregated status.
	//
	// Note: The default behavior treats unset (false) as critical for safety.
	// To mark a check as non-critical/warning, explicitly pass through WithNonCritical().
	Critical bool

	// criticalSet indicates whether Critical was explicitly set.
	// This allows us to distinguish between "not set" (default=critical) and "set to false".
	criticalSet bool
}

// CheckResult holds a single check's result.
type CheckResult struct {
	// Status is the check's availability status.
	Status AvailabilityStatus
	// Timestamp is when the check was executed.
	Timestamp time.Time
	// Error is the check error (nil if healthy).
	Error error
}
