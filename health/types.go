package health

import (
	"context"
	"time"
)

// CheckFunc is a function that performs a health check.
// It returns an error if the check fails.
type CheckFunc func(context.Context) error

// CheckOptions defines configuration for a specific check.
type CheckOptions struct {
	Name    string
	Timeout time.Duration
}

// HealthRegistrar allows services to register their health checks.
type HealthRegistrar interface {
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
