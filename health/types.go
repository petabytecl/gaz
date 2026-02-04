package health

import (
	"context"
	"time"

	"github.com/petabytecl/gaz/health/internal"
)

// Checker is the interface for executing health checks.
type Checker = internal.Checker

// CheckerOption configures the Checker.
type CheckerOption = internal.CheckerOption

// CheckerResult holds the aggregated health status and details.
type CheckerResult = internal.CheckerResult

// CheckResult holds the result of a single check.
type CheckResult = internal.CheckResult

// AvailabilityStatus represents system/component availability.
type AvailabilityStatus = internal.AvailabilityStatus

const (
	// StatusUnknown means the status is not yet known.
	StatusUnknown = internal.StatusUnknown
	// StatusUp means the system/component is available.
	StatusUp = internal.StatusUp
	// StatusDown means the system/component is unavailable.
	StatusDown = internal.StatusDown
)

// CheckFunc is a function that performs a health check.
// It returns an error if the check fails.
type CheckFunc func(context.Context) error

// CheckOptions defines configuration for a specific check.
type CheckOptions struct {
	Name    string
	Timeout time.Duration
}

// Registrar allows services to register their health checks.
type Registrar interface {
	// AddLivenessCheck registers a check for liveness probes (is app running?).
	// Failures here may cause the orchestrator to restart the container.
	AddLivenessCheck(name string, check CheckFunc)

	// AddReadinessCheck registers a check for readiness probes (can app accept traffic?).
	// Failures here cause the orchestrator to stop sending traffic.
	AddReadinessCheck(name string, check CheckFunc)

	// AddStartupCheck registers a check for startup probes (is app initialized?).
	// Failures here hold off liveness/readiness checks.
	AddStartupCheck(name string, check CheckFunc)
}
